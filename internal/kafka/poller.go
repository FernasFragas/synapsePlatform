//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/kafka/mocked_$GOFILE
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"synapsePlatform/internal/ingestor"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)
// KafkaConsumer handles consuming messages from Kafka using low-level Conn API
type KafkaConsumer struct {
	conn     *kafka.Conn  // Changed from reader to conn
	config   StreamingConfigs
	topic    string
	partition int
	mu       sync.Mutex
	lastPoll time.Time
	maxStale time.Duration
	// Track current offset for manual commit
	currentOffset int64
	offsetMu      sync.RWMutex
}
// StreamingConfigs holds configuration for message broker connections.
type StreamingConfigs struct {
	Brokers   []string
	Topics    []string
	GroupID   string  // Not used with Conn approach
	MinBytes  int
	MaxBytes  int
	Partition int     // NEW: Must specify partition
}
// NewConsumer creates a consumer for a specific topic and partition
func NewConsumer(config StreamingConfigs, topic string, maxStale time.Duration) *KafkaConsumer {
	return &KafkaConsumer{
		config:        config,
		topic:         topic,
		partition:     config.Partition,
		lastPoll:      time.Now(),
		maxStale:      maxStale,
		currentOffset: kafka.FirstOffset, // Start from beginning
	}
}
// ensureConnection establishes connection to the partition leader if not already connected
func (c *KafkaConsumer) ensureConnection(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return nil
	}
	// Dial the leader for this topic/partition
	conn, err := kafka.DialLeader(ctx, "tcp", c.config.Brokers[0], c.topic, c.partition)
	if err != nil {
		return fmt.Errorf("failed to dial leader: %w", err)
	}
	// Set the offset to start reading from
	c.offsetMu.RLock()
	offset := c.currentOffset
	c.offsetMu.RUnlock()
	if _, err := conn.Seek(offset, kafka.SeekAbsolute); err != nil {
		conn.Close()
		return fmt.Errorf("failed to seek to offset %d: %w", offset, err)
	}
	c.conn = conn
	return nil
}
// PollMessage reads a single message (for backward compatibility)
func (c *KafkaConsumer) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}
	// Use ReadMessage to get a single message
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	kafkaMsg, err := c.conn.ReadMessage(c.config.MaxBytes)
	if err != nil {
		return nil, err
	}
	var deviceMessage ingestor.DeviceMessage
	if err := json.Unmarshal(kafkaMsg.Value, &deviceMessage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	deviceMessage.Headers = c.convertHeaders(kafkaMsg.Headers)
	// Update current offset
	c.offsetMu.Lock()
	c.currentOffset = kafkaMsg.Offset + 1
	c.offsetMu.Unlock()
	return &deviceMessage, nil
}

// PollMessages uses ReadBatch to fetch multiple messages efficiently.
func (c *KafkaConsumer) PollMessages(ctx context.Context, maxMessages int) ([]*ingestor.DeviceMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Ensure connection is established
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	messages := make([]*ingestor.DeviceMessage, 0, maxMessages)

	// ReadBatch fetches a batch from Kafka
	batch := c.conn.ReadBatch(c.config.MinBytes, c.config.MaxBytes)

	// IMPORTANT: Don't defer Close here if you want to reuse the connection
	// Instead, close it explicitly after reading all messages

	// Check for batch-level errors BEFORE reading
	if err := batch.Err(); err != nil {
		batch.Close()  // Close the bad batch
		// Reconnect for next call
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()
		return nil, fmt.Errorf("batch error: %w", err)
	}

	// Read messages from the batch
	for i := 0; i < maxMessages; i++ {
		kafkaMsg, err := batch.ReadMessage()
		if err != nil {
			if err == io.EOF {
				// No more messages in batch - this is normal
				break
			}
			// Other error - close batch and return what we have
			break
		}

		var deviceMessage ingestor.DeviceMessage
		if err := json.Unmarshal(kafkaMsg.Value, &deviceMessage); err != nil {
			// Skip invalid messages but continue reading batch
			continue
		}

		deviceMessage.Headers = c.convertHeaders(kafkaMsg.Headers)
		messages = append(messages, &deviceMessage)

		// Track offset
		c.offsetMu.Lock()
		c.currentOffset = kafkaMsg.Offset + 1
		c.offsetMu.Unlock()
	}

	// NOW close the batch
	if err := batch.Close(); err != nil {
		// Log but don't fail - we got the messages
		// The connection might be in a bad state though
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil  // Force reconnect next time
		}
		c.mu.Unlock()
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no valid messages in batch")
	}

	return messages, nil
}
func (c *KafkaConsumer) Name() string {
	return fmt.Sprintf("kafka-partition-%d", c.partition)
}
func (c *KafkaConsumer) Check(ctx context.Context) error {
	// Try to establish connection to verify connectivity
	if err := c.ensureConnection(ctx); err != nil {
		return fmt.Errorf("kafka unreachable: %w", err)
	}
	return nil
}
func (c *KafkaConsumer) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
func (c *KafkaConsumer) convertHeaders(kafkaHeaders []kafka.Header) map[string]string {
	headers := make(map[string]string)
	for _, h := range kafkaHeaders {
		headers[h.Key] = string(h.Value)
	}
	return headers
}
// GetCurrentOffset returns the current offset for monitoring/debugging.
func (c *KafkaConsumer) GetCurrentOffset() int64 {
	c.offsetMu.RLock()
	defer c.offsetMu.RUnlock()

	return c.currentOffset
}

func GetTopicPartitions(ctx context.Context, brokers []string, topic string) (int, error) {
	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return 0, fmt.Errorf("failed to dial broker: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		return 0, fmt.Errorf("failed to read partitions: %w", err)
	}

	return len(partitions), nil
}