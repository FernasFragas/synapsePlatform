package log

import (
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

func (e *EventTransformer) Transform(msg *ingestor.DeviceMessage) (*ingestor.BaseEvent, error) {
	transformed, err := e.transformer.Transform(msg)
	if err != nil {
		e.logger.Error("failed to transform message", "msg", msg, "error", err)

		return nil, err
	}

	e.logger.Info("message transformed", "device_id", msg.DeviceID, "type", msg.Type, "transformed message", transformed)

	return transformed, nil
}
