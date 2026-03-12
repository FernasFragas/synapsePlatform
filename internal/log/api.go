package log

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/api"
	"synapsePlatform/internal/ingestor"
)

type EventReader struct {
	logger *slog.Logger

	api api.EventReader
}

func NewEventReader(logger *slog.Logger, api api.EventReader) *EventReader {
	return &EventReader{
		logger: logger,
		api:    api,
	}
}

func (e *EventReader) GetEvent(ctx context.Context, eventID string) (*ingestor.BaseEvent, error) {
	event, err := e.api.GetEvent(ctx, eventID)
	if err != nil {
		e.logger.Error("failed to get event", "event_id", eventID, "error", err)

		return nil, err
	}

	e.logger.Info("fetched event", "event_id", eventID)

	return event, nil
}

func (e *EventReader) ListEvents(ctx context.Context) ([]*ingestor.BaseEvent, error) {
	events, err := e.api.ListEvents(ctx)
	if err != nil {
		e.logger.Error("failed to list events", "error", err)

		return nil, err
	}

	e.logger.Info("listed events", "count", len(events))

	return events, nil
}
