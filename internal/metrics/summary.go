package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"synapsePlatform/internal/api"
)

type Summarizer struct {
	s        api.Summarizer
	tracer   trace.Tracer
	duration metric.Float64Histogram
	total    metric.Int64Counter
	errors   metric.Int64Counter
}

func NewSummarizer(meter metric.Meter, tracer trace.Tracer, s api.Summarizer) (*Summarizer, error) {
	duration, err := meter.Float64Histogram("summary.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to generate a summary"),
	)
	if err != nil {
		return nil, err
	}

	total, err := meter.Int64Counter("summary.request.total",
		metric.WithDescription("Total summary requests"),
	)
	if err != nil {
		return nil, err
	}

	errors, err := meter.Int64Counter("summary.request.errors",
		metric.WithDescription("Summary generation errors"),
	)
	if err != nil {
		return nil, err
	}

	return &Summarizer{
		s:        s,
		tracer:   tracer,
		duration: duration,
		total:    total,
		errors:   errors,
	}, nil
}

func (m *Summarizer) Summarize(ctx context.Context, req api.Request) (*api.Report, error) {
	ctx, span := m.tracer.Start(ctx, "summary.generate",
		trace.WithAttributes(
			attribute.String("domain", req.Domain),
		))
	defer span.End()

	start := time.Now()
	report, err := m.s.Summarize(ctx, req)
	elapsed := time.Since(start).Seconds()

	op := attribute.String(AttrOperation, "summarize")

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		m.errors.Add(ctx, 1, metric.WithAttributes(op))
		m.total.Add(ctx, 1, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))
		m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusError)))
		return nil, err
	}

	span.SetAttributes(
		attribute.String("model", report.Model),
		attribute.Int("content_len", len(report.Content)),
	)

	m.total.Add(ctx, 1, metric.WithAttributes(op, attribute.String(AttrStatus, StatusSuccess)))
	m.duration.Record(ctx, elapsed, metric.WithAttributes(op, attribute.String(AttrStatus, StatusSuccess)))

	return report, nil
}
