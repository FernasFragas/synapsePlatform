package summary

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"synapsePlatform/internal/api"
)

func TestBuildSummaryPromptIncludesJSONSchemaAndRules(t *testing.T) {
	prompt := BuildSummaryPrompt("energy", []api.DomainStat{
		{Domain: "energy", EventType: "auth_failure", Count: 12,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}, nil)

	assert.Contains(t, prompt, `"risk_level": "low" | "medium" | "high"`)
	assert.Contains(t, prompt, `"notable_events"`)
	assert.Contains(t, prompt, `"recommended_actions"`)
	assert.Contains(t, prompt, `"related_event_ids"`)
	assert.Contains(t, prompt, "Output JSON only")
	assert.Contains(t, prompt, "No markdown")
}

func TestBuildSummaryPromptEmbedsDomain(t *testing.T) {
	prompt := BuildSummaryPrompt("finance", nil, nil)
	assert.Contains(t, prompt, "Domain: finance")
}

func TestBuildSummaryPromptHandlesEmptyDomain(t *testing.T) {
	prompt := BuildSummaryPrompt("", nil, nil)
	assert.Contains(t, prompt, "Domain: (all domains)")
}

func TestBuildSummaryPromptEmbedsStats(t *testing.T) {
	stats := []api.DomainStat{
		{Domain: "energy", EventType: "reading", Count: 42,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}
	prompt := BuildSummaryPrompt("energy", stats, nil)
	assert.Contains(t, prompt, "domain=energy event_type=reading count=42")
	assert.Contains(t, prompt, "first_seen=2026-06-20T10:00:00Z")
	assert.Contains(t, prompt, "last_seen=2026-06-20T11:00:00Z")
}

func TestBuildSummaryPromptHandlesNoStats(t *testing.T) {
	prompt := BuildSummaryPrompt("energy", nil, nil)
	assert.Contains(t, prompt, "(none)\n")
}

func TestBuildSummaryPromptEmbedsEvidence(t *testing.T) {
	evidence := []api.SummaryEvidenceEvent{
		{EventID: "evt-1", Domain: "energy", EventType: "auth_failure",
			EntityID: "meter-001", OccurredAt: time.Date(2026, 6, 20, 10, 5, 0, 0, time.UTC)},
	}
	prompt := BuildSummaryPrompt("energy", nil, evidence)
	assert.Contains(t, prompt, "evt-1 domain=energy event_type=auth_failure entity_id=meter-001 occurred_at=2026-06-20T10:05:00Z")
}

func TestBuildSummaryPromptInstructsEmptyNotableEventsWhenNoEvidence(t *testing.T) {
	prompt := BuildSummaryPrompt("energy", nil, nil)
	assert.Contains(t, prompt, "(none — return an empty notable_events array")
}

func TestBuildSummaryPromptEndsWithJSONOnlyInstruction(t *testing.T) {
	prompt := BuildSummaryPrompt("energy", nil, nil)
	assert.True(t, strings.HasSuffix(strings.TrimSpace(prompt), "No markdown, no code fences, no commentary."))
}

func TestBuildSummaryPromptIsDeterministic(t *testing.T) {
	stats := []api.DomainStat{{Domain: "energy", EventType: "reading", Count: 5,
		FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
		LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)}}
	evidence := []api.SummaryEvidenceEvent{{EventID: "evt-1", Domain: "energy", EventType: "reading",
		OccurredAt: time.Date(2026, 6, 20, 10, 5, 0, 0, time.UTC)}}

	p1 := BuildSummaryPrompt("energy", stats, evidence)
	p2 := BuildSummaryPrompt("energy", stats, evidence)
	assert.Equal(t, p1, p2, "same inputs must produce identical prompts for stable cache keys")
}

func TestSummaryPromptVersionConstant(t *testing.T) {
	assert.Equal(t, "summary.v1", SummaryPromptVersion)
}

func TestAllowedEventIDsBuildsSetFromEvidence(t *testing.T) {
	evidence := []api.SummaryEvidenceEvent{
		{EventID: "evt-1"},
		{EventID: "evt-2"},
		{EventID: "evt-1"}, // duplicate should not double-count
	}
	set := AllowedEventIDs(evidence)
	require.Len(t, set, 2)
	assert.Contains(t, set, "evt-1")
	assert.Contains(t, set, "evt-2")
}

func TestAllowedEventIDsEmptyForNoEvidence(t *testing.T) {
	set := AllowedEventIDs(nil)
	assert.Empty(t, set)
}
