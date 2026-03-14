package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"synapsePlatform/internal"
)

const (
	ErrTypeNotFound internal.TypeOfError = iota
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
)

// RequestError provides structured error information for API failures.
type RequestError struct {
	TypeOfError            internal.TypeOfError
	ErrorOccurredBecauseOf ErrorOccurredBecauseOf
	Resource               string // e.g. "event"
	ResourceID             string // e.g. the id from the path — empty for list operations
	Err                    error
}

type ErrorResponse struct {
	Status    int    `json:"status"`
	Error     string `json:"error"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
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

func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Status:    status,
		Error:     http.StatusText(status),
		Message:   message,
		RequestID: requestIDFromContext(r.Context()),
	})
}
