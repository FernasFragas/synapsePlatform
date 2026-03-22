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

type MessageStorer struct {
	storer   ingestor.MessageStorer
	tracer   trace.Tracer
	duration metric.Float64Histogram
	total    metric.Int64Counter
	errors   metric.Int64Counter
	batchDuration metric.Float64Histogram
	batchSize     metric.Int64Histogram
}

func NewMessageStorer(meter metric.Meter, tracer trace.Tracer, storer ingestor.MessageStorer) (*MessageStorer, error) {
	duration, err := meter.Float64Histogram("ingestor.store_data.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to store a base event"),
	)
	if err != nil {
		return nil, err
	}

	total, err := meter.Int64Counter("ingestor.store_data.total",
		metric.WithDescription("Total store_data calls by domain and event type"),
	)
	if err != nil {
		return nil, err
	}

	errors, err := meter.Int64Counter("ingestor.store_data.errors",
		metric.WithDescription("Store errors by domain, event type, and entity type"),
	)
	if err != nil {
		return nil, err
	}

	batchDuration, err := meter.Float64Histogram("ingestor.store_batch.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to store a batch of events"),
	)
	if err != nil {
		return nil, err
	}
	batchSize, err := meter.Int64Histogram("ingestor.store_batch.size",
		metric.WithDescription("Number of events in each batch"),
	)
	if err != nil {
		return nil, err
	}
	return &MessageStorer{
		storer:        storer,
		tracer:        tracer,
		duration:      duration,
		total:         total,
		errors:        errors,
		batchDuration: batchDuration,
		batchSize:     batchSize,
	}, nil
}

func (m *MessageStorer) StoreData(ctx context.Context, data *ingestor.BaseEvent) error {
	ctx, span := m.tracer.Start(ctx, "ingestor.store_data",
		trace.WithAttributes(
			attribute.String(AttrDomain, data.Domain),
			attribute.String(AttrEventType, data.EventType),
			attribute.String("event_id", data.EventID.String()),
		))
	defer span.End()

	start := time.Now()

	err := m.storer.StoreData(ctx, data)

	elapsed := time.Since(start).Seconds()

	domainAttrs := []attribute.KeyValue{
		attribute.String(AttrDomain, data.Domain),
		attribute.String(AttrEventType, data.EventType),
		attribute.String(AttrEntityType, data.EntityType),
		attribute.String(AttrSource, data.Source),
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		m.errors.Add(ctx, 1, metric.WithAttributes(domainAttrs...))

		m.total.Add(ctx, 1, metric.WithAttributes(
			append(domainAttrs, attribute.String(AttrStatus, StatusError))...,
		))

		m.duration.Record(ctx, elapsed, metric.WithAttributes(
			attribute.String(AttrStatus, StatusError),
			attribute.String(AttrDomain, data.Domain),
		))

		return err
	}

	m.total.Add(ctx, 1, metric.WithAttributes(
		append(domainAttrs, attribute.String(AttrStatus, StatusSuccess))...,
	))

	m.duration.Record(ctx, elapsed, metric.WithAttributes(
		attribute.String(AttrStatus, StatusSuccess),
		attribute.String(AttrDomain, data.Domain),
	))

	return nil
}

func (m *MessageStorer) StoreBatch(ctx context.Context, events []*ingestor.BaseEvent) error {
	if len(events) == 0 {
		return nil
	}
	ctx, span := m.tracer.Start(ctx, "ingestor.store_batch",
		trace.WithAttributes(
			attribute.Int("batch_size", len(events)),
		))
	defer span.End()
	start := time.Now()
	err := m.storer.StoreBatch(ctx, events)
	elapsed := time.Since(start).Seconds()
	// Record batch size
	m.batchSize.Record(ctx, int64(len(events)))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		m.batchDuration.Record(ctx, elapsed, metric.WithAttributes(
			attribute.String(AttrStatus, StatusError),
		))
		// Count errors for each event in the batch
		for _, event := range events {
			domainAttrs := []attribute.KeyValue{
				attribute.String(AttrDomain, event.Domain),
				attribute.String(AttrEventType, event.EventType),
				attribute.String(AttrEntityType, event.EntityType),
				attribute.String(AttrSource, event.Source),
			}
			m.errors.Add(ctx, 1, metric.WithAttributes(domainAttrs...))
		}
		return err
	}
	m.batchDuration.Record(ctx, elapsed, metric.WithAttributes(
		attribute.String(AttrStatus, StatusSuccess),
	))
	// Count successful stores for each event
	for _, event := range events {
		domainAttrs := []attribute.KeyValue{
			attribute.String(AttrDomain, event.Domain),
			attribute.String(AttrEventType, event.EventType),
			attribute.String(AttrEntityType, event.EntityType),
			attribute.String(AttrSource, event.Source),
		}
		m.total.Add(ctx, 1, metric.WithAttributes(
			append(domainAttrs, attribute.String(AttrStatus, StatusSuccess))...,
		))
	}
	return nil
}
