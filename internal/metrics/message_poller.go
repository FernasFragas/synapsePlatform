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
	poller      ingestor.MessagePoller
	tracer      trace.Tracer
	duration    metric.Float64Histogram
	total       metric.Int64Counter
	errors      metric.Int64Counter
	ackDuration metric.Float64Histogram
	ackTotal    metric.Int64Counter
	ackErrors   metric.Int64Counter
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

	ackDuration, err := meter.Float64Histogram("ingestor.poller.ack.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to acknowledge messages"),
	)
	if err != nil {
		return nil, err
	}

	ackTotal, err := meter.Int64Counter("ingestor.poller.ack.total",
		metric.WithDescription("Total acknowledgment operations by status"),
	)
	if err != nil {
		return nil, err
	}

	ackErrors, err := meter.Int64Counter("ingestor.poller.ack.errors",
		metric.WithDescription("Acknowledgment errors"),
	)
	if err != nil {
		return nil, err
	}

	return &MessagePoller{
		poller:      poller,
		tracer:      tracer,
		duration:    duration,
		total:       total,
		errors:      errors,
		ackDuration: ackDuration,
		ackTotal:    ackTotal,
		ackErrors:   ackErrors,
	}, nil
}

func (m *MessagePoller) PollMessage(ctx context.Context) (*ingestor.Delivery, error) {
	ctx, span := m.tracer.Start(ctx, "poller.poll_message")
	defer span.End()

	start := time.Now()
	delivery, err := m.poller.PollMessage(ctx)
	elapsed := time.Since(start).Seconds()

	if err != nil {
		span.RecordError(err)
		m.errors.Add(ctx, 1, metric.WithAttributes(attribute.String(AttrOperation, "poll_message")))
		m.duration.Record(ctx, elapsed, metric.WithAttributes(attribute.String(AttrStatus, StatusError)))
		m.instrumentAck(delivery)

		return delivery, err
	}

	attrs := []attribute.KeyValue{attribute.String(AttrStatus, StatusSuccess)}
	if delivery != nil && delivery.Message != nil {
		attrs = append(attrs, attribute.String(AttrDeviceType, delivery.Message.Type))
	}

	m.total.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.duration.Record(ctx, elapsed, metric.WithAttributes(attrs...))

	m.instrumentAck(delivery)

	return delivery, nil
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

// maybe not needed ??
func (m *MessagePoller) instrumentAck(delivery *ingestor.Delivery) {
	if delivery == nil || delivery.Ack == nil {
		return
	}

	rawAck := delivery.Ack
	delivery.Ack = func(ctx context.Context) error {
		start := time.Now()
		err := rawAck(ctx)
		elapsed := time.Since(start).Seconds()

		if err != nil {
			m.ackErrors.Add(ctx, 1)
			m.ackTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(AttrStatus, StatusError)))
			m.ackDuration.Record(ctx, elapsed, metric.WithAttributes(attribute.String(AttrStatus, StatusError)))
			return err
		}

		m.ackTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(AttrStatus, StatusSuccess)))
		m.ackDuration.Record(ctx, elapsed, metric.WithAttributes(attribute.String(AttrStatus, StatusSuccess)))
		return nil
	}
}
