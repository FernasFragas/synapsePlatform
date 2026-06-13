package metrics_test

import (
	"context"
	"errors"
	"testing"

	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/metrics"
	"synapsePlatform/internal/utilstest"

	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestMessagePoller_PollMessage_TerminalErrorWithDeliveryWrapsAck(t *testing.T) {
	ack := utilstest.NewAck(t)
	delivery := &ingestor.Delivery{Ack: ack.Handler()}
	expectedErr := errors.New("bad payload")

	poller := utilstest.NewMessagePoller(t).WithTerminalDecodeFailure(delivery, expectedErr)
	subject, err := metrics.NewMessagePoller(
		metricnoop.NewMeterProvider().Meter("test"),
		tracenoop.NewTracerProvider().Tracer("test"),
		poller,
	)
	require.NoError(t, err)

	actual, err := subject.PollMessage(context.Background())

	require.Error(t, err)
	require.Same(t, delivery, actual)
	require.NoError(t, actual.Ack(context.Background()))
	ack.RequireCalls(1)
}

func TestMessagePoller_PollMessage_WrappedAckReturnsRawAckError(t *testing.T) {
	expectedErr := errors.New("ack failed")
	ack := utilstest.NewAck(t).WithError(expectedErr)
	delivery := &ingestor.Delivery{Ack: ack.Handler()}

	poller := utilstest.NewMessagePoller(t).WithDelivery(delivery)
	subject, err := metrics.NewMessagePoller(
		metricnoop.NewMeterProvider().Meter("test"),
		tracenoop.NewTracerProvider().Tracer("test"),
		poller,
	)
	require.NoError(t, err)

	actual, err := subject.PollMessage(context.Background())

	require.NoError(t, err)
	require.Same(t, delivery, actual)
	require.ErrorIs(t, actual.Ack(context.Background()), expectedErr)
	ack.RequireCalls(1)
}
