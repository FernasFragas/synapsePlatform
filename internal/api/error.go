package api

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors — used by handlers and tests with errors.Is.
var (
	ErrInvalidEventID = errors.New("event id must be a valid UUID")
)

// TypeOfError classifies the category of failure — maps directly to HTTP status codes.
type TypeOfError int

const (
	ErrTypeNotFound  TypeOfError = iota
	ErrTypeBadRequest
	ErrTypeInternal
	ErrTypeEncoding
)

// ErrorOccurredBecauseOf is the specific reason within a category.
type ErrorOccurredBecauseOf string

const (
	ErrFailedToGetEvent       ErrorOccurredBecauseOf = "failed to get event"
	ErrFailedToListEvents     ErrorOccurredBecauseOf = "failed to list events"
	ErrFailedToEncodeResponse ErrorOccurredBecauseOf = "failed to encode response"
	ErrInvalidPathParam       ErrorOccurredBecauseOf = "invalid path parameter"
)

// RequestError provides structured error information for API failures.
type RequestError struct {
	TypeOfError            TypeOfError
	ErrorOccurredBecauseOf ErrorOccurredBecauseOf
	Resource               string // e.g. "event"
	ResourceID             string // e.g. the id from the path — empty for list operations
	Err                    error
}

func (e RequestError) Error() string {
	if e.ResourceID != "" {
		return fmt.Sprintf("resource '%s' with id '%s': %s; cause: %s",
			e.Resource, e.ResourceID, e.ErrorOccurredBecauseOf, e.Err)
	}

	return fmt.Sprintf("resource '%s': %s; cause: %s",
		e.Resource, e.ErrorOccurredBecauseOf, e.Err)
}

// Unwrap allows errors.Is and errors.As to inspect the underlying cause.
func (e RequestError) Unwrap() error {
	return e.Err
}

// httpStatus maps a RequestError's TypeOfError to the correct HTTP status code.
// Falls back to 500 for unknown error types.
func httpStatus(err error) int {
	var reqErr RequestError
	if errors.As(err, &reqErr) {
		switch reqErr.TypeOfError { //nolint:exhaustive
		case ErrTypeNotFound:
			return http.StatusNotFound
		case ErrTypeBadRequest:
			return http.StatusBadRequest
		}
	}

	return http.StatusInternalServerError
}
