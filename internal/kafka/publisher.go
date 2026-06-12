//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/kafka/mocked_$GOFILE
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"synapsePlatform/internal/ingestor"
	"time"

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
		Error:   failed.ErrorMessage,
	})
	if err != nil {
		return err
	}

	return k.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(failed.Stage),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "X-Original-Topic", Value: []byte(failed.OriginalTopic)},
			{Key: "X-Original-Partition", Value: []byte(fmt.Sprintf("%d", failed.Partition))},
			{Key: "X-Original-Offset", Value: []byte(fmt.Sprintf("%d", failed.Offset))},
			{Key: "X-Error-Type", Value: []byte(failed.ErrorType)},
			{Key: "X-Stage", Value: []byte(failed.Stage)},
			{Key: "X-Retry-Count", Value: []byte(fmt.Sprintf("%d", failed.RetryCount))},
			{Key: "X-Failed-At", Value: []byte(failed.Timestamp.Format(time.RFC3339))},
		},
	})
}

func (k *KafkaDLQ) Close() error {
	return k.writer.Close()
}
