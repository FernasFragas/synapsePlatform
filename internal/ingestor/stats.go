//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
	"time"
)

type ConsumerStats struct {
	Lag        int64
	MinBytes   int64
	MaxBytes   int64
	Messages   int64
	Bytes      int64
	Rebalances int64
	Errors     int64
}

type ConsumerStatsReader interface {
	ReadStats(ctx context.Context) (ConsumerStats, error)
}

type StatsRunner struct {
	reader   ConsumerStatsReader
	interval time.Duration
}

func NewStatsRunner(reader ConsumerStatsReader, interval time.Duration) *StatsRunner {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	return &StatsRunner{reader: reader, interval: interval}
}

func (r *StatsRunner) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := r.reader.ReadStats(ctx); err != nil {
				return err
			}
		}
	}
}
