package log

import (
	"context"
	"log/slog"
	"synapsePlatform/internal/health"
	"time"
)

type HealthProbe struct {
	logger *slog.Logger
	probe  health.Probe
}

func NewHealthProbe(logger *slog.Logger, probe health.Probe) *HealthProbe {
	return &HealthProbe{logger: logger, probe: probe}
}

func (p *HealthProbe) Name() string { return p.probe.Name() }

func (p *HealthProbe) Check(ctx context.Context) error {
	start := time.Now()
	err := p.probe.Check(ctx)
	elapsed := time.Since(start)

	if err != nil {
		p.logger.Warn("health check failed",
			"probe", p.Name(),
			"duration", elapsed,
			"error", err,
		)
	} else {
		p.logger.Debug("health check passed",
			"probe", p.Name(),
			"duration", elapsed,
		)
	}

	return err
}
