package metrics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"synapsePlatform/internal/ingestor"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type ConsumerStatsReader struct {
	reader  ingestor.ConsumerStatsReader
	tracer  trace.Tracer
	lag     metric.Int64Gauge
	minByte metric.Int64Gauge
	maxByte metric.Int64Gauge
	msgs    metric.Int64Counter
	bytes   metric.Int64Counter
	rebal   metric.Int64Counter
	errs    metric.Int64Counter
	reads   metric.Int64Counter
	errors  metric.Int64Counter
	latency metric.Float64Histogram
}

func NewConsumerStatsReader(
	meter metric.Meter,
	tracer trace.Tracer,
	reader ingestor.ConsumerStatsReader,
) (*ConsumerStatsReader, error) {
	var joined error

	gauge := func(name, description string) metric.Int64Gauge {
		instrument, err := meter.Int64Gauge(name, metric.WithDescription(description))
		joined = errors.Join(joined, err)
		return instrument
	}

	counter := func(name, description string) metric.Int64Counter {
		instrument, err := meter.Int64Counter(name, metric.WithDescription(description))
		joined = errors.Join(joined, err)
		return instrument
	}

	histogram := func(name, description string) metric.Float64Histogram {
		instrument, err := meter.Float64Histogram(
			name,
			metric.WithUnit("s"),
			metric.WithDescription(description),
		)
		joined = errors.Join(joined, err)
		return instrument
	}

	result := &ConsumerStatsReader{
		reader:  reader,
		tracer:  tracer,
		lag:     gauge("kafka.consumer.lag", "Current Kafka consumer lag"),
		minByte: gauge("kafka.consumer.min_bytes", "Configured Kafka consumer min bytes"),
		maxByte: gauge("kafka.consumer.max_bytes", "Configured Kafka consumer max bytes"),
		msgs:    counter("kafka.consumer.messages", "Kafka messages observed since the last stats read"),
		bytes:   counter("kafka.consumer.bytes", "Kafka bytes observed since the last stats read"),
		rebal:   counter("kafka.consumer.rebalances", "Kafka consumer rebalances observed since the last stats read"),
		errs:    counter("kafka.consumer.errors", "Kafka consumer errors observed since the last stats read"),
		reads:   counter("kafka.consumer.stats.reads", "Kafka consumer stats reads by status"),
		errors:  counter("kafka.consumer.stats.errors", "Kafka consumer stats read errors"),
		latency: histogram("kafka.consumer.stats.duration", "Time to read Kafka consumer stats"),
	}

	if joined != nil {
		return nil, fmt.Errorf("create kafka consumer stats metrics: %w", joined)
	}

	return result, nil
}

func (m *ConsumerStatsReader) ReadStats(ctx context.Context) (ingestor.ConsumerStats, error) {
	ctx, span := m.tracer.Start(ctx, "kafka.consumer.read_stats")
	defer span.End()

	start := time.Now()
	stats, err := m.reader.ReadStats(ctx)
	elapsed := time.Since(start).Seconds()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		m.errors.Add(ctx, 1)
		m.reads.Add(ctx, 1, metric.WithAttributes(attribute.String(AttrStatus, StatusError)))
		m.latency.Record(ctx, elapsed, metric.WithAttributes(attribute.String(AttrStatus, StatusError)))
		return stats, err
	}

	m.lag.Record(ctx, stats.Lag)
	m.minByte.Record(ctx, stats.MinBytes)
	m.maxByte.Record(ctx, stats.MaxBytes)
	m.msgs.Add(ctx, stats.Messages)
	m.bytes.Add(ctx, stats.Bytes)
	m.rebal.Add(ctx, stats.Rebalances)
	m.errs.Add(ctx, stats.Errors)
	m.reads.Add(ctx, 1, metric.WithAttributes(attribute.String(AttrStatus, StatusSuccess)))
	m.latency.Record(ctx, elapsed, metric.WithAttributes(attribute.String(AttrStatus, StatusSuccess)))

	span.SetAttributes(
		attribute.Int64("kafka.consumer.lag", stats.Lag),
		attribute.Int64("kafka.consumer.messages", stats.Messages),
		attribute.Int64("kafka.consumer.bytes", stats.Bytes),
	)

	return stats, nil
}
