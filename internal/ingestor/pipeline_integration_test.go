//go:build integration

package ingestor_test

import (
	"context"
	"errors"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/kafka"
	"synapsePlatform/internal/sqllite"
	"synapsePlatform/internal/utilstest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type PipelineTestSuite struct {
	suite.Suite
	repo      *sqllite.Repo
	processor *utilstest.DataProcessor
	ctx       context.Context
}

func TestPipelineSuite(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

func (s *PipelineTestSuite) SetupTest() {
	var err error
	s.repo, err = sqllite.NewRepo(":memory:")
	s.Require().NoError(err)
	s.processor = utilstest.NewDataProcessor(s.T())
	s.ctx = context.Background()
}

func (s *PipelineTestSuite) TearDownTest() {
	s.repo.Close()
}

func (s *PipelineTestSuite) TestFullPipeline_MessageFlowsThroughToDatabase() {
	ctx, cancel := context.WithCancel(s.ctx)

	msg := &ingestor.DeviceMessage{
		DeviceID:  "pipeline-device",
		Type:      "energy_meter",
		Timestamp: time.Now(),
		Metrics: map[string]any{
			"power_w":    150.0,
			"energy_wh":  750.0,
			"voltage_v":  230.0,
			"current_ma": 652.0,
		},
	}

	s.processor.WithResult(msg)
	s.processor.WithCancel(cancel)

	domains := ingestor.AllDataTypes()
	transformer := ingestor.NewMessageTransformer(domains)
	failures := ingestor.NewFallbackFailureStorer(
		s.repo,
		kafka.NewKafkaDLQ(
			[]string{"localhost:9092"},
			"test-topic",
		),
	)

	ing := ingestor.New(
		ingestor.Config{CompatibleDataTypes: domains},
		s.processor,
		s.repo,
		transformer,
		failures,
	)

	s.Require().NoError(ing.Ingest(ctx))

	result, err := s.repo.ListEvents(context.Background(), ingestor.PageRequest{Limit: 10})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 1)

	stored := result.Items[0]
	s.Equal("energy_meter", stored.Domain)
	s.Equal("energy_meter", stored.EventType)
	s.Equal("pipeline-device", stored.EntityID)
	s.Equal("1.0.0", stored.SchemaVersion)

	reading, ok := stored.Data.(*ingestor.EnergyReading)
	s.Require().True(ok, "data should be *EnergyReading")
	s.InDelta(150.0, reading.PowerW, 0.01)
	s.InDelta(750.0, reading.EnergyWh, 0.01)
}

func (s *PipelineTestSuite) TestFullPipeline_ProcessorError_LandsInFailedMessages() {
	ctx, cancel := context.WithCancel(s.ctx)

	s.processor.WithError(errors.New("broker timeout"))
	s.processor.WithCancel(cancel)

	domains := ingestor.AllDataTypes()
	transformer := ingestor.NewMessageTransformer(domains)
	failures := ingestor.NewFallbackFailureStorer(
		s.repo,
		kafka.NewKafkaDLQ(
			[]string{"localhost:9092"},
			"test-topic",
		),
	)

	ing := ingestor.New(
		ingestor.Config{CompatibleDataTypes: domains},
		s.processor,
		s.repo,
		transformer,
		failures,
	)

	s.NoError(ing.Ingest(ctx))

	var count int
	s.Require().NoError(
		s.repo.Db.QueryRowContext(context.Background(),
			"SELECT COUNT(*) FROM failed_messages WHERE stage = 'process'").Scan(&count),
	)
	s.Equal(1, count)

	result, err := s.repo.ListEvents(context.Background(), ingestor.PageRequest{Limit: 10})
	s.Require().NoError(err)
	s.Empty(result.Items, "no events should be stored on processor error")
}

func (s *PipelineTestSuite) TestFullPipeline_TransformError_LandsInFailedMessages() {
	ctx, cancel := context.WithCancel(s.ctx)

	msg := &ingestor.DeviceMessage{
		DeviceID:  "unsupported-device",
		Type:      "totally_unknown_type",
		Timestamp: time.Now(),
		Metrics:   map[string]any{"foo": "bar"},
	}

	s.processor.WithResult(msg)
	s.processor.WithCancel(cancel)

	domains := ingestor.AllDataTypes()
	transformer := ingestor.NewMessageTransformer(domains)
	failures := ingestor.NewFallbackFailureStorer(
		s.repo,
		kafka.NewKafkaDLQ(
			[]string{"localhost:9092"},
			"test-topic",
		),
	)

	ing := ingestor.New(
		ingestor.Config{CompatibleDataTypes: domains},
		s.processor,
		s.repo,
		transformer,
		failures,
	)

	s.NoError(ing.Ingest(ctx))

	var count int
	s.Require().NoError(
		s.repo.Db.QueryRowContext(context.Background(),
			"SELECT COUNT(*) FROM failed_messages WHERE stage = 'transform'").Scan(&count),
	)
	s.Equal(1, count)
}

func (s *PipelineTestSuite) TestFullPipeline_MultipleMessages_AllPersisted() {
	ctx, cancel := context.WithCancel(s.ctx)

	msgs := []*ingestor.DeviceMessage{
		{DeviceID: "dev-1", Type: "energy_meter", Timestamp: time.Now(),
			Metrics: map[string]any{"power_w": 100.0, "energy_wh": 500.0, "voltage_v": 220.0, "current_ma": 455.0}},
		{DeviceID: "dev-2", Type: "energy_meter", Timestamp: time.Now(),
			Metrics: map[string]any{"power_w": 200.0, "energy_wh": 600.0, "voltage_v": 230.0, "current_ma": 870.0}},
	}

	for _, msg := range msgs {
		s.processor.WithResult(msg)
	}
	s.processor.WithCancel(cancel)

	domains := ingestor.AllDataTypes()
	transformer := ingestor.NewMessageTransformer(domains)
	failures := ingestor.NewFallbackFailureStorer(
		s.repo,
		kafka.NewKafkaDLQ(
			[]string{"localhost:9092"},
			"test-topic",
		),
	)

	ing := ingestor.New(
		ingestor.Config{CompatibleDataTypes: domains},
		s.processor,
		s.repo,
		transformer,
		failures,
	)

	s.Require().NoError(ing.Ingest(ctx))

	result, err := s.repo.ListEvents(context.Background(), ingestor.PageRequest{Limit: 10})
	s.Require().NoError(err)
	s.Len(result.Items, 2)
}
