package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	httpClient  *http.Client
	baseURL     string
	model       string
	temperature float64
	maxTokens   int
}

func NewOllamaClient(host, model string, temperature float64, maxTokens int, timeout time.Duration) *Client {
	if host == "" {
		host = "http://localhost:11434"
	}
	return &Client{
		httpClient:  &http.Client{Timeout: timeout},
		baseURL:     host,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

type generateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`  // false
	Options map[string]any `json:"options"` // {"temperature":..,"num_predict":..}
}
type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	reqBody := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]any{
			"temperature": c.temperature,
			"num_predict": c.maxTokens,
		},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal generate request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("create generate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama generate request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}
	var out generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}
	return out.Response, nil
}

func (c *Client) Model() string { return c.model }
