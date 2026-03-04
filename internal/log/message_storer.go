//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/log/mocked_$GOFILE
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

func NewMessageStorer(log *slog.Logger, storer ingestor.MessageStorer) MessageStorer {
	return MessageStorer{
		logger: log,
		storer: storer,
	}
}

func (s MessageStorer) StoreData(
	ctx context.Context, data *ingestor.BaseEvent) error {
	err := s.storer.StoreData(ctx, data)
	if err != nil {
		s.logger.Error("Was not able to save to the DataBase", "message", data, "error", err)

		return err
	}

	s.logger.Info("Saved message to the DataBase", "message", data)

	return nil
}
