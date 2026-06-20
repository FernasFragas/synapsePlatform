package sqllite_test

import (
	"context"
	"fmt"
	"synapsePlatform/internal/api"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/sqllite"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type StorerTestSuite struct {
	suite.Suite
	repo *sqllite.Repo
	ctx  context.Context
}

func TestStorerSuite(t *testing.T) {
	suite.Run(t, new(StorerTestSuite))
}

func (s *StorerTestSuite) SetupTest() {
	var err error
	s.repo, err = sqllite.NewRepo(":memory:")
	s.Require().NoError(err)
	s.ctx = context.Background()
}

func (s *StorerTestSuite) TearDownTest() {
	s.Require().NoError(s.repo.Close())
}

// --- StoreData + GetEvent ---

func (s *StorerTestSuite) TestStoreAndGetEvent_RoundTripsAllFields() {
	event := s.energyEvent("device-1", time.Now().UTC())

	s.Require().NoError(s.repo.StoreData(s.ctx, event))

	got, err := s.repo.GetEvent(s.ctx, event.EventID.String())
	s.Require().NoError(err)

	s.Equal(event.EventID, got.EventID)
	s.Equal(event.Domain, got.Domain)
	s.Equal(event.EventType, got.EventType)
	s.Equal(event.EntityID, got.EntityID)
	s.Equal(event.EntityType, got.EntityType)
	s.Equal(event.Source, got.Source)
	s.Equal(event.SchemaVersion, got.SchemaVersion)
}

func (s *StorerTestSuite) TestGetEvent_NotFound_ReturnsErrEventNotFound() {
	_, err := s.repo.GetEvent(s.ctx, uuid.NewString())

	s.ErrorIs(err, ingestor.ErrEventNotFound)
}

func (s *StorerTestSuite) TestStoreData_DuplicateEventID_RecordsDuplicate() {
	event := s.energyEvent("device-1", time.Now().UTC())

	s.Require().NoError(s.repo.StoreData(s.ctx, event))

	err := s.repo.StoreData(s.ctx, event)
	s.NoError(err)

	attempted, inserted, duplicates := s.storeAccounting()
	s.Equal(int64(2), attempted)
	s.Equal(int64(1), inserted)
	s.Equal(int64(1), duplicates)
}

// --- ListEvents pagination ---

func (s *StorerTestSuite) TestListEvents_Empty_ReturnsEmptyPage() {
	result, err := s.repo.ListEvents(s.ctx, ingestor.PageRequest{Limit: 10})

	s.Require().NoError(err)
	s.Empty(result.Items)
	s.False(result.HasMore)
	s.Empty(result.NextCursor)
}

func (s *StorerTestSuite) TestListEvents_ReturnsNewestFirst() {
	e1 := s.energyEvent("dev-1", time.Now().Add(-2*time.Hour))
	e2 := s.energyEvent("dev-2", time.Now().Add(-1*time.Hour))
	e3 := s.energyEvent("dev-3", time.Now())
	s.seedEvents(e1, e2, e3)

	result, err := s.repo.ListEvents(s.ctx, ingestor.PageRequest{Limit: 10})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 3)

	s.Equal(e3.EventID, result.Items[0].EventID, "most recent first")
	s.Equal(e1.EventID, result.Items[2].EventID, "oldest last")
	s.False(result.HasMore)
}

func (s *StorerTestSuite) TestListEvents_Pagination_WalksAllPages() {
	for i := 0; i < 5; i++ {
		s.Require().NoError(s.repo.StoreData(s.ctx,
			s.energyEvent("dev", time.Now().Add(time.Duration(i)*time.Second))))
	}

	var allIDs []string
	cursor := ""
	for {
		result, err := s.repo.ListEvents(s.ctx, ingestor.PageRequest{Cursor: cursor, Limit: 2})
		s.Require().NoError(err)

		for _, e := range result.Items {
			allIDs = append(allIDs, e.EventID.String())
		}

		if !result.HasMore {
			break
		}
		cursor = result.NextCursor
		s.NotEmpty(cursor)
	}

	s.Len(allIDs, 5, "all events reachable through pagination")

	unique := make(map[string]bool)
	for _, id := range allIDs {
		s.False(unique[id], "no duplicates across pages")
		unique[id] = true
	}
}

