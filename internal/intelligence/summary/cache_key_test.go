package summary

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"synapsePlatform/internal/api"
)

func TestComputeInputHashDeterministicForSameStats(t *testing.T) {
	stats := []api.DomainStat{
		{Domain: "energy", EventType: "reading", Count: 42,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}
	h1 := ComputeInputHash(stats, 0.2, 512)
	h2 := ComputeInputHash(stats, 0.2, 512)
	assert.Equal(t, h1, h2, "same stats and params must produce identical hash")
}

func TestComputeInputHashChangesWhenStatsChange(t *testing.T) {
	stats1 := []api.DomainStat{
		{Domain: "energy", EventType: "reading", Count: 42,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}
	stats2 := []api.DomainStat{
		{Domain: "energy", EventType: "reading", Count: 43,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}
	h1 := ComputeInputHash(stats1, 0.2, 512)
	h2 := ComputeInputHash(stats2, 0.2, 512)
	assert.NotEqual(t, h1, h2, "different stats must produce different hash")
}

func TestComputeInputHashChangesWhenTemperatureChanges(t *testing.T) {
	stats := []api.DomainStat{
		{Domain: "energy", EventType: "reading", Count: 42,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}
	h1 := ComputeInputHash(stats, 0.2, 512)
	h2 := ComputeInputHash(stats, 0.7, 512)
	assert.NotEqual(t, h1, h2, "different temperature must produce different hash")
}

func TestComputeInputHashChangesWhenMaxTokensChanges(t *testing.T) {
	stats := []api.DomainStat{
		{Domain: "energy", EventType: "reading", Count: 42,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}
	h1 := ComputeInputHash(stats, 0.2, 512)
	h2 := ComputeInputHash(stats, 0.2, 1024)
	assert.NotEqual(t, h1, h2, "different max_tokens must produce different hash")
}

func TestComputeInputHashIndependentOfStatsOrder(t *testing.T) {
	statsA := []api.DomainStat{
		{Domain: "energy", EventType: "reading", Count: 42,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
		{Domain: "finance", EventType: "transaction", Count: 10,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}
	statsB := []api.DomainStat{
		{Domain: "finance", EventType: "transaction", Count: 10,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
		{Domain: "energy", EventType: "reading", Count: 42,
			FirstSeen: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}
	hA := ComputeInputHash(statsA, 0.2, 512)
	hB := ComputeInputHash(statsB, 0.2, 512)
	assert.Equal(t, hA, hB, "hash must be independent of stats order")
}

func TestComputeInputHashEmptyStats(t *testing.T) {
	h := ComputeInputHash(nil, 0.2, 512)
	assert.NotEmpty(t, h)
	assert.Len(t, h, 64, "sha256 hex digest is 64 chars")
}

func TestBuildCacheKeyPopulatesAllFields(t *testing.T) {
	since := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	stats := []api.DomainStat{
		{Domain: "energy", EventType: "reading", Count: 5,
			FirstSeen: since, LastSeen: since.Add(time.Hour)},
	}
	key := BuildCacheKey("energy", since, "ollama", "mistral:7b", "summary.v1", 0.2, 512, stats)

	assert.Equal(t, "energy", key.Domain)
	assert.Equal(t, since, key.WindowFrom)
	assert.Equal(t, "ollama", key.Provider)
	assert.Equal(t, "mistral:7b", key.Model)
	assert.Equal(t, "summary.v1", key.PromptVersion)
	assert.Equal(t, 0.2, key.Temperature)
	assert.Equal(t, 512, key.MaxTokens)
	assert.NotEmpty(t, key.InputHash)
}

func TestBuildCacheKeyDifferentProviderProducesDifferentKey(t *testing.T) {
	since := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	stats := []api.DomainStat{{Domain: "energy", EventType: "reading", Count: 5,
		FirstSeen: since, LastSeen: since.Add(time.Hour)}}

	key1 := BuildCacheKey("energy", since, "ollama", "mistral:7b", "summary.v1", 0.2, 512, stats)
	key2 := BuildCacheKey("energy", since, "openai", "mistral:7b", "summary.v1", 0.2, 512, stats)
	assert.NotEqual(t, key1.Provider, key2.Provider)
}

func TestSummaryCacheKeyStringIncludesAllDimensions(t *testing.T) {
	key := SummaryCacheKey{
		Domain:        "energy",
		WindowFrom:    time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
		Provider:      "ollama",
		Model:         "mistral:7b",
		PromptVersion: "summary.v1",
		Temperature:   0.2,
		MaxTokens:     512,
		InputHash:     "abcdef0123456789",
	}
	s := key.String()
	assert.Contains(t, s, "domain=energy")
	assert.Contains(t, s, "provider=ollama")
	assert.Contains(t, s, "model=mistral:7b")
	assert.Contains(t, s, "prompt=summary.v1")
	assert.Contains(t, s, "temp=0.20")
	assert.Contains(t, s, "max_tokens=512")
}

func TestSummaryCacheKeyStringTruncatesHash(t *testing.T) {
	key := SummaryCacheKey{
		InputHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	s := key.String()
	assert.Contains(t, s, "input_hash=0123456789ab")
	assert.NotContains(t, s, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
}

func TestComputeInputHashIs64CharHex(t *testing.T) {
	h := ComputeInputHash(nil, 0, 0)
	require.Len(t, h, 64)
	for _, c := range h {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"hash must be lowercase hex, got %q", h)
	}
}