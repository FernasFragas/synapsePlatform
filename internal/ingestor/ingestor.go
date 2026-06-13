//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Ingestor accepts incoming data, validates and normalizes it, then persists facts.

type DataProcessor interface {
	ProcessData(ctx context.Context) (*Delivery, error)
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
		cfg.BatchSize = 1
	}

	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = time.Second
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
		if i.cfg.BatchSize == 1 {
			return i.ingestSingle(ctx)
		}

		return i.ingestBatch(ctx)
	}

	return i.ingestWithWorkerPool(ctx)
}

func (i *Ingestor) ingestSingle(ctx context.Context) error {
	for {
		delivery, err := i.processor.ProcessData(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.failTerminalAndAck(ctx, delivery, "process", err, 0)
			}

			continue
		}

		if delivery == nil || delivery.Message == nil {
			continue
		}

		event, err := i.transformer.Transform(ctx, delivery.Message)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.failTerminalAndAck(ctx, delivery, "transform", err, 0)
			}

			continue
		}

		err = retryTransient(ctx, func(ctx context.Context) error {
			return i.storer.StoreData(ctx, event)
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.failTerminalAndAck(ctx, delivery, "store", err, maxStoreAttempts)
			}

			continue
		}

		i.ack(ctx, delivery, "store")
	}
}

func (i *Ingestor) ingestBatch(ctx context.Context) error {
	eventCh := make(chan eventWithDelivery, i.cfg.BatchSize*2)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return i.handleBatch(ctx, eventCh) })

	batch := make([]eventWithDelivery, 0, i.cfg.BatchSize)
	ticker := time.NewTicker(i.cfg.BatchTimeout)
	defer ticker.Stop()

	if err := i.saveBatchWithDeliveries(ctx, ticker, eventCh, batch); err != nil {
		return err
	}

	return g.Wait()
}

func (i *Ingestor) ingestWithWorkerPool(ctx context.Context) error {
	deliveryCh := make(chan *Delivery, i.cfg.NumWorkers*10)
	eventCh := make(chan eventWithDelivery, i.cfg.NumWorkers*i.cfg.BatchSize)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return i.processDeliveries(ctx, deliveryCh)
	})

	var workersWg sync.WaitGroup
	workersWg.Add(i.cfg.NumWorkers)

	for workerID := 0; workerID < i.cfg.NumWorkers; workerID++ {
		workerID := workerID

		g.Go(func() error {
			defer workersWg.Done()

			return i.transformDeliveries(ctx, deliveryCh, eventCh, workerID)
		})
	}

	g.Go(func() error {
		workersWg.Wait()
		close(eventCh)

		return nil
	})

	g.Go(func() error {
		return i.runBatcherWithDeliveries(ctx, eventCh)
	})

	return g.Wait()
}

func (i *Ingestor) handleBatch(ctx context.Context, eventCh chan<- eventWithDelivery) error {
	defer close(eventCh)

	for {
		if ctx.Err() != nil {
			return nil
		}

		delivery, err := i.processor.ProcessData(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.failTerminalAndAck(ctx, delivery, "process", err, 0)
			}

			continue
		}

		if delivery == nil || delivery.Message == nil {
			continue
		}

		event, err := i.transformer.Transform(ctx, delivery.Message)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.failTerminalAndAck(ctx, delivery, "transform", err, 0)
			}

			continue
		}

		select {
		case eventCh <- eventWithDelivery{Event: event, Delivery: delivery}:
		case <-ctx.Done():
			return nil
		}
	}
}

func (i *Ingestor) processDeliveries(ctx context.Context, deliveryCh chan<- *Delivery) error {
	defer close(deliveryCh)

	for {
		if ctx.Err() != nil {
			return nil
		}

		delivery, err := i.processor.ProcessData(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if Classify(err) == ClassTerminal {
				i.failTerminalAndAck(ctx, delivery, "process", err, 0)
			}

			continue
		}

		if delivery == nil || delivery.Message == nil {
			continue
		}

		select {
		case deliveryCh <- delivery:
		case <-ctx.Done():
			return nil
		}
	}
}