func (s *StorerTestSuite) TestListEvents_DefaultLimit_Applies() {
	for i := 0; i < 25; i++ {
		s.Require().NoError(s.repo.StoreData(s.ctx,
			s.energyEvent("dev", time.Now().Add(time.Duration(i)*time.Millisecond))))
	}

	result, err := s.repo.ListEvents(s.ctx, ingestor.PageRequest{})
	s.Require().NoError(err)
	s.Len(result.Items, 20, "default page size is 20")
	s.True(result.HasMore)
}

func (s *StorerTestSuite) TestListEvents_InvalidCursor_ReturnsError() {
	_, err := s.repo.ListEvents(s.ctx, ingestor.PageRequest{Cursor: "not-valid-base64!!"})

	s.Error(err)
	s.Contains(err.Error(), "invalid cursor")
}

// --- StoreFailure ---

func (s *StorerTestSuite) TestStoreFailure_WithMessage_Persists() {
	msg := &ingestor.DeviceMessage{
		DeviceID:  "dev-1",
		Type:      "energy_meter",
		Timestamp: time.Now(),
	}

	err := s.repo.StoreFailure(s.ctx, ingestor.FailedMessage{
		Stage:        "transform",
		Message:      msg,
		ErrorMessage: "schema mismatch",
	})
	s.Require().NoError(err)

	var count int
	s.Require().NoError(
		s.repo.Db.QueryRowContext(s.ctx,
			"SELECT COUNT(*) FROM failed_messages WHERE stage = 'transform'").Scan(&count),
	)
	s.Equal(1, count)
}

func (s *StorerTestSuite) TestStoreFailure_NilMessage_DoesNotPanic() {
	err := s.repo.StoreFailure(s.ctx, ingestor.FailedMessage{
		Stage:        "process",
		ErrorMessage: "broker down",
	})
	s.NoError(err)
}

// --- AggregateByDomain ---

func (s *StorerTestSuite) TestAggregateByDomain_ReturnsEventTimeBounds() {
	first := time.Date(2026, 6, 13, 20, 35, 47, 0, time.UTC)
	last := first.Add(2 * time.Minute)
	beforeWindow := first.Add(-time.Hour)

	e1 := s.energyEvent("dev-1", first)
	e1.OccurredAt = first
	e2 := s.energyEvent("dev-2", last)
	e2.OccurredAt = last

	s.seedEvents(e1, e2)

	stats, err := s.repo.AggregateByDomain(s.ctx, beforeWindow)

	s.Require().NoError(err)
	s.Require().Len(stats, 1)
	s.Equal("energy", stats[0].Domain)
	s.Equal("energy_meter", stats[0].EventType)
	s.Equal(int64(2), stats[0].Count)
	s.True(stats[0].FirstSeen.Equal(first), "first_seen should not fall back to zero time")
	s.True(stats[0].LastSeen.Equal(last), "last_seen should not fall back to zero time")
}

// --- Health probe ---

func (s *StorerTestSuite) TestName_ReturnsDB() {
	s.Equal("db", s.repo.Name())
}

func (s *StorerTestSuite) TestCheck_OpenDB_ReturnsNoError() {
	s.NoError(s.repo.Check(s.ctx))
}

func (s *StorerTestSuite) TestCheck_ClosedDB_ReturnsError() {
	s.Require().NoError(s.repo.Close())

	s.Error(s.repo.Check(s.ctx))
}

// --- StoreBatch ---

func (s *StorerTestSuite) TestStoreBatch_EmptySlice_ReturnsNoError() {
	err := s.repo.StoreBatch(s.ctx, []*ingestor.BaseEvent{})

	s.NoError(err)
}

func (s *StorerTestSuite) TestStoreBatch_SingleEvent_Persists() {
	event := s.energyEvent("device-1", time.Now().UTC())

	err := s.repo.StoreBatch(s.ctx, []*ingestor.BaseEvent{event})
	s.Require().NoError(err)

	got, err := s.repo.GetEvent(s.ctx, event.EventID.String())
	s.Require().NoError(err)
	s.Equal(event.EventID, got.EventID)
}

