package summary

import (
	"fmt"
	"strings"
	"time"

	"synapsePlatform/internal/api"
)

// SummaryPromptVersion identifies the current prompt shape. Bumping this
// invalidates the cache and persisted-output comparability for summaries
// generated with an older prompt.
//
// The prompt built
// by BuildSummaryPrompt below is the v1 shape: strict JSON, no markdown, no
// prose outside JSON, fixed field names, oneof risk_level, evidence-only
// notable events, related event IDs on recommendations.
const SummaryPromptVersion = "summary.v1"

// BuildSummaryPrompt produces the versioned prompt that asks the model to
// return an EventSummary as JSON. It embeds the aggregate stats plus the
// bounded evidence set the model is allowed to cite.
//
// allowedEventIDs is the set of event IDs the model may reference in
// notable_events[].event_id and recommended_actions[].related_event_ids. When
// empty, the model is told to produce no notable events and no event-backed
// recommendations, since ValidateAgainstEvidence would reject any
// citation the model never saw.
func BuildSummaryPrompt(domain string, stats []api.DomainStat, evidence []api.SummaryEvidenceEvent) string {
	var b strings.Builder
	b.WriteString(summarySystemPreamble)
	b.WriteString(summaryJSONSchema)
	b.WriteString(summaryRules)

	b.WriteString("\nDomain: ")
	if domain != "" {
		b.WriteString(domain)
	} else {
		b.WriteString("(all domains)")
	}

	b.WriteString("\n\nEvent statistics for the requested window:\n")
	if len(stats) == 0 {
		b.WriteString("(none)\n")
	}

	for _, s := range stats {
		b.WriteString(fmt.Sprintf("- domain=%s event_type=%s count=%d first_seen=%s last_seen=%s\n",
			s.Domain, s.EventType, s.Count,
			s.FirstSeen.Format(time.RFC3339), s.LastSeen.Format(time.RFC3339),
		))
	}

	b.WriteString("\nAllowed event IDs you may cite in notable_events[].event_id and recommended_actions[].related_event_ids:\n")
	if len(evidence) == 0 {
		b.WriteString("(none — return an empty notable_events array and only recommendations not tied to specific events)\n")
	} else {
		for _, e := range evidence {
			b.WriteString(fmt.Sprintf("- %s domain=%s event_type=%s entity_id=%s occurred_at=%s\n",
				e.EventID, e.Domain, e.EventType, e.EntityID,
				e.OccurredAt.Format(time.RFC3339),
			))
		}
	}

	b.WriteString("\nReturn ONLY the JSON object. No markdown, no code fences, no commentary.")

	return b.String()
}

// AllowedEventIDs extracts the set of event IDs the model is permitted to cite,
// from the evidence slice. This is the input ValidateAgainstEvidence expects.
func AllowedEventIDs(evidence []api.SummaryEvidenceEvent) map[string]struct{} {
	set := make(map[string]struct{}, len(evidence))

	for _, e := range evidence {
		set[e.EventID] = struct{}{}
	}

	return set
}

const summarySystemPreamble = `You are an operational event analyst. Produce a structured summary of the event statistics and evidence provided below.

`

const summaryJSONSchema = `Respond with a single JSON object matching this schema (field names are exact):

{
  "summary": string,
  "risk_level": "low" | "medium" | "high",
  "notable_events": [
    {
      "event_id": string,
      "domain": string,
      "event_type": string,
      "entity_id": string,
      "occurred_at": RFC3339 timestamp,
      "reason": string
    }
  ],
  "recommended_actions": [
    {
      "action": string,
      "rationale": string,
      "related_event_ids": [string]
    }
  ]
}

`

const summaryRules = `Rules:
- Output JSON only. No markdown, no code fences, no prose before or after the JSON.
- "risk_level" must be one of: low, medium, high.
- "notable_events" must reference event IDs from the allowed list below ONLY. Do not invent event IDs.
- Each "notable_event" must include event_id, domain, event_type, occurred_at, and a reason.
- "recommended_actions" should include "related_event_ids" when the action is based on specific events, and those IDs must come from the allowed list.
- If no allowed event IDs are provided, return an empty "notable_events" array and only recommendations not tied to specific events.
- Use only the data provided. Do not assume values not present in the statistics or evidence.
`
