package log

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/ingestor"
)

type EventTransformer struct {
	logger      *slog.Logger
	transformer ingestor.Transformer
}

func NewEventTransformer(logger *slog.Logger, transformer ingestor.Transformer) *EventTransformer {
	return &EventTransformer{
		logger:      logger,
		transformer: transformer,
	}
}

func (e *EventTransformer) Transform(ctx context.Context, msg *ingestor.DeviceMessage) (*ingestor.BaseEvent, error) {
	transformed, err := e.transformer.Transform(ctx, msg)
	if err != nil {
		e.logger.Error("failed to transform message",
			"device_id", msg.DeviceID,
			"type", msg.Type,
			"error", err,
		)

		return nil, err
	}

	e.logger.Info("message transformed",
		"device_id", msg.DeviceID,
		"type", msg.Type,
		"event_id", transformed.EventID,
		"domain", transformed.Domain,
		"event_type", transformed.EventType,
	)

	return transformed, nil
}
