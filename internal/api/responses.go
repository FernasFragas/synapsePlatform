package api

import (
	"strconv"
	"synapsePlatform/internal/ingestor"
	"time"
)

type EventResponse struct {
	EventID       string    `json:"event_id"`
	Domain        string    `json:"domain"`
	EventType     string    `json:"event_type"`
	EntityID      string    `json:"entity_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Source        string    `json:"source"`
	SchemaVersion string    `json:"schema_version"`
	Data          any       `json:"data"`
}

type ListResponse struct {
	Data       []*EventResponse `json:"data"`
	NextCursor string           `json:"next_cursor,omitempty"`
	HasMore    bool             `json:"has_more"`
}

func toResponses(events []*ingestor.BaseEvent) []*EventResponse {
	responses := make([]*EventResponse, len(events))
	for i, event := range events {
		responses[i] = toResponse(event)
	}
	return responses
}

func toResponse(event *ingestor.BaseEvent) *EventResponse {
	return &EventResponse{
		EventID:       event.EventID.String(),
		Domain:        event.Domain,
		EventType:     event.EventType,
		EntityID:      event.EntityID,
		OccurredAt:    event.OccurredAt,
		Source:        event.Source,
		SchemaVersion: event.SchemaVersion,
		Data:          event.Data,
	}
}

func parseIntOrDefault(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return fallback
	}

	return v
}
