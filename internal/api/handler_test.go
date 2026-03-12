package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"synapsePlatform/internal"
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

	validator *utilstest.TokenValidator
	reader    *utilstest.EventReader
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (s *HandlerTestSuite) SetupTest() {
	s.validator = utilstest.NewTokenValidator(s.T())
	s.reader = utilstest.NewEventReader(s.T())
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

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/events"))

	s.Equal(http.StatusOK, rec.Code)
	s.Equal("application/json", rec.Header().Get("Content-Type"))

	var body []map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Len(body, 1)
}

func (s *HandlerTestSuite) TestListEvents_EmptyStore_Returns200WithEmptyArray() {
	s.withScope("read:events")
	s.reader.WithEvents([]*ingestor.BaseEvent{})

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/events"))

	s.Equal(http.StatusOK, rec.Code)
}

func (s *HandlerTestSuite) TestListEvents_MissingScope_Returns403() {
	s.withScope() // valid token, no scopes

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/events"))

	s.Equal(http.StatusForbidden, rec.Code)
}

func (s *HandlerTestSuite) TestListEvents_WrongScope_Returns403() {
	s.withScope("write:events") // wrong scope

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/events"))

	s.Equal(http.StatusForbidden, rec.Code)
}

func (s *HandlerTestSuite) TestListEvents_StorageError_Returns500() {
	s.withScope("read:events")
	s.reader.WithListError(errors.New("db connection lost"))

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/events"))

	s.Equal(http.StatusInternalServerError, rec.Code)
}

// --- GET /events/{id} ---

func (s *HandlerTestSuite) TestGetEvent_ValidTokenWithScope_Returns200() {
	event := validBaseEvent()
	s.withScope("read:events")
	s.reader.WithEvent(event)

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/events/"+event.EventID.String()))

	s.Equal(http.StatusOK, rec.Code)
	s.Equal("application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Equal(event.EventID.String(), body["event_id"])
}

func (s *HandlerTestSuite) TestGetEvent_MissingScope_Returns403() {
	s.withScope("write:events")

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/events/some-id"))

	s.Equal(http.StatusForbidden, rec.Code)
}

func (s *HandlerTestSuite) TestGetEvent_NotFound_Returns404() {
	s.withScope("read:events")
	s.reader.WithGetError(ingestor.ErrEventNotFound)

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/events/missing-id"))

	s.Equal(http.StatusNotFound, rec.Code)
}

func (s *HandlerTestSuite) TestGetEvent_StorageError_Returns500() {
	s.withScope("read:events")
	// A non-not-found error — should be 500, not 404
	s.reader.WithGetError(errors.New("db timeout"))

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, s.authorizedRequest(http.MethodGet, "/events/some-id"))

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
	}
}
