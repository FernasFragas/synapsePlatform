package ingestor_test

import (
	"context"
	"errors"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/utilstest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	IngestorTestDeviceID  = "device-123"
	IngestorTestEventType = "energy_meter"
)

type IngestorTestSuite struct {
	suite.Suite

	processor   *utilstest.DataProcessor
	transformer *utilstest.Transformer
	storer      *utilstest.MessageStorer
}

func TestIngestorSuite(t *testing.T) {
	suite.Run(t, new(IngestorTestSuite))
}

func (s *IngestorTestSuite) SetupTest() {
	s.processor = utilstest.NewDataProcessor(s.T())
	s.transformer = utilstest.NewTransformer(s.T())
	s.storer = utilstest.NewMessageStorerMock(s.T())
}

func (s *IngestorTestSuite) TestIngest_ProcessorError_SkipsAndContinues() {
	ctx, cancel := context.WithCancel(context.Background())

	s.processor.WithError(errors.New("poll failed"))
	s.processor.WithCancel(cancel)

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer)

	err := ing.Ingest(ctx)

	s.NoError(err)
}

func (s *IngestorTestSuite) TestIngest_TransformerError_SkipsAndContinues() {
	ctx, cancel := context.WithCancel(context.Background())

	s.processor.WithResult(&ingestor.DeviceMessage{DeviceID: IngestorTestDeviceID, Type: IngestorTestEventType, Timestamp: time.Now()})
	s.transformer.WithError(errors.New("transform failed"))
	s.processor.WithCancel(cancel)

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer)

	s.NoError(ing.Ingest(ctx))
}

func (s *IngestorTestSuite) TestIngest_HappyPath_EventPersistedInRealDB() {
	ctx, cancel := context.WithCancel(context.Background())
	event := validEvent()

	s.processor.WithResult(&ingestor.DeviceMessage{DeviceID: IngestorTestDeviceID, Type: IngestorTestEventType, Timestamp: time.Now()})
	s.transformer.WithResult(event)
	s.processor.WithCancel(cancel)
	s.storer.WithSuccess()

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer)

	s.NoError(ing.Ingest(ctx))
}

// validEvent returns a fully-populated BaseEvent suitable for DB storage.
func validEvent() *ingestor.BaseEvent {
	return &ingestor.BaseEvent{
		EventID:       uuid.New(),
		Domain:        "energy",
		EventType:     IngestorTestEventType,
		EntityID:      IngestorTestDeviceID,
		EntityType:    "sensor",
		OccurredAt:    time.Now().UTC(),
		IngestedAt:    time.Now().UTC(),
		Source:        "mqtt-bridge",
		SchemaVersion: "1.0.0",
		Data: &ingestor.EnergyReading{
			PowerW:    100,
			EnergyWh:  500,
			VoltageV:  220,
			CurrentMA: 455,
		},
	}
}
