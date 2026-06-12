package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/metric"
)

type StatsCollector struct {
	reader  *kafka.Reader
	lag     metric.Int64Gauge
	minByte metric.Int64Gauge
	maxByte metric.Int64Gauge
	msgs    metric.Int64Counter
	bytes   metric.Int64Counter
	rebal   metric.Int64Counter
	errs    metric.Int64Counter
}

// NewStatsCollector builds the instruments once, up front. Instrument creation
// can fail (bad name), so it is surfaced via errors.Join rather than discarded
// with `_ =` at seven call sites and risking a nil-gauge panic later.
func NewStatsCollector(kafkaConsumer *KafkaConsumer, meter metric.Meter) (*StatsCollector, error) {
	var err error
	g := func(name string) metric.Int64Gauge {
		gauge, e := meter.Int64Gauge(name)
		err = errors.Join(err, e)
		return gauge
	}
	c := func(name string) metric.Int64Counter {
		counter, e := meter.Int64Counter(name)
		err = errors.Join(err, e)
		return counter
	}

	s := &StatsCollector{
		reader:  kafkaConsumer.reader,
		lag:     g("kafka.consumer.lag"),
		minByte: g("kafka.consumer.min_bytes"),
		maxByte: g("kafka.consumer.max_bytes"),
		msgs:    c("kafka.consumer.message_count"),
		bytes:   c("kafka.consumer.bytes_count"),
		rebal:   c("kafka.consumer.rebalance_count"),
		errs:    c("kafka.consumer.errors_count"),
	}
	if err != nil {
		return nil, fmt.Errorf("create kafka stats instruments: %w", err)
	}
	return s, nil
}

// Run polls kafka.Reader.Stats() every 5s until ctx is cancelled, then returns.
// Run blocks, so callers drive lifecycle with the same ctx as the rest of the
// pipeline (e.g. an errgroup).
func (s *StatsCollector) Run(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// kafka-go's counter fields are deltas since the last Stats() call,
			// which maps cleanly onto OTel counters; Lag/bytes are gauges.
			stats := s.reader.Stats()
			s.lag.Record(ctx, stats.Lag)
			s.minByte.Record(ctx, int64(stats.MinBytes))
			s.maxByte.Record(ctx, int64(stats.MaxBytes))
			s.msgs.Add(ctx, stats.Messages)
			s.bytes.Add(ctx, stats.Bytes)
			s.rebal.Add(ctx, stats.Rebalances)
			s.errs.Add(ctx, stats.Errors)
		}
	}
}
