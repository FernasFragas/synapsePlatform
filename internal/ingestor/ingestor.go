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
	ProcessData(ctx context.Context) (*DeviceMessage, AckHandler, error)
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
		msg, ack, err := i.processor.ProcessData(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.dlqAndCommit(ctx, ack, FailedMessage{
					Stage: "process", Message: msg, ErrorMessage: err.Error(), ErrorType: ClassTerminal.String(),
				})
			}

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

			if Classify(err) == ClassTerminal {
				i.dlqAndCommit(ctx, ack, FailedMessage{
					Stage: "transform", Message: msg, ErrorMessage: err.Error(), ErrorType: ClassTerminal.String(),
				})
			}

			continue
		}

		// Retry transient store failures with ctx-aware backoff.
		err = retryTransient(ctx, func(ctx context.Context) error {
			return i.storer.StoreData(ctx, transformedData)
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if Classify(err) == ClassTerminal {
				i.dlqAndCommit(ctx, ack, FailedMessage{
					Stage: "store", Message: msg, ErrorMessage: err.Error(),
					ErrorType: ClassTerminal.String(), RetryCount: maxStoreAttempts,
				})

				continue
			}

			continue
		}

		// Stored durably -- NOW commit the offset.
		i.commit(ctx, ack, "store")
	}
}

// Batch processing with proper channel-based coordination
func (i *Ingestor) ingestBatch(ctx context.Context) error {
	eventCh := make(chan eventWithAck, i.cfg.BatchSize*2)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return i.handleBatch(ctx, eventCh) })

	batch := make([]eventWithAck, 0, i.cfg.BatchSize)
	ticker := time.NewTicker(i.cfg.BatchTimeout)
	defer ticker.Stop()

	return i.saveBatchWithAcks(ctx, ticker, eventCh, batch)
}

// Worker pool implementation
func (i *Ingestor) ingestWithWorkerPool(ctx context.Context) error {
	msgCh := make(chan messageWithAck, i.cfg.NumWorkers*10)
	eventCh := make(chan eventWithAck, i.cfg.NumWorkers*i.cfg.BatchSize)

	g, ctx := errgroup.WithContext(ctx)
	var workersWg sync.WaitGroup

	g.Go(func() error {
		return i.processWithAck(ctx, msgCh)
	})

	workersWg.Add(i.cfg.NumWorkers)
	for w := 0; w < i.cfg.NumWorkers; w++ {
		workerID := w
		g.Go(func() error {
			defer workersWg.Done()
			return i.transformWithAck(ctx, msgCh, eventCh, workerID)
		})
	}

	g.Go(func() error {
		workersWg.Wait()
		close(eventCh)
		return nil
	})

	g.Go(func() error { return i.runBatcherWithAck(ctx, eventCh) })

	return g.Wait()
}

// Batcher collects events and flushes in batches
func (i *Ingestor) runBatcher(ctx context.Context, eventCh <-chan *BaseEvent) error {
	batch := make([]*BaseEvent, 0, i.cfg.BatchSize)

	ticker := time.NewTicker(i.cfg.BatchTimeout)

	defer ticker.Stop()

	return i.saveBatch(ctx, ticker, eventCh, batch)
}

func (i *Ingestor) handleBatch(ctx context.Context, eventCh chan<- eventWithAck) error {
	defer close(eventCh)

	for {
		if ctx.Err() != nil {
			return nil
		}

		msg, ack, err := i.processor.ProcessData(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.dlqAndCommit(ctx, ack, FailedMessage{
					Stage:        "process",
					Message:      msg,
					ErrorMessage: err.Error(),
					ErrorType:    ClassTerminal.String(),
				})
			}

			continue
		}
		if msg == nil {
			continue
		}

		event, err := i.transformer.Transform(ctx, msg)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.dlqAndCommit(ctx, ack, FailedMessage{
					Stage:        "transform",
					Message:      msg,
					ErrorMessage: err.Error(),
					ErrorType:    ClassTerminal.String(),
				})
			}

			continue
		}

		select {
		case eventCh <- eventWithAck{Event: event, Ack: ack}:
		case <-ctx.Done():
			return nil
		}
	}
}

func (i *Ingestor) processWithAck(ctx context.Context, msgCh chan<- messageWithAck) error {
	defer close(msgCh)

	for {
		if ctx.Err() != nil {
			return nil
		}

		msg, ack, err := i.processor.ProcessData(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.dlqAndCommit(ctx, ack, FailedMessage{
					Stage:        "process",
					Message:      msg,
					ErrorMessage: err.Error(),
					ErrorType:    ClassTerminal.String(),
				})
			}

			continue
		}

		if msg == nil {
			continue
		}

		select {
		case msgCh <- messageWithAck{Message: msg, Ack: ack}:
		case <-ctx.Done():
			return nil
		}
	}
}

