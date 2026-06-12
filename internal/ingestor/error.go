//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"synapsePlatform/internal"
)

// Common errors for the ingestor.
var (
	ErrUnknownDataType       = errors.New("unknown data type")
	ErrNilMessage            = errors.New("message is nil")
	ErrMissingFieldDeviceID  = errors.New("missing required field DeviceId")
	ErrMissingFieldType      = errors.New("missing field Type")
	ErrMissingFieldTimestamp = errors.New("missing field Timestamp")
	ErrEventNotFound         = errors.New("event not found")
)

const (
	ErrValidatingMsg internal.TypeOfError = iota
	ErrStoringMsg
	ErrProcessingMsg
	ErrPollingMsg
	ErrValidatingData
	ErrMarshalingMsg
	ErrUnmarshallingMsg
)

type ErrorOccurredBecauseOf string

const (
	ErrFailedToPollMsg      ErrorOccurredBecauseOf = "failed to poll message"
	ErrFailedToProcessMsg   ErrorOccurredBecauseOf = "failed to process message"
	ErrFailedToValidateMsg  ErrorOccurredBecauseOf = "failed to validate message"
	ErrFailedToStoreMsg     ErrorOccurredBecauseOf = "failed to store message"
	ErrFailedToUnmarshalMsg ErrorOccurredBecauseOf = "failed to unmarshal message"
	ErrFailedToMarshalMsg   ErrorOccurredBecauseOf = "failed to marshal message"
	ErrFailedToValidateData ErrorOccurredBecauseOf = "failed to validate data"
)

// Custom Error types for the ingestor.

// ProcessorError provides detailed field extraction error info
type ProcessorError struct {
	TypeOfError            internal.TypeOfError
	ErrorOccurredBecauseOf ErrorOccurredBecauseOf
	Field                  string
	Expected               string
	Got                    any
	Err                    error
}

func (e ProcessorError) Error() string {
	return fmt.Sprintf("field '%s': expected %s, got %T, because of %s; detailed error: \n %s",
		e.Field, e.Expected, e.Got, e.ErrorOccurredBecauseOf, e.Err)
}

// ErrorClass describes whether retrying an operation could ever succeed. It
// replaces the stringly-typed "transient"/"terminal" values: a real type gives
// the compiler a say, makes the zero value meaningful (transient), and
// documents the contract at every call site:
// RetryableSinkError vs NonRetryableSinkError split.
type ErrorClass int

const (
	// ClassTransient the downstream may recover (broker timeout, DB locked,
	// connection refused). Leave the message uncommitted so Kafka redelivers it.
	ClassTransient ErrorClass = iota

	// ClassTerminal the message is fundamentally broken (malformed payload,
	// missing required field) and will never succeed. Route it to the DLQ and
	// commit so it stops blocking the partition.
	ClassTerminal
)

func (c ErrorClass) String() string {
	if c == ClassTerminal {
		return "terminal"
	}
	return "transient"
}

// classified is implemented by errors that know their own retry semantics.
// Letting the producing layer tag its errors (e.g. the SQLite storer marking
// SQLITE_BUSY transient) keeps the decision next to the code with the most
// context, instead of a central switch that scrapes err.Error() text.
type classified interface {
	ErrorClass() ErrorClass
}

// Classify reports whether err is transient or terminal. It inspects the error
// CHAIN, never the rendered message string, so it stays robust when a
// dependency rewords its errors:
//
//  1. an error that implements classified decides for itself;
//  2. known sentinels (missing fields, unknown type) are terminal;
//  3. a JSON syntax/type error means the payload will never parse, terminal;
//  4. everything else defaults to transient: retrying is safer than dropping
//     data, and a genuinely stuck partition surfaces through lag metrics.
//
// Because the stdlib errors machinery unwraps, this works even when the cause
// is buried inside a ProcessorError (which implements Unwrap): a decode error
// wrapped as a "poll" failure is still classified terminal via its json cause.
func Classify(err error) ErrorClass {
	if err == nil {
		return ClassTransient
	}

	var c classified
	if errors.As(err, &c) {
		return c.ErrorClass()
	}

	switch {
	case errors.Is(err, ErrMissingFieldDeviceID),
		errors.Is(err, ErrMissingFieldType),
		errors.Is(err, ErrMissingFieldTimestamp),
		errors.Is(err, ErrUnknownDataType):

		return ClassTerminal
	}

	// JSON unmarshal errors are always terminal.
	if strings.Contains(err.Error(), "invalid character") ||
		strings.Contains(err.Error(), "unexpected end of JSON") ||
		strings.Contains(err.Error(), "cannot unmarshal") {

		return ClassTerminal
	}
	// Transient: external system might recover.
	if errors.Is(err, sql.ErrConnDone) ||
		errors.Is(err, sql.ErrTxDone) ||
		strings.Contains(err.Error(), "database is locked") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "context canceled") ||
		strings.Contains(err.Error(), "temporary") {

		return ClassTransient
	}

	var (
		typeErr *json.UnmarshalTypeError
	)
	if _, ok := errors.AsType[*json.SyntaxError](err); ok || errors.As(err, &typeErr) {
		return ClassTerminal
	}

	return ClassTransient
}

func (e ProcessorError) Unwrap() error {
	return e.Err
}
