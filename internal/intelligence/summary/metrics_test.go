package summary

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"synapsePlatform/internal/api"
	"synapsePlatform/internal/intelligence/provider"
	"synapsePlatform/internal/utilstest"
)

// recordingCacheMetrics captures calls for assertion in tests.
type recordingCacheMetrics struct {
	hits               []string
	misses             []string
	validationFailures []string
}

func (r *recordingCacheMetrics) CacheHit(_ context.Context, domain string) {
	r.hits = append(r.hits, domain)
}
func (r *recordingCacheMetrics) CacheMiss(_ context.Context, domain string) {
	r.misses = append(r.misses, domain)
}
func (r *recordingCacheMetrics) ValidationFailure(_ context.Context, domain string) {
	r.validationFailures = append(r.validationFailures, domain)
}

func TestServiceEmitCacheMissOnNoCachedSummary(t *testing.T) {
	recorder := &recordingCacheMetrics{}
	mockCompleter := utilstest.NewMock()
	mockCompleter.CompletionOutput = "generated summary"

	// Build a reader mock that returns no cached summary and empty stats so
	// the service goes through the empty-events path without calling the
	// completer.
	reader := &stubAggregateReader{
		latestResult: nil,
		latestFound:  false,
		statsResult:  nil,
	}

	svc := New(reader, mockCompleter, "ollama", "mistral:7b", 512, 0.2,
		WithCacheMetrics(recorder),
	)

	report, err := svc.Summarize(context.Background(), api.Request{
		Domain: "energy",
		Since:  time.Now().UTC().Truncate(time.Hour),
	})
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Empty(t, recorder.hits, "no cache hit expected")
	assert.Equal(t, []string{"energy"}, recorder.misses, "one cache miss expected for energy domain")
}

func TestServiceEmitCacheHitOnCachedSummary(t *testing.T) {
	recorder := &recordingCacheMetrics{}
	cachedReport := &api.Report{
		Domain:     "energy",
		Content:    "cached summary",
		Model:      "mistral:7b",
		Provider:   "ollama",
		InputHash:  "some-hash",
		WindowFrom: time.Now().UTC().Truncate(time.Hour),
	}
	reader := &stubAggregateReader{
		latestResult: cachedReport,
		latestFound:  true,
		statsResult: []api.DomainStat{
			{Domain: "energy", EventType: "reading", Count: 5,
				FirstSeen: time.Now().UTC(), LastSeen: time.Now().UTC()},
		},
	}

	svc := New(reader, utilstest.NewMock(), "ollama", "mistral:7b", 512, 0.2,
		WithCacheMetrics(recorder),
	)

	report, err := svc.Summarize(context.Background(), api.Request{
		Domain: "energy",
		Since:  cachedReport.WindowFrom,
	})
	require.NoError(t, err)
	assert.Equal(t, "cached summary", report.Content)
	assert.Equal(t, []string{"energy"}, recorder.hits, "one cache hit expected")
	assert.Empty(t, recorder.misses, "no cache miss expected on hit")
}

func TestServiceWithoutCacheMetricsDoesNotPanic(t *testing.T) {
	reader := &stubAggregateReader{
		latestResult: nil,
		latestFound:  false,
		statsResult:  nil,
	}

	svc := New(reader, utilstest.NewMock(), "ollama", "mistral:7b", 512, 0.2)

	report, err := svc.Summarize(context.Background(), api.Request{
		Domain: "energy",
		Since:  time.Now().UTC().Truncate(time.Hour),
	})
	require.NoError(t, err)
	assert.NotNil(t, report, "service should work without cache metrics")
}

// stubAggregateReader satisfies api.AggregateReader for summary service tests
// without needing a real database or generated mock.
type stubAggregateReader struct {
	latestResult *api.Report
	latestFound  bool
	statsResult  []api.DomainStat
}

func (s *stubAggregateReader) AggregateByDomain(_ context.Context, _ time.Time) ([]api.DomainStat, error) {
	return s.statsResult, nil
}

func (s *stubAggregateReader) LatestSummary(_ context.Context, _ api.SummaryLookup) (*api.Report, bool, error) {
	return s.latestResult, s.latestFound, nil
}

func (s *stubAggregateReader) SaveSummary(_ context.Context, _ *api.Report) (int64, error) {
	return 1, nil
}

func (s *stubAggregateReader) SaveSummaryEvidenceLinks(_ context.Context, _ []api.SummaryEvidenceLink) error {
	return nil
}

// Ensure the stub satisfies the interface at compile time.
var _ api.AggregateReader = (*stubAggregateReader)(nil)

// Ensure the recordingCacheMetrics satisfies the interface.
var _ CacheMetrics = (*recordingCacheMetrics)(nil)

// Ensure *provider.Mock satisfies Completer at compile time.
var _ Completer = (*utilstest.Mock)(nil)
var _ provider.ModelProvider = (*utilstest.Mock)(nil)