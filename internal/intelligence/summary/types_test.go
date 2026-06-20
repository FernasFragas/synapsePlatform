package summary

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validEventSummary() EventSummary {
	return EventSummary{
		Summary:   "Spike in auth failures across the energy domain.",
		RiskLevel: RiskLevelHigh,
		NotableEvents: []NotableEvent{
			{
				EventID:    "evt-1",
				Domain:     "energy",
				EventType:  "auth_failure",
				EntityID:   "grid-meter-001",
				OccurredAt: time.Now().UTC(),
				Reason:     "repeated 401 responses",
			},
		},
		RecommendedActions: []RecommendedAction{
			{
				Action:          "rotate API credentials",
				Rationale:       "failures started after the last deploy",
				RelatedEventIDs: []string{"evt-1"},
			},
		},
	}
}

func TestEventSummaryValidateValid(t *testing.T) {
	es := validEventSummary()
	require.NoError(t, es.Validate())
}

func TestEventSummaryValidateRejectsMissingSummary(t *testing.T) {
	es := validEventSummary()
	es.Summary = ""
	require.Error(t, es.Validate())
}

func TestEventSummaryValidateRejectsMissingRiskLevel(t *testing.T) {
	es := validEventSummary()
	es.RiskLevel = ""
	require.Error(t, es.Validate())
}

func TestEventSummaryValidateRejectsUnknownRiskLevel(t *testing.T) {
	es := validEventSummary()
	es.RiskLevel = "critical"
	require.Error(t, es.Validate())
}

func TestEventSummaryValidateAcceptsAllRiskLevels(t *testing.T) {
	for _, level := range []string{RiskLevelLow, RiskLevelMedium, RiskLevelHigh} {
		es := validEventSummary()
		es.RiskLevel = level
		assert.NoError(t, es.Validate(), "risk level %q should be valid", level)
	}
}

func TestEventSummaryValidateRejectsNotableEventMissingEventID(t *testing.T) {
	es := validEventSummary()
	es.NotableEvents[0].EventID = ""
	require.Error(t, es.Validate())
}

func TestEventSummaryValidateRejectsNotableEventMissingDomain(t *testing.T) {
	es := validEventSummary()
	es.NotableEvents[0].Domain = ""
	require.Error(t, es.Validate())
}

func TestEventSummaryValidateRejectsNotableEventMissingReason(t *testing.T) {
	es := validEventSummary()
	es.NotableEvents[0].Reason = ""
	require.Error(t, es.Validate())
}

func TestEventSummaryValidateRejectsNotableEventZeroOccurredAt(t *testing.T) {
	es := validEventSummary()
	es.NotableEvents[0].OccurredAt = time.Time{}
	require.Error(t, es.Validate())
}

func TestEventSummaryValidateRejectsRecommendedActionMissingAction(t *testing.T) {
	es := validEventSummary()
	es.RecommendedActions[0].Action = ""
	require.Error(t, es.Validate())
}

func TestEventSummaryValidateRejectsRecommendedActionMissingRationale(t *testing.T) {
	es := validEventSummary()
	es.RecommendedActions[0].Rationale = ""
	require.Error(t, es.Validate())
}

func TestEventSummaryValidateAllowsEmptyCollections(t *testing.T) {
	es := EventSummary{
		Summary:            "all clear",
		RiskLevel:          RiskLevelLow,
		NotableEvents:      nil,
		RecommendedActions: nil,
	}
	require.NoError(t, es.Validate())
}

func TestEventSummaryValidateNilReceiverRejected(t *testing.T) {
	var es *EventSummary
	require.Error(t, es.Validate())
}

func TestValidateAgainstEvidenceAllCitationsKnown(t *testing.T) {
	es := validEventSummary()
	allowed := map[string]struct{}{"evt-1": {}}
	require.NoError(t, es.ValidateAgainstEvidence(allowed))
}

func TestValidateAgainstEvidenceRejectsUnknownNotableEventID(t *testing.T) {
	es := validEventSummary()
	es.NotableEvents[0].EventID = "evt-hallucinated"
	allowed := map[string]struct{}{"evt-1": {}}
	err := es.ValidateAgainstEvidence(allowed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evt-hallucinated")
}

func TestValidateAgainstEvidenceRejectsUnknownRelatedEventID(t *testing.T) {
	es := validEventSummary()
	es.RecommendedActions[0].RelatedEventIDs = []string{"evt-1", "evt-fake"}
	allowed := map[string]struct{}{"evt-1": {}}
	err := es.ValidateAgainstEvidence(allowed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evt-fake")
}

func TestValidateAgainstEvidenceRejectsCitationsWhenNoEvidenceProvided(t *testing.T) {
	es := validEventSummary()
	err := es.ValidateAgainstEvidence(nil)
	require.Error(t, err)
}

func TestValidateAgainstEvidenceAllowsEmptyCitationsWhenNoEvidence(t *testing.T) {
	es := EventSummary{
		Summary:            "no notable events",
		RiskLevel:          RiskLevelLow,
		NotableEvents:      nil,
		RecommendedActions: nil,
	}
	require.NoError(t, es.ValidateAgainstEvidence(nil))
}

func TestValidateAgainstEvidenceAllowsActionWithoutRelatedIDs(t *testing.T) {
	es := validEventSummary()
	es.RecommendedActions[0].RelatedEventIDs = nil
	allowed := map[string]struct{}{"evt-1": {}}
	require.NoError(t, es.ValidateAgainstEvidence(allowed))
}

func TestValidateAgainstEvidenceNilReceiverRejected(t *testing.T) {
	var es *EventSummary
	require.Error(t, es.ValidateAgainstEvidence(map[string]struct{}{"evt-1": {}}))
}