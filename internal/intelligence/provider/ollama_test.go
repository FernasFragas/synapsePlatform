package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientComplete(t *testing.T) {
	wantPrompt := "summarize these stats"
	wantResponse := "There were 42 energy events."

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/generate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var req generateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "mistral:7b", req.Model)
		assert.Equal(t, wantPrompt, req.Prompt)
		assert.False(t, req.Stream)

		_ = json.NewEncoder(w).Encode(generateResponse{
			Response:        wantResponse,
			Done:            true,
			PromptEvalCount: 12,
			EvalCount:       8,
		})
	}))
	defer srv.Close()

	client := NewOllamaClient(srv.URL, "mistral:7b", "nomic-embed-text", 0.2, 512, 30*time.Second)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Model:       "mistral:7b",
		Prompt:      wantPrompt,
		MaxTokens:   512,
		Temperature: 0.2,
	})

	require.NoError(t, err)
	assert.Equal(t, wantResponse, resp.Content)
	assert.Equal(t, "mistral:7b", resp.Model)
	assert.Equal(t, "ollama", resp.Provider)
	assert.Equal(t, "ollama", client.Name())
	assert.Equal(t, 12, resp.TokenUsage.PromptTokens)
	assert.Equal(t, 8, resp.TokenUsage.CompletionTokens)
	assert.Equal(t, 20, resp.TokenUsage.TotalTokens)
	assert.GreaterOrEqual(t, resp.LatencyMs, int64(0))
}

func TestClientCompleteUsesDefaultModelAndParams(t *testing.T) {
	var captured generateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))
		_ = json.NewEncoder(w).Encode(generateResponse{Response: "ok", Done: true})
	}))
	defer srv.Close()

	client := NewOllamaClient(srv.URL, "default-model", "embed-model", 0.7, 100, 30*time.Second)
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "default-model", captured.Model)
	assert.Equal(t, float64(100), captured.Options["num_predict"])
	assert.Equal(t, 0.7, captured.Options["temperature"])
}

func TestClientCompleteErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewOllamaClient(srv.URL, "x", "y", 0.2, 512, 30*time.Second)
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: "prompt"})
	require.Error(t, err)
}

func TestClientCompleteUnavailableMapsToTypedError(t *testing.T) {
	// Point at a closed port to force a network error.
	client := NewOllamaClient("http://127.0.0.1:0", "x", "y", 0.2, 512, 50*time.Millisecond)
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: "prompt"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnavailableProvider), "want ErrUnavailableProvider, got %v", err)
}

func TestClientCompleteEmptyPromptRejected(t *testing.T) {
	client := NewOllamaClient("http://x", "m", "e", 0.2, 512, time.Second)
	_, err := client.Complete(context.Background(), CompletionRequest{Prompt: ""})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedOperation))
}

func TestClientEmbed(t *testing.T) {
	wantTexts := []string{"alpha", "beta"}
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/embeddings", r.URL.Path)
		var req embeddingsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "nomic-embed-text", req.Model)
		assert.Equal(t, wantTexts[callCount], req.Prompt)
		callCount++
		_ = json.NewEncoder(w).Encode(embeddingsResponse{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
	defer srv.Close()

	client := NewOllamaClient(srv.URL, "mistral:7b", "nomic-embed-text", 0.2, 512, 30*time.Second)
	resp, err := client.Embed(context.Background(), EmbedRequest{Input: wantTexts})
	require.NoError(t, err)
	assert.Equal(t, 2, len(resp.Embeddings))
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, resp.Embeddings[0])
	assert.Equal(t, "nomic-embed-text", resp.Model)
	assert.Equal(t, "ollama", resp.Provider)
	// Ollama embeddings endpoint does not report token usage; must be zero.
	assert.Equal(t, 0, resp.TokenUsage.TotalTokens)
}

func TestClientEmbedEmptyInputRejected(t *testing.T) {
	client := NewOllamaClient("http://x", "m", "e", 0.2, 512, time.Second)
	_, err := client.Embed(context.Background(), EmbedRequest{Input: nil})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedOperation))
}

func TestClientEmbedNoModelConfiguredRejected(t *testing.T) {
	client := NewOllamaClient("http://x", "completion-only", "", 0.2, 512, time.Second)
	_, err := client.Embed(context.Background(), EmbedRequest{Input: []string{"text"}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedOperation))
}
