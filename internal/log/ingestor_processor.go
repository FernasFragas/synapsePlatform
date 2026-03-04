//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/log/mocked_$GOFILE
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
		il.logger.Error("failed to transform message", "msg", msg, "error", err, "detailed err", err.Error())

		return nil, err
	}

	il.logger.Info("message processed", "device_id", msg.DeviceID, "type", msg.Type, "message", msg)

	return msg, err
}
