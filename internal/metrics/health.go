package metrics

import (
	"context"
	"synapsePlatform/internal/health"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type HealthProbe struct {
	probe    health.Probe
	duration metric.Float64Histogram
	total    metric.Int64Counter
}

func NewHealthProbe(probe health.Probe, meter metric.Meter) (*HealthProbe, error) {
	duration, err := meter.Float64Histogram("health_check_duration_seconds",
		metric.WithDescription("Duration of health check probes"),
	)
	if err != nil {
		return nil, err
	}

	total, err := meter.Int64Counter("health_check_total",
		metric.WithDescription("Total health check invocations"),
	)
	if err != nil {
		return nil, err
	}

	return &HealthProbe{probe: probe, duration: duration, total: total}, nil
}

func (p *HealthProbe) Name() string { return p.probe.Name() }

func (p *HealthProbe) Check(ctx context.Context) error {
	start := time.Now()
	err := p.probe.Check(ctx)
	elapsed := time.Since(start).Seconds()

	probeName := attribute.String("probe", p.Name())
	status := attribute.String("status", "ok")
	if err != nil {
		status = attribute.String("status", "error")
	}

	p.duration.Record(ctx, elapsed, metric.WithAttributes(probeName, status))
	p.total.Add(ctx, 1, metric.WithAttributes(probeName, status))

	return err
}
