package main

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/kafka"
	synnapLog "synapsePlatform/internal/log"
	"synapsePlatform/internal/metrics"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func newIngestionPipeline(
	logger *slog.Logger,
	meter metric.Meter,
	tracer trace.Tracer,
	consumer *kafka.KafkaConsumer,
	storer ingestor.MessageStorer,
	transformer ingestor.Transformer,
	failures ingestor.FailureStorer,
	domains []ingestor.DataTypes,
) func(ctx context.Context) error {
	topicLogger := logger.With("topic", consumer.Name())

	// Poller: consumer → log → metrics
	var poller ingestor.MessagePoller = synnapLog.NewMessagePoller(topicLogger, consumer)
	metricsPoller, err := metrics.NewMessagePoller(meter, tracer, poller)
	if err != nil {
		logger.Error("failed to build metrics poller", "error", err)
		return func(ctx context.Context) error { return err }
	}

	// Processor: poller → core → log → metrics
	proc := ingestor.NewProcessor(metricsPoller)
	var dataProc ingestor.DataProcessor = synnapLog.NewIngestorProcessor(topicLogger, proc)
	metricsProc, err := metrics.NewIngestorProcessor(meter, tracer, dataProc)
	if err != nil {
		logger.Error("failed to build metrics processor", "error", err)
		return func(ctx context.Context) error { return err }
	}

	ing := ingestor.New(
		ingestor.Config{CompatibleDataTypes: domains},
		metricsProc, storer, transformer, failures,
	)

	return ing.Ingest
}
