package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"synapsePlatform/internal/api"
	"synapsePlatform/internal/ingestor"
)

type API struct {
	reader   api.EventReader
	tracer   trace.Tracer
	duration metric.Float64Histogram
	total    metric.Int64Counter
	errors   metric.Int64Counter
}

func NewAPI(meter metric.Meter, tracer trace.Tracer, reader api.EventReader) (*API, error) {
	duration, err := meter.Float64Histogram("api.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to serve an API request"),
	)
	if err != nil {
		return nil, err
	}

	total, err := meter.Int64Counter("api.request.total",
		metric.WithDescription("Total API requests by operation and status"),
	)
	if err != nil {
		return nil, err
	}

	errors, err := meter.Int64Counter("api.request.errors",
		metric.WithDescription("API errors by operation"),
	)
	if err != nil {
		return nil, err
	}

	return &API{
		reader:   reader,
		tracer:   tracer,
		duration: duration,
		total:    total,
		errors:   errors,
	}, nil
}

func (m *API) GetEvent(ctx context.Context, eventID string) (*ingestor.BaseEvent, error) {
	ctx, span := m.tracer.Start(ctx, "api.get_event",
		trace.WithAttributes(attribute.String("event_id", eventID)))
	defer span.End()

	start := time.Now()

	event, err := m.reader.GetEvent(ctx, eventID)

	elapsed := time.Since(start).Seconds()

	op := attribute.String(AttrOperation, "get_event")

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		m.errors.Add(ctx, 1, metric.WithAttributes(op))

		m.total.Add(ctx, 1, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))

		m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))

		return nil, err
	}

	span.SetAttributes(
		attribute.String(AttrDomain, event.Domain),
		attribute.String(AttrEventType, event.EventType),
	)

	m.total.Add(ctx, 1, metric.WithAttributes(op, attribute.String(AttrStatus, StatusSuccess)))

	m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusSuccess)))

	return event, nil
}

func (m *API) ListEvents(ctx context.Context, page ingestor.PageRequest) (*ingestor.PageResponse[*ingestor.BaseEvent], error) {
	ctx, span := m.tracer.Start(ctx, "api.list_events",
		trace.WithAttributes(attribute.String("cursor", page.Cursor)))
	defer span.End()

	start := time.Now()
	events, err := m.reader.ListEvents(ctx, page)
	elapsed := time.Since(start).Seconds()

	op := attribute.String(AttrOperation, "list_events")

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		m.errors.Add(ctx, 1, metric.WithAttributes(op))
		m.total.Add(ctx, 1, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))
		m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))

		return nil, err
	}

	span.SetAttributes(
		attribute.String("cursor", page.Cursor),
		attribute.Int("limit", page.Limit),
	)

	m.total.Add(ctx, 1, metric.WithAttributes(op, attribute.String(AttrStatus, StatusSuccess)))
	m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusSuccess)))

	return events, nil
}
