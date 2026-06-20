package provider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"synapsePlatform/internal/intelligence/provider"
	"synapsePlatform/internal/utilstest"
)

func TestRouterRoutesToRequestedProvider(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")
	primary.CompletionOutput = "from-primary"
	fallback := utilstest.NewMock().WithName("fallback")

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary":  primary,
		"fallback": fallback,
	}, provider.RouterConfig{
		DefaultCompletionProvider: "primary",
		Fallbacks:                 map[string]string{"primary": "fallback"},
	})

	resp, err := r.Complete(context.Background(), provider.CompletionRequest{
		Prompt:   "hi",
		Metadata: map[string]string{"provider": "primary"},
	})
	require.NoError(t, err)
	assert.Equal(t, "from-primary", resp.Content)
	assert.Equal(t, "primary", resp.Provider)
	assert.Equal(t, 1, primary.CompleteCalls())
	assert.Equal(t, 0, fallback.CompleteCalls(), "fallback must not be called when primary succeeds")
}

func TestRouterUsesDefaultWhenNoProviderRequested(t *testing.T) {
	defaultP := utilstest.NewMock().WithName("default")
	defaultP.CompletionOutput = "from-default"
	other := utilstest.NewMock().WithName("other")

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"default": defaultP,
		"other":   other,
	}, provider.RouterConfig{DefaultCompletionProvider: "default"})

	resp, err := r.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "from-default", resp.Content)
	assert.Equal(t, 1, defaultP.CompleteCalls())
	assert.Equal(t, 0, other.CompleteCalls())
}

func TestRouterFallsBackOnUnavailableError(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")
	primary.CompleteErr = provider.ErrUnavailableProvider
	primary.CompletionOutput = "should-not-see"
	fallback := utilstest.NewMock().WithName("fallback")
	fallback.CompletionOutput = "from-fallback"

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary":  primary,
		"fallback": fallback,
	}, provider.RouterConfig{
		DefaultCompletionProvider: "primary",
		Fallbacks:                 map[string]string{"primary": "fallback"},
	})

	resp, err := r.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "from-fallback", resp.Content)
	assert.Equal(t, "fallback", resp.Provider)
	assert.Equal(t, 1, primary.CompleteCalls())
	assert.Equal(t, 1, fallback.CompleteCalls(), "fallback must be tried once")
}

func TestRouterDoesNotFallbackOnUnsupportedOperation(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")
	primary.CompleteErr = provider.ErrUnsupportedOperation
	fallback := utilstest.NewMock().WithName("fallback")
	fallback.CompletionOutput = "should-not-see"

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary":  primary,
		"fallback": fallback,
	}, provider.RouterConfig{
		DefaultCompletionProvider: "primary",
		Fallbacks:                 map[string]string{"primary": "fallback"},
	})

	_, err := r.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrUnsupportedOperation))
	assert.Equal(t, 0, fallback.CompleteCalls(), "must not fallback on unsupported-operation errors")
}

func TestRouterDoesNotFallbackOnInvalidResponse(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")
	primary.CompleteErr = provider.ErrInvalidResponse
	fallback := utilstest.NewMock().WithName("fallback")

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary":  primary,
		"fallback": fallback,
	}, provider.RouterConfig{
		DefaultCompletionProvider: "primary",
		Fallbacks:                 map[string]string{"primary": "fallback"},
	})

	_, err := r.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrInvalidResponse))
	assert.Equal(t, 0, fallback.CompleteCalls())
}

func TestRouterFallsBackOnWrappedUnavailableError(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")
	primary.CompleteErr = &wrappedErr{inner: provider.ErrUnavailableProvider, msg: "connection refused"}
	fallback := utilstest.NewMock().WithName("fallback")
	fallback.CompletionOutput = "rescued"

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary":  primary,
		"fallback": fallback,
	}, provider.RouterConfig{
		DefaultCompletionProvider: "primary",
		Fallbacks:                 map[string]string{"primary": "fallback"},
	})

	resp, err := r.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "rescued", resp.Content)
	assert.Equal(t, 1, fallback.CompleteCalls())
}

func TestRouterNoFallbackConfiguredReturnsPrimaryError(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")
	primary.CompleteErr = provider.ErrUnavailableProvider

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary": primary,
	}, provider.RouterConfig{DefaultCompletionProvider: "primary"})

	_, err := r.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrUnavailableProvider))
}

