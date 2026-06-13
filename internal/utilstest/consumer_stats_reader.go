package utilstest

import (
	"errors"
	"synapsePlatform/internal/ingestor"
	mock_ingestor "synapsePlatform/internal/utilstest/mocksgen/ingestor"
	"testing"

	"go.uber.org/mock/gomock"
)

type ConsumerStatsReader struct {
	*mock_ingestor.MockConsumerStatsReader
	t *testing.T
}

func NewConsumerStatsReader(t *testing.T) *ConsumerStatsReader {
	return &ConsumerStatsReader{
		MockConsumerStatsReader: mock_ingestor.NewMockConsumerStatsReader(gomock.NewController(t)),
		t:                       t,
	}
}

func (r *ConsumerStatsReader) WithStats(stats ingestor.ConsumerStats) *ConsumerStatsReader {
	r.MockConsumerStatsReader.EXPECT().
		ReadStats(gomock.Any()).
		Return(stats, nil)
	return r
}

func (r *ConsumerStatsReader) WithStatsAnyTimes(stats ingestor.ConsumerStats) *ConsumerStatsReader {
	r.MockConsumerStatsReader.EXPECT().
		ReadStats(gomock.Any()).
		Return(stats, nil).
		AnyTimes()
	return r
}

func (r *ConsumerStatsReader) WithError(err error) *ConsumerStatsReader {
	if err == nil {
		err = errors.New("stats read failed")
	}

	r.MockConsumerStatsReader.EXPECT().
		ReadStats(gomock.Any()).
		Return(ingestor.ConsumerStats{}, err)
	return r
}
