package llm

import (
	"context"
	"encoding/json"
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
		assert.Equal(t, "llama3.2:3b", req.Model)
		assert.Equal(t, wantPrompt, req.Prompt)
		assert.False(t, req.Stream)

		_ = json.NewEncoder(w).Encode(generateResponse{Response: wantResponse, Done: true})
	}))
	defer srv.Close()

	client := NewOllamaClient(srv.URL, "llama3.2:3b", 0.2, 512, 30*time.Second)
	got, err := client.Complete(context.Background(), wantPrompt)
	require.NoError(t, err)
	assert.Equal(t, wantResponse, got)
}

func TestClientCompleteErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewOllamaClient(srv.URL, "x", 0.2, 512, 30*time.Second)
	_, err := client.Complete(context.Background(), "prompt")
	require.Error(t, err)
}
