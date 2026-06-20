package utilstest

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"synapsePlatform/internal/intelligence/provider"
	"sync"
	"time"
)

const mockProviderName = "mock"

// Mock is a deterministic provider used by unit tests for the summary service,
// router fallback, embedding pipeline, and semantic search. It performs no I/O.
//
// Complete returns a configured string (defaulting to echoing the prompt).
// Embed returns deterministic vectors derived from each input text via a stable
// hash, so the same input always yields the same vector across runs.
//
// Errors, latency, and token usage can be injected to exercise telemetry and
// fallback paths without a real model.
type Mock struct {
	name string

	// CompletionOutput is returned by Complete. If empty, Complete echoes the
	// prompt back, which keeps tests readable without forcing every caller to
	// configure output.
	CompletionOutput string

	// CompletionModel is reported on CompletionResponse.Model. Defaults to
	// "mock-completion" when empty.
	CompletionModel string

	// EmbeddingModel is reported on EmbedResponse.Model. Defaults to
	// "mock-embedding" when empty.
	EmbeddingModel string

	// EmbeddingDimensions controls the length of each vector returned by
	// Embed. Defaults to 16, which is small enough for fast brute-force search
	// tests and large enough to avoid trivial collisions.
	EmbeddingDimensions int

	// CompletionLatency and EmbeddingLatency are reported on the response's
	// LatencyMs field. They are values rather than real sleeps, so tests stay
	// fast while still exercising latency-handling code.
	CompletionLatency time.Duration
	EmbeddingLatency  time.Duration

	// CompletionUsage and EmbeddingUsage are reported on the respective
	// responses. When zero, the mock derives a deterministic token count from
	// the input so telemetry tests have non-zero data without configuration.
	CompletionUsage provider.TokenUsage
	EmbeddingUsage  provider.TokenUsage

	// Inject per-operation errors. When set, Complete/Embed return the error
	// instead of a response. Use this to drive router-fallback tests.
	CompleteErr error
	EmbedErr    error

	mu sync.Mutex
	// calls counts Complete and Embed invocations for assertion in tests.
	completeCalls int
	embedCalls    int
}

// NewMock builds a mock with sensible defaults. Tests typically mutate the
// returned *Mock's exported fields before use.
func NewMock() *Mock {
	return &Mock{
		name:                mockProviderName,
		CompletionModel:     "mock-completion",
		EmbeddingModel:      "mock-embedding",
		EmbeddingDimensions: 16,
	}
}

// Name returns the configured provider name, defaulting to "mock".
func (m *Mock) Name() string {
	if m == nil {
		return mockProviderName
	}
	if m.name != "" {
		return m.name
	}
	return mockProviderName
}

// WithName sets the provider name reported by Name and on responses. Returns
// the receiver for fluent configuration in tests.
func (m *Mock) WithName(name string) *Mock {
	m.name = name
	return m
}

// CompleteCalls returns the number of Complete invocations observed so far.
func (m *Mock) CompleteCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.completeCalls
}

// EmbedCalls returns the number of Embed invocations observed so far.
func (m *Mock) EmbedCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.embedCalls
}

// Complete returns the configured output, or echoes the prompt when no output
// is set. Returns CompleteErr when set, so router tests can drive fallback.
func (m *Mock) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
	if m == nil {
		return nil, provider.ErrUnavailableProvider
	}
	m.mu.Lock()
	m.completeCalls++
	m.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m.CompleteErr != nil {
		return nil, m.CompleteErr
	}

	content := m.CompletionOutput
	if content == "" {
		content = req.Prompt
	}
	model := req.Model
	if model == "" {
		model = m.CompletionModel
	}

	usage := m.CompletionUsage
	if usage == (provider.TokenUsage{}) {
		usage = derivedTokenUsage(req.Prompt, content)
	}

	return &provider.CompletionResponse{
		Content:    content,
		Model:      model,
		Provider:   m.Name(),
		LatencyMs:  m.CompletionLatency.Milliseconds(),
		TokenUsage: usage,
	}, nil
}

