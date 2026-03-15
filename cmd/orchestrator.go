package main

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/kafka"
	synnapLog "synapsePlatform/internal/log"
)

func newIngestionPipeline(
	logger *slog.Logger,
	kafkaCfg kafka.StreamingConfigs,
	topic string,
	storer ingestor.MessageStorer,
	transformer ingestor.Transformer,
	failures ingestor.FailureStorer,
	domains []ingestor.DataTypes,
) func(ctx context.Context) error {
	topicLogger := logger.With("topic", topic)

	poller := kafka.NewConsumer(kafkaCfg, topic)
	loggedPoller := synnapLog.NewMessagePoller(topicLogger, poller)

	proc := ingestor.NewProcessor(loggedPoller)
	loggedProc := synnapLog.NewIngestorProcessor(topicLogger, proc)

	ing := ingestor.New(
		ingestor.Config{CompatibleDataTypes: domains},
		loggedProc, storer, transformer, failures,
	)

	return ing.Ingest
}
