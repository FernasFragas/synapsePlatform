package log

import (
	"context"
	"log/slog"
	"unicode/utf8"
)

const defaultRedacted = "[REDACTED]"

// Options configures the RedactingHandler.
type Options struct {
	// RedactKeys is the set of attribute key names whose values will be
	// replaced with "[REDACTED]" regardless of type. Case-sensitive.
	// Common entries: "token", "password", "secret", "authorization".
	RedactKeys []string

	// MaxValueBytes caps the byte length of any string or []byte attribute
	// value. Values exceeding this are truncated and suffixed with "…[truncated]".
	// Zero means no cap.
	MaxValueBytes int
}

// RedactingHandler is a slog.Handler wrapper
// that sits between slog.Logger and the actual output handler.
// It catches anything that slipped past Layer 1 — for example,
// a new field added to a struct without updating LogValue(),
// or a future decorator that accidentally logs a raw sensitive value.
type RedactingHandler struct {
	inner   slog.Handler
	redact  map[string]struct{}
	maxSize int
}

func NewRedactingHandler(inner slog.Handler, opts Options) *RedactingHandler {
	redact := make(map[string]struct{}, len(opts.RedactKeys))
	for _, k := range opts.RedactKeys {
		redact[k] = struct{}{}
	}

	return &RedactingHandler{inner: inner, redact: redact, maxSize: opts.MaxValueBytes}
}

func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Build a new record with transformed attributes.
	// We do not mutate the original — slog.Record is not safe to modify in place.
	next := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		next.AddAttrs(h.transformAttr(a))

		return true
	})

	return h.inner.Handle(ctx, next)
}

func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	transformed := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		transformed[i] = h.transformAttr(a)
	}

	return &RedactingHandler{inner: h.inner.WithAttrs(transformed), redact: h.redact, maxSize: h.maxSize}
}

func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{inner: h.inner.WithGroup(name), redact: h.redact, maxSize: h.maxSize}
}

func (h *RedactingHandler) transformAttr(a slog.Attr) slog.Attr {
	// Resolve LogValuer first so the resolved value is also checked.
	a.Value = a.Value.Resolve()

	// Redact by key.
	if _, sensitive := h.redact[a.Key]; sensitive {
		return slog.String(a.Key, defaultRedacted)
	}

	// Recurse into groups (e.g. slog.Group, or a LogValue that returns GroupValue).
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()

		transformed := make([]slog.Attr, len(attrs))

		for i, ga := range attrs {
			transformed[i] = h.transformAttr(ga)
		}

		return slog.Attr{Key: a.Key, Value: slog.GroupValue(transformed...)}
	}

	// Cap string length.
	if h.maxSize > 0 && a.Value.Kind() == slog.KindString {
		s := a.Value.String()
		if len(s) > h.maxSize {
			// Truncate cleanly on a UTF-8 boundary.
			cut := h.maxSize
			for cut > 0 && !utf8.RuneStart(s[cut]) {
				cut--
			}

			return slog.String(a.Key, s[:cut]+"…[truncated]")
		}
	}

	return a
}
