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
	if err != nil || msg == nil {
		attrs := []any{"error", err}
		if msg != nil {
			attrs = append(attrs,
				"device_id", msg.DeviceID,
				"type", msg.Type,
				"timestamp", msg.Timestamp.String(),
			)
		}

		mp.logger.Error("failed to poll message", attrs...)
	}

	if msg == nil {
		mp.logger.Debug("no message available")

		return nil, nil
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

func (mp *MessagePoller) PollMessages(ctx context.Context, maxMessages int) ([]*ingestor.DeviceMessage, error) {
	// Try to call PollMessages on the underlying poller
	batchPoller, ok := mp.poller.(interface {
		PollMessages(ctx context.Context, maxMessages int) ([]*ingestor.DeviceMessage, error)
	})

	var msgs []*ingestor.DeviceMessage
	var err error

	if ok {
		// Use batch polling if available
		msgs, err = batchPoller.PollMessages(ctx, maxMessages)
	} else {
		// Fallback to single message
		msg, singleErr := mp.poller.PollMessage(ctx)
		if singleErr != nil {
			err = singleErr
		} else if msg != nil {
			msgs = []*ingestor.DeviceMessage{msg}
		}
	}

	if err != nil {
		mp.logger.Error("failed to poll messages batch",
			"max_messages", maxMessages,
			"error", err,
		)

		return msgs, err
	}

	if len(msgs) == 0 {
		mp.logger.Debug("no messages available in batch")
		return msgs, nil
	}

	// Count message types
	typeCount := make(map[string]int)

	for _, msg := range msgs {
		if msg != nil {
			typeCount[msg.Type]++
		}
	}

	mp.logger.Info("polled message batch",
		"batch_size", len(msgs),
		"max_requested", maxMessages,
		"types", typeCount,
	)

	return msgs, nil
}
