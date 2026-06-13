package kafka

import (
	"context"
	"synapsePlatform/internal/ingestor"
	"sync"

	"github.com/segmentio/kafka-go"
)

// OffsetCommitter turns "this message is done" into a correct Kafka offset
// commit. it has to account for a fact with no producer equivalent:
// Kafka stores ONE offset per partition,
// so committing offset N implicitly acknowledges every offset <= N on that partition.
//
// A naive per-message commit (reader.CommitMessages(ctx, msg)) skips
// over an earlier, still-failing message the moment a later one succeeds,
// silently destroying the redelivery that at-least-once delivery relies on.
//
// OffsetCommitter only advances the committed position across the highest
// CONTIGUOUS run of completed offsets, so an un-acked (transient-failed) offset
// is a gap that blocks every later commit on its partition until it succeeds on
// redelivery. This is what makes the closure-per-message API safe under the
// out-of-order completion of the batch and worker-pool paths.
type OffsetCommitter struct {
	reader *kafka.Reader

	mu    sync.Mutex
	parts map[int]*partitionProgress
}

type partitionProgress struct {
	next      int64                   // next offset eligible to be committed
	completed map[int64]kafka.Message // completed offsets >= next, awaiting contiguity
}

func NewOffsetCommitter(reader *kafka.Reader) *OffsetCommitter {
	return &OffsetCommitter{
		reader: reader,
		parts:  make(map[int]*partitionProgress),
	}
}

// Ack returns an ingestor.AckHandler bound to msg. It is called at FETCH time
// (synchronously, inside PollMessage), which is what makes the watermark
// correct: FetchMessage delivers a partition's offsets strictly in order, so
// the first offset we register for a partition is its lowest uncommitted one,
// the right starting watermark. Pinning it at completion time instead would be
// a bug: a higher offset finishing first (workers complete out of order) would
// commit over earlier, not-yet-stored messages.
//
// The returned handler is invoked later, once msg is durably stored (or DLQ'd);
// it records completion and commits the new contiguous watermark.
func (c *OffsetCommitter) Ack(msg kafka.Message) ingestor.AckHandler {
	c.register(msg)

	return func(ctx context.Context) error {
		return c.complete(ctx, msg)
	}
}

func (c *OffsetCommitter) register(msg kafka.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.parts[msg.Partition] != nil {
		return
	}

	c.parts[msg.Partition] = &partitionProgress{
		next:      msg.Offset,
		completed: make(map[int64]kafka.Message),
	}
}

func (c *OffsetCommitter) complete(ctx context.Context, msg kafka.Message) error {
	c.mu.Lock()
	p := c.parts[msg.Partition] // always present: Ack registered it at fetch time
	p.completed[msg.Offset] = msg

	// Walk the contiguous run of completed offsets; commit only the highest one.
	var (
		commit   kafka.Message
		advanced bool
	)
	for {
		m, ok := p.completed[p.next]
		if !ok {
			break
		}
		commit, advanced = m, true
		delete(p.completed, p.next)
		p.next++
	}
	c.mu.Unlock()

	if !advanced {
		// Completed out of order: an earlier offset is still in flight. Holding
		// the commit is exactly the desired behaviour -- never skip the gap.
		return nil
	}
	return c.reader.CommitMessages(ctx, commit)
}
