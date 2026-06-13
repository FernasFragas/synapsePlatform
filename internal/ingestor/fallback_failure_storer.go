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
	ID           string
	Stage        string
	ErrorType    string
	ErrorMessage string
	RetryCount   int
	Message      *DeviceMessage
	Metadata     MessageMetadata
	FailedAt     time.Time
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
