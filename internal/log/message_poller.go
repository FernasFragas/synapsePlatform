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

// Subscribe logs the registers topics/queues to consume from.
func (mp *MessagePoller) Subscribe(_ context.Context, topics string) error {
	err := mp.poller.Subscribe(nil, topics)
	if err != nil {
		mp.logger.Error("failed to subscribe", "topic", topics, "error", err)

		return err
	}

	mp.logger.Info("subscribed to topic", "topic", topics)

	return nil
}

// PollMessage logs the consuming messages, calling handler for each.
func (mp *MessagePoller) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, error) {
	msg, err := mp.poller.PollMessage(ctx)
	if err != nil {
		mp.logger.Error("failed to poll message",
			"device_id", msg.DeviceID,
			"type",      msg.Type,
			"timestamp", msg.Timestamp.String(),
			"error", err,
		)

		return msg, err
	}

	mp.logger.Info("polled message",
		"device_id", msg.DeviceID,
		"type",      msg.Type,
		"timestamp", msg.Timestamp.String(),
	)

	return msg, nil
}

// Close logs gracefully shuts down the consumer.
func (mp *MessagePoller) Close(context.Context) error {
	err := mp.poller.Close(nil)
	if err != nil {
		mp.logger.Error("failed to close connection", "error", err)

		return err
	}

	mp.logger.Info("closed connection")

	return nil
}
