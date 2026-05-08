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
		s.logger.ErrorContext(ctx, "failed to store event",
			"event_id", data.EventID,
			"domain", data.Domain,
			"event_type", data.EventType,
			"error", err,
		)

		return err
	}

	s.logger.InfoContext(ctx, "stored event",
		"event_id", data.EventID,
		"domain", data.Domain,
		"event_type", data.EventType,
		"entity_id", data.EntityID,
	)

	return nil
}

func (s *MessageStorer) StoreBatch(ctx context.Context, events []*ingestor.BaseEvent) error {
	if len(events) == 0 {
		return nil
	}

	s.logger.DebugContext(ctx, "storing batch",
		"batch_size", len(events),
	)

	err := s.storer.StoreBatch(ctx, events)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to store batch",
			"batch_size", len(events),
			"error", err,
		)

		return err
	}
	// Log summary of what was stored
	domainCounts := make(map[string]int)

	for _, event := range events {
		domainCounts[event.Domain]++
	}

	s.logger.InfoContext(ctx, "stored batch",
		"batch_size", len(events),
		"domains", domainCounts,
	)

	return nil
}
