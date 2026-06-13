//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/kafka/mocked_$GOFILE
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

func (c *KafkaConsumer) PollMessage(ctx context.Context) (*ingestor.Delivery, error) {
	kafkaMsg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return nil, ingestor.NewTransientError(fmt.Errorf("fetch message: %w", err))
	}

	c.markPolled()

	ack := c.committer.Ack(kafkaMsg)

	var msg ingestor.DeviceMessage
	if err := json.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		return &ingestor.Delivery{
			Metadata: kafkaMetadata(kafkaMsg),
			Ack:      ack,
		}, ingestor.NewTerminalError(fmt.Errorf("decode device message: %w", err))
	}

	return &ingestor.Delivery{
		Message:  &msg,
		Metadata: kafkaMetadata(kafkaMsg),
		Ack:      ack,
	}, nil
}

func kafkaMetadata(msg kafka.Message) ingestor.MessageMetadata {
	return ingestor.MessageMetadata{
		Source:  "kafka",
		Headers: convertHeaders(msg.Headers),
		Labels: map[string]string{
			"topic":     msg.Topic,
			"partition": strconv.Itoa(msg.Partition),
			"offset":    strconv.FormatInt(msg.Offset, 10),
		},
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

func (c *KafkaConsumer) markPolled() {
	c.mu.Lock()
	c.lastPoll = time.Now()
	c.mu.Unlock()
}

func convertHeaders(kafkaHeaders []kafka.Header) map[string]string {
	if len(kafkaHeaders) == 0 {
		return nil
	}

	headers := make(map[string]string, len(kafkaHeaders))
	for _, h := range kafkaHeaders {
		headers[h.Key] = string(h.Value)
	}
	return headers
}
