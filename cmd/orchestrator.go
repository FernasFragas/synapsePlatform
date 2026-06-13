package main

import (
	"context"
	"fmt"
	"log/slog"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/kafka"
	synnapLog "synapsePlatform/internal/log"
	"synapsePlatform/internal/metrics"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func newIngestionPipeline(
	logger *slog.Logger, meter metric.Meter, tracer trace.Tracer,
	consumer *kafka.KafkaConsumer, storer ingestor.MessageStorer,
	transformer ingestor.Transformer, failures ingestor.FailureStorer,
	domains []ingestor.DataTypes, batchSize int,
	batchTimeout time.Duration, workersNumb int,
) (func(ctx context.Context) error, error) {
	topicLogger := logger.With("topic", consumer.Name())

	var poller ingestor.MessagePoller = synnapLog.NewMessagePoller(topicLogger, consumer)
	metricsPoller, err := metrics.NewMessagePoller(meter, tracer, poller)
	if err != nil {
		return nil, fmt.Errorf("metrics poller: %w", err)
	}

	proc := ingestor.NewProcessor(metricsPoller)
	var dataProc ingestor.DataProcessor = synnapLog.NewIngestorProcessor(topicLogger, proc)
	metricsProc, err := metrics.NewIngestorProcessor(meter, tracer, dataProc)
	if err != nil {
		return nil, fmt.Errorf("metrics processor: %w", err)
	}

	ing := ingestor.New(ingestor.Config{
		CompatibleDataTypes: domains,
		BatchSize:           batchSize,
		BatchTimeout:        batchTimeout,
		NumWorkers:          workersNumb,
	}, metricsProc, storer, transformer, failures)

	return ing.Ingest, nil
}
