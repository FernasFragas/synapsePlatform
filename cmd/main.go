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
	"synapsePlatform/internal/health"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/kafka"
	synnapLog "synapsePlatform/internal/log"
	"synapsePlatform/internal/metrics"
	"synapsePlatform/internal/sqllite"
	"syscall"
	"time"

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

	providers, err := metrics.NewProviders(context.Background(), "synapse-platform", cfg.Tracing.Endpoint)
	if err != nil {
		logger.Error("Failed to create observability providers", "error", err)
		os.Exit(1)
	}
	defer providers.Shutdown(context.Background())

	meter := providers.Meter.Meter("synapse-platform")
	tracer := providers.Tracer.Tracer("synapse-platform")

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
	metricsStore, err := metrics.NewMessageStorer(meter, tracer, storer)
	if err != nil {
		logger.Error("failed to build metric storer", "error", err)

		os.Exit(1)
	}

	domains := ingestor.AllDataTypes()
	transformer := synnapLog.NewEventTransformer(logger, ingestor.NewMessageTransformer(domains))
	metricsTransformer, err := metrics.NewEventTransformer(meter, tracer, transformer)
	if err != nil {
		logger.Error("failed to build metrics transformer", "error", err)

		os.Exit(1)
	}

	var kafkaFailures ingestor.FailureStorer
	kafkaFailures = kafka.NewKafkaDLQ(cfg.Kafka.Brokers, cfg.Kafka.DLQTopics)
	kafkaFailures = synnapLog.NewFailurePublisher(logger.With("failures", "kafka"), kafkaFailures)

	failures := ingestor.NewFallbackFailureStorer(db, kafkaFailures)

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
	metricsEventReader, err := metrics.NewAPI(meter, tracer, eventReader)
	if err != nil {
		logger.Error("failed to build metrics event reader", "error", err)

		os.Exit(1)
	}

	// Health probes — db and kafka are the same objects used for business logic.
	// No separate wrapper: db *is* a Probe, each consumer *is* a Probe.
	healthLogger := logger.With("component", "health")
	var dbProbe health.Probe = db
	dbProbe = synnapLog.NewHealthProbe(healthLogger, dbProbe)
	// Collect kafka consumer probes as we build pipelines
	var kafkaProbes []health.Probe

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
		consumer := kafka.NewConsumer(kafkaConfig, topic, 2*time.Minute)
		var consumerProbe health.Probe = consumer

		consumerProbe = synnapLog.NewHealthProbe(healthLogger, consumerProbe)
		kafkaProbes = append(kafkaProbes, consumerProbe)

		run := newIngestionPipeline(logger, meter, tracer, consumer, metricsStore, metricsTransformer, failures, domains)

		g.Go(func() error { return run(ctx) })
	}

	// Build health checker with all probes
	allProbes := append([]health.Probe{dbProbe}, kafkaProbes...)
	checker := health.NewChecker(2*time.Second, allProbes...)

	apiServer := api.NewServer(
		cfg.Server,
		metricsEventReader,
		authenticator,
		synnapLog.NewHTTPHandlerLogger(logger),
		checker,
	)

	logger.Info("system starting",
		"topics", cfg.Kafka.Topics,
		"brokers", cfg.Kafka.Brokers,
		"db", cfg.Database.Path,
		"server", cfg.Server.Address,
	)

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
