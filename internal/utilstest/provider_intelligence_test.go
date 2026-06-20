package utilstest

import (
	"context"
	"errors"
	"synapsePlatform/internal/intelligence/provider"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockCompleteReturnsConfiguredOutput(t *testing.T) {
	m := NewMock()
	m.CompletionOutput = "configured summary"

	resp, err := m.Complete(context.Background(), provider.CompletionRequest{Prompt: "anything"})
	require.NoError(t, err)
	assert.Equal(t, "configured summary", resp.Content)
	assert.Equal(t, "mock-completion", resp.Model)
	assert.Equal(t, "mock", resp.Provider)
	assert.Equal(t, 1, m.CompleteCalls())
}

func TestMockCompleteEchoesPromptWhenOutputUnset(t *testing.T) {
	m := NewMock()
	resp, err := m.Complete(context.Background(), provider.CompletionRequest{Prompt: "echo me"})
	require.NoError(t, err)
	assert.Equal(t, "echo me", resp.Content)
}

func TestMockCompleteRequestModelOverridesDefault(t *testing.T) {
	m := NewMock()
	resp, err := m.Complete(context.Background(), provider.CompletionRequest{Model: "custom-model", Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "custom-model", resp.Model)
}

func TestMockCompleteDerivesNonZeroTokenUsage(t *testing.T) {
	m := NewMock()
	resp, err := m.Complete(context.Background(), provider.CompletionRequest{Prompt: "a prompt with words"})
	require.NoError(t, err)
	assert.Greater(t, resp.TokenUsage.PromptTokens, 0)
	assert.Greater(t, resp.TokenUsage.CompletionTokens, 0)
	assert.Equal(t, resp.TokenUsage.PromptTokens+resp.TokenUsage.CompletionTokens, resp.TokenUsage.TotalTokens)
}

func TestMockCompleteUsesConfiguredLatencyAndUsage(t *testing.T) {
	m := NewMock()
	m.CompletionLatency = 42_000_000 // 42ms in ns -> 42ms reported
	m.CompletionUsage = provider.TokenUsage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5}

	resp, err := m.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.LatencyMs)
	assert.Equal(t, 3, resp.TokenUsage.PromptTokens)
	assert.Equal(t, 2, resp.TokenUsage.CompletionTokens)
	assert.Equal(t, 5, resp.TokenUsage.TotalTokens)
}

func TestMockCompleteInjectsError(t *testing.T) {
	m := NewMock()
	m.CompleteErr = provider.ErrUnavailableProvider

	_, err := m.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrUnavailableProvider))
	// Failed calls are still counted, so router tests can assert attempt counts.
	assert.Equal(t, 1, m.CompleteCalls())
}

func TestMockCompleteRespectsContextCancellation(t *testing.T) {
	m := NewMock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := m.Complete(ctx, provider.CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMockEmbedDeterministic(t *testing.T) {
	m := NewMock()

	resp1, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"alpha", "beta"}})
	require.NoError(t, err)
	resp2, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"alpha", "beta"}})
	require.NoError(t, err)

	require.Len(t, resp1.Embeddings, 2)
	require.Len(t, resp2.Embeddings, 2)
	assert.Equal(t, resp1.Embeddings[0], resp2.Embeddings[0], "same input must produce identical vectors")
	assert.Equal(t, resp1.Embeddings[1], resp2.Embeddings[1])
	assert.NotEqual(t, resp1.Embeddings[0], resp1.Embeddings[1], "different inputs should differ")
}

func TestMockEmbedDifferentTextsProduceDifferentVectors(t *testing.T) {
	m := NewMock()
	resp, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"power spike", "auth failure"}})
	require.NoError(t, err)
	assert.NotEqual(t, resp.Embeddings[0], resp.Embeddings[1])
}

func TestMockEmbedIsL2Normalized(t *testing.T) {
	m := NewMock()
	m.EmbeddingDimensions = 8

	resp, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"normalized vector"}})
	require.NoError(t, err)
	require.Len(t, resp.Embeddings, 1)

	var sumSq float32
	for _, x := range resp.Embeddings[0] {
		sumSq += x * x
	}
	assert.InDelta(t, 1.0, sumSq, 0.001, "vector should be L2-normalized for cosine search")
}

func TestMockEmbedUsesConfiguredDimensions(t *testing.T) {
	m := NewMock()
	m.EmbeddingDimensions = 32

	resp, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"x"}})
	require.NoError(t, err)
	assert.Len(t, resp.Embeddings[0], 32)
}

func TestMockEmbedDefaultsDimensionsWhenUnset(t *testing.T) {
	m := NewMock()
	m.EmbeddingDimensions = 0

	resp, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"x"}})
	require.NoError(t, err)
	assert.Len(t, resp.Embeddings[0], 16)
}

func TestMockEmbedReportsModelAndProvider(t *testing.T) {
	m := NewMock()
	resp, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"x"}})
	require.NoError(t, err)
	assert.Equal(t, "mock-embedding", resp.Model)
	assert.Equal(t, "mock", resp.Provider)
	assert.Equal(t, 1, m.EmbedCalls())
}

func TestMockEmbedDerivesNonZeroTokenUsage(t *testing.T) {
	m := NewMock()
	resp, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"alpha", "beta"}})
	require.NoError(t, err)
	assert.Greater(t, resp.TokenUsage.PromptTokens, 0)
	assert.Equal(t, resp.TokenUsage.PromptTokens, resp.TokenUsage.TotalTokens)
}

func TestMockEmbedUsesConfiguredUsage(t *testing.T) {
	m := NewMock()
	m.EmbeddingUsage = provider.TokenUsage{PromptTokens: 9, TotalTokens: 9}

	resp, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"a", "b"}})
	require.NoError(t, err)
	assert.Equal(t, 9, resp.TokenUsage.PromptTokens)
	assert.Equal(t, 9, resp.TokenUsage.TotalTokens)
}

func TestMockEmbedInjectsError(t *testing.T) {
	m := NewMock()
	m.EmbedErr = provider.ErrInvalidResponse

	_, err := m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"x"}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrInvalidResponse))
}

func TestMockWithNameOverridesReportedName(t *testing.T) {
	m := NewMock().WithName("mock-primary")
	assert.Equal(t, "mock-primary", m.Name())

	resp, err := m.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "mock-primary", resp.Provider)
}

func TestMockNilGuardsReturnUnavailable(t *testing.T) {
	var m *Mock
	_, err := m.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrUnavailableProvider))

	_, err = m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"x"}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrUnavailableProvider))
}

func TestMockDeterministicVectorEmptyInput(t *testing.T) {
	v := deterministicVector("", 4)
	assert.Len(t, v, 4)
	assert.Equal(t, []float32{0, 0, 0, 0}, v)
}

func TestMockCallCountersIncrementIndependently(t *testing.T) {
	m := NewMock()

	_, _ = m.Complete(context.Background(), provider.CompletionRequest{Prompt: "a"})
	_, _ = m.Complete(context.Background(), provider.CompletionRequest{Prompt: "b"})
	_, _ = m.Embed(context.Background(), provider.EmbedRequest{Input: []string{"x"}})

	assert.Equal(t, 2, m.CompleteCalls())
	assert.Equal(t, 1, m.EmbedCalls())
}
