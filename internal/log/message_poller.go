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
func (mp *MessagePoller) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, ingestor.AckHandler, error) {
	msg, ack, err := mp.poller.PollMessage(ctx)
	if err != nil {
		attrs := []any{"error", err}
		if msg != nil {
			attrs = append(attrs,
				"device_id", msg.DeviceID,
				"type", msg.Type,
				"timestamp", msg.Timestamp.String(),
			)
		}

		mp.logger.ErrorContext(ctx, "failed to poll message", attrs...)

		return msg, ack, err
	}
	if msg == nil {
		mp.logger.DebugContext(ctx, "no message available")

		return nil, ack, nil
	}

	mp.logger.InfoContext(ctx, "polled message",
		"device_id", msg.DeviceID,
		"type", msg.Type,
		"timestamp", msg.Timestamp.String(),
	)

	return msg, ack, nil
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
