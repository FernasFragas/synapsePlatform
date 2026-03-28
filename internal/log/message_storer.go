package log

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/ingestor"

	"go.opentelemetry.io/otel/trace"
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
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()

	err := s.storer.StoreData(ctx, data)
	if err != nil {
		s.logger.Error("failed to store event",
			"trace_id", traceID,
			"event_id", data.EventID,
			"domain", data.Domain,
			"event_type", data.EventType,
			"error", err,
		)

		return err
	}

	s.logger.Info("stored event",
		"trace_id", traceID,
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

	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()

	s.logger.Debug("storing batch",
		"batch_size", len(events),
	)

	err := s.storer.StoreBatch(ctx, events)
	if err != nil {
		s.logger.Error("failed to store batch",
			"trace_id", traceID,
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

	s.logger.Info("stored batch",
		"trace_id", traceID,
		"batch_size", len(events),
		"domains", domainCounts,
	)

	return nil
}