func (s *StorerTestSuite) TestStoreBatch_MultipleEvents_AllPersist() {
	now := time.Now().UTC()
	events := []*ingestor.BaseEvent{
		s.energyEvent("device-1", now),
		s.energyEvent("device-2", now.Add(1*time.Second)),
		s.energyEvent("device-3", now.Add(2*time.Second)),
	}

	err := s.repo.StoreBatch(s.ctx, events)
	s.Require().NoError(err)

	// Verify all events were stored
	for _, event := range events {
		got, err := s.repo.GetEvent(s.ctx, event.EventID.String())
		s.Require().NoError(err)
		s.Equal(event.EventID, got.EventID)
		s.Equal(event.EntityID, got.EntityID)
	}
}

func (s *StorerTestSuite) TestStoreBatch_LargeBatch_Persists() {
	now := time.Now().UTC()
	events := make([]*ingestor.BaseEvent, 100)
	for i := 0; i < 100; i++ {
		events[i] = s.energyEvent(fmt.Sprintf("device-%d", i), now.Add(time.Duration(i)*time.Millisecond))
	}

	err := s.repo.StoreBatch(s.ctx, events)
	s.Require().NoError(err)

	// Verify count
	var count int
	s.Require().NoError(
		s.repo.Db.QueryRowContext(s.ctx, "SELECT COUNT(*) FROM events").Scan(&count),
	)
	s.Equal(100, count)

	attempted, inserted, duplicates := s.storeAccounting()
	s.Equal(int64(100), attempted)
	s.Equal(int64(100), inserted)
	s.Equal(int64(0), duplicates)
}

func (s *StorerTestSuite) TestStoreBatch_DuplicateEventID_RecordsDuplicate() {
	event1 := s.energyEvent("device-1", time.Now().UTC())
	event2 := s.energyEvent("device-2", time.Now().UTC())
	event2.EventID = event1.EventID // Duplicate ID

	// Store first event
	s.Require().NoError(s.repo.StoreData(s.ctx, event1))

	// Try to batch insert with duplicate
	err := s.repo.StoreBatch(s.ctx, []*ingestor.BaseEvent{event2})
	s.NoError(err)

	// Verify only one event exists
	var count int
	s.Require().NoError(
		s.repo.Db.QueryRowContext(s.ctx, "SELECT COUNT(*) FROM events").Scan(&count),
	)
	s.Equal(1, count)

	attempted, inserted, duplicates := s.storeAccounting()
	s.Equal(int64(2), attempted)
	s.Equal(int64(1), inserted)
	s.Equal(int64(1), duplicates)
}

func (s *StorerTestSuite) TestStoreBatch_MixedEventTypes_AllPersist() {
	now := time.Now().UTC()
	events := []*ingestor.BaseEvent{
		s.energyEvent("device-1", now),
		s.financialEvent("account-1", now.Add(1*time.Second)),
		s.environmentalEvent("sensor-1", now.Add(2*time.Second)),
	}

	err := s.repo.StoreBatch(s.ctx, events)
	s.Require().NoError(err)

	// Verify all events with different types were stored
	for _, event := range events {
		got, err := s.repo.GetEvent(s.ctx, event.EventID.String())
		s.Require().NoError(err)
		s.Equal(event.EventType, got.EventType)
	}
}

// --- Structured summary persistence (3.2.4 Option A) ---