func (i *Ingestor) transformWithAck(ctx context.Context, msgCh <-chan messageWithAck,
	eventCh chan<- eventWithAck, workerID int) error {

	for {
		select {
		case <-ctx.Done():
			return nil
		case item, ok := <-msgCh:
			if !ok {
				return nil
			}

			event, err := i.transformer.Transform(ctx, item.Message)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}

				if Classify(err) == ClassTerminal {
					i.dlqAndCommit(ctx, item.Ack, FailedMessage{
						Stage:        "transform",
						Message:      item.Message,
						ErrorMessage: fmt.Sprintf("worker %d: %s", workerID, err),
						ErrorType:    ClassTerminal.String(),
					})
				}

				continue
			}

			select {
			case eventCh <- eventWithAck{Event: event, Ack: item.Ack}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (i *Ingestor) runBatcherWithAck(ctx context.Context, eventCh <-chan eventWithAck) error {
	batch := make([]eventWithAck, 0, i.cfg.BatchSize)

	ticker := time.NewTicker(i.cfg.BatchTimeout)

	defer ticker.Stop()

	return i.saveBatchWithAcks(ctx, ticker, eventCh, batch)
}

func (i *Ingestor) saveBatchWithAcks(ctx context.Context, ticker *time.Ticker,
	eventCh <-chan eventWithAck, batch []eventWithAck) error {

	for {
		select {
		case <-ctx.Done():
			_ = i.storeBatchWithAcks(ctx, batch, ticker)
			batch = batch[:0]

			return nil
		case <-ticker.C:
			if err := i.storeBatchWithAcks(ctx, batch, ticker); err != nil {
				batch = batch[:0]

				continue
			}

			batch = batch[:0]
		case item, ok := <-eventCh:
			if !ok {
				_ = i.storeBatchWithAcks(ctx, batch, ticker)
				batch = batch[:0]

				return nil
			}

			batch = append(batch, item)
			if len(batch) >= i.cfg.BatchSize {
				if err := i.storeBatchWithAcks(ctx, batch, ticker); err != nil {
					batch = batch[:0]

					continue
				}

				batch = batch[:0]
			}
		}
	}
}
func (i *Ingestor) storeBatchWithAcks(ctx context.Context, batch []eventWithAck,
	ticker *time.Ticker) error {
	defer ticker.Reset(i.cfg.BatchTimeout)

	if len(batch) == 0 {
		return nil
	}

	events := make([]*BaseEvent, len(batch))

	for n, item := range batch {
		events[n] = item.Event
	}

	err := retryTransient(ctx, func(ctx context.Context) error {
		if len(events) == 1 {
			return i.storer.StoreData(ctx, events[0])
		}
		return i.storer.StoreBatch(ctx, events)
	})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if Classify(err) == ClassTerminal {
			for _, item := range batch {
				i.dlqAndCommit(ctx, item.Ack, FailedMessage{
					Stage: "store_batch",
					Message: &DeviceMessage{
						DeviceID:  item.Event.EntityID,
						Type:      item.Event.EventType,
						Timestamp: item.Event.OccurredAt,
					},
					ErrorMessage: fmt.Sprintf("batch store failed: %s", err),
					ErrorType:    ClassTerminal.String(),
					RetryCount:   maxStoreAttempts,
				})
			}
		}

		return err
	}

	for _, item := range batch {
		i.commit(ctx, item.Ack, "store_batch")
	}

	return nil
}

func (i *Ingestor) saveBatch(ctx context.Context, ticker *time.Ticker, eventCh <-chan *BaseEvent, batch []*BaseEvent) error {
	for {
		select {
		case <-ctx.Done():
			_ = i.storeBatch(ctx, batch, ticker)
			batch = batch[:0]

			return nil

		case <-ticker.C:
			if err := i.storeBatch(ctx, batch, ticker); err != nil {
				batch = batch[:0]

				continue
			}
			batch = batch[:0]

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
				Stage:        "store_batch",
				Message:      failedMsg,
				ErrorMessage: fmt.Sprintf("batch store failed: %s", err),
			})
		}
	}

	ticker.Reset(i.cfg.BatchTimeout)

	return err
}

// messageWithAck wraps a raw device message with its Kafka offset commit handle.
// This carries the ack through the poll → validate stage.
type messageWithAck struct {
	Message *DeviceMessage
	Ack     AckHandler
}

// eventWithAck pairs a transformed event with its Kafka offset commit handle.
// This carries the ack through the transform → store stage.
type eventWithAck struct {
	Event *BaseEvent
	Ack   AckHandler
}

const maxStoreAttempts = 3

// retryTransient runs op until it succeeds, hits a terminal error, or exhausts
// maxStoreAttempts. Unlike time.Sleep, the backoff honours ctx so shutdown is
// immediate -- a stuck store never pins a goroutine past cancellation.
func retryTransient(ctx context.Context, op func(context.Context) error) error {
	var err error
	for attempt := 0; attempt < maxStoreAttempts; attempt++ {
		if err = op(ctx); err == nil {
			return nil
		}
		if Classify(err) == ClassTerminal {
			return err // retrying a terminal error only burns time
		}
		backoff := time.Duration(1<<attempt) * time.Second // 1s, 2s, 4s
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

// dlqAndCommit routes a terminally-failed message to the DLQ and commits its
// offset so it stops blocking the partition. Both steps are best-effort but
// LOGGED: a lost DLQ write or a failed commit is an operational signal, not
// something to silently discard.
func (i *Ingestor) dlqAndCommit(ctx context.Context, ack AckHandler, failed FailedMessage) {
	_ = i.failures.StoreFailure(ctx, failed)

	i.commit(ctx, ack, failed.Stage)
}

// commit awaits the offset commit and logs a genuine failure (ignoring the
// expected error during shutdown). PostHog equivalent: awaiting the AckFuture.
func (i *Ingestor) commit(ctx context.Context, ack AckHandler, stage string) {
	if ack == nil {
		return
	}
	if err := ack(ctx); err != nil && ctx.Err() == nil {
		_ = i.failures.StoreFailure(ctx, FailedMessage{
			Stage:        stage,
			ErrorType:    "commit_failed",
			ErrorMessage: fmt.Sprintf("offset commit failed: %s", err),
		})
	}
}
