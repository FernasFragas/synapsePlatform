package metrics

import (
	"context"
	"errors"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/utilstest"
	"testing"

	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestConsumerStatsReader_ReadStats_ReturnsWrappedStats(t *testing.T) {
	expected := ingestor.ConsumerStats{
		Lag:        42,
		MinBytes:   1,
		MaxBytes:   100,
		Messages:   10,
		Bytes:      2048,
		Rebalances: 1,
		Errors:     0,
	}

	reader := utilstest.NewConsumerStatsReader(t).WithStats(expected)
	meter := metricnoop.NewMeterProvider().Meter("test")
	tracer := tracenoop.NewTracerProvider().Tracer("test")

	subject, err := NewConsumerStatsReader(meter, tracer, reader)
	require.NoError(t, err)

	actual, err := subject.ReadStats(context.Background())

	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestConsumerStatsReader_ReadStats_ReturnsWrappedError(t *testing.T) {
	expectedErr := errors.New("stats failed")
	reader := utilstest.NewConsumerStatsReader(t).WithError(expectedErr)
	meter := metricnoop.NewMeterProvider().Meter("test")
	tracer := tracenoop.NewTracerProvider().Tracer("test")

	subject, err := NewConsumerStatsReader(meter, tracer, reader)
	require.NoError(t, err)

	_, err = subject.ReadStats(context.Background())

	require.ErrorIs(t, err, expectedErr)
}
