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
		http.Error(w, "unauthorized", http.StatusUnauthorized)

		return
	}

	if !identity.HasScope("read:events") {
		http.Error(w, "forbidden", http.StatusForbidden)

		return
	}

	events, err := s.events.ListEvents(r.Context())
	if err != nil {
		reqErr := RequestError{
			TypeOfError:            ErrTypeInternal,
			ErrorOccurredBecauseOf: ErrFailedToListEvents,
			Resource:               "events",
			Err:                    err,
		}

		http.Error(w, reqErr.Error(), httpStatus(reqErr))

		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toResponses(events)); err != nil {
		reqErr := RequestError{
			TypeOfError:            ErrTypeEncoding,
			ErrorOccurredBecauseOf: ErrFailedToEncodeResponse,
			Resource:               "events",
			Err:                    err,
		}

		http.Error(w, reqErr.Error(), httpStatus(reqErr))
	}
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	identity, err := auth.IdentityFromContext(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)

		return
	}

	if !identity.HasScope("read:events") {
		http.Error(w, "forbidden", http.StatusForbidden)

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
		http.Error(w, reqErr.Error(), httpStatus(reqErr))

		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toResponse(event)); err != nil {
		// Encoding failure — resource was found but response couldn't be written
		reqErr := RequestError{
			TypeOfError:            ErrTypeEncoding,
			ErrorOccurredBecauseOf: ErrFailedToEncodeResponse,
			Resource:               "event",
			ResourceID:             id,
			Err:                    err,
		}
		http.Error(w, reqErr.Error(), httpStatus(reqErr))
	}
}

func toResponses(events []*ingestor.BaseEvent) []*EventResponse {
	responses := make([]*EventResponse, len(events))
	for i, event := range events {
		responses[i] = toResponse(event)
	}
	return responses
}

func toResponse(event *ingestor.BaseEvent) *EventResponse {
	return &EventResponse{
		EventID:       event.EventID.String(),
		Domain:        event.Domain,
		EventType:     event.EventType,
		EntityID:      event.EntityID,
		OccurredAt:    event.OccurredAt,
		Source:        event.Source,
		SchemaVersion: event.SchemaVersion,
		Data:          event.Data,
	}
}
