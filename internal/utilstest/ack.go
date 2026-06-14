package utilstest

import (
	"context"
	"synapsePlatform/internal/ingestor"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type Ack struct {
	t     *testing.T
	calls atomic.Int64
	err   error
}

func NewAck(t *testing.T) *Ack {
	return &Ack{t: t}
}

func (a *Ack) WithError(err error) *Ack {
	a.err = err
	return a
}

func (a *Ack) Handler() ingestor.AckHandler {
	return func(context.Context) error {
		a.calls.Add(1)
		return a.err
	}
}

func (a *Ack) RequireCalls(expected int64) {
	require.Equal(a.t, expected, a.calls.Load())
}
