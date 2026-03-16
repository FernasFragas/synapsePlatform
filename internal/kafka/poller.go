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
	reader *kafka.Reader
	config StreamingConfigs

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
		Brokers:        config.Brokers,
		GroupID:        config.GroupID,
		Topic:          topic,
		MinBytes:       config.MinBytes,
		MaxBytes:       config.MaxBytes,
		CommitInterval: time.Second,
	})

	return &KafkaConsumer{
		config:   config,
		reader:   reader,
		lastPoll: time.Now(),
		maxStale: maxStale,
	}
}

func (c *KafkaConsumer) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		kafkaMsg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return nil, err
		}

		c.mu.Lock()
		c.lastPoll = time.Now()
		c.mu.Unlock()

		var deviceMessage ingestor.DeviceMessage
		if err := json.Unmarshal(kafkaMsg.Value, &deviceMessage); err != nil {
			return nil, err
		}

		deviceMessage.Headers = c.convertHeaders(kafkaMsg.Headers)

		return &deviceMessage, nil
	}
}

func (c *KafkaConsumer) Name() string { return "kafka" }

func (c *KafkaConsumer) Check(_ context.Context) error {
	c.mu.Lock()
	last := c.lastPoll
	c.mu.Unlock()

	stale := time.Since(last)
	if stale > c.maxStale {
		return fmt.Errorf("last successful poll %s ago (threshold %s)",
			stale.Round(time.Second), c.maxStale)
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
