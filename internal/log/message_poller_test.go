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

func TestMessagePoller_PollMessage_TerminalErrorReturnsDeliveryAndLogsMetadata(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	delivery := &ingestor.Delivery{
		Metadata: ingestor.MessageMetadata{
			Source:  "kafka",
			Headers: map[string]string{"trace-id": "abc"},
			Labels:  map[string]string{"partition": "0"},
		},
	}
	expectedErr := errors.New("bad payload")

	poller := utilstest.NewMessagePoller(t).WithTerminalDecodeFailure(delivery, expectedErr)
	subject := ingestorlog.NewMessagePoller(logger, poller)

	actual, err := subject.PollMessage(context.Background())

	require.Error(t, err)
	require.Same(t, delivery, actual)

	logged := logs.String()
	require.Contains(t, logged, "failed to poll message")
	require.Contains(t, logged, `"source":"kafka"`)
	require.Contains(t, logged, `"trace-id":"abc"`)
	require.Contains(t, logged, `"partition":"0"`)
}
