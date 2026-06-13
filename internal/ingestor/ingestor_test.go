package ingestor_test

import (
	"context"
	"errors"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/utilstest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
	failures    *utilstest.FailureStorer
}

func (s *IngestorTestSuite) SetupTest() {
	s.processor = utilstest.NewDataProcessor(s.T())
	s.transformer = utilstest.NewTransformer(s.T())
	s.storer = utilstest.NewMessageStorerMock(s.T())
	s.failures = utilstest.NewFailureStorer(s.T())
}

func (s *IngestorTestSuite) TestIngest_ProcessorError_SkipsAndContinues() {
	ctx, cancel := context.WithCancel(context.Background())

	s.processor.WithError(errors.New("poll failed"))
	s.processor.WithCancel(cancel)

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer, s.failures)

	err := ing.Ingest(ctx)

	s.NoError(err)
}

func (s *IngestorTestSuite) TestIngest_TransformerError_SkipsAndContinues() {
	ctx, cancel := context.WithCancel(context.Background())

	s.processor.WithResult(&ingestor.DeviceMessage{DeviceID: IngestorTestDeviceID, Type: IngestorTestEventType, Timestamp: time.Now()})
	s.transformer.WithError(errors.New("transform failed"))
	s.processor.WithCancel(cancel)

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer, s.failures)

	s.NoError(ing.Ingest(ctx))
}

func (s *IngestorTestSuite) TestIngest_HappyPath_EventPersistedInRealDB() {
	ctx, cancel := context.WithCancel(context.Background())
	event := validEvent()

	s.processor.WithResult(&ingestor.DeviceMessage{DeviceID: IngestorTestDeviceID, Type: IngestorTestEventType, Timestamp: time.Now()})
	s.transformer.WithResult(event)
	s.processor.WithCancel(cancel)
	s.storer.WithSuccess()

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer, s.failures)

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

func validMessage() *ingestor.DeviceMessage {
	return &ingestor.DeviceMessage{
		DeviceID:  IngestorTestDeviceID,
		Type:      IngestorTestEventType,
		Timestamp: time.Now(),
	}
}

func (s *IngestorTestSuite) TestIngest_HappyPath_AcksAfterStore() {
	ctx, cancel := context.WithCancel(context.Background())
	ack := utilstest.NewAck(s.T())

	msg := validMessage()
	event := validEvent()

	s.processor.WithDeliveryAndAck(msg, ack.Handler())
	s.transformer.WithResult(event)
	s.storer.WithSuccess()
	s.processor.WithCancel(cancel)
	s.failures.ExpectNoFailure()

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer, s.failures)
	s.Require().NoError(ing.Ingest(ctx))
	ack.RequireCalls(1)
}

func (s *IngestorTestSuite) TestIngest_TransientProcessError_DoesNotAckOrStoreFailure() {
	ctx, cancel := context.WithCancel(context.Background())
	ack := utilstest.NewAck(s.T())

	delivery := &ingestor.Delivery{Ack: ack.Handler()}
	s.processor.WithTransientError(delivery, errors.New("broker timeout"))
	s.processor.WithCancel(cancel)
	s.failures.ExpectNoFailure()

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer, s.failures)
	s.Require().NoError(ing.Ingest(ctx))
	ack.RequireCalls(0)
}

func (s *IngestorTestSuite) TestIngest_TerminalProcessError_StoresFailureThenAcks() {
	ctx, cancel := context.WithCancel(context.Background())
	ack := utilstest.NewAck(s.T())

	delivery := &ingestor.Delivery{Ack: ack.Handler()}
	s.processor.WithTerminalError(delivery, errors.New("bad payload"))
	s.failures.ExpectFailure("process", ingestor.ClassTerminal.String())
	s.processor.WithCancel(cancel)

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer, s.failures)
	s.Require().NoError(ing.Ingest(ctx))
	ack.RequireCalls(1)
}

func (s *IngestorTestSuite) TestIngest_TerminalFailureStoreFails_DoesNotAck() {
	ctx, cancel := context.WithCancel(context.Background())
	ack := utilstest.NewAck(s.T())

	delivery := &ingestor.Delivery{Ack: ack.Handler()}
	s.processor.WithTerminalError(delivery, errors.New("bad payload"))
	s.failures.ExpectFailureStoreError(errors.New("dlq unavailable"))
	s.processor.WithCancel(cancel)

	ing := ingestor.New(ingestor.Config{}, s.processor, s.storer, s.transformer, s.failures)
	s.Require().NoError(ing.Ingest(ctx))
	ack.RequireCalls(0)
}

func (s *IngestorTestSuite) TestIngest_BatchTransientStoreError_DoesNotAck() {
	ctx, cancel := context.WithCancel(context.Background())
	ack1 := utilstest.NewAck(s.T())
	ack2 := utilstest.NewAck(s.T())

	s.processor.WithDeliveryAndAck(validMessage(), ack1.Handler())
	s.processor.WithDeliveryAndAck(validMessage(), ack2.Handler())
	s.transformer.WithResult(validEvent())
	s.transformer.WithResult(validEvent())
	s.storer.MockMessageStorer.EXPECT().
		StoreBatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, []*ingestor.BaseEvent) error {
			cancel()
			return errors.New("database unavailable")
		})
	s.failures.ExpectNoFailure()

	ing := ingestor.New(ingestor.Config{
		BatchSize:  2,
		NumWorkers: 1,
	}, s.processor, s.storer, s.transformer, s.failures)

	s.Require().NoError(ing.Ingest(ctx))
	ack1.RequireCalls(0)
	ack2.RequireCalls(0)
}
