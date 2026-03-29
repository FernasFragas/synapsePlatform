//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Ingestor:
// It accepts incoming data.
// It validates and normalizes.
// It persists facts.

// DataProcessor interface responsible for processing the readings received.
type DataProcessor interface {
	ProcessData(ctx context.Context) (*DeviceMessage, error)
	ProcessDataBatch(ctx context.Context, maxMessages int) ([]*DeviceMessage, error)
}

type MessageStorer interface {
	StoreData(ctx context.Context, data *BaseEvent) error
	StoreBatch(ctx context.Context, events []*BaseEvent) error
}

type Transformer interface {
	Transform(ctx context.Context, msg *DeviceMessage) (*BaseEvent, error)
}

type NormalizedData interface {
	Validate() error
	Normalize() error
}

type FailureStorer interface {
	StoreFailure(ctx context.Context, failed FailedMessage) error
}

type Config struct {
	CompatibleDataTypes []DataTypes
	BatchSize           int
	BatchTimeout        time.Duration
	NumWorkers          int
}

type Ingestor struct {
	cfg         Config
	processor   DataProcessor
	storer      MessageStorer
	transformer Transformer
	failures    FailureStorer
}

func New(cfg Config, processor DataProcessor, storer MessageStorer, transformer Transformer, failures FailureStorer) *Ingestor {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1 // Default to no batching
	}

	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 1 * time.Second
	}

	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 1
	}

	return &Ingestor{
		cfg:         cfg,
		processor:   processor,
		storer:      storer,
		transformer: transformer,
		failures:    failures,
	}

}

func (i *Ingestor) Ingest(ctx context.Context) error {
	if i.cfg.NumWorkers == 1 {
		// Use existing single-worker implementation
		if i.cfg.BatchSize == 1 {
			return i.ingestSingle(ctx)
		}

		return i.ingestBatch(ctx)
	}

	// Use worker pool implementation
	return i.ingestWithWorkerPool(ctx)
}

// Simple single-message processing (no batching overhead)
func (i *Ingestor) ingestSingle(ctx context.Context) error {
	for {
		msg, err := i.processor.ProcessData(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			_ = i.failures.StoreFailure(ctx, FailedMessage{
				Stage: "process", Message: msg, Err: err,
			})

			continue
		}

		if msg == nil {
			continue
		}

		transformedData, err := i.transformer.Transform(ctx, msg)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			_ = i.failures.StoreFailure(ctx, FailedMessage{
				Stage: "transform", Message: msg, Err: err,
			})

			continue
		}

		err = i.storer.StoreData(ctx, transformedData)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			_ = i.failures.StoreFailure(ctx, FailedMessage{
				Stage: "store", Message: msg, Err: err,
			})

			continue
		}
	}
}

// Batch processing with proper channel-based coordination
func (i *Ingestor) ingestBatch(ctx context.Context) error {
	batch := make([]*BaseEvent, 0, i.cfg.BatchSize)
	ticker := time.NewTicker(i.cfg.BatchTimeout)
	defer ticker.Stop()

	eventCh := make(chan *BaseEvent, i.cfg.BatchSize*2)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error { return i.handleBatch(ctx, eventCh) })

	return i.saveBatch(ctx, ticker, eventCh, batch)
}

// Worker pool implementation
func (i *Ingestor) ingestWithWorkerPool(ctx context.Context) error {
	msgCh := make(chan *DeviceMessage, i.cfg.NumWorkers*10)

	eventCh := make(chan *BaseEvent, i.cfg.NumWorkers*i.cfg.BatchSize)

	g, ctx := errgroup.WithContext(ctx)
	var workersWg sync.WaitGroup

	g.Go(func() error {
		return i.process(ctx, msgCh)
	})

	workersWg.Add(i.cfg.NumWorkers)
	for w := 0; w < i.cfg.NumWorkers; w++ {
		workerID := w

		g.Go(func() error {
			defer workersWg.Done()

			return i.transform(ctx, msgCh, eventCh,workerID)
		})
	}

	g.Go(func() error {
		workersWg.Wait()

		close(eventCh)

		return nil
	})

	g.Go(func() error { return i.runBatcher(ctx, eventCh) })

	return g.Wait()
}

