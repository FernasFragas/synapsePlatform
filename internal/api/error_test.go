package api_test

import (
	"errors"
	"net/http"
	"synapsePlatform/internal/ingestor"
	"testing"

	"synapsePlatform/internal/api"

	"github.com/stretchr/testify/suite"
)

type ErrorTestSuite struct {
	suite.Suite
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}

func (s *ErrorTestSuite) TestRequestError_WithResourceID_ContainsBoth() {
	err := api.RequestError{
		TypeOfError:            api.ErrTypeNotFound,
		ErrorOccurredBecauseOf: api.ErrFailedToGetEvent,
		Resource:               "event",
		ResourceID:             "abc-123",
		Err:                    errors.New("no rows"),
	}

	msg := err.Error()

	s.Contains(msg, "event")
	s.Contains(msg, "abc-123")
	s.Contains(msg, "no rows")
}

func (s *ErrorTestSuite) TestRequestError_WithoutResourceID_OmitsID() {
	err := api.RequestError{
		TypeOfError:            api.ErrTypeInternal,
		ErrorOccurredBecauseOf: api.ErrFailedToListEvents,
		Resource:               "events",
		Err:                    errors.New("timeout"),
	}

	s.NotContains(err.Error(), "with id")
}

func (s *ErrorTestSuite) TestRequestError_Unwrap_ExposesUnderlyingCause() {
	cause := errors.New("root cause")
	err := api.RequestError{Err: cause}

	s.ErrorIs(err, cause)
}

func (s *ErrorTestSuite) TestRequestError_ErrorsAs_ExtractsThroughWrap() {
	original := api.RequestError{
		Resource:   "event",
		ResourceID: "xyz",
	}
	wrapped := errors.Join(errors.New("context"), original)

	var extracted api.RequestError
	s.Require().True(errors.As(wrapped, &extracted))
	s.Equal("xyz", extracted.ResourceID)
}

func (s *ErrorTestSuite) TestHTTPStatus_NotFound_Returns404() {
	err := api.RequestError{TypeOfError: api.ErrTypeNotFound}
	s.Equal(http.StatusNotFound, httpStatus(err)) // see note below
}

func (s *ErrorTestSuite) TestHTTPStatus_BadRequest_Returns400() {
	err := api.RequestError{TypeOfError: api.ErrTypeBadRequest}
	s.Equal(http.StatusBadRequest, httpStatus(err))
}

func (s *ErrorTestSuite) TestHTTPStatus_Internal_Returns500() {
	err := api.RequestError{TypeOfError: api.ErrTypeInternal}
	s.Equal(http.StatusInternalServerError, httpStatus(err))
}

func (s *ErrorTestSuite) TestHTTPStatus_UnknownError_Returns500() {
	s.Equal(http.StatusInternalServerError, httpStatus(errors.New("unknown")))
}

func (s *ErrorTestSuite) TestProcessorError_Unwrap_AllowsErrorsIs() {
	wrapped := ingestor.ProcessorError{Err: ingestor.ErrNilMessage}
	s.ErrorIs(wrapped, ingestor.ErrNilMessage, "errors.Is should traverse Unwrap")
}

func httpStatus(err error) int {
	var reqErr api.RequestError
	if errors.As(err, &reqErr) {
		switch reqErr.TypeOfError { //nolint:exhaustive
		case api.ErrTypeNotFound:
			return http.StatusNotFound
		case api.ErrTypeBadRequest:
			return http.StatusBadRequest
		}
	}

	return http.StatusInternalServerError
}
