package utilstest

import (
	"context"
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
		DoAndReturn(func(_ context.Context) (*ingestor.Delivery, error) {
			cancel()
			return nil, context.Canceled
		})

	return r
}

// WithResult sets the mock to return the given messages.
func (r *DataProcessor) WithResult(messages *ingestor.DeviceMessage) *DataProcessor {
	return r.WithDelivery(&ingestor.Delivery{Message: messages})
}

// WithDeliveryAndAck returns the given message with a specific ack handler.
func (r *DataProcessor) WithDeliveryAndAck(messages *ingestor.DeviceMessage, ack ingestor.AckHandler) *DataProcessor {
	return r.WithDelivery(&ingestor.Delivery{Message: messages, Ack: ack})
}

// WithResultAndAck returns the given message with a specific ack handler.
func (r *DataProcessor) WithResultAndAck(messages *ingestor.DeviceMessage, ack ingestor.AckHandler) *DataProcessor {
	return r.WithDeliveryAndAck(messages, ack)
}

func (r *DataProcessor) WithDelivery(delivery *ingestor.Delivery) *DataProcessor {
	r.MockDataProcessor.EXPECT().ProcessData(gomock.Any()).Return(delivery, nil)
	return r
}

func (r *DataProcessor) WithTerminalError(delivery *ingestor.Delivery, err error) *DataProcessor {
	r.MockDataProcessor.EXPECT().
		ProcessData(gomock.Any()).
		Return(delivery, ingestor.NewTerminalError(err))
	return r
}

func (r *DataProcessor) WithTransientError(delivery *ingestor.Delivery, err error) *DataProcessor {
	r.MockDataProcessor.EXPECT().
		ProcessData(gomock.Any()).
		Return(delivery, ingestor.NewTransientError(err))
	return r
}
