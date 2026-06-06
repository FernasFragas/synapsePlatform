package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"synapsePlatform/internal"
	"synapsePlatform/internal/health"
	"testing"
	"time"

	"synapsePlatform/internal/api"
	"synapsePlatform/internal/auth"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/utilstest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type HandlerTestSuite struct {
	suite.Suite

	validator  *utilstest.TokenValidator
	reader     *utilstest.EventReader
	summarizer *utilstest.Summarizer
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (s *HandlerTestSuite) SetupTest() {
	s.validator = utilstest.NewTokenValidator(s.T())
	s.reader = utilstest.NewEventReader(s.T())
	s.summarizer = utilstest.NewSummarizer(s.T())
}

func (s *HandlerTestSuite) newTestServer() *api.Server {
	return api.NewServer(
		testServerConfig(),
		s.reader,
		nil,
		s.validator,
		noopMiddleware,
		health.NewChecker(time.Second),
		nil,
	)
}

func (s *HandlerTestSuite) newTestServerWithSummarizer() *api.Server {
	return api.NewServer(
		testServerConfig(),
		s.reader,
		s.summarizer, // real mock
		s.validator,
		noopMiddleware,
		health.NewChecker(time.Second),
		nil,
	)
}

var noopMiddleware api.Middleware = func(next http.Handler) http.Handler {
	return next
}

// --- helpers ---

func (s *HandlerTestSuite) authorizedRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Authorization", "Bearer valid.token")
	return req
}

func (s *HandlerTestSuite) withScope(scopes ...string) {
	s.validator.WithIdentity(auth.Identity{Subject: "svc-1", Scopes: scopes})
}

// --- GET /events ---

func (s *HandlerTestSuite) TestListEvents_ValidTokenWithScope_Returns200() {
	s.withScope("read:events")
	s.reader.WithEvents([]*ingestor.BaseEvent{validBaseEvent()})

	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/events"))

	s.Equal(http.StatusOK, rec.Code)
	s.Equal("application/json", rec.Header().Get("Content-Type"))

	var body struct {
		Data       []map[string]any `json:"data"`
		NextCursor string           `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
	}
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Len(body.Data, 1)
}

func (s *HandlerTestSuite) TestListEvents_EmptyStore_Returns200WithEmptyArray() {
	s.withScope("read:events")
	s.reader.WithEvents([]*ingestor.BaseEvent{})

	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/events"))

	s.Equal(http.StatusOK, rec.Code)
}

func (s *HandlerTestSuite) TestListEvents_MissingScope_Returns403() {
	s.withScope() // valid token, no scopes

	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/events"))

	s.Equal(http.StatusForbidden, rec.Code)
}

func (s *HandlerTestSuite) TestListEvents_WrongScope_Returns403() {
	s.withScope("write:events") // wrong scope

	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/events"))

	s.Equal(http.StatusForbidden, rec.Code)
}

func (s *HandlerTestSuite) TestListEvents_StorageError_Returns500() {
	s.withScope("read:events")
	s.reader.WithListError(errors.New("db connection lost"))

	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/events"))

	s.Equal(http.StatusInternalServerError, rec.Code)
}

// --- GET /events/{id} ---

func (s *HandlerTestSuite) TestGetEvent_ValidTokenWithScope_Returns200() {
	event := validBaseEvent()
	s.withScope("read:events")
	s.reader.WithEvent(event)

	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/events/"+event.EventID.String()))

	s.Equal(http.StatusOK, rec.Code)
	s.Equal("application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Equal(event.EventID.String(), body["event_id"])
}

func (s *HandlerTestSuite) TestGetEvent_MissingScope_Returns403() {
	s.withScope("write:events")

	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/events/some-id"))

	s.Equal(http.StatusForbidden, rec.Code)
}

func (s *HandlerTestSuite) TestGetEvent_NotFound_Returns404() {
	s.withScope("read:events")
	s.reader.WithGetError(ingestor.ErrEventNotFound)

	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/events/missing-id"))

	s.Equal(http.StatusNotFound, rec.Code)
}

func (s *HandlerTestSuite) TestGetEvent_StorageError_Returns500() {
	s.withScope("read:events")
	// A non-not-found error — should be 500, not 404
	s.reader.WithGetError(errors.New("db timeout"))

	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/events/some-id"))

	s.Equal(http.StatusInternalServerError, rec.Code)
}

// --- GET /summary ---

func (s *HandlerTestSuite) TestGetSummary_ValidTokenWithScope_Returns200() {
	s.withScope("read:events")

	report := &api.Report{
		Domain:     "energy",
		WindowFrom: time.Now().UTC().Add(-24 * time.Hour),
		Model:      "llama3.2:3b",
		Content:    "There were 42 energy events.",
		CreatedAt:  time.Now().UTC(),
	}
	s.summarizer.WithReport(report)

	srv := s.newTestServerWithSummarizer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/summary?domain=energy"))

	s.Equal(http.StatusOK, rec.Code)
	s.Equal("application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Equal("energy", body["domain"])
	s.Equal("llama3.2:3b", body["model"])
	s.Equal("There were 42 energy events.", body["content"])
}

func (s *HandlerTestSuite) TestGetSummary_MissingScope_Returns403() {
	s.withScope() // no scopes

	srv := s.newTestServerWithSummarizer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/summary"))

	s.Equal(http.StatusForbidden, rec.Code)
}

func (s *HandlerTestSuite) TestGetSummary_SummarizerNil_Returns503() {
	s.withScope("read:events")
	// Use the old newTestServer which passes nil for summarizer
	srv := s.newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/summary"))

	s.Equal(http.StatusServiceUnavailable, rec.Code)
}

func (s *HandlerTestSuite) TestGetSummary_SummarizeError_Returns500() {
	s.withScope("read:events")
	s.summarizer.WithError(errors.New("ollama connection refused"))

	srv := s.newTestServerWithSummarizer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/v1/summary"))

	s.Equal(http.StatusInternalServerError, rec.Code)
}

// --- shared fixture ---

func validBaseEvent() *ingestor.BaseEvent {
	return &ingestor.BaseEvent{
		EventID:       uuid.New(),
		Domain:        "energy",
		EventType:     "energy_meter",
		EntityID:      "device-001",
		EntityType:    "sensor",
		OccurredAt:    time.Now().UTC(),
		IngestedAt:    time.Now().UTC(),
		Source:        "iot-gateway",
		SchemaVersion: "1.0.0",
		Data:          &ingestor.EnergyReading{PowerW: 100, EnergyWh: 500, VoltageV: 220, CurrentMA: 455},
	}
}

func testServerConfig() internal.ServerConfig {
	return internal.ServerConfig{
		Address: ":0",
		RateLimit: internal.RateLimitConfig{
			RequestsPerSecond: 100,
			Burst:             100,
		},
	}
}
