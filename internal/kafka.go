//go:generate mockgen -source=$GOFILE -destination=../internal/utilstest/mocksgen/mocked_$GOFILE
package internal

import (
	"context"
	"encoding/json"
	"synapsePlatform/internal/ingestor"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaConsumer handles consuming messages from Kafka
type KafkaConsumer struct {
	reader []*kafka.Reader
	config StreamingConfigs
}

// StreamingConfigs holds configuration for message broker connections.
type StreamingConfigs struct {
	Brokers  []string
	Topics   []string
	GroupID  string
	MinBytes int
	MaxBytes int
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg StreamingConfigs) *KafkaConsumer {
	return &KafkaConsumer{
		config: cfg,
		reader: make([]*kafka.Reader, 0),
	}
}

// PollMessage begins consuming messages, calling handler for each message
func (c *KafkaConsumer) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		for _, reader := range c.reader {
			kafkaMsg, err := reader.ReadMessage(ctx)
			if err != nil {
				return nil, err
			}

			var deviceMessage ingestor.DeviceMessage
			err = json.Unmarshal(kafkaMsg.Value, &deviceMessage)
			if err != nil {
				return nil, err
			}

			return &deviceMessage, nil
		}
	}

	return nil, nil
}

// Close gracefully shuts down all reader
func (c *KafkaConsumer) Close(context.Context) error {
	for _, reader := range c.reader {
		if err := reader.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Subscribe registers topics to consume from
func (c *KafkaConsumer) Subscribe(_ context.Context, topics string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.config.Brokers,
		GroupID:        c.config.GroupID,
		Topic:          topics,
		MinBytes:       c.config.MinBytes,
		MaxBytes:       c.config.MaxBytes,
		CommitInterval: time.Second, // tune
	})

	c.reader = append(c.reader, reader)

	return nil
}

// convertHeaders converts Kafka headers to generic map
func (c *KafkaConsumer) convertHeaders(kafkaHeaders []kafka.Header) map[string]string {
	headers := make(map[string]string)
	for _, h := range kafkaHeaders {
		headers[h.Key] = string(h.Value)
	}

	return headers
}
