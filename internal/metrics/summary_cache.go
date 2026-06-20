package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// SummaryCacheMetrics implements summary.CacheMetrics using OpenTelemetry
// counters. It emits the three metrics:
//
//   - intelligence_summary_cache_hits_total
//   - intelligence_summary_cache_misses_total
//   - intelligence_summary_validation_failures_total
//
// Each counter is labeled with the domain so cache behavior can be inspected
// per domain.
type SummaryCacheMetrics struct {
	hits               metric.Int64Counter
	misses             metric.Int64Counter
	validationFailures metric.Int64Counter
}

// NewSummaryCacheMetrics builds the cache metrics recorder from the project's
// OTel meter. It returns an error if any counter cannot be registered (for
// example, if a name collision occurs).
func NewSummaryCacheMetrics(meter metric.Meter) (*SummaryCacheMetrics, error) {
	hits, err := meter.Int64Counter("intelligence_summary_cache_hits_total",
		metric.WithDescription("Summary cache hits"),
	)
	if err != nil {
		return nil, err
	}

	misses, err := meter.Int64Counter("intelligence_summary_cache_misses_total",
		metric.WithDescription("Summary cache misses"),
	)
	if err != nil {
		return nil, err
	}

	validationFailures, err := meter.Int64Counter("intelligence_summary_validation_failures_total",
		metric.WithDescription("Summary output validation failures"),
	)
	if err != nil {
		return nil, err
	}

	return &SummaryCacheMetrics{
		hits:               hits,
		misses:             misses,
		validationFailures: validationFailures,
	}, nil
}

func (m *SummaryCacheMetrics) CacheHit(ctx context.Context, domain string) {
	m.hits.Add(ctx, 1, metric.WithAttributes(attribute.String(AttrDomain, domain)))
}

func (m *SummaryCacheMetrics) CacheMiss(ctx context.Context, domain string) {
	m.misses.Add(ctx, 1, metric.WithAttributes(attribute.String(AttrDomain, domain)))
}

func (m *SummaryCacheMetrics) ValidationFailure(ctx context.Context, domain string) {
	m.validationFailures.Add(ctx, 1, metric.WithAttributes(attribute.String(AttrDomain, domain)))
}
