package ingestor_test

import (
	"context"
	"errors"
	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/utilstest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestStatsRunner_Run_ReadsStatsUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := utilstest.NewConsumerStatsReader(t)
	reader.MockConsumerStatsReader.EXPECT().
		ReadStats(gomock.Any()).
		DoAndReturn(func(context.Context) (ingestor.ConsumerStats, error) {
			cancel()
			return ingestor.ConsumerStats{Lag: 10}, nil
		})

	runner := ingestor.NewStatsRunner(reader, time.Millisecond)

	require.NoError(t, runner.Run(ctx))
}

func TestStatsRunner_Run_ReturnsReadError(t *testing.T) {
	ctx := context.Background()
	expected := errors.New("stats unavailable")
	reader := utilstest.NewConsumerStatsReader(t).WithError(expected)

	runner := ingestor.NewStatsRunner(reader, time.Millisecond)

	require.ErrorIs(t, runner.Run(ctx), expected)
}
