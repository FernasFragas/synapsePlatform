package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testOpenAIKey = "test-secret-key"

func newTestOpenAIConfig(t *testing.T, baseURL, apiKeyEnv string) OpenAICompatibleConfig {
	t.Helper()
	if apiKeyEnv == "" {
		apiKeyEnv = "SYNAPSE_TEST_OPENAI_KEY"
	}
	t.Setenv(apiKeyEnv, testOpenAIKey)
	return OpenAICompatibleConfig{
		Name:            "openai",
		BaseURL:         baseURL,
		APIKeyEnv:       apiKeyEnv,
		CompletionModel: "gpt-4o-mini",
		EmbeddingModel:  "text-embedding-3-small",
		Timeout:         10 * time.Second,
	}
}

func TestOpenAICompleteSuccess(t *testing.T) {
	var (
		gotAuth     string
		gotModel    string
		gotPrompt   string
		gotMaxTokens int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		gotAuth = r.Header.Get("Authorization")

		var req oaiChatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		gotModel = req.Model
		gotPrompt = req.Messages[0].Content
		gotMaxTokens = req.MaxTokens
		assert.False(t, req.Stream)

		_ = json.NewEncoder(w).Encode(oaiChatResponse{
			Choices: []struct {
				Message      oaiMessage `json:"message"`
				FinishReason string     `json:"finish_reason"`
			}{
				{Message: oaiMessage{Role: "assistant", Content: "summary text"}, FinishReason: "stop"},
			},
			Usage: oaiUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		})
	}))
	defer srv.Close()

	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, srv.URL, ""))
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Prompt:    "summarize",
		MaxTokens: 256,
	})
	require.NoError(t, err)
	assert.Equal(t, "summary text", resp.Content)
	assert.Equal(t, "gpt-4o-mini", resp.Model)
	assert.Equal(t, "openai", resp.Provider)
	assert.Equal(t, "Bearer "+testOpenAIKey, gotAuth)
	assert.Equal(t, "gpt-4o-mini", gotModel)
	assert.Equal(t, "summarize", gotPrompt)
	assert.Equal(t, 256, gotMaxTokens)
	assert.Equal(t, 10, resp.TokenUsage.PromptTokens)
	assert.Equal(t, 5, resp.TokenUsage.CompletionTokens)
	assert.Equal(t, 15, resp.TokenUsage.TotalTokens)
	assert.Equal(t, "openai", client.Name())
	assert.GreaterOrEqual(t, resp.LatencyMs, int64(0))
}

func TestOpenAICompleteRequestOverridesModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req oaiChatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "gpt-4o", req.Model)
		_ = json.NewEncoder(w).Encode(oaiChatResponse{
			Choices: []struct {
				Message      oaiMessage `json:"message"`
				FinishReason string     `json:"finish_reason"`
			}{{Message: oaiMessage{Content: "ok"}}},
		})
	}))
	defer srv.Close()

	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, srv.URL, ""))
	_, err := client.Complete(context.Background(), CompletionRequest{Model: "gpt-4o", Prompt: "hi"})
	require.NoError(t, err)
}

func TestOpenAICompleteEmptyPromptRejected(t *testing.T) {
	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, "http://example", ""))
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: ""})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedOperation))
}

func TestOpenAICompleteNoModelConfiguredRejected(t *testing.T) {
	cfg := newTestOpenAIConfig(t, "http://example", "")
	cfg.CompletionModel = ""
	client := NewOpenAICompatibleClient(cfg)
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedOperation))
}

func TestOpenAICompleteErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, srv.URL, ""))
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
}

func TestOpenAICompleteUnavailableMapsToTypedError(t *testing.T) {
	// Closed port forces a network error.
	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, "http://127.0.0.1:0", ""))
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnavailableProvider), "want ErrUnavailableProvider, got %v", err)
}

func TestOpenAICompleteGatewayTimeoutMapsToUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer srv.Close()

	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, srv.URL, ""))
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnavailableProvider))
}

func TestOpenAICompleteAuthFailureMapsToUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, srv.URL, ""))
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedOperation))
}

func TestOpenAIMissingAPIKeyEnvRejected(t *testing.T) {
	// Use an env var that is never set.
	client := NewOpenAICompatibleClient(OpenAICompatibleConfig{
		Name:            "openai",
		BaseURL:         "http://example",
		APIKeyEnv:       "SYNAPSE_DEFINITELY_UNSET_KEY",
		CompletionModel: "gpt-4o-mini",
	})
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedOperation))
}

func TestOpenAIEmbedSuccess(t *testing.T) {
	var gotInput []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/embeddings", r.URL.Path)
		var req oaiEmbeddingsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		gotInput = req.Input
		_ = json.NewEncoder(w).Encode(oaiEmbeddingsResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index    int        `json:"index"`
			}{
				{Embedding: []float32{0.1, 0.2}, Index: 0},
				{Embedding: []float32{0.3, 0.4}, Index: 1},
			},
			Usage: oaiUsage{PromptTokens: 7, TotalTokens: 7},
		})
	}))
	defer srv.Close()

	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, srv.URL, ""))
	resp, err := client.Embed(context.Background(), EmbedRequest{Input: []string{"a", "b"}})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, gotInput)
	assert.Equal(t, 2, len(resp.Embeddings))
	assert.Equal(t, []float32{0.1, 0.2}, resp.Embeddings[0])
	assert.Equal(t, []float32{0.3, 0.4}, resp.Embeddings[1])
	assert.Equal(t, "text-embedding-3-small", resp.Model)
	assert.Equal(t, 7, resp.TokenUsage.PromptTokens)
	assert.Equal(t, 7, resp.TokenUsage.TotalTokens)
}

func TestOpenAIEmbedCountMismatchRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(oaiEmbeddingsResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index    int        `json:"index"`
			}{
				{Embedding: []float32{0.1}, Index: 0},
			},
		})
	}))
	defer srv.Close()

	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, srv.URL, ""))
	_, err := client.Embed(context.Background(), EmbedRequest{Input: []string{"a", "b"}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidResponse))
}

func TestOpenAIEmbedEmptyInputRejected(t *testing.T) {
	client := NewOpenAICompatibleClient(newTestOpenAIConfig(t, "http://example", ""))
	_, err := client.Embed(context.Background(), EmbedRequest{Input: nil})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedOperation))
}

func TestNewOpenAICompatibleClientDefaults(t *testing.T) {
	t.Setenv("SYNAPSE_TEST_OPENAI_KEY", testOpenAIKey)
	client := NewOpenAICompatibleClient(OpenAICompatibleConfig{
		Name:            "",
		APIKeyEnv:       "SYNAPSE_TEST_OPENAI_KEY",
		CompletionModel: "m",
		EmbeddingModel:  "e",
	})
	assert.Equal(t, "https://api.openai.com", client.cfg.BaseURL)
	assert.Equal(t, openAICompatibleProviderName, client.Name())
}

func TestNewOpenAICompatibleClientTrimsTrailingSlash(t *testing.T) {
	client := NewOpenAICompatibleClient(OpenAICompatibleConfig{
		BaseURL:         "https://api.example.com/",
		APIKeyEnv:       "SYNAPSE_TEST_OPENAI_KEY",
		CompletionModel: "m",
	})
	assert.False(t, strings.HasSuffix(client.cfg.BaseURL, "/"))
}

// Ensure the package still compiles when the env var helper is unused on
// systems where os.Getenv is stubbed out.
var _ = os.Getenv