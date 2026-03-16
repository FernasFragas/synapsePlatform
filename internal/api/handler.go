package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"synapsePlatform/internal/auth"
	"synapsePlatform/internal/ingestor"
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
