package provider

import (
	"context"
	"errors"
	"fmt"
)

// ModelProvider is the target interface the router routes between. It is
// exported so external test packages (and the future telemetry decorator in
// 3.3) can construct routers with mock providers without importing the
// provider package's internal test helpers.
//
// Business packages (summary, embedding, search) should still declare their
// own smaller Completer or Embedder interfaces at the point of consumption and
// must not depend on this one.
type ModelProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error)
	Name() string
}

// RouterConfig configures a Router's defaults and fallback chain.
//
// DefaultCompletionProvider is used when a CompletionRequest does not name a
// provider (via Metadata["provider"]). DefaultEmbeddingProvider is the
// equivalent for EmbedRequest. Fallbacks maps a primary provider name to the
// single provider that should be tried when the primary fails with a
// transient error.
type RouterConfig struct {
	DefaultCompletionProvider string
	DefaultEmbeddingProvider  string
	Fallbacks                 map[string]string
}

// Router selects a provider for completion and embedding requests, applies
// configured defaults, and falls back to a configured secondary provider on
// transient errors.
//
// Routing is intentionally simple in Tier 3: explicit selection, defaults,
// and a single fallback hop. Cost-aware or latency-aware routing can be added
// later once inference telemetry (3.3) has enough data to inform it.
type Router struct {
	providers         map[string]ModelProvider
	defaultCompletion string
	defaultEmbedding  string
	fallbacks         map[string]string
}

// NewRouter builds a Router from the given providers and config.
//
// Each provider must have a non-empty, unique Name(); duplicate names are
// rejected to prevent silent routing ambiguity. If DefaultCompletionProvider
// or DefaultEmbeddingProvider is set, the corresponding provider must be
// present in the map.
func NewRouter(providers map[string]ModelProvider, cfg RouterConfig) (*Router, error) {
	if len(providers) == 0 {
		return nil, errors.New("router requires at least one provider")
	}

	seen := make(map[string]struct{}, len(providers))
	for name, p := range providers {
		if p == nil {
			return nil, fmt.Errorf("provider %q is nil", name)
		}
		if p.Name() == "" {
			return nil, fmt.Errorf("provider registered as %q has an empty Name()", name)
		}
		if p.Name() != name {
			return nil, fmt.Errorf("provider key %q does not match its Name() %q", name, p.Name())
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("duplicate provider name %q", name)
		}
		seen[name] = struct{}{}
	}

	if cfg.DefaultCompletionProvider != "" {
		if _, ok := providers[cfg.DefaultCompletionProvider]; !ok {
			return nil, fmt.Errorf("default completion provider %q is not registered", cfg.DefaultCompletionProvider)
		}
	}

	if cfg.DefaultEmbeddingProvider != "" {
		if _, ok := providers[cfg.DefaultEmbeddingProvider]; !ok {
			return nil, fmt.Errorf("default embedding provider %q is not registered", cfg.DefaultEmbeddingProvider)
		}
	}

	for primary, fb := range cfg.Fallbacks {
		if primary == fb {
			return nil, fmt.Errorf("fallback for %q points to itself", primary)
		}
		if _, ok := providers[fb]; !ok {
			return nil, fmt.Errorf("fallback %q for provider %q is not registered", fb, primary)
		}
	}

	return &Router{
		providers:         providers,
		defaultCompletion: cfg.DefaultCompletionProvider,
		defaultEmbedding:  cfg.DefaultEmbeddingProvider,
		fallbacks:         cfg.Fallbacks,
	}, nil
}

// Complete routes a completion request to the requested provider, or to the
// configured default when the request does not name one. On a transient
// provider error, it tries the configured fallback provider once. Validation,
// unsupported-operation, and invalid-response errors are returned without
// fallback, since a second provider will not fix a malformed local request.
func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	primary, err := r.selectProvider(req.Metadata, r.defaultCompletion)
	if err != nil {
		return nil, err
	}

	resp, err := primary.Complete(ctx, req)
	if err == nil {
		return resp, nil
	}

	if fb, ok := r.fallbackFor(primary.Name(), err); ok {
		return fb.Complete(ctx, req)
	}

	return nil, err
}

// Embed routes an embedding request to the requested provider, or to the
// configured default when the request does not name one. Fallback behavior
// matches Complete.
func (r *Router) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	primary, err := r.selectProvider(req.Metadata, r.defaultEmbedding)
	if err != nil {
		return nil, err
	}

	resp, err := primary.Embed(ctx, req)
	if err == nil {
		return resp, nil
	}

	if fb, ok := r.fallbackFor(primary.Name(), err); ok {
		return fb.Embed(ctx, req)
	}

	return nil, err
}

// Name reports the router identity. A router is not itself a ModelProvider by
// default; this exists so telemetry decorators (3.3) can label routed calls.
func (r *Router) Name() string { return "router" }

// selectProvider resolves which provider handles a request. The requested
// provider is read from Metadata["provider"]; when absent, the configured
// default is used. An explicit request for an unknown provider is an error,
// not a fallback candidate, since it indicates a caller misconfiguration.
func (r *Router) selectProvider(meta map[string]string, defaultName string) (ModelProvider, error) {
	requested := ""
	if meta != nil {
		requested = meta["provider"]
	}

	name := requested
	if name == "" {
		name = defaultName
	}

	if name == "" {
		return nil, fmt.Errorf("%w: no provider requested and no default configured", ErrUnsupportedOperation)
	}

	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: provider %q is not registered", ErrUnsupportedOperation, name)
	}

	return p, nil
}

// fallbackFor returns the fallback provider for the given primary when the
// error is eligible for fallback. Only transient unavailability is eligible;
// validation, unsupported-operation, and invalid-response errors are not.
func (r *Router) fallbackFor(primaryName string, err error) (ModelProvider, bool) {
	if errors.Is(err, ErrUnavailableProvider) {
		if fbName, ok := r.fallbacks[primaryName]; ok {
			if fb, ok := r.providers[fbName]; ok {
				return fb, true
			}
		}
	}

	return nil, false
}
