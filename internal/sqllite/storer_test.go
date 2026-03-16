package sqllite_test

import (
	"context"
	"errors"
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

func (s *StorerTestSuite) TestStoreData_DuplicateEventID_ReturnsError() {
	event := s.energyEvent("device-1", time.Now().UTC())

	s.Require().NoError(s.repo.StoreData(s.ctx, event))

	err := s.repo.StoreData(s.ctx, event)
	s.Error(err)
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
		Stage:   "transform",
		Message: msg,
		Err:     errors.New("schema mismatch"),
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
		Stage: "process",
		Err:   errors.New("broker down"),
	})
	s.NoError(err)
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
