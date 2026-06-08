package log

import (
	"context"
	"log/slog"
	"time"

	"synapsePlatform/internal/api"
)

type Summarizer struct {
	logger *slog.Logger
	s      api.Summarizer
}

func NewSummarizer(logger *slog.Logger, s api.Summarizer) *Summarizer {
	return &Summarizer{
		logger: logger.With("component", "summarizer"),
		s:      s,
	}
}

func (l *Summarizer) Summarize(ctx context.Context, req api.Request) (*api.Report, error) {
	start := time.Now()
	report, err := l.s.Summarize(ctx, req)
	elapsed := time.Since(start)

	if err != nil {
		l.logger.ErrorContext(ctx, "failed to summarize",
			"domain", req.Domain,
			"since", req.Since,
			"elapsed_ms", elapsed.Milliseconds(),
			"error", err,
		)
		return nil, err
	}

	l.logger.InfoContext(ctx, "summarized",
		"domain", req.Domain,
		"since", req.Since,
		"model", report.Model,
		"elapsed_ms", elapsed.Milliseconds(),
	)
	return report, nil
}
