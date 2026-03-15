package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"synapsePlatform/internal"
	"synapsePlatform/internal/api"
	"synapsePlatform/internal/auth"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/kafka"
	synnapLog "synapsePlatform/internal/log"
	"synapsePlatform/internal/sqllite"
	"syscall"

	"golang.org/x/sync/errgroup"
)

func main() {
	cfg, err := internal.LoadConfig("config.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	baseHandler := slog.NewJSONHandler(os.Stdout, nil)
	safeHandler := synnapLog.NewRedactingHandler(baseHandler, synnapLog.Options{
		RedactKeys:    cfg.Log.RedactKeys,
		MaxValueBytes: cfg.Log.MaxValueBytes,
	})
	logger := slog.New(safeHandler)

	kafkaConfig := kafka.StreamingConfigs{
		Brokers:  cfg.Kafka.Brokers,
		GroupID:  cfg.Kafka.GroupID,
		MinBytes: cfg.Kafka.MinBytes,
		MaxBytes: cfg.Kafka.MaxBytes,
	}

	db, err := sqllite.NewRepo(cfg.Database.Path)
	if err != nil {
		logger.Error("Failed to open database", "error", err)

		os.Exit(1)
	}
	defer func(Db *sql.DB) {
		if err := Db.Close(); err != nil {
			logger.Error("Failed to close DB", "error", err)
		}
	}(db.Db)

	storer := synnapLog.NewMessageStorer(logger, db)

	domains := ingestor.AllDataTypes()

	transformer := synnapLog.NewEventTransformer(logger, ingestor.NewMessageTransformer(domains))

	var dbFailures ingestor.FailureStorer
	dbFailures = sqllite.NewFailureStorer(db)
	dbFailures = synnapLog.NewFailureStorer(logger.With("failures", "db"), dbFailures)

	var kafkaFailures ingestor.FailureStorer
	kafkaFailures = kafka.NewKafkaDLQ(cfg.Kafka.Brokers, cfg.Kafka.DLQTopics)
	kafkaFailures = synnapLog.NewFailureStorer(logger.With("failures", "kafka"), kafkaFailures)

	failures := ingestor.NewFallbackFailureStorer(dbFailures, kafkaFailures)

	authenticator, err := auth.NewJWTValidator(
		[]byte(cfg.Auth.JWT.Secret),
		cfg.Auth.JWT.Issuer,
		cfg.Auth.JWT.Audience,
	)
	if err != nil {
		logger.Error("Failed to create JWT validator", "error", err)
		os.Exit(1)
	}

	eventReader := synnapLog.NewEventReader(logger, db)

	apiServer := api.NewServer(
		cfg.Server,
		eventReader,
		authenticator,
		synnapLog.NewHTTPHandlerLogger(logger),
	)

	logger.Info("system starting",
		"topics", cfg.Kafka.Topics,
		"brokers", cfg.Kafka.Brokers,
		"db", cfg.Database.Path,
		"server", cfg.Server.Address,
	)

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		ch := make(chan os.Signal, 1)

		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-ch:
			logger.Info("received signal", "signal", sig.String())

			return fmt.Errorf("received signal: %s", sig)
		case <-ctx.Done():
			return nil
		}
	})

	for _, topic := range cfg.Kafka.Topics {
		run := newIngestionPipeline(logger, kafkaConfig, topic, storer, transformer, failures, domains)

		g.Go(func() error { return run(ctx) })
	}

	g.Go(func() error {
		return apiServer.Start()
	})

	g.Go(func() error {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.Shutdown.Timeout)

		defer cancel()

		return apiServer.Shutdown(shutdownCtx)
	})

	waitErr := g.Wait()
	if waitErr != nil {
		logger.Error("shutdown", "reason", waitErr)
	} else {
		logger.Info("system stopped gracefully")
	}
}