func TestRouterUnknownRequestedProviderRejected(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary": primary,
	}, provider.RouterConfig{DefaultCompletionProvider: "primary"})

	_, err := r.Complete(context.Background(), provider.CompletionRequest{
		Prompt:   "hi",
		Metadata: map[string]string{"provider": "ghost"},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrUnsupportedOperation))
}

func TestRouterNoDefaultAndNoRequestRejected(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary": primary,
	}, provider.RouterConfig{})

	_, err := r.Complete(context.Background(), provider.CompletionRequest{Prompt: "hi"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrUnsupportedOperation))
}

func TestRouterEmbedUsesDefaultAndFallsBack(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")
	primary.EmbedErr = provider.ErrUnavailableProvider
	fallback := utilstest.NewMock().WithName("fallback")

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary":  primary,
		"fallback": fallback,
	}, provider.RouterConfig{
		DefaultEmbeddingProvider: "primary",
		Fallbacks:                map[string]string{"primary": "fallback"},
	})

	resp, err := r.Embed(context.Background(), provider.EmbedRequest{Input: []string{"x"}})
	require.NoError(t, err)
	assert.Equal(t, "fallback", resp.Provider)
	assert.Equal(t, 1, primary.EmbedCalls())
	assert.Equal(t, 1, fallback.EmbedCalls())
}

func TestRouterEmbedNoFallbackOnUnsupportedOperation(t *testing.T) {
	primary := utilstest.NewMock().WithName("primary")
	primary.EmbedErr = provider.ErrUnsupportedOperation
	fallback := utilstest.NewMock().WithName("fallback")

	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"primary":  primary,
		"fallback": fallback,
	}, provider.RouterConfig{
		DefaultEmbeddingProvider: "primary",
		Fallbacks:                map[string]string{"primary": "fallback"},
	})

	_, err := r.Embed(context.Background(), provider.EmbedRequest{Input: []string{"x"}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrUnsupportedOperation))
	assert.Equal(t, 0, fallback.EmbedCalls())
}

func TestRouterName(t *testing.T) {
	r := mustNewRouter(t, map[string]provider.ModelProvider{
		"p": utilstest.NewMock().WithName("p"),
	}, provider.RouterConfig{DefaultCompletionProvider: "p"})
	assert.Equal(t, "router", r.Name())
}

// --- NewRouter validation ---

func TestNewRouterRejectsEmptyProviders(t *testing.T) {
	_, err := provider.NewRouter(nil, provider.RouterConfig{})
	require.Error(t, err)
}

func TestNewRouterRejectsNilProvider(t *testing.T) {
	_, err := provider.NewRouter(map[string]provider.ModelProvider{"p": nil}, provider.RouterConfig{})
	require.Error(t, err)
}

func TestNewRouterRejectsNameMismatch(t *testing.T) {
	m := utilstest.NewMock().WithName("real-name")
	_, err := provider.NewRouter(map[string]provider.ModelProvider{"wrong-key": m}, provider.RouterConfig{})
	require.Error(t, err)
}

func TestNewRouterRejectsUnknownDefault(t *testing.T) {
	_, err := provider.NewRouter(
		map[string]provider.ModelProvider{"p": utilstest.NewMock().WithName("p")},
		provider.RouterConfig{DefaultCompletionProvider: "missing"},
	)
	require.Error(t, err)
}

func TestNewRouterRejectsSelfFallback(t *testing.T) {
	_, err := provider.NewRouter(
		map[string]provider.ModelProvider{"p": utilstest.NewMock().WithName("p")},
		provider.RouterConfig{Fallbacks: map[string]string{"p": "p"}},
	)
	require.Error(t, err)
}

func TestNewRouterRejectsUnknownFallback(t *testing.T) {
	_, err := provider.NewRouter(
		map[string]provider.ModelProvider{"p": utilstest.NewMock().WithName("p")},
		provider.RouterConfig{Fallbacks: map[string]string{"p": "ghost"}},
	)
	require.Error(t, err)
}

// --- helpers ---

func mustNewRouter(t *testing.T, providers map[string]provider.ModelProvider, cfg provider.RouterConfig) *provider.Router {
	t.Helper()
	r, err := provider.NewRouter(providers, cfg)
	require.NoError(t, err)
	return r
}

// wrappedErr lets us assert errors.Is works through wrapping, matching how the
// real providers wrap ErrUnavailableProvider with %w.
type wrappedErr struct {
	inner error
	msg   string
}

func (w *wrappedErr) Error() string { return w.msg + ": " + w.inner.Error() }
func (w *wrappedErr) Unwrap() error { return w.inner }