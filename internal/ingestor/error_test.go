package ingestor_test

import (
	"errors"
	"fmt"
	"synapsePlatform/internal/ingestor"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ErrorTestSuite struct {
	suite.Suite
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}

func (s *ErrorTestSuite) TestProcessorError_ImplementsError() {
	var err error = ingestor.ProcessorError{}
	s.NotNil(err)
}

func (s *ErrorTestSuite) TestProcessorError_Error_ContainsField() {
	err := ingestor.ProcessorError{
		Field:                  "device_id",
		Expected:               "string",
		Got:                    42,
		ErrorOccurredBecauseOf: ingestor.ErrFailedToValidateMsg,
		Err:                    errors.New("cannot be empty"),
	}

	result := err.Error()

	s.Contains(result, "device_id")
	s.Contains(result, "string")
	s.Contains(result, "cannot be empty")
}

func (s *ErrorTestSuite) TestProcessorError_ErrorsAs() {
	original := ingestor.ProcessorError{
		Field:    "device_id",
		Expected: "string",
	}

	wrapped := fmt.Errorf("context: %w", original)

	var extracted ingestor.ProcessorError
	s.True(errors.As(wrapped, &extracted))
	s.Equal("device_id", extracted.Field)
}