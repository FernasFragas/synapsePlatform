package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const ollamaProviderName = "ollama"

// Client is the Ollama implementation of the provider target behavior.
// It supports completion (via /api/generate) and embedding (via /api/embeddings)
// against a single configured host. The model used for each call is taken from
// the request when set, falling back to the client's configured default.
type Client struct {
	httpClient       *http.Client
	baseURL          string
	completionModel  string
	embeddingModel   string
	defaultMaxTokens int
	defaultTemp      float64
}

// NewOllamaClient builds an Ollama provider client.
//
// host defaults to http://localhost:11434 when empty. completionModel is used
// when a CompletionRequest does not specify a Model. embeddingModel is used
// when an EmbedRequest does not specify a Model. timeout keeps the request
// timeout configurable as required by the plan.
func NewOllamaClient(host, completionModel, embeddingModel string, temperature float64, maxTokens int, timeout time.Duration) *Client {
	if host == "" {
		host = "http://localhost:11434"
	}

	return &Client{
		httpClient:       &http.Client{Timeout: timeout},
		baseURL:          host,
		completionModel:  completionModel,
		embeddingModel:   embeddingModel,
		defaultMaxTokens: maxTokens,
		defaultTemp:      temperature,
	}
}

// Name returns the provider identifier used for routing and telemetry.
func (c *Client) Name() string { return ollamaProviderName }

type generateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Options map[string]any `json:"options"`
}

// generateResponse maps the Ollama /api/generate payload. Token usage fields
// are populated when the server reports them; callers must tolerate zero
// values when the server omits them.
type generateResponse struct {
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	EvalCount          int    `json:"eval_count"`
	TotalDuration      int64  `json:"total_duration"` // nanoseconds
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
}

// embeddingsRequest maps the Ollama /api/embeddings payload.
type embeddingsRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// embeddingsResponse maps the Ollama /api/embeddings payload. Ollama returns a
// single embedding vector per call, so Embed batches one request per input.
type embeddingsResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Complete calls Ollama's generation endpoint and returns a structured
// completion response with provider name, model, latency, and token usage.
func (c *Client) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if c == nil {
		return nil, ErrUnavailableProvider
	}

	if req.Prompt == "" {
		return nil, fmt.Errorf("%w: empty prompt", ErrUnsupportedOperation)
	}

	model := req.Model
	if model == "" {
		model = c.completionModel
	}

	if model == "" {
		return nil, fmt.Errorf("%w: no completion model configured", ErrUnsupportedOperation)
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = c.defaultMaxTokens
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.defaultTemp
	}

	body := generateRequest{
		Model:  model,
		Prompt: req.Prompt,
		Stream: false,
		Options: map[string]any{
			"temperature": temperature,
			"num_predict": maxTokens,
		},
	}

	start := time.Now()
	raw, err := c.doPost(ctx, "/api/generate", body)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}

	var out generateResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: decode ollama generate response: %v", ErrInvalidResponse, err)
	}

	return &CompletionResponse{
		Content:   out.Response,
		Model:     model,
		Provider:  ollamaProviderName,
		LatencyMs: latencyMs,
		TokenUsage: TokenUsage{
			PromptTokens:     out.PromptEvalCount,
			CompletionTokens: out.EvalCount,
			TotalTokens:      out.PromptEvalCount + out.EvalCount,
		},
		Raw: raw,
	}, nil
}

// Embed calls Ollama's embeddings endpoint for each input text and returns one
// embedding vector per input. The configured embedding model is used unless
// the request overrides it.
func (c *Client) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	if c == nil {
		return nil, ErrUnavailableProvider
	}

	if len(req.Input) == 0 {
		return nil, fmt.Errorf("%w: empty embedding input", ErrUnsupportedOperation)
	}

	model := req.Model
	if model == "" {
		model = c.embeddingModel
	}

	if model == "" {
		return nil, fmt.Errorf("%w: no embedding model configured", ErrUnsupportedOperation)
	}

	embeddings := make([][]float32, 0, len(req.Input))
	var totalPromptTokens int
	start := time.Now()

	for _, text := range req.Input {
		body := embeddingsRequest{Model: model, Prompt: text}

		raw, err := c.doPost(ctx, "/api/embeddings", body)
		if err != nil {
			return nil, err
		}

		var out embeddingsResponse
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("%w: decode ollama embeddings response: %v", ErrInvalidResponse, err)
		}
		embeddings = append(embeddings, toFloat32Slice(out.Embedding))
		// Ollama's /api/embeddings does not report token counts; leave usage at
		// zero rather than inventing numbers, per the plan's guidance.
	}

	return &EmbedResponse{
		Embeddings: embeddings,
		Model:      model,
		Provider:   ollamaProviderName,
		LatencyMs:  time.Since(start).Milliseconds(),
		TokenUsage: TokenUsage{
			PromptTokens: totalPromptTokens,
			TotalTokens:  totalPromptTokens,
		},
	}, nil
}

// Model returns the configured default completion model. Retained for
// backwards compatibility while the summary service is migrated to request
// scoped models.
func (c *Client) Model() string { return c.completionModel }

func (c *Client) doPost(ctx context.Context, path string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%w: create request: %v", ErrUnavailableProvider, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network/timeout errors mean the provider is unavailable — eligible
		// for fallback by the router.
		return nil, fmt.Errorf("%w: ollama request failed: %v", ErrUnavailableProvider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout {
		return nil, fmt.Errorf("%w: ollama returned status %d", ErrUnavailableProvider, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func toFloat32Slice(in []float64) []float32 {
	out := make([]float32, len(in))

	for i, v := range in {
		out[i] = float32(v)
	}

	return out
}
