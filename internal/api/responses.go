package api

import "time"

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
