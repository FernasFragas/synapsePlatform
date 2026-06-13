package log

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/ingestor"
)

type Publisher struct {
	logger *slog.Logger

	failuresPublisher ingestor.FailureStorer
}

func NewFailurePublisher(logger *slog.Logger, storer ingestor.FailureStorer) *Publisher {
	return &Publisher{logger: logger, failuresPublisher: storer}
}

func (f *Publisher) StoreFailure(ctx context.Context, failed ingestor.FailedMessage) error {
	err := f.failuresPublisher.StoreFailure(ctx, failed)
	attrs := []any{
		"stage", failed.Stage,
		"error_type", failed.ErrorType,
		"message", failed.Message,
		"error_message", failed.ErrorMessage,
		"retry_count", failed.RetryCount,
		"source", failed.Metadata.Source,
		"labels", failed.Metadata.Labels,
		"failed_at", failed.FailedAt,
	}

	if err != nil {
		f.logger.ErrorContext(ctx, "failed to store failure", append(attrs, "error", err)...)
		return err
	}

	f.logger.WarnContext(ctx, "failure stored", attrs...)
	return nil
}
