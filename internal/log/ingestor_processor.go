package log

import (
	"context"
	"log/slog"

	"synapsePlatform/internal/ingestor"
)

type IngestorProcessor struct {
	logger    *slog.Logger
	processor ingestor.DataProcessor
}

func NewIngestorProcessor(logger *slog.Logger, processor ingestor.DataProcessor) *IngestorProcessor {
	return &IngestorProcessor{
		logger:    logger,
		processor: processor,
	}
}

func (il *IngestorProcessor) ProcessData(ctx context.Context) (*ingestor.Delivery, error) {
	delivery, err := il.processor.ProcessData(ctx)
	if err != nil {
		il.logger.ErrorContext(ctx, "failed to process message",
			"delivery", deliveryLogValue(delivery),
			"error", err,
		)
		return delivery, err
	}

	if delivery == nil || delivery.Message == nil {
		il.logger.WarnContext(ctx, "processed delivery is empty")
		return delivery, nil
	}

	il.logger.InfoContext(ctx, "message processed",
		"device_id", delivery.Message.DeviceID,
		"type", delivery.Message.Type,
		"source", delivery.Metadata.Source,
	)

	return delivery, nil
}

func deliveryLogValue(delivery *ingestor.Delivery) slog.Value {
	if delivery == nil {
		return slog.StringValue("<nil>")
	}

	if delivery.Message == nil {
		return slog.GroupValue(
			slog.String("source", delivery.Metadata.Source),
			slog.Any("labels", delivery.Metadata.Labels),
		)
	}

	return slog.GroupValue(
		slog.String("device_id", delivery.Message.DeviceID),
		slog.String("type", delivery.Message.Type),
		slog.String("source", delivery.Metadata.Source),
		slog.Any("labels", delivery.Metadata.Labels),
	)
}
