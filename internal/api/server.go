//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/api/mocked_$GOFILE
package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"synapsePlatform/internal"
	"synapsePlatform/internal/auth"
	"synapsePlatform/internal/health"
	"synapsePlatform/internal/ingestor"
	"time"
)

type EventReader interface {
	GetEvent(ctx context.Context, eventID string) (*ingestor.BaseEvent, error)
	ListEvents(ctx context.Context, page ingestor.PageRequest) (*ingestor.PageResponse[*ingestor.BaseEvent], error)
}

type Summarizer interface {
	Summarize(ctx context.Context, req Request) (*Report, error)
}

// AggregateReader is satisfied by *sqllite.Repo (Phase 1.4).
type AggregateReader interface {
	AggregateByDomain(ctx context.Context, since time.Time) ([]DomainStat, error)
	LatestSummary(ctx context.Context, lookup SummaryLookup) (*Report, bool, error)
	SaveSummary(ctx context.Context, r *Report) (int64, error)
	SaveSummaryEvidenceLinks(ctx context.Context, links []SummaryEvidenceLink) error
}

// SummaryLookup carries the full cache key dimensions for a summary lookup.
// The repository matches on all fields so that changing any one of them
// (model, provider, prompt version, inference params, or input data) results
// in a cache miss rather than returning a stale summary generated under
// different conditions.
type SummaryLookup struct {
	Domain        string
	WindowFrom    time.Time
	Provider      string
	Model         string
	PromptVersion string
	Temperature   float64
	MaxTokens     int
	InputHash     string
}

type Request struct {
	Domain string
	Since  time.Time
}
type Report struct {
	Domain            string    `json:"domain"`
	WindowFrom        time.Time `json:"window_from"`
	Model             string    `json:"model"`
	Content           string    `json:"content"`
	StructuredContent string    `json:"structured_content,omitempty"`
	Provider          string    `json:"provider,omitempty"`
	PromptVersion     string    `json:"prompt_version,omitempty"`
	InputHash         string    `json:"input_hash,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}
type DomainStat struct {
	Domain, EventType   string
	Count               int64
	FirstSeen, LastSeen time.Time
}

// DefaultMaxEvidenceEvents is the bounded set size for the curated evidence
// passed to the model.
const DefaultMaxEvidenceEvents = 30

// EvidenceRequest selects the bounded evidence set for a summary.
type EvidenceRequest struct {
	Domain     string
	Since      time.Time
	Until      time.Time // zero value means "now"
	MaxResults int       // <= 0 falls back to DefaultMaxEvidenceEvents
}

// SummaryEvidenceEvent is a redacted, stable event field set the model is
// allowed to see and cite. It includes no raw payloads or secrets. The
// evidence selection step produces these.
type SummaryEvidenceEvent struct {
	EventID    string
	Domain     string
	EventType  string
	EntityID   string
	OccurredAt time.Time
}

// Evidence relationship constants for the intelligence_summary_events join
// table. The relationship column records why a particular event is linked to
// a summary: it was part of the evidence set, it was flagged as a notable
// event, or it was referenced by a recommended action.
const (
	EvidenceRelationshipEvidence       = "evidence"
	EvidenceRelationshipNotable        = "notable"
	EvidenceRelationshipRecommendation = "recommendation"
)

// SummaryEvidenceLink is a single row in the intelligence_summary_events join
// table. It connects a persisted summary to an event it cited, with the
// relationship explaining the connection.
type SummaryEvidenceLink struct {
	SummaryID    int64
	EventID      string
	Relationship string
}

type Server struct {
	server              *http.Server
	mux                 *http.ServeMux
	events              EventReader
	validator           auth.TokenValidator
	loggerMiddleware    Middleware
	rateLimitMiddleware Middleware
	corsMiddleware      Middleware
	health              *health.Checker
	addr                string
	metricsHandler      http.Handler
	summarizer          Summarizer
}

func NewServer(cfg internal.ServerConfig, events EventReader, summarizer Summarizer, validator auth.TokenValidator, loggerMiddleware Middleware, healthChecker *health.Checker, metricsHandler http.Handler) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: cfg.Timeouts.ReadHeader,
			ReadTimeout:       cfg.Timeouts.Read,
			WriteTimeout:      cfg.Timeouts.Write,
			IdleTimeout:       cfg.Timeouts.Idle,
		},
		mux:              mux,
		events:           events,
		summarizer:       summarizer,
		validator:        validator,
		loggerMiddleware: loggerMiddleware,
		health:           healthChecker,
		addr:             cfg.Address,
		metricsHandler:   metricsHandler,
	}

	s.rateLimitMiddleware = s.rateLimiter(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.Burst)
	s.corsMiddleware = s.cors(cfg.CORS.AllowedOrigins)

	s.routes()

	return s
}

// Start starts the httpserver set up by NewService.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("can't create listener: %w", err)
	}

	if err := s.server.Serve(ln); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server failed: %w", err)
		}
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /livez", health.LivezHandler())
	s.mux.HandleFunc("GET /readyz", s.health.ReadyzHandler())
	s.mux.HandleFunc("GET /healthz", s.health.ReadyzHandler())

	if s.metricsHandler != nil {
		s.mux.Handle("GET /metrics", s.metricsHandler)
	}

	chain := func(h http.HandlerFunc) http.Handler {
		return s.recoverPanic(
			s.requestID(
				s.traceRequest(
					s.rateLimitMiddleware(
						s.corsMiddleware(
							s.loggerMiddleware(
								s.authenticate(h)))))))
	}

	s.mux.Handle("GET /v1/events", chain(s.handleListEvents))
	s.mux.Handle("GET /v1/events/{id}", chain(s.handleGetEvent))

	s.mux.Handle("GET /v1/summary", chain(s.handleGetSummary))
}
