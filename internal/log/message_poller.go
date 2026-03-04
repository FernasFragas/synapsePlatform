//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/log/mocked_$GOFILE
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

func NewMessagePoller(log *slog.Logger, poller ingestor.MessagePoller) MessagePoller {
	return MessagePoller{
		logger: log,
		poller: poller,
	}
}

// Subscribe logs the registers topics/queues to consume from.
func (mp MessagePoller) Subscribe(topics string) error {
	err := mp.poller.Subscribe(topics)
	if err != nil {
		mp.logger.Error("Was not able to subscribe", "topic", topics, "error", err)
	}

	mp.logger.Info("Subscribed to topic", "topic", topics)

	return err
}

// PollMessage logs the consuming messages, calling handler for each.
func (mp MessagePoller) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, error) {
	msg, err := mp.poller.PollMessage(ctx)
	if err != nil {
		mp.logger.Error("Was not able to poll the message from", "message", msg, "error", err)
	}

	mp.logger.Info("Polled message", "message", msg)

	return msg, err
}

// Close logs gracefully shuts down the consumer.
func (mp MessagePoller) Close() error {
	err := mp.poller.Close()
	if err != nil {
		mp.logger.Error("Was not able to close connection from queue", "error", err)
	}

	mp.logger.Info("Closed connection from queue")

	return err
}
