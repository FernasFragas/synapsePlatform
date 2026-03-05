package ingestor_test

import (
	"context"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/utilstest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

const TransformerDeviceID = "device-001"

type TransformerTestSuite struct {
	suite.Suite

	realRepo *utilstest.TestRepo
}

func TestTransformerSuite(t *testing.T) {
	suite.Run(t, new(TransformerTestSuite))
}

func (s *TransformerTestSuite) SetupTest() {
	// TestRepo is created here so every test can optionally use the real DB.
	s.realRepo = utilstest.NewTestRepo(s.T())
}

func (s *TransformerTestSuite) TestTransform_EmptyDomainsList_AllowsAnyDomain() {
	transformer := ingestor.NewMessageTransformer([]ingestor.DataTypes{})

	event, err := transformer.Transform(energyMeterMsg())

	s.Require().NoError(err, "empty domain filter should allow any message type through")
	s.NotNil(event)
}

func (s *TransformerTestSuite) TestTransform_NilDomainsList_AllowsAnyDomain() {
	transformer := ingestor.NewMessageTransformer(nil)

	event, err := transformer.Transform(energyMeterMsg())

	s.Require().NoError(err, "nil domain filter should behave like empty — allow all")
	s.NotNil(event)
}

func (s *TransformerTestSuite) TestTransform_ConfiguredDomain_AllowsThrough() {
	transformer := ingestor.NewMessageTransformer([]ingestor.DataTypes{
		ingestor.DataTypeEnergyMeter,
	})

	event, err := transformer.Transform(energyMeterMsg())

	s.Require().NoError(err, "energy_meter is configured as supported — should be allowed")
	s.NotNil(event)
}

func (s *TransformerTestSuite) TestTransform_UnconfiguredDomain_ReturnsProcessorError() {
	transformer := ingestor.NewMessageTransformer([]ingestor.DataTypes{
		ingestor.DataTypeFinancialStream,
	})

	event, err := transformer.Transform(energyMeterMsg())

	s.Nil(event, "no event should be returned for an unsupported domain")
	s.Require().Error(err, "unsupported domain must return an error")

	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr, "error should be wrapped in ProcessorError")
	s.Equal(ingestor.ErrValidatingData, procErr.TypeOfError,
		"domain rejection should use ErrValidatingData type")
	s.Equal("domain", procErr.Field,
		"error field should identify 'domain' as the offending field")
}

func (s *TransformerTestSuite) TestTransform_MultipleSupportedDomains_RejectsUnlisted() {
	transformer := ingestor.NewMessageTransformer([]ingestor.DataTypes{
		ingestor.DataTypeEnergyMeter,
		ingestor.DataTypeEnvironmentalSensor,
	})

	event, err := transformer.Transform(financialMsg())

	s.Nil(event, "financial_stream should be rejected when not in the supported list")
	s.Require().Error(err, "unlisted domain should return an error")
}

func (s *TransformerTestSuite) TestTransform_DeviceID_MappedToEntityID() {
	transformer := ingestor.NewMessageTransformer(nil)
	msg := energyMeterMsg()

	event, err := transformer.Transform(msg)

	s.Require().NoError(err)
	s.Equal(TransformerDeviceID, event.EntityID,
		"msg.DeviceID should be mapped to event.EntityID")
}

func (s *TransformerTestSuite) TestTransform_Timestamp_MappedToOccurredAt() {
	transformer := ingestor.NewMessageTransformer(nil)
	fixedTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	msg := energyMeterMsg()
	msg.Timestamp = fixedTime

	event, err := transformer.Transform(msg)

	s.Require().NoError(err)
	s.Equal(fixedTime, event.OccurredAt,
		"msg.Timestamp should be mapped to event.OccurredAt unchanged")
}

func (s *TransformerTestSuite) TestTransform_SchemaVersion_AlwaysV1() {
	transformer := ingestor.NewMessageTransformer(nil)

	event, err := transformer.Transform(energyMeterMsg())

	s.Require().NoError(err)
	s.Equal(ingestor.SchemaVersion1, event.SchemaVersion,
		"schema version should always be set to the current SchemaVersion1 constant")
}

func (s *TransformerTestSuite) TestTransform_EventID_IsNonZeroUUID() {
	transformer := ingestor.NewMessageTransformer(nil)

	event, err := transformer.Transform(energyMeterMsg())

	s.Require().NoError(err)
	s.NotEqual(uuid.UUID{}, event.EventID,
		"each Transform call must assign a new non-zero EventID")
}

func (s *TransformerTestSuite) TestTransform_TwoCalls_ProduceDifferentEventIDs() {
	transformer := ingestor.NewMessageTransformer(nil)

	e1, err1 := transformer.Transform(energyMeterMsg())
	e2, err2 := transformer.Transform(energyMeterMsg())

	s.Require().NoError(err1)
	s.Require().NoError(err2)
	s.NotEqual(e1.EventID, e2.EventID, "consecutive calls must produce unique EventIDs")
}

func (s *TransformerTestSuite) TestTransform_EnergyMeterType_DomainIsEnergyMeter() {
	transformer := ingestor.NewMessageTransformer(nil)

	event, err := transformer.Transform(energyMeterMsg())

	s.Require().NoError(err)
	s.Equal(ingestor.DataTypeEnergyMeter.String(), event.Domain)
	s.Equal(string(ingestor.DataTypeEnergyMeter), event.EventType)
}

