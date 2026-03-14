package log

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/ingestor"
)

type FailureStorer struct {
	logger  *slog.Logger

	storer  ingestor.FailureStorer
}

func NewFailureStorer(logger *slog.Logger, storer ingestor.FailureStorer) *FailureStorer {
	return &FailureStorer{logger: logger, storer: storer}
}

func (f *FailureStorer) StoreFailure(ctx context.Context, failed ingestor.FailedMessage) error {
	err := f.storer.StoreFailure(ctx, failed)
	if err != nil {
		f.logger.Error("failed to store failure",
			"stage",   failed.Stage,
			"message", failed.Message,
			"cause",   failed.Err,
			"error",   err,
		)
		return err
	}

	f.logger.Warn("failure stored",
		"stage",   failed.Stage,
		"message", failed.Message,
		"cause",   failed.Err,
	)

	return nil
}
