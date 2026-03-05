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

func (il *IngestorProcessor) ProcessData(ctx context.Context) (*ingestor.DeviceMessage, error) {
	msg, err := il.processor.ProcessData(ctx)
	if err != nil {
		il.logger.Error("failed to process message", "msg", msg, "error", err)

		return nil, err
	}

	if msg == nil {
		il.logger.Warn("msg received from processing is empty", "msg", msg)

		return msg, nil
	}

	il.logger.Info("message processed", "device_id", msg.DeviceID, "type", msg.Type, "message", msg)

	return msg, nil
}
