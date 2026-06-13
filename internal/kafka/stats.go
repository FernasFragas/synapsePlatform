package kafka

import (
	"context"
	"fmt"

	"synapsePlatform/internal/ingestor"

	"github.com/segmentio/kafka-go"
)

type kafkaStatsReader interface {
	Stats() kafka.ReaderStats
}

type StatsReader struct {
	reader kafkaStatsReader
}

func NewStatsReader(consumer *KafkaConsumer) *StatsReader {
	return &StatsReader{reader: consumer.reader}
}

func NewStatsReaderFromReader(reader kafkaStatsReader) *StatsReader {
	return &StatsReader{reader: reader}
}

func (s *StatsReader) ReadStats(ctx context.Context) (ingestor.ConsumerStats, error) {
	if err := ctx.Err(); err != nil {
		return ingestor.ConsumerStats{}, err
	}

	if s.reader == nil {
		return ingestor.ConsumerStats{}, fmt.Errorf("kafka stats reader is nil")
	}

	stats := s.reader.Stats()

	return ingestor.ConsumerStats{
		Lag:        stats.Lag,
		MinBytes:   int64(stats.MinBytes),
		MaxBytes:   int64(stats.MaxBytes),
		Messages:   stats.Messages,
		Bytes:      stats.Bytes,
		Rebalances: stats.Rebalances,
		Errors:     stats.Errors,
	}, nil
}
