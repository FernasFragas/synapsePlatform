package provider

// CompletionRequest is the shared, provider-agnostic input for a completion call.
type CompletionRequest struct {
	Model       string
	Prompt      string
	MaxTokens   int
	Temperature float64
	Metadata    map[string]string
}

// CompletionResponse is the shared, provider-agnostic output of a completion call.
//
// Raw is optional and intended for debugging provider-specific behavior. It must
// not be returned directly from public APIs.
type CompletionResponse struct {
	Content    string
	Model      string
	TokenUsage TokenUsage
	LatencyMs  int64
	Provider   string
	Raw        []byte
}

// EmbedRequest is the shared, provider-agnostic input for an embedding call.
type EmbedRequest struct {
	Model    string
	Input    []string
	Metadata map[string]string
}

// EmbedResponse is the shared, provider-agnostic output of an embedding call.
//
// Raw is optional and intended for debugging provider-specific behavior. It must
// not be returned directly from public APIs.
type EmbedResponse struct {
	Embeddings [][]float32
	Model      string
	TokenUsage TokenUsage
	LatencyMs  int64
	Provider   string
	Raw        []byte
}

// TokenUsage reports the token accounting for a single model call.
// Fields that a provider does not report should be left as zero values rather
// than invented.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}