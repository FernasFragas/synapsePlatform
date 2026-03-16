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

type IngestorProcessor struct {
	processor ingestor.DataProcessor
	tracer    trace.Tracer
	duration  metric.Float64Histogram
	total     metric.Int64Counter
	errors    metric.Int64Counter
}

func NewIngestorProcessor(meter metric.Meter, tracer trace.Tracer, processor ingestor.DataProcessor) (*IngestorProcessor, error) {
	duration, err := meter.Float64Histogram("ingestor.process_data.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to process a device message"),
	)
	if err != nil {
		return nil, err
	}
	total, err := meter.Int64Counter("ingestor.process_data.total",
		metric.WithDescription("Total process_data calls by status"),
	)
	if err != nil {
		return nil, err
	}
	errorsCounter, err := meter.Int64Counter("ingestor.process_data.errors",
		metric.WithDescription("Process errors with device context"),
	)
	if err != nil {
		return nil, err
	}
	return &IngestorProcessor{
		processor: processor,
		tracer:    tracer,
		duration:  duration,
		total:     total,
		errors:    errorsCounter,
	}, nil
}

func (m *IngestorProcessor) ProcessData(ctx context.Context) (*ingestor.DeviceMessage, error) {
	ctx, span := m.tracer.Start(ctx, "ingestor.process_data")
	defer span.End()

	start := time.Now()

	msg, err := m.processor.ProcessData(ctx)

	elapsed := time.Since(start).Seconds()

	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		span.RecordError(err)

		m.errors.Add(ctx, 1, metric.WithAttributes(
			attribute.String(AttrOperation, "process_data"),
		))

		m.total.Add(ctx, 1, metric.WithAttributes(
			attribute.String(AttrOperation, "process_data"),
			attribute.String(AttrStatus, StatusError),
		))

		m.duration.Record(ctx, elapsed, metric.WithAttributes(
			attribute.String(AttrStatus, StatusError),
		))

		return nil, err
	}

	if msg != nil {
		span.SetAttributes(
			attribute.String(AttrDeviceID, msg.DeviceID),
			attribute.String(AttrDeviceType, msg.Type),
		)
	}

	m.total.Add(ctx, 1, metric.WithAttributes(
		attribute.String(AttrOperation, "process_data"),
		attribute.String(AttrStatus, StatusSuccess),
	))

	m.duration.Record(ctx, elapsed, metric.WithAttributes(
		attribute.String(AttrStatus, StatusSuccess),
	))

	return msg, nil
}
