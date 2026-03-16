package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"synapsePlatform/internal/ingestor"
)

type MessagePoller struct {
	poller   ingestor.MessagePoller
	tracer   trace.Tracer
	duration metric.Float64Histogram
	total    metric.Int64Counter
	errors   metric.Int64Counter
}

func NewMessagePoller(meter metric.Meter, tracer trace.Tracer, poller ingestor.MessagePoller) (*MessagePoller, error) {
	duration, err := meter.Float64Histogram("ingestor.poller.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time per poller operation"),
	)
	if err != nil {
		return nil, err
	}

	total, err := meter.Int64Counter("ingestor.poller.total",
		metric.WithDescription("Total poller operations by operation and status"),
	)
	if err != nil {
		return nil, err
	}

	errors, err := meter.Int64Counter("ingestor.poller.errors",
		metric.WithDescription("Poller errors by operation"),
	)
	if err != nil {
		return nil, err
	}

	return &MessagePoller{
		poller:   poller,
		tracer:   tracer,
		duration: duration,
		total:    total,
		errors:   errors,
	}, nil
}

func (m *MessagePoller) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, error) {
	start := time.Now()

	msg, err := m.poller.PollMessage(ctx)

	elapsed := time.Since(start).Seconds()

	op := attribute.String(AttrOperation, "poll_message")

	if err != nil {
		m.errors.Add(ctx, 1, metric.WithAttributes(op))
		m.total.Add(ctx, 1, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))
		m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))

		return msg, err
	}

	attrs := []attribute.KeyValue{op, attribute.String(AttrStatus, StatusSuccess)}
	if msg != nil {
		attrs = append(attrs, attribute.String(AttrDeviceType, msg.Type))
	}

	m.total.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusSuccess)))

	return msg, nil
}

func (m *MessagePoller) Close(ctx context.Context) error {
	start := time.Now()

	err := m.poller.Close(ctx)

	elapsed := time.Since(start).Seconds()

	op := attribute.String(AttrOperation, "close")

	if err != nil {
		m.errors.Add(ctx, 1, metric.WithAttributes(op))
		m.total.Add(ctx, 1, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))
		m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))

		return err
	}

	m.total.Add(ctx, 1, metric.WithAttributes(op, attribute.String(AttrStatus, StatusSuccess)))
	m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusSuccess)))

	return nil
}
