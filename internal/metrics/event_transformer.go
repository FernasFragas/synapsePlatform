package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"synapsePlatform/internal/ingestor"
)

type EventTransformer struct {
	transformer ingestor.Transformer
	tracer      trace.Tracer
	duration    metric.Float64Histogram
	total       metric.Int64Counter
	errors      metric.Int64Counter
}

func NewEventTransformer(meter metric.Meter, tracer trace.Tracer, transformer ingestor.Transformer) (*EventTransformer, error) {
	duration, err := meter.Float64Histogram("ingestor.transform.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to transform a device message into a base event"),
	)
	if err != nil {
		return nil, err
	}

	total, err := meter.Int64Counter("ingestor.transform.total",
		metric.WithDescription("Total transform calls by status"),
	)
	if err != nil {
		return nil, err
	}

	errors, err := meter.Int64Counter("ingestor.transform.errors",
		metric.WithDescription("Transform errors by device type"),
	)
	if err != nil {
		return nil, err
	}

	return &EventTransformer{
		transformer: transformer,
		tracer:      tracer,
		duration:    duration,
		total:       total,
		errors:      errors,
	}, nil
}

func (m *EventTransformer) Transform(ctx context.Context, msg *ingestor.DeviceMessage) (*ingestor.BaseEvent, error) {
	ctx, span := m.tracer.Start(ctx, "ingestor.transform",
		trace.WithAttributes(
			attribute.String(AttrDeviceID, msg.DeviceID),
			attribute.String(AttrDeviceType, msg.Type),
		))
	defer span.End()

	start := time.Now()

	event, err := m.transformer.Transform(ctx, msg)

	elapsed := time.Since(start).Seconds()

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		m.errors.Add(ctx, 1, metric.WithAttributes(
			attribute.String(AttrDeviceType, msg.Type),
			attribute.String(AttrDeviceID, msg.DeviceID),
		))

		m.total.Add(ctx, 1, metric.WithAttributes(
			attribute.String(AttrStatus, StatusError),
			attribute.String(AttrDeviceType, msg.Type),
		))

		m.duration.Record(ctx, elapsed, metric.WithAttributes(
			attribute.String(AttrStatus, StatusError),
		))

		return nil, err
	}

	span.SetAttributes(
		attribute.String(AttrDomain, event.Domain),
		attribute.String(AttrEventType, event.EventType),
	)

	m.total.Add(ctx, 1, metric.WithAttributes(
		attribute.String(AttrStatus, StatusSuccess),
		attribute.String(AttrDomain, event.Domain),
		attribute.String(AttrEventType, event.EventType),
	))
	m.duration.Record(ctx, elapsed, metric.WithAttributes(
		attribute.String(AttrStatus, StatusSuccess),
		attribute.String(AttrDomain, event.Domain),
	))

	return event, nil
}
