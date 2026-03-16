package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"synapsePlatform/internal/auth"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
)

type Middleware func(http.Handler) http.Handler

type requestIDKey struct{}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearer(r)
		if token == "" {
			// No token at all — tell the client what scheme to use
			w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)

			writeError(w, r, http.StatusUnauthorized, "unauthorized")

			return
		}

		identity, err := s.validator.Validate(token)
		if err != nil {
			var valErr auth.ValidationError
			if errors.As(err, &valErr) && valErr.TypeOfError == auth.ErrTypeTokenExpired {
				w.Header().Set("WWW-Authenticate", `Bearer error="token_expired"`)

				writeError(w, r, http.StatusUnauthorized, "token expired")

				return
			}

			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			writeError(w, r, http.StatusUnauthorized, "unauthorized")

			return
		}

		next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	})
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	// to make sure the auth scheme is case-insensitive — "bearer", "Bearer", "BEARER" all valid
	if len(h) < 7 || !strings.EqualFold(h[:7], "bearer ") {
		return ""
	}

	return h[7:]
}

func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				writeError(w, r, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), requestIDKey{}, id)

		w.Header().Set("X-Request-ID", id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) rateLimiter(rps float64, burst int) Middleware {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Retry-After", "1")

				writeError(w, r, http.StatusTooManyRequests, "rate limit exceeded")

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) cors(allowedOrigins []string) Middleware {
	origins := make(map[string]bool, len(allowedOrigins))

	for _, o := range allowedOrigins {
		origins[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

				w.WriteHeader(http.StatusNoContent)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) traceRequest(next http.Handler) http.Handler {
	tracer := otel.Tracer("synapse.api")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.URLPath(r.URL.Path),
			))
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)

	return id
}
