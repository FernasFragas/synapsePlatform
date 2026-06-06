//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/summary/mocked_$GOFILE
package summary

import (
	"context"
	"fmt"
	"strings"
	"time"

	"synapsePlatform/internal/api"
)

type Service struct {
	reader api.AggregateReader
	llm    api.LLMClient
	model  string
}

func New(reader api.AggregateReader, llm api.LLMClient, model string) *Service {
	return &Service{
		reader: reader,
		llm:    llm,
		model:  model,
	}
}

func (s *Service) Summarize(ctx context.Context, req api.Request) (*api.Report, error) {
	// 1. Check cache
	cached, ok, err := s.reader.LatestSummary(ctx, req.Domain, req.Since)
	if err != nil {
		return nil, fmt.Errorf("check summary cache: %w", err)
	}
	if ok {
		return cached, nil
	}

	// 2. Aggregate
	stats, err := s.reader.AggregateByDomain(ctx, req.Since)
	if err != nil {
		return nil, fmt.Errorf("aggregate events: %w", err)
	}

	// Filter to requested domain (or keep all if empty)
	var domainStats []api.DomainStat
	for _, st := range stats {
		if req.Domain == "" || st.Domain == req.Domain {
			domainStats = append(domainStats, st)
		}
	}
	if len(domainStats) == 0 {
		return &api.Report{
			Domain:     req.Domain,
			WindowFrom: req.Since,
			Model:      s.model,
			Content:    "No events found for the requested period.",
			CreatedAt:  time.Now().UTC(),
		}, nil
	}

	// 3. Build prompt
	prompt := buildPrompt(req.Domain, domainStats)

	// 4. Call LLM
	content, err := s.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	report := &api.Report{
		Domain:     req.Domain,
		WindowFrom: req.Since,
		Model:      s.model,
		Content:    strings.TrimSpace(content),
		CreatedAt:  time.Now().UTC(),
	}

	// 5. Cache
	if err := s.reader.SaveSummary(ctx, report); err != nil {
		return report, fmt.Errorf("save summary: %w", err)
	}

	return report, nil
}

func buildPrompt(domain string, stats []api.DomainStat) string {
	var b strings.Builder
	b.WriteString("You are a data analyst. Summarize the following event statistics in 2-3 sentences.\n")
	if domain != "" {
		b.WriteString(fmt.Sprintf("Domain: %s\n", domain))
	}
	b.WriteString("Event summary:\n")
	for _, s := range stats {
		b.WriteString(fmt.Sprintf("- %s/%s: %d events (from %s to %s)\n",
			s.Domain, s.EventType, s.Count,
			s.FirstSeen.Format(time.RFC3339), s.LastSeen.Format(time.RFC3339),
		))
	}
	b.WriteString("\nProvide a concise summary.")
	return b.String()
}