func (s *StorerTestSuite) TestSaveAndLatestSummary_RoundTripsStructuredFields() {
	since := time.Now().UTC().Truncate(time.Hour)
	report := &api.Report{
		Domain:            "energy",
		WindowFrom:        since,
		Model:             "mistral:7b",
		Content:           "Spike in auth failures.",
		StructuredContent: `{"summary":"Spike in auth failures.","risk_level":"high"}`,
		Provider:          "ollama",
		PromptVersion:     "summary.v1",
		InputHash:         "abc123def456",
		CreatedAt:         time.Now().UTC(),
	}

	_, err := s.repo.SaveSummary(s.ctx, report)
	s.Require().NoError(err)

	got, ok, err := s.repo.LatestSummary(s.ctx, api.SummaryLookup{
		Domain:        report.Domain,
		WindowFrom:    report.WindowFrom,
		Provider:      report.Provider,
		Model:         report.Model,
		PromptVersion: report.PromptVersion,
		InputHash:     report.InputHash,
	})
	s.Require().NoError(err)
	s.True(ok)
	s.Equal(report.Domain, got.Domain)
	s.Equal(report.WindowFrom, got.WindowFrom)
	s.Equal(report.Model, got.Model)
	s.Equal(report.Content, got.Content)
	s.Equal(report.StructuredContent, got.StructuredContent)
	s.Equal(report.Provider, got.Provider)
	s.Equal(report.PromptVersion, got.PromptVersion)
	s.Equal(report.InputHash, got.InputHash)
}

func (s *StorerTestSuite) TestSaveSummary_LegacyFieldsOnly_StillPersists() {
	since := time.Now().UTC().Truncate(time.Hour)
	report := &api.Report{
		Domain:     "energy",
		WindowFrom: since,
		Model:      "mistral:7b",
		Content:    "Free-form summary without structured data.",
		CreatedAt:  time.Now().UTC(),
	}

	_, err := s.repo.SaveSummary(s.ctx, report)
	s.Require().NoError(err)

	got, ok, err := s.repo.LatestSummary(s.ctx, api.SummaryLookup{
		Domain:     report.Domain,
		WindowFrom: report.WindowFrom,
		Model:      report.Model,
		InputHash:  report.InputHash,
	})
	s.Require().NoError(err)
	s.True(ok)
	s.Equal(report.Content, got.Content)
	s.Empty(got.StructuredContent, "legacy summaries store NULL structured_content")
	s.Empty(got.Provider)
	s.Empty(got.PromptVersion)
	s.Empty(got.InputHash)
}

func (s *StorerTestSuite) TestLatestSummary_NotFound_ReturnsFalse() {
	got, ok, err := s.repo.LatestSummary(s.ctx, api.SummaryLookup{
		Domain:     "nonexistent",
		WindowFrom: time.Now().UTC(),
		Model:      "mistral:7b",
		InputHash:  "no-such-hash",
	})
	s.Require().NoError(err)
	s.False(ok)
	s.Nil(got)
}

func (s *StorerTestSuite) TestLatestSummary_ReturnsMostRecent() {
	since := time.Now().UTC().Truncate(time.Hour)
	older := &api.Report{
		Domain: "energy", WindowFrom: since, Model: "mistral:7b",
		Content: "older summary", InputHash: "shared-hash",
		CreatedAt: time.Now().UTC().Add(-2 * time.Minute),
	}
	newer := &api.Report{
		Domain: "energy", WindowFrom: since, Model: "mistral:7b",
		Content: "newer summary", InputHash: "shared-hash",
		CreatedAt: time.Now().UTC(),
	}

	_, err := s.repo.SaveSummary(s.ctx, older)
	s.Require().NoError(err)
	_, err = s.repo.SaveSummary(s.ctx, newer)
	s.Require().NoError(err)

	got, ok, err := s.repo.LatestSummary(s.ctx, api.SummaryLookup{
		Domain:     "energy",
		WindowFrom: since,
		Model:      "mistral:7b",
		InputHash:  "shared-hash",
	})
	s.Require().NoError(err)
	s.True(ok)
	s.Equal("newer summary", got.Content)
}

