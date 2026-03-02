package ingestor_test

import (
	"strings"
	"synapsePlatform/internal/ingestor"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	ValidEntityID      = "device-001"
	ValidDomain        = "energy"
	ValidEventType     = "energy_meter"
	ValidEntityType    = "sensor"
	ValidSource        = "mqtt-bridge"
	ValidSchemaVersion = "1.0.0"
)

type NormalizedMessageTestSuite struct {
	suite.Suite
}

func TestNormalizedMessageSuite(t *testing.T) {
	suite.Run(t, new(NormalizedMessageTestSuite))
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_HappyPath_ReturnsNil() {
	event := validBaseEvent(validEnergyReading())

	err := event.Validate()

	s.NoError(err, "a fully populated BaseEvent with valid data should pass validation")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_MissingEventID_ReturnsProcessorError() {
	event := validBaseEvent(validEnergyReading())
	event.EventID = uuid.UUID{} // zero UUID

	err := event.Validate()

	s.Require().Error(err, "missing EventID should fail validation")

	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr, "error should be wrapped in ProcessorError")
	s.Equal(ingestor.ErrValidatingData, procErr.TypeOfError, "error type should be ErrValidatingData")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_MissingDomain_ReturnsProcessorError() {
	event := validBaseEvent(validEnergyReading())
	event.Domain = ""

	err := event.Validate()

	s.Require().Error(err, "missing Domain should fail validation")

	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr, "error should be wrapped in ProcessorError")
	s.Equal(ingestor.ErrValidatingData, procErr.TypeOfError)
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_MissingEventType_ReturnsProcessorError() {
	event := validBaseEvent(validEnergyReading())
	event.EventType = ""
	s.Require().Error(event.Validate(), "missing EventType should fail validation")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_MissingEntityID_ReturnsProcessorError() {
	event := validBaseEvent(validEnergyReading())
	event.EntityID = ""
	s.Require().Error(event.Validate(), "missing EntityID should fail validation")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_EntityIDTooLong_ReturnsProcessorError() {
	// --- Arrange ---
	event := validBaseEvent(validEnergyReading())
	event.EntityID = strings.Repeat("x", 256) // max=255

	// --- Act + Assert ---
	s.Require().Error(event.Validate(), "EntityID longer than 255 chars should fail validation")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_MissingEntityType_ReturnsProcessorError() {
	event := validBaseEvent(validEnergyReading())
	event.EntityType = ""
	s.Require().Error(event.Validate(), "missing EntityType should fail validation")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_ZeroOccurredAt_ReturnsProcessorError() {
	event := validBaseEvent(validEnergyReading())
	event.OccurredAt = time.Time{}
	s.Require().Error(event.Validate(), "zero OccurredAt should fail validation")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_ZeroIngestedAt_ReturnsProcessorError() {
	event := validBaseEvent(validEnergyReading())
	event.IngestedAt = time.Time{}
	s.Require().Error(event.Validate(), "zero IngestedAt should fail validation")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_MissingSource_ReturnsProcessorError() {
	event := validBaseEvent(validEnergyReading())
	event.Source = ""
	s.Require().Error(event.Validate(), "missing Source should fail validation")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_InvalidSchemaVersion_ReturnsProcessorError() {
	event := validBaseEvent(validEnergyReading())
	event.SchemaVersion = "1.0" // not semver — requires format "MAJOR.MINOR.PATCH"

	s.Require().Error(event.Validate(), "non-semver SchemaVersion should fail validation")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_Validate_InvalidDataPayload_ReturnsError() {
	invalidData := &ingestor.EnergyReading{
		PowerW:    0, // required → fails
		EnergyWh:  0,
		VoltageV:  0,
		CurrentMA: 0,
	}
	event := validBaseEvent(invalidData)

	s.Require().Error(event.Validate(), "invalid nested data should cause Validate to fail")
}

func validEnergyReading() *ingestor.EnergyReading {
	return &ingestor.EnergyReading{
		PowerW:    100,
		EnergyWh:  500,
		VoltageV:  220,
		CurrentMA: 455,
	}
}

func validBaseEvent(data ingestor.NormalizedData) ingestor.BaseEvent {
	return ingestor.BaseEvent{
		EventID:       uuid.New(),
		Domain:        ValidDomain,
		EventType:     ValidEventType,
		EntityID:      ValidEntityID,
		EntityType:    ValidEntityType,
		OccurredAt:    time.Now().UTC(),
		IngestedAt:    time.Now().UTC(),
		Source:        ValidSource,
		SchemaVersion: ValidSchemaVersion,
		Data:          data,
	}
}
