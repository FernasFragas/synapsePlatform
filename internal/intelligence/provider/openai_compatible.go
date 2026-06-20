package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const openAICompatibleProviderName = "openai_compatible"

// OpenAICompatibleConfig configures an OpenAI-compatible provider.
//
// BaseURL is the API root without a trailing slash (for example
// "https://api.openai.com" or "https://api.vllm.ai"). APIKeyEnv names the
// environment variable that holds the bearer token; the key itself is read
// lazily on each call so rotation does not require a restart. CompletionModel
// and EmbeddingModel are defaults used when a request does not set Model.
type OpenAICompatibleConfig struct {
	Name            string
	BaseURL         string
	APIKeyEnv       string
	CompletionModel string
	EmbeddingModel  string
	Timeout         time.Duration
}

// OpenAICompatibleClient is a provider target implementation that speaks the
// OpenAI Chat Completions and Embeddings API shapes. It works against OpenAI
// itself and against compatible endpoints (vLLM, Together, LiteLLM, etc.) as
// long as their request/response shape matches.
//
// Provider-specific request/response mapping lives here, never in the summary
// or embedding business packages.
type OpenAICompatibleClient struct {
	cfg        OpenAICompatibleConfig
	httpClient *http.Client
}

// NewOpenAICompatibleClient builds a provider against an OpenAI-compatible API.
// The API key is read from the environment on each call, so a missing key is
// not a constructor-time error.
func NewOpenAICompatibleClient(cfg OpenAICompatibleConfig) *OpenAICompatibleClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com"
	}

	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &OpenAICompatibleClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// Name returns the configured provider name, defaulting to
// openai_compatible. The router keys providers by name; two OpenAI-compatible
// providers with different base URLs must have different names.
func (c *OpenAICompatibleClient) Name() string {
	if c == nil {
		return openAICompatibleProviderName
	}

	if c.cfg.Name != "" {
		return c.cfg.Name
	}

	return openAICompatibleProviderName
}

// --- Chat Completions ---

type oaiChatRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	Temperature float64      `json:"temperature,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Stream      bool         `json:"stream"`
}

type oaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaiChatResponse struct {
	Choices []struct {
		Message      oaiMessage `json:"message"`
		FinishReason string     `json:"finish_reason"`
	} `json:"choices"`
	Usage oaiUsage `json:"usage"`
}

type oaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Complete calls POST /v1/chat/completions and maps the first choice into a
// structured completion response.
func (c *OpenAICompatibleClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if c == nil {
		return nil, ErrUnavailableProvider
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("%w: empty prompt", ErrUnsupportedOperation)
	}

	model := req.Model
	if model == "" {
		model = c.cfg.CompletionModel
	}

	if model == "" {
		return nil, fmt.Errorf("%w: no completion model configured", ErrUnsupportedOperation)
	}

	maxTokens := req.MaxTokens
	if maxTokens < 0 {
		maxTokens = 0
	}

	body := oaiChatRequest{
		Model:     model,
		Stream:    false,
		MaxTokens: maxTokens,
		Messages: []oaiMessage{
			{Role: "user", Content: req.Prompt},
		},
	}

	if req.Temperature > 0 {
		body.Temperature = req.Temperature
	}

	start := time.Now()
	raw, err := c.doPost(ctx, "/v1/chat/completions", body)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}

	var out oaiChatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: decode openai chat response: %v", ErrInvalidResponse, err)
	}

	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("%w: openai chat response had no choices", ErrInvalidResponse)
	}

	return &CompletionResponse{
		Content:    out.Choices[0].Message.Content,
		Model:      model,
		Provider:   c.Name(),
		LatencyMs:  latencyMs,
		TokenUsage: toTokenUsage(out.Usage),
		Raw:        raw,
	}, nil
}

// --- Embeddings ---

type oaiEmbeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type oaiEmbeddingsResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage oaiUsage `json:"usage"`
}

// Embed calls POST /v1/embeddings with the full input batch and returns one
// vector per input, ordered by index. The OpenAI embeddings endpoint accepts
// an array of inputs, so Embed is a single HTTP call rather than one per
// input like the Ollama provider.
func (c *OpenAICompatibleClient) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	if c == nil {
		return nil, ErrUnavailableProvider
	}

	if len(req.Input) == 0 {
		return nil, fmt.Errorf("%w: empty embedding input", ErrUnsupportedOperation)
	}

	model := req.Model
	if model == "" {
		model = c.cfg.EmbeddingModel
	}

	if model == "" {
		return nil, fmt.Errorf("%w: no embedding model configured", ErrUnsupportedOperation)
	}

	body := oaiEmbeddingsRequest{Model: model, Input: req.Input}

	start := time.Now()
	raw, err := c.doPost(ctx, "/v1/embeddings", body)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}

	var out oaiEmbeddingsResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: decode openai embeddings response: %v", ErrInvalidResponse, err)
	}

	if len(out.Data) != len(req.Input) {
		return nil, fmt.Errorf("%w: expected %d embeddings, got %d", ErrInvalidResponse, len(req.Input), len(out.Data))
	}

	// Server may return embeddings out of order; sort by index so callers get
	// vectors aligned to their input order.
	embeddings := make([][]float32, len(out.Data))
	for i, item := range out.Data {
		if item.Index < 0 || item.Index >= len(embeddings) {
			return nil, fmt.Errorf("%w: embedding index %d out of range", ErrInvalidResponse, item.Index)
		}
		embeddings[i] = item.Embedding
	}

	return &EmbedResponse{
		Embeddings: embeddings,
		Model:      model,
		Provider:   c.Name(),
		LatencyMs:  latencyMs,
		TokenUsage: toTokenUsage(out.Usage),
	}, nil
}

// Model returns the configured default completion model.
func (c *OpenAICompatibleClient) Model() string { return c.cfg.CompletionModel }

// doPost issues an authenticated POST and returns the raw body. Auth failures
// (missing env var) are reported as ErrUnsupportedOperation because retrying
// against another provider will not fix a missing local secret. Network and
// 5xx/gateway errors map to ErrUnavailableProvider so the router can fall
// back. 4xx errors (other than auth-related) are surfaced as generic errors
// since they typically indicate a malformed request.
func (c *OpenAICompatibleClient) doPost(ctx context.Context, path string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	url := c.cfg.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%w: create request: %v", ErrUnavailableProvider, err)
	}
	req.Header.Set("Content-Type", "application/json")

	if c.cfg.APIKeyEnv != "" {
		token := os.Getenv(c.cfg.APIKeyEnv)
		if token == "" {
			return nil, fmt.Errorf("%w: api key env %q is not set", ErrUnsupportedOperation, c.cfg.APIKeyEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: openai request failed: %v", ErrUnavailableProvider, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusServiceUnavailable, resp.StatusCode == http.StatusGatewayTimeout,
		resp.StatusCode == http.StatusBadGateway, resp.StatusCode == http.StatusRequestTimeout:
		return nil, fmt.Errorf("%w: openai returned status %d", ErrUnavailableProvider, resp.StatusCode)
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w: openai auth failed with status %d", ErrUnsupportedOperation, resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("openai returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func toTokenUsage(u oaiUsage) TokenUsage {
	return TokenUsage{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		TotalTokens:      u.TotalTokens,
	}
}
