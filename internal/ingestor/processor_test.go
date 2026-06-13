package ingestor_test

import (
	"context"
	"errors"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/utilstest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const (
	TestDeviceID = "device-001"
	TestType     = "energy_meter"
)

type ProcessorTestSuite struct {
	suite.Suite

	poller  *utilstest.MessagePoller
	subject *ingestor.Processor
}

func TestProcessorSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}

func (s *ProcessorTestSuite) SetupTest() {
	s.poller = utilstest.NewMessagePoller(s.T())
	s.subject = ingestor.NewProcessor(s.poller)
}

func (s *ProcessorTestSuite) TestProcessData_ValidMessage_ReturnsMessage() {
	msg := &ingestor.DeviceMessage{
		DeviceID:  TestDeviceID,
		Type:      TestType,
		Timestamp: time.Now(),
	}
	s.poller.WithResult(msg)

	delivery, err := s.subject.ProcessData(context.Background())

	s.Require().NoError(err, "valid message should not produce an error")
	s.Require().NotNil(delivery)
	s.Equal(msg, delivery.Message, "returned message should be the one from the poller")
}

func (s *ProcessorTestSuite) TestProcessData_PollerError_ReturnsProcessorError() {
	pollerErr := errors.New("broker unavailable")
	s.poller.WithError(pollerErr)

	delivery, err := s.subject.ProcessData(context.Background())

	s.Nil(delivery, "delivery should be nil when poller fails")
	s.Require().Error(err, "poller error should be propagated")

	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr, "error should be wrapped in ProcessorError")
	s.Equal(ingestor.ErrPollingMsg, procErr.TypeOfError, "error type should be ErrPollingMsg")
	s.Equal(ingestor.ErrFailedToPollMsg, procErr.ErrorOccurredBecauseOf, "error reason should be ErrFailedToPollMsg")
	s.Equal("delivery", procErr.Field, "field should identify the delivery field")
	s.ErrorIs(procErr.Err, pollerErr, "original poller error should be preserved in the chain")
}

func (s *ProcessorTestSuite) TestProcessData_TerminalPollErrorWithDelivery_ReturnsDelivery() {
	delivery := &ingestor.Delivery{
		Metadata: ingestor.MessageMetadata{Source: "kafka"},
	}
	pollerErr := errors.New("bad payload")
	s.poller.WithTerminalDecodeFailure(delivery, pollerErr)

	actual, err := s.subject.ProcessData(context.Background())

	s.Require().Error(err)
	s.Same(delivery, actual, "terminal poll errors with a delivery must keep the delivery for failure storage and ack")

	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr)
	s.Equal(ingestor.ErrPollingMsg, procErr.TypeOfError)
	s.ErrorIs(procErr.Err, pollerErr)
}

func (s *ProcessorTestSuite) TestProcessData_PollerContextCancelled_ReturnsProcessorError() {
	s.poller.WithError(context.Canceled)

	delivery, err := s.subject.ProcessData(context.Background())

	s.Nil(delivery, "delivery should be nil on context cancellation")
	s.Require().Error(err, "cancelled context error should be propagated")

	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr, "error should be wrapped in ProcessorError")
	s.Equal(ingestor.ErrPollingMsg, procErr.TypeOfError)
	s.ErrorIs(procErr.Err, context.Canceled, "wrapped error should be context.Canceled")
}

func (s *ProcessorTestSuite) TestProcessData_NilMessage_ReturnsProcessorError() {
	s.poller.WithNoResult()

	delivery, err := s.subject.ProcessData(context.Background())

	s.Nil(delivery, "delivery should be nil when poller returns nil message")
	s.Require().Error(err, "nil message should produce an error")

	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr, "error should be wrapped in ProcessorError")
	s.Equal(ingestor.ErrProcessingMsg, procErr.TypeOfError)
	s.Equal(ingestor.ErrFailedToProcessMsg, procErr.ErrorOccurredBecauseOf)
	s.ErrorIs(procErr.Err, ingestor.ErrNilMessage, "wrapped error should be ErrNilMessage")
}

func (s *ProcessorTestSuite) TestProcessData_MissingDeviceID_ReturnsValidationError() {
	s.poller.WithResult(&ingestor.DeviceMessage{
		DeviceID:  "",
		Type:      TestType,
		Timestamp: time.Now(),
	})

	delivery, err := s.subject.ProcessData(context.Background())

	s.Require().NotNil(delivery, "delivery should be returned when validation fails so it can be acked after DLQ")
	s.Require().Error(err, "missing DeviceID should produce a validation error")

	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr)
	s.Equal(ingestor.ErrValidatingMsg, procErr.TypeOfError)
	s.Equal(ingestor.ErrFailedToValidateMsg, procErr.ErrorOccurredBecauseOf)
	s.ErrorIs(procErr.Err, ingestor.ErrMissingFieldDeviceID)
}

func (s *ProcessorTestSuite) TestProcessData_MissingType_ReturnsValidationError() {
	s.poller.WithResult(&ingestor.DeviceMessage{
		DeviceID:  TestDeviceID,
		Type:      "",
		Timestamp: time.Now(),
	})

	delivery, err := s.subject.ProcessData(context.Background())

	s.Require().NotNil(delivery)
	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr)
	s.Equal(ingestor.ErrValidatingMsg, procErr.TypeOfError)
	s.ErrorIs(procErr.Err, ingestor.ErrMissingFieldType)
}

func (s *ProcessorTestSuite) TestProcessData_MissingTimestamp_ReturnsValidationError() {
	s.poller.WithResult(&ingestor.DeviceMessage{
		DeviceID:  TestDeviceID,
		Type:      TestType,
		Timestamp: time.Time{},
	})

	delivery, err := s.subject.ProcessData(context.Background())

	s.Require().NotNil(delivery)
	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr)
	s.Equal(ingestor.ErrValidatingMsg, procErr.TypeOfError)
	s.ErrorIs(procErr.Err, ingestor.ErrMissingFieldTimestamp)
}
