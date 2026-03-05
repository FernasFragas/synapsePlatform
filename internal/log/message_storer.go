package log

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/ingestor"
)

type MessageStorer struct {
	logger *slog.Logger
	storer ingestor.MessageStorer
}

func NewMessageStorer(log *slog.Logger, storer ingestor.MessageStorer) *MessageStorer {
	return &MessageStorer{
		logger: log,
		storer: storer,
	}
}

func (s *MessageStorer) StoreData(
	ctx context.Context, data *ingestor.BaseEvent) error {
	err := s.storer.StoreData(ctx, data)
	if err != nil {
		s.logger.Error("failed to store event", "message", data, "error", err)

		return err
	}

	s.logger.Info("stored event", "message", data)

	return nil
}