func (s *TransformerTestSuite) TestTransform_FinancialStreamType_DomainIsFinancialStream() {
	transformer := ingestor.NewMessageTransformer(nil)

	event, err := transformer.Transform(financialMsg())

	s.Require().NoError(err)
	s.Equal(ingestor.DataTypeFinancialStream.String(), event.Domain)
}

func (s *TransformerTestSuite) TestTransform_EnvironmentalSensorType_DomainIsEnvironmental() {
	transformer := ingestor.NewMessageTransformer(nil)

	event, err := transformer.Transform(environmentalMsg())

	s.Require().NoError(err)
	s.Equal(ingestor.DataTypeEnvironmentalSensor.String(), event.Domain)
}

func (s *TransformerTestSuite) TestTransform_UnknownMsgType_MapsToUnknownDomain() {
	transformer := ingestor.NewMessageTransformer(nil)
	msg := &ingestor.DeviceMessage{
		DeviceID:  TransformerDeviceID,
		Type:      "completely_unknown_device",
		Timestamp: time.Now(),
		Metrics:   map[string]any{},
	}

	event, err := transformer.Transform(msg)

	s.Require().NoError(err, "unknown types should map to DataTypeUnknown without error")
	s.Equal(ingestor.DataTypeUnknown.String(), event.Domain)
}

func (s *TransformerTestSuite) TestTransform_EmptyMetrics_EnergyMeter_ValidationFails() {
	transformer := ingestor.NewMessageTransformer(nil)
	msg := &ingestor.DeviceMessage{
		DeviceID:  TransformerDeviceID,
		Type:      string(ingestor.DataTypeEnergyMeter),
		Timestamp: time.Now(),
		Metrics:   map[string]any{},
	}

	event, err := transformer.Transform(msg)

	s.Nil(event)
	s.Require().Error(err, "zero-value EnergyReading should fail required constraints")

	var procErr ingestor.ProcessorError
	s.Require().ErrorAs(err, &procErr)
	s.Equal(ingestor.ErrValidatingData, procErr.TypeOfError)
}

func (s *TransformerTestSuite) TestTransform_EmptyMetrics_FinancialStream_ValidationFails() {
	transformer := ingestor.NewMessageTransformer(nil)
	msg := &ingestor.DeviceMessage{
		DeviceID:  TransformerDeviceID,
		Type:      string(ingestor.DataTypeFinancialStream),
		Timestamp: time.Now(),
		Metrics:   map[string]any{},
	}

	_, err := transformer.Transform(msg)

	s.Require().Error(err, "empty FinancialTransaction fields should fail required constraints")
}

func (s *TransformerTestSuite) TestTransform_EnergyMeter_OutputCanBeStoredInDB() {
	// --- Arrange ---
	repo := s.realRepo.Build()
	transformer := ingestor.NewMessageTransformer(nil)

	// --- Act ---
	event, err := transformer.Transform(energyMeterMsg())
	s.Require().NoError(err, "transform must succeed before we can test persistence")

	storeErr := repo.StoreData(context.Background(), event)

	// --- Assert ---
	s.NoError(storeErr,
		"an event produced by Transform should be storable in the DB without schema errors")
}

func (s *TransformerTestSuite) TestTransform_FinancialStream_OutputCanBeStoredInDB() {
	// --- Arrange ---
	repo := s.realRepo.Build()
	transformer := ingestor.NewMessageTransformer(nil)

	// --- Act ---
	event, err := transformer.Transform(financialMsg())
	s.Require().NoError(err)

	storeErr := repo.StoreData(context.Background(), event)

	// --- Assert ---
	s.NoError(storeErr,
		"a FinancialStream event produced by Transform should be storable in the DB")
}

func (s *TransformerTestSuite) TestAllDataTypes_ReturnsAllSupportedDomains() {
	types := ingestor.AllDataTypes()
	s.Len(types, 9, "should return all 4 domain types")
	s.Contains(types, ingestor.DataTypeEnergyMeter)
	s.Contains(types, ingestor.DataTypeFinancialStream)
}

func energyMeterMsg() *ingestor.DeviceMessage {
	return &ingestor.DeviceMessage{
		DeviceID:  TransformerDeviceID,
		Type:      string(ingestor.DataTypeEnergyMeter),
		Timestamp: time.Now(),
		Metrics: map[string]any{
			"power_w":    int64(100),
			"energy_wh":  int64(500),
			"voltage_v":  int32(220),
			"current_ma": int32(455),
		},
	}
}

func financialMsg() *ingestor.DeviceMessage {
	return &ingestor.DeviceMessage{
		DeviceID:  TransformerDeviceID,
		Type:      string(ingestor.DataTypeFinancialStream),
		Timestamp: time.Now(),
		Metrics: map[string]any{
			"amount_minor": int64(1000),
			"currency":     "USD",
			"merchant":     "ACME Corp",
			"status":       "completed",
		},
	}
}

func environmentalMsg() *ingestor.DeviceMessage {
	return &ingestor.DeviceMessage{
		DeviceID:  TransformerDeviceID,
		Type:      string(ingestor.DataTypeEnvironmentalSensor),
		Timestamp: time.Now(),
		Metrics: map[string]any{
			"temperature_c":     int64(22),
			"humidity_percent":  int64(65),
			"air_quality_index": int64(50),
		},
	}
}