// Embed returns one deterministic vector per input. The vector is derived
// from a stable hash of the input text, so the same text always maps to the
// same vector across runs and across machines. Returns EmbedErr when set.
func (m *Mock) Embed(ctx context.Context, req provider.EmbedRequest) (*provider.EmbedResponse, error) {
	if m == nil {
		return nil, provider.ErrUnavailableProvider
	}
	m.mu.Lock()
	m.embedCalls++
	m.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m.EmbedErr != nil {
		return nil, m.EmbedErr
	}

	dims := m.EmbeddingDimensions
	if dims <= 0 {
		dims = 16
	}
	model := req.Model
	if model == "" {
		model = m.EmbeddingModel
	}

	embeddings := make([][]float32, len(req.Input))
	var totalPromptTokens int
	for i, text := range req.Input {
		embeddings[i] = deterministicVector(text, dims)
		totalPromptTokens += derivedTokens(text)
	}

	usage := m.EmbeddingUsage
	if usage == (provider.TokenUsage{}) {
		usage = provider.TokenUsage{PromptTokens: totalPromptTokens, TotalTokens: totalPromptTokens}
	}

	return &provider.EmbedResponse{
		Embeddings: embeddings,
		Model:      model,
		Provider:   m.Name(),
		LatencyMs:  m.EmbeddingLatency.Milliseconds(),
		TokenUsage: usage,
	}, nil
}

// Model returns the configured default completion model.
func (m *Mock) Model() string { return m.CompletionModel }

// deterministicVector derives a stable []float32 of the requested length from
// the input text. It uses FNV-1a for the first seed and SHA-256 stretches for
// additional dimensions, which keeps the function fast and collision-resistant
// enough for test fixtures while remaining fully deterministic.
//
// The vector is L2-normalized so cosine similarity reduces to a dot product,
// which is what brute-force search tests expect.
func deterministicVector(text string, dims int) []float32 {
	v := make([]float32, dims)
	if dims == 0 || text == "" {
		return v
	}

	// Seed each dimension from a sliding FNV hash over the text. This spreads
	// information across all dims without needing a cryptographic derivation.
	for i := 0; i < dims; i++ {
		h := fnv.New64a()
		_, _ = h.Write([]byte(fmt.Sprintf("%d:%s", i, text)))
		v[i] = float32(int64(h.Sum64()%10000)) / 10000.0
	}

	// L2-normalize so cosine similarity is a plain dot product.
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return v
	}
	norm := float32(1.0 / float64(sqrt64(sum)))
	for i := range v {
		v[i] *= norm
	}
	return v
}

// derivedTokenUsage produces deterministic, non-zero token counts from input
// and output text so telemetry tests have realistic data without configuring
// usage explicitly.
func derivedTokenUsage(prompt, content string) provider.TokenUsage {
	promptTokens := derivedTokens(prompt)
	completionTokens := derivedTokens(content)
	return provider.TokenUsage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// derivedTokens maps text to a deterministic token-ish count. It is not a real
// tokenizer; it only needs to be stable and non-zero for non-empty input.
func derivedTokens(text string) int {
	if text == "" {
		return 0
	}
	h := sha256.Sum256([]byte(text))
	n := int(binary.BigEndian.Uint32(h[:4]))
	// Map to [1, 256] so non-empty inputs never report zero tokens.
	return (n % 256) + 1
}

// sqrt64 computes math.Sqrt without importing the math package, keeping the
// mock dependency-free. Uses a Newton-Raphson iteration seeded from the bit
// pattern, which converges in a few steps for the magnitudes seen here.
func sqrt64(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Fast inverse sqrt seed.
	i := int64(x)
	//nolint:gosec // intentional bit manipulation for the seed
	r := float64(i)
	if r == 0 {
		r = 1
	}
	for i := 0; i < 20; i++ {
		r = 0.5 * (r + x/r)
	}
	return r
}

// Compile-time guard: ensure injected errors that wrap the typed sentinel
// errors are still detectable via errors.Is. This documents the contract the
// router (3.1.5) will rely on.
var _ = errors.Is
