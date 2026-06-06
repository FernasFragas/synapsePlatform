package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"synapsePlatform/internal/auth"
	"synapsePlatform/internal/ingestor"
	"time"
)

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	identity, err := auth.IdentityFromContext(r.Context())
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")

		return
	}

	if !identity.HasScope("read:events") {
		writeError(w, r, http.StatusForbidden, "forbidden")

		return
	}

	events, err := s.events.ListEvents(r.Context(), ingestor.PageRequest{
		Cursor: r.URL.Query().Get("cursor"),
		Limit:  parseIntOrDefault(r.URL.Query().Get("limit"), 20),
	})
	if err != nil {
		reqErr := RequestError{
			TypeOfError:            ErrTypeInternal,
			ErrorOccurredBecauseOf: ErrFailedToListEvents,
			Resource:               "events",
			Err:                    err,
		}

		writeError(w, r, httpStatus(reqErr), string(reqErr.ErrorOccurredBecauseOf))

		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(ListResponse{
		Data:       toResponses(events.Items),
		NextCursor: events.NextCursor,
		HasMore:    events.HasMore,
	})
	if err != nil {
		reqErr := RequestError{
			TypeOfError:            ErrTypeEncoding,
			ErrorOccurredBecauseOf: ErrFailedToEncodeResponse,
			Resource:               "events",
			Err:                    err,
		}

		writeError(w, r, httpStatus(reqErr), string(reqErr.ErrorOccurredBecauseOf))
	}
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	identity, err := auth.IdentityFromContext(r.Context())
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")

		return
	}

	if !identity.HasScope("read:events") {
		writeError(w, r, http.StatusForbidden, "forbidden")

		return
	}

	id := r.PathValue("id")

	event, err := s.events.GetEvent(r.Context(), id)
	if err != nil {
		errType := ErrTypeInternal
		if errors.Is(err, ingestor.ErrEventNotFound) {
			errType = ErrTypeNotFound
		}

		reqErr := RequestError{
			TypeOfError:            errType,
			ErrorOccurredBecauseOf: ErrFailedToGetEvent,
			Resource:               "event",
			ResourceID:             id,
			Err:                    err,
		}

		writeError(w, r, httpStatus(reqErr), string(reqErr.ErrorOccurredBecauseOf))

		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(toResponse(event))
	if err != nil {
		// Encoding failure — resource was found but response couldn't be written
		reqErr := RequestError{
			TypeOfError:            ErrTypeEncoding,
			ErrorOccurredBecauseOf: ErrFailedToEncodeResponse,
			Resource:               "event",
			ResourceID:             id,
			Err:                    err,
		}

		writeError(w, r, httpStatus(reqErr), string(reqErr.ErrorOccurredBecauseOf))
	}
}

func (s *Server) handleGetSummary(w http.ResponseWriter, r *http.Request) {
	identity, err := auth.IdentityFromContext(r.Context())
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !identity.HasScope("read:events") {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	domain := r.URL.Query().Get("domain")
	sinceRaw := r.URL.Query().Get("since")
	// Default to last 24 hours
	since := time.Now().UTC().Add(-24 * time.Hour)
	if sinceRaw != "" {
		if parsed, err := time.Parse(time.RFC3339, sinceRaw); err == nil {
			since = parsed
		}
	}
	if s.summarizer == nil {
		writeError(w, r, http.StatusServiceUnavailable, "summarizer_disabled")
		return
	}
	report, err := s.summarizer.Summarize(r.Context(), Request{
		Domain: domain,
		Since:  since,
	})
	if err != nil {
		reqErr := RequestError{
			TypeOfError:            ErrTypeInternal,
			ErrorOccurredBecauseOf: ErrFailedToGetEvent, // or add a new ErrFailedToSummarize
			Resource:               "summary",
			Err:                    err,
		}
		writeError(w, r, httpStatus(reqErr), string(reqErr.ErrorOccurredBecauseOf))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}
