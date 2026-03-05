package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"synapsePlatform/internal"
	"synapsePlatform/internal/ingestor"
	synnapLog "synapsePlatform/internal/log"
	"synapsePlatform/internal/sqllite"
	"syscall"
)

type application struct {
	db       *sqllite.Repo
	consumer ingestor.MessagePoller
}

func main() {
	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure Kafka msgPoller
	kafkaConfig := internal.StreamingConfigs{
		Brokers:  []string{"localhost:9092"},
		GroupID:  "synapse-platform-msgPoller",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	}

	// create logger
	baseHandler := slog.NewJSONHandler(os.Stdout, nil)
	safeHandler := synnapLog.NewRedactingHandler(baseHandler, synnapLog.Options{
		RedactKeys:    []string{"token", "password", "secret", "authorization"},
		MaxValueBytes: 512,
	})
	logger := slog.New(safeHandler)

	var (
		processor   ingestor.DataProcessor
		msgPoller   ingestor.MessagePoller
		transformer ingestor.Transformer
	)

	// Create msgPoller
	msgPoller = internal.NewConsumer(kafkaConfig)
	msgPoller = synnapLog.NewMessagePoller(logger, msgPoller)
	// Subscribe to the ingestion topic
	if err := msgPoller.Subscribe("ingestion.raw"); err != nil {
		log.Fatalf("Failed to subscribe to topics: %v", err)
	}

	// create processors for logging and process message
	processor = ingestor.NewProcessor(msgPoller)
	processor = synnapLog.NewIngestorProcessor(logger, processor)

	// create storer for logging and storing message
	// NewRepo database
	db, err := sqllite.NewRepo("data.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func(Db *sql.DB) {
		err := Db.Close()
		if err != nil {
			log.Fatalf("Failed to close DB because: %v", err)
		}
	}(db.Db)

	storer := synnapLog.NewMessageStorer(logger, db)

	domains := ingestor.AllDataTypes()

	// create transformers for logging and transform message
	transformer = ingestor.NewMessageTransformer(domains)
	transformer = synnapLog.NewEventTransformer(logger, transformer)

	// Create ingestor and start ingesting message
	err = ingestor.New(ingestor.Config{CompatibleDataTypes: domains}, processor, storer, transformer).
		Ingest(ctx)
	if err != nil {
		logger.Error("Ingest failed and exit", "error", err)
	}

	//onShutdown(func() {
	//	cancel()
	//	//_ = hz.Stop(ctx)
	//	time.Sleep(5 * time.Second)
	//}, logger)

}

func onShutdown(fn func(), log *slog.Logger) {
	ch := make(chan os.Signal, 1)

	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s := <-ch

		log.Info("got a exit signal", "signal", s.String())

		fn()
	}()
}
