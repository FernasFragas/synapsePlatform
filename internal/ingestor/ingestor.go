//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
)

// Ingestor:
// It accepts incoming data.
// It validates and normalizes.
// It persists facts.

// DataProcessor interface responsible for processing the readings received.
type DataProcessor interface {
	ProcessData(ctx context.Context) (*DeviceMessage, error)
}

type MessageStorer interface {
	StoreData(ctx context.Context, data *BaseEvent) error
}

type Transformer interface {
	Transform(ctx context.Context, msg *DeviceMessage) (*BaseEvent, error)
}

type NormalizedData interface {
	Validate() error
	Normalize() error
}

type Config struct {
	CompatibleDataTypes []DataTypes
}

type Ingestor struct {
	cfg         Config
	processor   DataProcessor
	storer      MessageStorer
	transformer Transformer
}

func New(cfg Config, processor DataProcessor, storer MessageStorer, transformer Transformer) *Ingestor {
	return &Ingestor{
		cfg:         cfg,
		processor:   processor,
		storer:      storer,
		transformer: transformer,
	}

}

func (i *Ingestor) Ingest(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := i.processor.ProcessData(ctx)
			if err != nil {
				continue
			}

			if msg == nil {
				continue
			}

			transformedData, err := i.transformer.Transform(ctx, msg)
			if err != nil {
				continue
			}

			err = i.storer.StoreData(ctx, transformedData)
			if err != nil {
				continue
			}
		}
	}
}
