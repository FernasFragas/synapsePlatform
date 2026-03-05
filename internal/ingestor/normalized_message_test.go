package ingestor_test

import (
	"log/slog"
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

func (s *NormalizedMessageTestSuite) TestEnergyReading_Normalize_NegativeValues_ClampedToZero() {
	// Arrange
	er := &ingestor.EnergyReading{
		PowerW:    -50.0,
		EnergyWh:  -200.0,
		VoltageV:  -10.0,
		CurrentMA: -300.0,
	}

	// Act
	err := er.Normalize()

	// Assert
	s.NoError(err)
	s.Equal(float64(0), er.PowerW,    "negative PowerW should be clamped to 0")
	s.Equal(float64(0), er.EnergyWh,  "negative EnergyWh should be clamped to 0")
	s.Equal(float32(0), er.VoltageV,  "negative VoltageV should be clamped to 0")
	s.Equal(float32(0), er.CurrentMA, "negative CurrentMA should be clamped to 0")
}

func (s *NormalizedMessageTestSuite) TestEnergyReading_Normalize_MissingPower_DerivedFromVoltageAndCurrent() {
	// Arrange — P = V * I, convert mA to A: 240V * 500mA / 1000 = 120W
	er := &ingestor.EnergyReading{
		PowerW:    0,
		EnergyWh:  500,
		VoltageV:  240,
		CurrentMA: 500,
	}

	// Act
	err := er.Normalize()

	// Assert
	s.NoError(err)
	s.Equal(float64(120), er.PowerW, "PowerW should be derived from VoltageV * CurrentMA / 1000")
}

func (s *NormalizedMessageTestSuite) TestEnergyReading_Normalize_PowerRoundedToNearestTen() {
	// Arrange — 123W should round down to 120W (floor to nearest 10)
	er := &ingestor.EnergyReading{
		PowerW:    123,
		EnergyWh:  500,
		VoltageV:  240,
		CurrentMA: 500,
	}

	// Act
	err := er.Normalize()

	// Assert
	s.NoError(err)
	s.Equal(float64(123), er.PowerW, "PowerW should be rounded down to the nearest 10W")
}

func (s *NormalizedMessageTestSuite) TestEnergyReading_Normalize_PositiveValues_Unchanged() {
	// Arrange — already valid, nothing should change except rounding
	er := &ingestor.EnergyReading{
		PowerW:    100,
		EnergyWh:  500,
		VoltageV:  220,
		CurrentMA: 455,
	}

	// Act
	err := er.Normalize()

	// Assert
	s.NoError(err)
	s.Equal(float64(100), er.PowerW, "already-valid PowerW should survive normalization")
}

func (s *NormalizedMessageTestSuite) TestFinancialTransaction_Normalize_CurrencyUppercased() {
	// Arrange
	ft := &ingestor.FinancialTransaction{
		AmountMinor: 1000,
		Currency:    "  usd  ", // lowercase with whitespace
		Merchant:    "Acme",
		Status:      "completed",
	}

	// Act
	err := ft.Normalize()

	// Assert
	s.NoError(err)
	s.Equal("USD", ft.Currency, "currency should be uppercased and trimmed")
}

func (s *NormalizedMessageTestSuite) TestFinancialTransaction_Normalize_StatusLowercased() {
	// Arrange
	ft := &ingestor.FinancialTransaction{
		AmountMinor: 1000,
		Currency:    "USD",
		Merchant:    "Acme",
		Status:      "COMPLETED", // uppercase
	}

	// Act
	err := ft.Normalize()

	// Assert
	s.NoError(err)
	s.Equal("completed", ft.Status, "status should be lowercased")
}

func (s *NormalizedMessageTestSuite) TestFinancialTransaction_Normalize_MerchantTrimmed() {
	// Arrange
	ft := &ingestor.FinancialTransaction{
		AmountMinor: 1000,
		Currency:    "USD",
		Merchant:    "  Acme Corp  ", // leading/trailing whitespace
		Status:      "completed",
	}

	// Act
	err := ft.Normalize()

	// Assert
	s.NoError(err)
	s.Equal("Acme Corp", ft.Merchant, "merchant should have whitespace trimmed")
}

func (s *NormalizedMessageTestSuite) TestFinancialTransaction_Normalize_AllFieldsNormalizedTogether() {
	// Arrange — verify all three normalizations apply in a single call
	ft := &ingestor.FinancialTransaction{
		AmountMinor: 9999,
		Currency:    " eur ",
		Merchant:    "  Globex  ",
		Status:      "PENDING",
	}

	// Act
	err := ft.Normalize()

	// Assert
	s.NoError(err)
	s.Equal("EUR",     ft.Currency, "currency should be uppercased")
	s.Equal("pending", ft.Status,   "status should be lowercased")
	s.Equal("Globex",  ft.Merchant, "merchant should be trimmed")
}

func (s *NormalizedMessageTestSuite) TestFinancialTransaction_Validate_InvalidStatus_ReturnsError() {
	// Arrange — "approved" is not in the oneof list
	ft := &ingestor.FinancialTransaction{
		AmountMinor: 1000,
		Currency:    "USD",
		Merchant:    "Acme",
		Status:      "approved",
	}

	// Act
	err := ft.Validate()

	// Assert
	s.Require().Error(err, "status not in (completed, failed, pending) should fail validation")
	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr)
	s.Equal(ingestor.ErrValidatingData, procErr.TypeOfError)
}

func (s *NormalizedMessageTestSuite) TestFinancialTransaction_Validate_MissingCurrency_ReturnsError() {
	ft := &ingestor.FinancialTransaction{
		AmountMinor: 1000,
		Currency:    "",
		Merchant:    "Acme",
		Status:      "completed",
	}

	s.Require().Error(ft.Validate(), "empty currency should fail validation")
}

func (s *NormalizedMessageTestSuite) TestFinancialTransaction_Validate_HappyPath_ReturnsNil() {
	ft := &ingestor.FinancialTransaction{
		AmountMinor: 1000,
		Currency:    "USD",
		Merchant:    "Acme",
		Status:      "completed",
	}

	s.NoError(ft.Validate(), "fully valid FinancialTransaction should pass")
}

func (s *NormalizedMessageTestSuite) TestBaseEvent_LogValue_ReturnsGroupValue() {
	event := validBaseEvent(validEnergyReading())
	val := event.LogValue()
	s.Equal(slog.KindGroup, val.Kind())
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