func (i *Ingestor) transformDeliveries(
	ctx context.Context,
	deliveryCh <-chan *Delivery,
	eventCh chan<- eventWithDelivery,
	workerID int,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case delivery, ok := <-deliveryCh:
			if !ok {
				return nil
			}

			if delivery == nil || delivery.Message == nil {
				continue
			}

			event, err := i.transformer.Transform(ctx, delivery.Message)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}

				if Classify(err) == ClassTerminal {
					i.failTerminalAndAck(
						ctx,
						delivery,
						"transform",
						fmt.Errorf("worker %d: %w", workerID, err),
						0,
					)
				}

				continue
			}

			select {
			case eventCh <- eventWithDelivery{Event: event, Delivery: delivery}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (i *Ingestor) runBatcherWithDeliveries(ctx context.Context, eventCh <-chan eventWithDelivery) error {
	batch := make([]eventWithDelivery, 0, i.cfg.BatchSize)
	ticker := time.NewTicker(i.cfg.BatchTimeout)
	defer ticker.Stop()

	return i.saveBatchWithDeliveries(ctx, ticker, eventCh, batch)
}

func (i *Ingestor) saveBatchWithDeliveries(
	ctx context.Context,
	ticker *time.Ticker,
	eventCh <-chan eventWithDelivery,
	batch []eventWithDelivery,
) error {
	for {
		select {
		case <-ctx.Done():
			i.flushBatchWithDeliveries(ctx, batch)

			return nil

		case <-ticker.C:
			i.flushBatchWithDeliveries(ctx, batch)
			batch = batch[:0]

		case item, ok := <-eventCh:
			if !ok {
				i.flushBatchWithDeliveries(ctx, batch)

				return nil
			}

			batch = append(batch, item)
			if len(batch) >= i.cfg.BatchSize {
				i.flushBatchWithDeliveries(ctx, batch)
				batch = batch[:0]
				resetTicker(ticker, i.cfg.BatchTimeout)
			}
		}
	}
}

// flushBatchWithDeliveries intentionally keeps the ingestion loop alive after
// store failures. Terminal errors are routed to failure storage and acked by
// storeBatchWithDeliveries. Transient errors leave deliveries unacked so the
// inbound adapter can redeliver them; storer log/metrics decorators record the
// operational failure.
func (i *Ingestor) flushBatchWithDeliveries(ctx context.Context, batch []eventWithDelivery) {
	if len(batch) == 0 {
		return
	}

	_ = i.storeBatchWithDeliveries(ctx, batch)
}

func (i *Ingestor) storeBatchWithDeliveries(ctx context.Context, batch []eventWithDelivery) error {
	if len(batch) == 0 {
		return nil
	}

	events := make([]*BaseEvent, len(batch))
	for idx, item := range batch {
		events[idx] = item.Event
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
				i.failTerminalAndAck(ctx, item.Delivery, "store_batch", err, maxStoreAttempts)
			}
		}

		return err
	}

	for _, item := range batch {
		i.ack(ctx, item.Delivery, "store_batch")
	}

	return nil
}

type eventWithDelivery struct {
	Event    *BaseEvent
	Delivery *Delivery
}

const maxStoreAttempts = 3

func retryTransient(ctx context.Context, op func(context.Context) error) error {
	var err error

	for attempt := 0; attempt < maxStoreAttempts; attempt++ {
		if err = op(ctx); err == nil {
			return nil
		}

		if Classify(err) == ClassTerminal {
			return err
		}

		timer := time.NewTimer(time.Duration(1<<attempt) * time.Second)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			return ctx.Err()
		}
	}

	return err
}

func (i *Ingestor) failTerminalAndAck(
	ctx context.Context,
	delivery *Delivery,
	stage string,
	err error,
	retryCount int,
) {
	if delivery == nil {
		return
	}

	failed := FailedMessage{
		Stage:        stage,
		ErrorType:    ClassTerminal.String(),
		ErrorMessage: err.Error(),
		RetryCount:   retryCount,
		Message:      delivery.Message,
		Metadata:     delivery.Metadata,
		FailedAt:     time.Now().UTC(),
	}

	if storeErr := i.failures.StoreFailure(ctx, failed); storeErr != nil {
		// Do not ack. Redelivery is safer than losing the original message.
		return
	}

	i.ack(ctx, delivery, stage)
}

func (i *Ingestor) ack(ctx context.Context, delivery *Delivery, stage string) {
	if delivery == nil || delivery.Ack == nil {
		return
	}

	if err := delivery.Ack(ctx); err != nil && ctx.Err() == nil {
		_ = i.failures.StoreFailure(ctx, FailedMessage{
			Stage:        stage,
			ErrorType:    "ack_failed",
			ErrorMessage: err.Error(),
			Metadata:     delivery.Metadata,
			FailedAt:     time.Now().UTC(),
		})
	}
}

func resetTicker(ticker *time.Ticker, timeout time.Duration) {
	if !ticker.Stop() {
		select {
		case <-ticker.C:
		default:
		}
	}

	ticker.Reset(timeout)
}
