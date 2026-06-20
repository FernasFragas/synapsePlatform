package summary

import "context"

// CacheMetrics records cache hit/miss and validation failure events for the
// summary service. The service calls these at the relevant decision points;
// the concrete implementation lives in the metrics package and uses the
// project's OpenTelemetry meter.
//
// Declared here (where consumed) so the summary package does not import the
// metrics package. Callers pass a *metrics.SummaryCacheMetrics (or a test
// double) via the ServiceOptions pattern.
type CacheMetrics interface {
	CacheHit(ctx context.Context, domain string)
	CacheMiss(ctx context.Context, domain string)
	ValidationFailure(ctx context.Context, domain string)
}

// noopCacheMetrics is the default when no metrics recorder is configured. All
// methods are no-ops so the service can call them unconditionally without
// nil checks.
type noopCacheMetrics struct{}

func (noopCacheMetrics) CacheHit(_ context.Context, _ string)            {}
func (noopCacheMetrics) CacheMiss(_ context.Context, _ string)           {}
func (noopCacheMetrics) ValidationFailure(_ context.Context, _ string)   {}

// ServiceOption configures a Service at construction time.
type ServiceOption func(*Service)

// WithCacheMetrics sets the cache metrics recorder for the service. When not
// set, the service uses a no-op recorder so metric calls are always safe.
func WithCacheMetrics(m CacheMetrics) ServiceOption {
	return func(s *Service) {
		if m != nil {
			s.cacheMetrics = m
		}
	}
}