// Batcher collects events and flushes in batches
func (i *Ingestor) runBatcher(ctx context.Context, eventCh <-chan *BaseEvent) error {
	batch := make([]*BaseEvent, 0, i.cfg.BatchSize)

	ticker := time.NewTicker(i.cfg.BatchTimeout)

	defer ticker.Stop()

	return i.saveBatch(ctx, ticker, eventCh, batch)
}

func (i *Ingestor) handleBatch(ctx context.Context, eventCh chan *BaseEvent) error {
	for {
		if ctx.Err() != nil {
			close(eventCh)

			return nil
		}

		msg, err := i.processor.ProcessData(ctx)
		if err != nil {
			_ = i.failures.StoreFailure(ctx, FailedMessage{
				Stage: "process", Message: msg, Err: err,
			})

			continue
		}

		if msg == nil {
			continue
		}

		transformedData, err := i.transformer.Transform(ctx, msg)
		if err != nil {
			if ctx.Err() != nil {
				close(eventCh)

				return nil
			}
			_ = i.failures.StoreFailure(ctx, FailedMessage{
				Stage: "transform", Message: msg, Err: err,
			})

			continue
		}

		select {
		case eventCh <- transformedData:
		case <-ctx.Done():
			close(eventCh)

			return nil
		}
	}
}

// In internal/ingestor/ingestor.go - modify the process() function
func (i *Ingestor) process(ctx context.Context, msgCh chan *DeviceMessage) error {
	defer close(msgCh)

	for {
		if ctx.Err() != nil {
			return nil
		}


		// Fetch a batch of messages (e.g., 10 at a time)
		msgs, err := i.processor.ProcessDataBatch(ctx, 10)
		if err != nil {
			_ = i.failures.StoreFailure(ctx, FailedMessage{
				Stage: "process_batch", Message: nil, Err: err,
			})
			continue
		}

		// Send all messages to workers
		for _, msg := range msgs {
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (i *Ingestor) transform(ctx context.Context, msgCh chan *DeviceMessage, eventCh chan *BaseEvent, workerID int) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case msg, ok := <-msgCh:
			if !ok {
				return nil
			}

			transformedData, err := i.transformer.Transform(ctx, msg)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				_ = i.failures.StoreFailure(ctx, FailedMessage{
					Stage:   "transform",
					Message: msg,
					Err:     fmt.Errorf("worker %d: %w", workerID, err),
				})
				continue
			}

			select {
			case eventCh <- transformedData:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (i *Ingestor) saveBatch(ctx context.Context, ticker *time.Ticker, eventCh <-chan *BaseEvent, batch []*BaseEvent) error {
	for {
		select {
		case <-ctx.Done():
			_ = i.storeBatch(ctx, batch, ticker)

			return nil

		case <-ticker.C:
			if err := i.storeBatch(ctx, batch, ticker); err != nil {
				batch = batch[:0]

				continue
			}

		case event, ok := <-eventCh:
			if !ok {
				_ = i.storeBatch(ctx, batch, ticker)
				batch = batch[:0]

				return nil
			}

			batch = append(batch, event)

			if len(batch) >= i.cfg.BatchSize {
				if err := i.storeBatch(ctx, batch, ticker); err != nil {
					batch = batch[:0]

					continue
				}

				batch = batch[:0]
			}
		}
	}
}

func (i *Ingestor) storeBatch(ctx context.Context, batch []*BaseEvent, ticker *time.Ticker) error {
	if len(batch) == 0 {
		return nil
	}

	var err error
	if len(batch) == 1 {
		err = i.storer.StoreData(ctx, batch[0])
	} else {
		err = i.storer.StoreBatch(ctx, batch)
	}

	if err != nil {
		for _, event := range batch {
			failedMsg := &DeviceMessage{
				DeviceID:  event.EntityID,
				Type:      event.EventType,
				Timestamp: event.OccurredAt,
			}

			_ = i.failures.StoreFailure(ctx, FailedMessage{
				Stage:   "store_batch",
				Message: failedMsg,
				Err:     fmt.Errorf("batch store failed: %w", err),
			})
		}
	}

	ticker.Reset(i.cfg.BatchTimeout)

	return err
}
