//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/api/mocked_server.go
package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"synapsePlatform/internal"
	"synapsePlatform/internal/auth"
	"synapsePlatform/internal/ingestor"
)

type EventReader interface {
	GetEvent(ctx context.Context, eventID string) (*ingestor.BaseEvent, error)
	ListEvents(ctx context.Context, page ingestor.PageRequest) (*ingestor.PageResponse[*ingestor.BaseEvent], error)
}

type Server struct {
	server           *http.Server
	mux              *http.ServeMux
	events           EventReader
	validator        auth.TokenValidator
	loggerMiddleware Middleware
	addr             string
}

func NewServer(cfg internal.ServerConfig, events EventReader, validator auth.TokenValidator, loggerMiddleware Middleware) *Server {
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
		validator:        validator,
		loggerMiddleware: loggerMiddleware,
		addr:             cfg.Address,
	}
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
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	s.mux.Handle("GET /events", s.recoverPanic(s.loggerMiddleware(s.authenticate(http.HandlerFunc(s.handleListEvents)))))
	s.mux.Handle("GET /events/{id}", s.recoverPanic(
		s.loggerMiddleware(s.authenticate(http.HandlerFunc(s.handleGetEvent)))),
	)
}
