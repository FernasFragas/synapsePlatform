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
func (mp *MessagePoller) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, error) {
	msg, err := mp.poller.PollMessage(ctx)
	if err != nil {
		attrs := []any{"error", err}
		if msg != nil {
			attrs = append(attrs,
				"device_id", msg.DeviceID,
				"type", msg.Type,
				"timestamp", msg.Timestamp.String(),
			)
		}

		mp.logger.Error("failed to poll message", attrs...)

		return msg, err
	}

	mp.logger.Info("polled message",
		"device_id", msg.DeviceID,
		"type", msg.Type,
		"timestamp", msg.Timestamp.String(),
	)

	return msg, nil
}

// Close logs gracefully shuts down the consumer.
func (mp *MessagePoller) Close(ctx context.Context) error {
	err := mp.poller.Close(ctx)
	if err != nil {
		mp.logger.Error("failed to close connection", "error", err)

		return err
	}

	mp.logger.Info("closed connection")

	return nil
}
