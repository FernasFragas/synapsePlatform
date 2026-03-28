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
func (mp *MessagePoller) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, string, error) {
	msg, receiptHandle, err := mp.poller.PollMessage(ctx)
	if err != nil || msg == nil {
		attrs := []any{"error", err}
		if msg != nil {
			attrs = append(attrs,
				"device_id", msg.DeviceID,
				"type", msg.Type,
				"timestamp", msg.Timestamp.String(),
			)
		}

		if receiptHandle != "" {
			attrs = append(attrs, "receipt_handle", receiptHandle)
		}

		mp.logger.Error("failed to poll message", attrs...)

		return msg, receiptHandle, err
	}

	if msg == nil {
		mp.logger.Debug("no message available")

		return nil, "", nil
	}

	mp.logger.Info("polled message",
		"device_id", msg.DeviceID,
		"type", msg.Type,
		"timestamp", msg.Timestamp.String(),
		"receipt_handle", receiptHandle,
	)

	return msg, receiptHandle, nil
}

// AckMessageSuccess logs message acknowledgment.
func (mp *MessagePoller) AckMessageSuccess(ctx context.Context, receiptHandle string) error {
	err := mp.poller.AckMessageSuccess(ctx, receiptHandle)
	if err != nil {
		mp.logger.Error("failed to acknowledge message",
			"receipt_handle", receiptHandle,
			"error", err,
		)

		return err
	}

	mp.logger.Debug("acknowledged message",
		"receipt_handle", receiptHandle,
	)

	return nil
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
