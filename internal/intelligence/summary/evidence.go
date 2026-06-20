package summary

import (
	"context"
	"time"

	"synapsePlatform/internal/api"
)

// SummaryEvidenceReader loads a bounded, curated set of events for the summary
// prompt. The service calls this after aggregate stats are loaded and before
// the prompt is built, then passes the result to BuildSummaryPrompt and
// ValidateAgainstEvidence.
//
// Declared here so the summary package does not depend on the
// repository. *sqllite.Repo satisfies this interface. The request and event
// types live in the api package to avoid an import cycle between sqllite and
// summary.
type SummaryEvidenceReader interface {
	ListSummaryEvidence(ctx context.Context, req api.EvidenceRequest) ([]api.SummaryEvidenceEvent, error)
}

// noopEvidenceReader returns no evidence, used when no reader is configured.
func (noopEvidenceReader) ListSummaryEvidence(_ context.Context, _ api.EvidenceRequest) ([]api.SummaryEvidenceEvent, error) {
	return nil, nil
}

// WithEvidenceReader sets the evidence reader for the service. When not set,
// the service uses a no-op reader so prompts are built with empty evidence.
func WithEvidenceReader(r SummaryEvidenceReader) ServiceOption {
	return func(s *Service) {
		if r != nil {
			s.evidenceReader = r
		}
	}
}

// noopEvidenceReader is the default when no evidence reader is configured.
type noopEvidenceReader struct{}

// evidenceRequestFromAPI converts the service-level request to the api-level
// evidence request, applying defaults. This is used internally by Summarize.
func evidenceRequestFromAPI(domain string, since time.Time) api.EvidenceRequest {
	return api.EvidenceRequest{
		Domain:     domain,
		Since:      since,
		MaxResults: api.DefaultMaxEvidenceEvents,
	}
}
