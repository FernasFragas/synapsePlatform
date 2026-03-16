package api_test

import (
	"net/http"
	"net/http/httptest"
	"synapsePlatform/internal/health"
	"testing"
	"time"

	"synapsePlatform/internal/api"
	"synapsePlatform/internal/auth"
	"synapsePlatform/internal/utilstest"

	"github.com/stretchr/testify/suite"
)

func noopMiddleware(next http.Handler) http.Handler { return next }

type MiddlewareTestSuite struct {
	suite.Suite

	validator *utilstest.TokenValidator
	reader    *utilstest.EventReader
}

func TestMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

func (s *MiddlewareTestSuite) SetupTest() {
	s.validator = utilstest.NewTokenValidator(s.T())
	s.reader = utilstest.NewEventReader(s.T())
}

func (s *MiddlewareTestSuite) TestAuthenticate_NoAuthHeader_Returns401WithWWWAuthenticate() {
	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware, health.NewChecker(time.Second))
	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
	s.Equal(`Bearer realm="api"`, rec.Header().Get("WWW-Authenticate"))
}

func (s *MiddlewareTestSuite) TestAuthenticate_WrongScheme_Returns401() {
	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware, health.NewChecker(time.Second))
	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *MiddlewareTestSuite) TestAuthenticate_BearerLowercase_IsAccepted() {
	s.validator.WithIdentity(auth.Identity{
		Subject: "user-1",
		Scopes:  []string{"read:events"},
	})
	s.reader.WithEvents(nil)

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware, health.NewChecker(time.Second))
	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	req.Header.Set("Authorization", "bearer some-token")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
}

func (s *MiddlewareTestSuite) TestAuthenticate_InvalidToken_Returns401WithWWWAuthenticate() {
	s.validator.WithError(auth.ValidationError{
		TypeOfError:            auth.ErrTypeTokenInvalid,
		ErrorOccurredBecauseOf: auth.ErrTokenSignatureInvalid,
	})

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware, health.NewChecker(time.Second))
	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	req.Header.Set("Authorization", "Bearer tampered.token.here")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
	s.Equal(`Bearer error="invalid_token"`, rec.Header().Get("WWW-Authenticate"))
}

func (s *MiddlewareTestSuite) TestAuthenticate_ExpiredToken_Returns401() {
	s.validator.WithError(auth.ValidationError{
		TypeOfError:            auth.ErrTypeTokenExpired,
		ErrorOccurredBecauseOf: auth.ErrTokenExpired,
	})

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware, health.NewChecker(time.Second))
	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	req.Header.Set("Authorization", "Bearer expired.token")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *MiddlewareTestSuite) TestAuthenticate_ValidToken_PassesIdentityInContext() {
	expected := auth.Identity{Subject: "user-1", Scopes: []string{"read:events"}}
	s.validator.WithIdentity(expected)
	s.reader.WithEvents(nil)

	srv := api.NewServer(testServerConfig(), s.reader, s.validator, noopMiddleware, health.NewChecker(time.Second))
	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	req.Header.Set("Authorization", "Bearer valid.token")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
}
