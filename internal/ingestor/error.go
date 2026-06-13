//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"errors"
	"fmt"
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

func (e ProcessorError) Unwrap() error {
	return e.Err
}

type ErrorClass int

const (
	ClassTransient ErrorClass = iota
	ClassTerminal
)

type ClassifiedError interface {
	error
	ErrorClass() ErrorClass
}

type classifiedError struct {
	class ErrorClass
	err   error
}

func (e classifiedError) Error() string { return e.err.Error() }
func (e classifiedError) Unwrap() error { return e.err }
func (e classifiedError) ErrorClass() ErrorClass {
	return e.class
}

func NewTransientError(err error) error {
	if err == nil {
		return nil
	}

	return classifiedError{class: ClassTransient, err: err}
}

func NewTerminalError(err error) error {
	if err == nil {
		return nil
	}

	return classifiedError{class: ClassTerminal, err: err}
}

func Classify(err error) ErrorClass {
	if err == nil {
		return ClassTransient
	}

	var classified ClassifiedError
	if errors.As(err, &classified) {
		return classified.ErrorClass()
	}

	switch {
	case errors.Is(err, ErrMissingFieldDeviceID),
		errors.Is(err, ErrMissingFieldType),
		errors.Is(err, ErrMissingFieldTimestamp),
		errors.Is(err, ErrUnknownDataType):
		return ClassTerminal
	default:
		return ClassTransient
	}
}

func (c ErrorClass) String() string {
	switch c {
	case ClassTerminal:
		return "terminal"
	default:
		return "transient"
	}
}
