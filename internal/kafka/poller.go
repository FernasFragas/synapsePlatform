//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/kafka/mocked_$GOFILE
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"synapsePlatform/internal/ingestor"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaConsumer handles consuming messages from Kafka
type KafkaConsumer struct {
	reader    *kafka.Reader
	committer *OffsetCommitter
	config    StreamingConfigs

	mu       sync.Mutex
	lastPoll time.Time
	maxStale time.Duration
}

// StreamingConfigs holds configuration for message broker connections.
type StreamingConfigs struct {
	Brokers  []string
	Topics   []string
	GroupID  string
	MinBytes int
	MaxBytes int
}

func NewConsumer(config StreamingConfigs, topic string, maxStale time.Duration) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:          config.Brokers,
		GroupID:          config.GroupID,
		Topic:            topic,
		MinBytes:         config.MinBytes,
		MaxBytes:         config.MaxBytes,
		MaxWait:          100 * time.Millisecond, // Don't wait too long
		ReadBatchTimeout: 100 * time.Millisecond,
	})

	return &KafkaConsumer{
		config:    config,
		reader:    reader,
		committer: NewOffsetCommitter(reader),
		lastPoll:  time.Now(),
		maxStale:  maxStale,
	}
}

func (c *KafkaConsumer) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, ingestor.AckHandler, error) {
	select {
	case <-ctx.Done():
		return nil, nil, nil
	default:
		// ReadMessage It needs to commit after succefully stored the data
		kafkaMsg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch message: %w", err)
		}

		// Update staleness tracker (Tier 1.3.2)
		c.mu.Lock()
		c.lastPoll = time.Now()
		c.mu.Unlock()

		// Bind the commit handle up front so EVERY outcome -- including a terminal
		// decode error -- routes through the contiguous committer. Committing the
		// raw offset directly here would skip any earlier in-flight message on this
		// partition (see OffsetCommitter).
		ack := c.committer.Ack(kafkaMsg)

		var deviceMessage ingestor.DeviceMessage
		if err := json.Unmarshal(kafkaMsg.Value, &deviceMessage); err != nil {
			// Terminal: the payload will never parse. So we hand the ack back so the
			// caller can DLQ then commit it, exactly like any other terminal error.
			return nil, ack, fmt.Errorf("decode device message: %w", err)
		}

		deviceMessage.Headers = c.convertHeaders(kafkaMsg.Headers)

		return &deviceMessage, ack, nil
	}
}

func (c *KafkaConsumer) Name() string { return "kafka" }

func (c *KafkaConsumer) Check(ctx context.Context) error {
	// Try to fetch metadata to verify connectivity
	conn, err := kafka.DialContext(ctx, "tcp", c.config.Brokers[0])
	if err != nil {
		return fmt.Errorf("kafka unreachable: %w", err)
	}

	defer conn.Close()

	// Check 2: Consumer staleness:
	// called from stats every 10s; if stats stop, liveness expires.
	c.mu.Lock()
	stale := time.Since(c.lastPoll) > c.maxStale
	c.mu.Unlock()
	if stale {
		return fmt.Errorf("consumer stale: no poll in %v", c.maxStale)
	}

	return nil
}

func (c *KafkaConsumer) Close(context.Context) error {
	return c.reader.Close()
}

func (c *KafkaConsumer) convertHeaders(kafkaHeaders []kafka.Header) map[string]string {
	headers := make(map[string]string)
	for _, h := range kafkaHeaders {
		headers[h.Key] = string(h.Value)
	}
	return headers
}
