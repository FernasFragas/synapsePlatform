package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"synapsePlatform/internal/ingestor"
)

type IngestorProcessor struct {
	processor ingestor.DataProcessor
	duration  metric.Float64Histogram
	total     metric.Int64Counter
	errors    metric.Int64Counter
}

func NewIngestorProcessor(meter metric.Meter, processor ingestor.DataProcessor) (*IngestorProcessor, error) {
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

	errors, err := meter.Int64Counter("ingestor.process_data.errors",
		metric.WithDescription("Process errors with device context"),
	)
	if err != nil {
		return nil, err
	}

	return &IngestorProcessor{
		processor: processor,
		duration:  duration,
		total:     total,
		errors:    errors,
	}, nil
}

func (m *IngestorProcessor) ProcessData(ctx context.Context) (*ingestor.DeviceMessage, error) {
	start := time.Now()
	msg, err := m.processor.ProcessData(ctx)
	elapsed := time.Since(start).Seconds()

	if err != nil {
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

	attrs := []attribute.KeyValue{
		attribute.String(AttrOperation, "process_data"),
		attribute.String(AttrStatus, StatusSuccess),
	}
	if msg != nil {
		attrs = append(attrs,
			attribute.String(AttrDeviceType, msg.Type),
		)
	}

	m.total.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.duration.Record(ctx, elapsed, metric.WithAttributes(
		attribute.String(AttrStatus, StatusSuccess),
	))

	return msg, err
}
