//go:generate mockgen -source=$GOFILE -destination=../internal/utilstest/mocksgen/mocked_$GOFILE
package kafka

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

// StreamingConfigs holds configuration for message broker connections.
type StreamingConfigs struct {
	Brokers  []string
	Topics   []string
	GroupID  string
	MinBytes int
	MaxBytes int
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config StreamingConfigs, topic string) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		GroupID:        config.GroupID,
		Topic:          topic,
		MinBytes:       config.MinBytes,
		MaxBytes:       config.MaxBytes,
		CommitInterval: time.Second, // tune
	})

	return &KafkaConsumer{
		config: config,
		reader: reader,
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
			return nil, err
		}

		var deviceMessage ingestor.DeviceMessage
		err = json.Unmarshal(kafkaMsg.Value, &deviceMessage)
		if err != nil {
			return nil, err
		}

		return &deviceMessage, nil
	}

	return nil, nil
}

// Close gracefully shuts down all reader
func (c *KafkaConsumer) Close(context.Context) error {
	if err := c.reader.Close(); err != nil {
		return err
	}

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
