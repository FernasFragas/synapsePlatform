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
	reader *kafka.Reader
	config StreamingConfigs
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg StreamingConfigs) *KafkaConsumer {
	return &KafkaConsumer{
		config: cfg,
	}
}

// PollMessage begins consuming messages, calling handler for each message
func (c *KafkaConsumer) PollMessage(ctx context.Context) (*ingestor.DeviceMessage, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		kafkaMsg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			// todo
		}

		var deviceMessage ingestor.DeviceMessage
		if err := json.Unmarshal(kafkaMsg.Value, &deviceMessage); err != nil {
			// Todo
		}

		return &deviceMessage, nil
	}
}

// Close gracefully shuts down all reader
func (c *KafkaConsumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return err
	}

	return nil
}

// Subscribe registers topics to consume from
func (c *KafkaConsumer) Subscribe(topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.config.Brokers,
		GroupID:        c.config.GroupID,
		Topic:          topic,
		MinBytes:       c.config.MinBytes,
		MaxBytes:       c.config.MaxBytes,
		CommitInterval: time.Second, // tune
	})

	c.reader = reader

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
