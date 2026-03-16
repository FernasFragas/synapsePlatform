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
	if err != nil {
		f.logger.Error("failed to store failure",
			"stage", failed.Stage,
			"message", failed.Message,
			"cause", failed.Err,
			"error", err,
		)
		return err
	}

	f.logger.Warn("failure stored",
		"stage", failed.Stage,
		"message", failed.Message,
		"cause", failed.Err,
	)

	return nil
}
