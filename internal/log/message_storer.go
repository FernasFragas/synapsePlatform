package log

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/ingestor"
)

type MessageStorer struct {
	logger *slog.Logger

	storer        ingestor.MessageStorer
	failureStorer ingestor.FailureStorer
}

func NewMessageStorer(log *slog.Logger, storer ingestor.MessageStorer) *MessageStorer {
	return &MessageStorer{
		logger: log,
		storer: storer,
	}
}

func (s *MessageStorer) StoreData(ctx context.Context, data *ingestor.BaseEvent) error {
	err := s.storer.StoreData(ctx, data)
	if err != nil {
		s.logger.Error("failed to store event",
			"event_id", data.EventID,
			"domain", data.Domain,
			"event_type", data.EventType,
			"error", err,
		)

		return err
	}

	s.logger.Info("stored event",
		"event_id", data.EventID,
		"domain", data.Domain,
		"event_type", data.EventType,
		"entity_id", data.EntityID,
	)

	return nil
}

func (s *MessageStorer) StoreFailure(ctx context.Context, failed ingestor.FailedMessage) error {
	err := s.failureStorer.StoreFailure(ctx, failed)
	if err != nil {
		s.logger.Error("failed to store failure",
			"stage", failed.Stage,
			"message", failed.Message,
			"cause", failed.Err,
			"error", err,
		)

		return err
	}

	s.logger.Warn("failure stored",
		"stage", failed.Stage,
		"message", failed.Message,
		"cause", failed.Err,
	)

	return nil
}
