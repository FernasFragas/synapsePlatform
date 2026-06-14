//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/kafka/mocked_$GOFILE
package kafka

import (
	"context"
	"encoding/json"
	"strconv"
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
		Stage    string                   `json:"stage"`
		Message  *ingestor.DeviceMessage  `json:"message,omitempty"`
		Error    string                   `json:"error,omitempty"`
		Type     string                   `json:"type,omitempty"`
		Metadata ingestor.MessageMetadata `json:"metadata"`
		FailedAt time.Time                `json:"failed_at"`
	}{
		Stage:    failed.Stage,
		Message:  failed.Message,
		Error:    failed.ErrorMessage,
		Type:     failed.ErrorType,
		Metadata: failed.Metadata,
		FailedAt: failed.FailedAt,
	})
	if err != nil {
		return err
	}

	headers := []kafka.Header{
		{Key: "X-Error-Type", Value: []byte(failed.ErrorType)},
		{Key: "X-Stage", Value: []byte(failed.Stage)},
		{Key: "X-Retry-Count", Value: []byte(strconv.Itoa(failed.RetryCount))},
		{Key: "X-Failed-At", Value: []byte(failed.FailedAt.Format(time.RFC3339))},
	}

	for key, value := range failed.Metadata.Labels {
		headers = append(headers, kafka.Header{
			Key:   "X-Source-" + key,
			Value: []byte(value),
		})
	}

	return k.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(failed.Stage),
		Value:   payload,
		Headers: headers,
	})
}

func (k *KafkaDLQ) Close() error {
	return k.writer.Close()
}
