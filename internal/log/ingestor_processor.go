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
		il.logger.Error("failed to process message", "msg", msg, "error", err)

		return nil, err
	}

	if msg == nil {
		il.logger.Warn("msg received from processing is empty", "msg", msg)

		return msg, nil
	}

	il.logger.Info("message processed",
		"device_id", msg.DeviceID,
		"type", msg.Type,
		"message", msg,
	)

	return msg, nil
}

func (il *IngestorProcessor) ProcessDataBatch(ctx context.Context, maxMessages int) ([]*ingestor.DeviceMessage, error) {
	// Try to call ProcessDataBatch on the underlying processor
	batchProcessor, ok := il.processor.(interface {
		ProcessDataBatch(ctx context.Context, maxMessages int) ([]*ingestor.DeviceMessage, error)
	})

	var msgs []*ingestor.DeviceMessage
	var err error

	if ok {
		// Use batch processing if available
		msgs, err = batchProcessor.ProcessDataBatch(ctx, maxMessages)
	} else {
		// Fallback to single message
		msg, singleErr := il.processor.ProcessData(ctx)
		if singleErr != nil {
			err = singleErr
		} else if msg != nil {
			msgs = []*ingestor.DeviceMessage{msg}
		}
	}

	if err != nil {
		il.logger.Error("failed to process message batch",
			"max_messages", maxMessages,
			"error", err,
		)

		return msgs, err
	}

	if len(msgs) == 0 {
		il.logger.Warn("no messages received from batch processing")

		return msgs, nil
	}

	// Count message types and validation results
	typeCount := make(map[string]int)

	for _, msg := range msgs {
		if msg != nil {
			typeCount[msg.Type]++
		}
	}

	il.logger.Info("message batch processed",
		"batch_size", len(msgs),
		"max_requested", maxMessages,
		"types", typeCount,
	)

	return msgs, nil
}
