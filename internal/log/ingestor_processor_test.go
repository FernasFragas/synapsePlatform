package log_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"synapsePlatform/internal/ingestor"
	ingestorlog "synapsePlatform/internal/log"
	"synapsePlatform/internal/utilstest"

	"github.com/stretchr/testify/require"
)

func TestIngestorProcessor_ProcessData_PollErrorIsNotLoggedTwice(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	expectedErr := ingestor.ProcessorError{
		TypeOfError:            ingestor.ErrPollingMsg,
		ErrorOccurredBecauseOf: ingestor.ErrFailedToPollMsg,
		Field:                  "delivery",
		Expected:               "Delivery",
		Err:                    errors.New("broker unavailable"),
	}
	processor := utilstest.NewDataProcessor(t).WithError(expectedErr)
	subject := ingestorlog.NewIngestorProcessor(logger, processor)

	delivery, err := subject.ProcessData(context.Background())

	require.Nil(t, delivery)
	require.ErrorIs(t, err, expectedErr.Err)
	require.Empty(t, logs.String())
}