func (s *StorerTestSuite) TestLatestSummary_DifferentInputHash_IsCacheMiss() {
	since := time.Now().UTC().Truncate(time.Hour)
	report := &api.Report{
		Domain: "energy", WindowFrom: since, Model: "mistral:7b",
		Content: "original", InputHash: "hash-A",
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.repo.SaveSummary(s.ctx, report)
	s.Require().NoError(err)

	// Same domain/window/model but different input_hash should miss.
	got, ok, err := s.repo.LatestSummary(s.ctx, api.SummaryLookup{
		Domain:     "energy",
		WindowFrom: since,
		Model:      "mistral:7b",
		InputHash:  "hash-B",
	})
	s.Require().NoError(err)
	s.False(ok, "different input_hash must be a cache miss")
	s.Nil(got)
}

func (s *StorerTestSuite) TestLatestSummary_DifferentModel_IsCacheMiss() {
	since := time.Now().UTC().Truncate(time.Hour)
	report := &api.Report{
		Domain: "energy", WindowFrom: since, Model: "mistral:7b",
		Content: "original", InputHash: "shared-hash",
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.repo.SaveSummary(s.ctx, report)
	s.Require().NoError(err)

	got, ok, err := s.repo.LatestSummary(s.ctx, api.SummaryLookup{
		Domain:     "energy",
		WindowFrom: since,
		Model:      "llama3",
		InputHash:  "shared-hash",
	})
	s.Require().NoError(err)
	s.False(ok, "different model must be a cache miss")
	s.Nil(got)
}

// --- Summary evidence ---

func (s *StorerTestSuite) TestListSummaryEvidence_ReturnsRecentEventsForDomain() {
	since := time.Now().UTC().Truncate(time.Hour)
	s.seedEvents(
		s.energyEvent("meter-1", since.Add(5*time.Minute)),
		s.energyEvent("meter-2", since.Add(10*time.Minute)),
		s.financialEvent("account-1", since.Add(15*time.Minute)),
	)

	evidence, err := s.repo.ListSummaryEvidence(s.ctx, api.EvidenceRequest{
		Domain:     "energy",
		Since:      since,
		MaxResults: 10,
	})
	s.Require().NoError(err)
	s.Len(evidence, 2)
	for _, e := range evidence {
		s.Equal("energy", e.Domain)
		s.NotEmpty(e.EventID)
		s.NotEmpty(e.EventType)
	}
}

func (s *StorerTestSuite) TestListSummaryEvidence_AllDomainsWhenDomainEmpty() {
	since := time.Now().UTC().Truncate(time.Hour)
	s.seedEvents(
		s.energyEvent("meter-1", since.Add(5*time.Minute)),
		s.financialEvent("account-1", since.Add(10*time.Minute)),
	)

	evidence, err := s.repo.ListSummaryEvidence(s.ctx, api.EvidenceRequest{
		Domain:     "",
		Since:      since,
		MaxResults: 10,
	})
	s.Require().NoError(err)
	s.Len(evidence, 2)
}

func (s *StorerTestSuite) TestListSummaryEvidence_BoundedByMaxResults() {
	since := time.Now().UTC().Truncate(time.Hour)
	for i := 0; i < 5; i++ {
		s.Require().NoError(s.repo.StoreData(s.ctx, s.energyEvent(fmt.Sprintf("meter-%d", i), since.Add(time.Duration(i)*time.Minute))))
	}

	evidence, err := s.repo.ListSummaryEvidence(s.ctx, api.EvidenceRequest{
		Domain:     "energy",
		Since:      since,
		MaxResults: 3,
	})
	s.Require().NoError(err)
	s.Len(evidence, 3, "evidence set must be bounded by MaxResults")
}

func (s *StorerTestSuite) TestListSummaryEvidence_DefaultsMaxResultsWhenZero() {
	since := time.Now().UTC().Truncate(time.Hour)
	evidence, err := s.repo.ListSummaryEvidence(s.ctx, api.EvidenceRequest{
		Domain:     "energy",
		Since:      since,
		MaxResults: 0,
	})
	s.Require().NoError(err)
	s.Empty(evidence, "no events seeded, should return empty slice")
}

func (s *StorerTestSuite) TestListSummaryEvidence_OrdersByOccurredAtDesc() {
	since := time.Now().UTC().Truncate(time.Hour)
	older := s.energyEvent("meter-old", since.Add(1*time.Minute))
	newer := s.energyEvent("meter-new", since.Add(30*time.Minute))
	s.seedEvents(older, newer)

	evidence, err := s.repo.ListSummaryEvidence(s.ctx, api.EvidenceRequest{
		Domain:     "energy",
		Since:      since,
		MaxResults: 10,
	})
	s.Require().NoError(err)
	s.Require().Len(evidence, 2)
	s.True(evidence[0].OccurredAt.After(evidence[1].OccurredAt), "most recent event should be first")
}

func (s *StorerTestSuite) TestListSummaryEvidence_EmptyWhenNoEventsInWindow() {
	evidence, err := s.repo.ListSummaryEvidence(s.ctx, api.EvidenceRequest{
		Domain:     "energy",
		Since:      time.Now().UTC(),
		MaxResults: 10,
	})
	s.Require().NoError(err)
	s.Empty(evidence)
}

func (s *StorerTestSuite) TestListSummaryEvidence_IncludesStableFieldsOnly() {
	since := time.Now().UTC().Truncate(time.Hour)
	s.seedEvents(s.energyEvent("meter-1", since.Add(5*time.Minute)))

	evidence, err := s.repo.ListSummaryEvidence(s.ctx, api.EvidenceRequest{
		Domain:     "energy",
		Since:      since,
		MaxResults: 10,
	})
	s.Require().NoError(err)
	s.Require().Len(evidence, 1)
	e := evidence[0]
	s.NotEmpty(e.EventID)
	s.Equal("energy", e.Domain)
	s.NotEmpty(e.EventType)
	s.NotEmpty(e.EntityID)
	s.False(e.OccurredAt.IsZero(), "occurred_at must be populated")
}

// --- Summary evidence links (3.2.8) ---

func (s *StorerTestSuite) TestSaveSummaryReturnsID() {
	report := &api.Report{
		Domain: "energy", WindowFrom: time.Now().UTC().Truncate(time.Hour),
		Model: "mistral:7b", Content: "test", CreatedAt: time.Now().UTC(),
	}
	id, err := s.repo.SaveSummary(s.ctx, report)
	s.Require().NoError(err)
	s.Greater(id, int64(0), "SaveSummary must return a valid row ID")
}

func (s *StorerTestSuite) TestSaveSummaryEvidenceLinks_PersistsLinks() {
	report := &api.Report{
		Domain: "energy", WindowFrom: time.Now().UTC().Truncate(time.Hour),
		Model: "mistral:7b", Content: "test", CreatedAt: time.Now().UTC(),
	}
	summaryID, err := s.repo.SaveSummary(s.ctx, report)
	s.Require().NoError(err)

	links := []api.SummaryEvidenceLink{
		{SummaryID: summaryID, EventID: "evt-1", Relationship: api.EvidenceRelationshipEvidence},
		{SummaryID: summaryID, EventID: "evt-2", Relationship: api.EvidenceRelationshipEvidence},
		{SummaryID: summaryID, EventID: "evt-1", Relationship: api.EvidenceRelationshipNotable},
	}
	s.Require().NoError(s.repo.SaveSummaryEvidenceLinks(s.ctx, links))

	var count int
	s.Require().NoError(s.repo.Db.QueryRowContext(s.ctx,
		`SELECT COUNT(*) FROM intelligence_summary_events WHERE summary_id = ?`,
		summaryID,
	).Scan(&count))
	s.Equal(3, count)
}

func (s *StorerTestSuite) TestSaveSummaryEvidenceLinks_IdempotentOnDuplicate() {
	report := &api.Report{
		Domain: "energy", WindowFrom: time.Now().UTC().Truncate(time.Hour),
		Model: "mistral:7b", Content: "test", CreatedAt: time.Now().UTC(),
	}
	summaryID, err := s.repo.SaveSummary(s.ctx, report)
	s.Require().NoError(err)

	links := []api.SummaryEvidenceLink{
		{SummaryID: summaryID, EventID: "evt-1", Relationship: api.EvidenceRelationshipEvidence},
	}
	s.Require().NoError(s.repo.SaveSummaryEvidenceLinks(s.ctx, links))
	// Save the same link again — should not error or duplicate.
	s.Require().NoError(s.repo.SaveSummaryEvidenceLinks(s.ctx, links))

	var count int
	s.Require().NoError(s.repo.Db.QueryRowContext(s.ctx,
		`SELECT COUNT(*) FROM intelligence_summary_events WHERE summary_id = ? AND event_id = ? AND relationship = ?`,
		summaryID, "evt-1", api.EvidenceRelationshipEvidence,
	).Scan(&count))
	s.Equal(1, count, "INSERT OR IGNORE must prevent duplicates")
}

func (s *StorerTestSuite) TestSaveSummaryEvidenceLinks_EmptySliceIsNoOp() {
	s.Require().NoError(s.repo.SaveSummaryEvidenceLinks(s.ctx, nil))
}

func (s *StorerTestSuite) TestSaveSummaryEvidenceLinks_IndexedByEventID() {
	report := &api.Report{
		Domain: "energy", WindowFrom: time.Now().UTC().Truncate(time.Hour),
		Model: "mistral:7b", Content: "test", CreatedAt: time.Now().UTC(),
	}
	summaryID, err := s.repo.SaveSummary(s.ctx, report)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.SaveSummaryEvidenceLinks(s.ctx, []api.SummaryEvidenceLink{
		{SummaryID: summaryID, EventID: "evt-shared", Relationship: api.EvidenceRelationshipEvidence},
	}))

	// Query by event_id using the index to verify it works.
	var foundSummaryID int64
	s.Require().NoError(s.repo.Db.QueryRowContext(s.ctx,
		`SELECT summary_id FROM intelligence_summary_events WHERE event_id = ?`,
		"evt-shared",
	).Scan(&foundSummaryID))
	s.Equal(summaryID, foundSummaryID)
}

// --- helpers ---

func (s *StorerTestSuite) energyEvent(entityID string, ingestedAt time.Time) *ingestor.BaseEvent {
	return &ingestor.BaseEvent{
		EventID:       uuid.New(),
		Domain:        "energy",
		EventType:     "energy_meter",
		EntityID:      entityID,
		EntityType:    "sensor",
		OccurredAt:    time.Now().UTC(),
		IngestedAt:    ingestedAt,
		Source:        "test",
		SchemaVersion: "1.0.0",
		Data:          &ingestor.EnergyReading{PowerW: 100, EnergyWh: 500, VoltageV: 220, CurrentMA: 455},
	}
}

func (s *StorerTestSuite) seedEvents(events ...*ingestor.BaseEvent) {
	for _, e := range events {
		s.Require().NoError(s.repo.StoreData(s.ctx, e))
	}
}

func (s *StorerTestSuite) storeAccounting() (int64, int64, int64) {
	var attempted, inserted, duplicates int64
	s.Require().NoError(s.repo.Db.QueryRowContext(s.ctx, `
SELECT attempted_events, inserted_events, duplicate_events
FROM store_accounting
WHERE id = 1
`).Scan(&attempted, &inserted, &duplicates))

	return attempted, inserted, duplicates
}

func (s *StorerTestSuite) financialEvent(entityID string, ingestedAt time.Time) *ingestor.BaseEvent {
	return &ingestor.BaseEvent{
		EventID:       uuid.New(),
		Domain:        "financial",
		EventType:     "financial_stream",
		EntityID:      entityID,
		EntityType:    "device",
		OccurredAt:    time.Now().UTC(),
		IngestedAt:    ingestedAt,
		Source:        "test",
		SchemaVersion: "1.0.0",
		Data: &ingestor.FinancialTransaction{
			AmountMinor: 10050,
			Currency:    "USD",
			Merchant:    "Test Store",
			Status:      "completed",
		},
	}
}

func (s *StorerTestSuite) environmentalEvent(entityID string, ingestedAt time.Time) *ingestor.BaseEvent {
	return &ingestor.BaseEvent{
		EventID:       uuid.New(),
		Domain:        "environmental",
		EventType:     "environmental_sensor",
		EntityID:      entityID,
		EntityType:    "sensor",
		OccurredAt:    time.Now().UTC(),
		IngestedAt:    ingestedAt,
		Source:        "test",
		SchemaVersion: "1.0.0",
		Data: &ingestor.EnvironmentalSensor{
			TemperatureC:    22.5,
			HumidityPercent: 65.0,
			AirQualityIndex: 45.0,
		},
	}
}
