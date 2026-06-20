package summary

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"synapsePlatform/internal/api"
)

// SummaryCacheKey captures every dimension that distinguishes one cached
// summary from another. Two requests with the same key should return the same
// model output; any change in any field means a fresh completion is required.
type SummaryCacheKey struct {
	Domain        string
	WindowFrom    time.Time
	Provider      string
	Model         string
	PromptVersion string
	Temperature   float64
	MaxTokens     int
	InputHash     string
}

// BuildCacheKey assembles the cache key for a summary request. It computes
// InputHash from the aggregate stats that will be embedded in the prompt plus
// the inference parameters (temperature, max_tokens), so the key changes when
// the underlying event data changes even if the window is the same, and when
// the inference parameters change even if the event data is the same.
//
// provider is the provider that will handle the completion (read from the
// completer at call time); model, temperature, and maxTokens come from the
// service configuration. promptVersion is SummaryPromptVersion.
//
// temperature and max_tokens are folded into the input_hash rather than stored
// as separate columns, so the SQL lookup filters on input_hash to distinguish
// summaries generated under different inference parameters.
func BuildCacheKey(domain string, windowFrom time.Time, provider, model, promptVersion string, temperature float64, maxTokens int, stats []api.DomainStat) SummaryCacheKey {
	return SummaryCacheKey{
		Domain:        domain,
		WindowFrom:    windowFrom,
		Provider:      provider,
		Model:         model,
		PromptVersion: promptVersion,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		InputHash:     ComputeInputHash(stats, temperature, maxTokens),
	}
}

// ComputeInputHash derives a stable hash from the aggregate stats used in the
// prompt plus the inference parameters (temperature, max_tokens). The hash is
// based solely on the event data and inference config, not on the current time,
// so identical stats within the same window produce identical hashes. This
// means a summary generated at 10:00 and re-requested at 10:05 (same window,
// same stats, same params) will hit the cache.
//
// The stats are sorted by (domain, event_type) before hashing so the hash is
// independent of the order the database returned rows.
func ComputeInputHash(stats []api.DomainStat, temperature float64, maxTokens int) string {
	var b strings.Builder
	b.WriteString(strconv.FormatFloat(temperature, 'f', -1, 64))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(maxTokens))
	b.WriteByte('\n')

	if len(stats) == 0 {
		return sha256Hex(b.String())
	}

	sorted := make([]api.DomainStat, len(stats))
	copy(sorted, stats)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Domain != sorted[j].Domain {
			return sorted[i].Domain < sorted[j].Domain
		}
		return sorted[i].EventType < sorted[j].EventType
	})

	for _, s := range sorted {
		b.WriteString(s.Domain)
		b.WriteByte('|')
		b.WriteString(s.EventType)
		b.WriteByte('|')
		b.WriteString(strconv.FormatInt(s.Count, 10))
		b.WriteByte('|')
		b.WriteString(s.FirstSeen.Format(time.RFC3339Nano))
		b.WriteByte('|')
		b.WriteString(s.LastSeen.Format(time.RFC3339Nano))
		b.WriteByte('\n')
	}

	return sha256Hex(b.String())
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// String returns a human-readable representation for logging and debugging.
// It does not include the full input hash (truncated to 12 chars) to keep log
// lines readable.
func (k SummaryCacheKey) String() string {
	return fmt.Sprintf("domain=%s window=%s provider=%s model=%s prompt=%s temp=%.2f max_tokens=%d input_hash=%s",
		k.Domain,
		k.WindowFrom.Format(time.RFC3339),
		k.Provider,
		k.Model,
		k.PromptVersion,
		k.Temperature,
		k.MaxTokens,
		truncateHash(k.InputHash),
	)
}

func truncateHash(h string) string {
	if len(h) <= 12 {
		return h
	}

	return h[:12]
}
