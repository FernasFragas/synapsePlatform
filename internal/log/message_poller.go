package log

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/ingestor"
)

type MessagePoller struct {
	logger *slog.Logger
	poller ingestor.MessagePoller
}

func NewMessagePoller(log *slog.Logger, poller ingestor.MessagePoller) *MessagePoller {
	return &MessagePoller{
		logger: log,
		poller: poller,
	}
}

// PollMessage logs the consuming messages, calling handler for each.
func (mp *MessagePoller) PollMessage(ctx context.Context) (*ingestor.Delivery, error) {
	delivery, err := mp.poller.PollMessage(ctx)
	if err != nil {
		mp.logger.ErrorContext(ctx, "failed to poll message",
			"metadata", deliveryMetadata(delivery),
			"error", err,
		)
		return delivery, err
	}

	if delivery == nil || delivery.Message == nil {
		mp.logger.DebugContext(ctx, "no message available")
		return delivery, nil
	}

	mp.logger.InfoContext(ctx, "polled message",
		"device_id", delivery.Message.DeviceID,
		"type", delivery.Message.Type,
		"source", delivery.Metadata.Source,
	)

	return delivery, nil
}

// Close logs gracefully shuts down the consumer.
func (mp *MessagePoller) Close(ctx context.Context) error {
	err := mp.poller.Close(ctx)
	if err != nil {
		mp.logger.ErrorContext(ctx, "failed to close connection", "error", err)

		return err
	}

	mp.logger.InfoContext(ctx, "closed connection")

	return nil
}

func deliveryMetadata(delivery *ingestor.Delivery) any {
	if delivery == nil {
		return nil
	}

	return map[string]any{
		"source":  delivery.Metadata.Source,
		"headers": delivery.Metadata.Headers,
		"labels":  delivery.Metadata.Labels,
	}
}
