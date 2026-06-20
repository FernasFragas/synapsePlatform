package summary

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// Risk level values allowed by the EventSummary schema.
const (
	RiskLevelLow    = "low"
	RiskLevelMedium = "medium"
	RiskLevelHigh   = "high"
)

// EventSummary is the structured output schema for an evidence-backed event
// summary. It is the validated shape the summary service persists and returns;
// free-form model text is parsed into this struct before anything is stored.
//
// Every NotableEvent must reference a stored event by ID. RecommendedAction
// entries should reference the event IDs that support the recommendation. This
// keeps the summary useful during incidents and prevents structured JSON from
// becoming structured hallucination.
type EventSummary struct {
	Summary            string              `json:"summary" validate:"required"`
	RiskLevel          string              `json:"risk_level" validate:"required,oneof=low medium high"`
	NotableEvents      []NotableEvent      `json:"notable_events" validate:"dive"`
	RecommendedActions []RecommendedAction `json:"recommended_actions" validate:"dive"`
}

// NotableEvent is a single event the model flagged as operationally
// significant. EventID must identify an event present in the summary's input
// evidence set.
type NotableEvent struct {
	EventID    string    `json:"event_id" validate:"required"`
	Domain     string    `json:"domain" validate:"required"`
	EventType  string    `json:"event_type" validate:"required"`
	EntityID   string    `json:"entity_id,omitempty"`
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	Reason     string    `json:"reason" validate:"required"`
}

// RecommendedAction is a suggested next step grounded in the summarized events.
// RelatedEventIDs should reference event IDs from the input evidence set.
type RecommendedAction struct {
	Action          string   `json:"action" validate:"required"`
	Rationale       string   `json:"rationale" validate:"required"`
	RelatedEventIDs []string `json:"related_event_ids"`
}

// summaryValidate is the package-level validator used for EventSummary. It
// mirrors the pattern in internal/ingestor/normalized_message.go so the
// summary package validates with the same library configuration as the rest
// of the project.
var summaryValidate = validator.New(validator.WithRequiredStructEnabled())

// Validate checks the structured summary against its schema tags. It returns
// the underlying validator error so callers can wrap it with context.
func (es *EventSummary) Validate() error {
	if es == nil {
		return fmt.Errorf("event summary is nil")
	}

	if err := summaryValidate.Struct(es); err != nil {
		return fmt.Errorf("validate event summary: %w", err)
	}

	return nil
}

// ValidateAgainstEvidence checks that every notable event and every
// recommended-action related-event-ID references an event present in the
// provided allowed set. The model output
// that cites event IDs the model never saw must be rejected or repaired
// before persistence.
//
// allowedEventIDs is the set of event IDs the model received as evidence. Pass
// nil or an empty set to reject any citation (useful when the evidence set is
// empty by construction).
func (es *EventSummary) ValidateAgainstEvidence(allowedEventIDs map[string]struct{}) error {
	if es == nil {
		return fmt.Errorf("event summary is nil")
	}
	if len(allowedEventIDs) == 0 {
		// With no evidence, any citation is unsupported. Notable events and
		// recommended actions with related IDs are rejected; an empty summary
		// with no citations is allowed.
		if len(es.NotableEvents) > 0 {
			return fmt.Errorf("notable events reference event IDs but no evidence was provided")
		}

		for _, a := range es.RecommendedActions {
			if len(a.RelatedEventIDs) > 0 {
				return fmt.Errorf("recommended action %q references event IDs but no evidence was provided", a.Action)
			}
		}

		return nil
	}

	for i, ne := range es.NotableEvents {
		if _, ok := allowedEventIDs[ne.EventID]; !ok {
			return fmt.Errorf("notable event %d references unknown event_id %q", i, ne.EventID)
		}
	}

	for i, a := range es.RecommendedActions {
		for j, id := range a.RelatedEventIDs {
			if _, ok := allowedEventIDs[id]; !ok {
				return fmt.Errorf("recommended action %d references unknown event_id %q at position %d", i, id, j)
			}
		}
	}

	return nil
}
