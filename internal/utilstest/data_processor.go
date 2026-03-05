package utilstest

import (
	"context"
	"errors"
	"synapsePlatform/internal/ingestor"
	mock_ingestor "synapsePlatform/internal/utilstest/mocksgen/ingestor"
	"testing"

	"go.uber.org/mock/gomock"
)

type DataProcessor struct {
	*mock_ingestor.MockDataProcessor

	t *testing.T
}

func NewDataProcessor(t *testing.T) *DataProcessor {
	return &DataProcessor{
		MockDataProcessor: mock_ingestor.NewMockDataProcessor(gomock.NewController(t)),
		t:                 t,
	}
}

// WithError sets the mock to return an error.
func (r *DataProcessor) WithError(err error) *DataProcessor {
	r.MockDataProcessor.EXPECT().ProcessData(gomock.Any()).Return(nil, err)

	return r
}

func (r *DataProcessor) WithCancel(cancel context.CancelFunc) *DataProcessor {
	r.MockDataProcessor.EXPECT().
		ProcessData(gomock.Any()).
		DoAndReturn(func(_ context.Context) (*ingestor.DeviceMessage, error) {
			cancel()

			return nil, errors.New("cancelled") //nolint:err113
	})

	return r
}

// WithResult sets the mock to return the given messages.
func (r *DataProcessor) WithResult(messages *ingestor.DeviceMessage) *DataProcessor {
	r.MockDataProcessor.EXPECT().ProcessData(gomock.Any()).Return(messages, nil)

	return r
}
