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
	processor      ingestor.DataProcessor
	tracer         trace.Tracer
	duration       metric.Float64Histogram
	total          metric.Int64Counter
	errors         metric.Int64Counter
	ackDuration    metric.Float64Histogram
	ackTotal       metric.Int64Counter
	ackErrors      metric.Int64Counter
	batchSize      metric.Int64Histogram
	validationErrs metric.Int64Counter
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

	ackDuration, err := meter.Float64Histogram("ingestor.ack_data.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to acknowledge processed data"),
	)
	if err != nil {
		return nil, err
	}

	ackTotal, err := meter.Int64Counter("ingestor.ack_data.total",
		metric.WithDescription("Total acknowledgment operations by status"),
	)
	if err != nil {
		return nil, err
	}

	ackErrors, err := meter.Int64Counter("ingestor.ack_data.errors",
		metric.WithDescription("Acknowledgment errors"),
	)
	if err != nil {
		return nil, err
	}

	batchSize, err := meter.Int64Histogram("ingestor.process_batch.size",
		metric.WithDescription("Number of messages processed per batch"),
	)
	if err != nil {
		return nil, err
	}

	validationErrs, err := meter.Int64Counter("ingestor.process_batch.validation_errors",
		metric.WithDescription("Messages that failed validation in batch processing"),
	)
	if err != nil {
		return nil, err
	}

	return &IngestorProcessor{
		processor:   processor,
		tracer:      tracer,
		duration:    duration,
		total:       total,
		errors:      errorsCounter,
		ackDuration: ackDuration,
		ackTotal:    ackTotal,
		ackErrors:   ackErrors,
		batchSize:      batchSize,
		validationErrs: validationErrs,
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

func (m *IngestorProcessor) ProcessDataBatch(ctx context.Context, maxMessages int) ([]*ingestor.DeviceMessage, error) {
	ctx, span := m.tracer.Start(ctx, "ingestor.process_data_batch",
		trace.WithAttributes(
			attribute.Int("max_messages", maxMessages),
		))
	defer span.End()

	start := time.Now()
	// Try to call ProcessDataBatch on the underlying processor
	batchProcessor, ok := m.processor.(interface {
		ProcessDataBatch(ctx context.Context, maxMessages int) ([]*ingestor.DeviceMessage, error)
	})

	var msgs []*ingestor.DeviceMessage
	var err error

	if ok {
		// Use batch processing if available
		msgs, err = batchProcessor.ProcessDataBatch(ctx, maxMessages)
	} else {
		// Fallback to single message
		msg, singleErr := m.processor.ProcessData(ctx)
		if singleErr != nil {
			err = singleErr
		} else if msg != nil {
			msgs = []*ingestor.DeviceMessage{msg}
		}
	}

	elapsed := time.Since(start).Seconds()

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		m.errors.Add(ctx, 1, metric.WithAttributes(
			attribute.String(AttrOperation, "process_data_batch"),
		))
		m.total.Add(ctx, 1, metric.WithAttributes(
			attribute.String(AttrOperation, "process_data_batch"),
			attribute.String(AttrStatus, StatusError),
		))
		m.duration.Record(ctx, elapsed, metric.WithAttributes(
			attribute.String(AttrStatus, StatusError),
		))

		return msgs, err
	}
	// Record batch size
	batchSize := len(msgs)
	m.batchSize.Record(ctx, int64(batchSize))

	span.SetAttributes(
		attribute.Int("messages_processed", batchSize),
	)
	m.total.Add(ctx, 1, metric.WithAttributes(
		attribute.String(AttrOperation, "process_data_batch"),
		attribute.String(AttrStatus, StatusSuccess),
	))
	m.duration.Record(ctx, elapsed, metric.WithAttributes(
		attribute.String(AttrStatus, StatusSuccess),
	))

	return msgs, nil
}
