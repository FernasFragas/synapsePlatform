//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/kafka/mocked_$GOFILE
package kafka

import (
	"context"
	"encoding/json"
	"synapsePlatform/internal/ingestor"

	"github.com/segmentio/kafka-go"
)

type KafkaDLQ struct {
	writer *kafka.Writer
}

func NewKafkaDLQ(brokers []string, topic string) *KafkaDLQ {
	return &KafkaDLQ{
		writer: &kafka.Writer{
			Addr:  kafka.TCP(brokers...),
			Topic: topic,
		},
	}
}

func (k *KafkaDLQ) StoreFailure(ctx context.Context, failed ingestor.FailedMessage) error {
	payload, err := json.Marshal(struct {
		Stage   string                  `json:"stage"`
		Message *ingestor.DeviceMessage `json:"message,omitempty"`
		Error   string                  `json:"error,omitempty"`
	}{
		Stage:   failed.Stage,
		Message: failed.Message,
		Error:   failed.Err.Error(),
	})
	if err != nil {
		return err
	}

	return k.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(failed.Stage),
		Value: payload,
	})
}

func (k *KafkaDLQ) Close() error {
	return k.writer.Close()
}
