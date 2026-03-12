package api

import (
	"errors"
	"net/http"
	"strings"
	"synapsePlatform/internal/auth"
)

type Middleware func(http.Handler) http.Handler

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearer(r)
		if token == "" {
			// No token at all — tell the client what scheme to use
			w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)

			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		identity, err := s.validator.Validate(token)
		if err != nil {
			var valErr auth.ValidationError
			if errors.As(err, &valErr) && valErr.TypeOfError == auth.ErrTypeTokenExpired {
				// Could return a more specific WWW-Authenticate hint
			}

			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)

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
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
