package ingestor

import (
	"context"
	"fmt"
	"time"
)

type FallbackFailureStorer struct {
	primary   FailureStorer
	secondary FailureStorer
}

type FailedMessage struct {
	ID            string // unique ID for DLQ lookup
	OriginalTopic string // e.g., "ingestion.raw"
	Partition     int    // Kafka partition
	Offset        int64  // Kafka offset
	Stage         string // "process", "transform", "store"
	ErrorType     string // "transient", "terminal", "poison"
	ErrorMessage  string
	RetryCount    int // how many times retried
	Message       *DeviceMessage
	Headers       map[string]string // original Kafka headers
	Timestamp     time.Time         // when failure occurred
}

func NewFallbackFailureStorer(primary, secondary FailureStorer) *FallbackFailureStorer {
	return &FallbackFailureStorer{primary: primary, secondary: secondary}
}

func (f *FallbackFailureStorer) StoreFailure(ctx context.Context, failed FailedMessage) error {
	if err := f.primary.StoreFailure(ctx, failed); err == nil {
		return nil
	}

	if err := f.secondary.StoreFailure(ctx, failed); err != nil {
		return fmt.Errorf("all failure backends unavailable: %w", err)
	}

	return nil
}
