//nolint:depguard
package utilstest

import (
	"testing"

	"synapsePlatform/internal/ingestor"
	mock_ingestor "synapsePlatform/internal/utilstest/mocksgen/ingestor"

	"go.uber.org/mock/gomock"
)

type MessagePoller struct {
	*mock_ingestor.MockMessagePoller

	t *testing.T
}

func NewMessagePoller(t *testing.T) *MessagePoller {
	return &MessagePoller{
		MockMessagePoller: mock_ingestor.NewMockMessagePoller(gomock.NewController(t)),
		t:                 t,
	}
}

// WithError sets the mock to return an error.
func (r *MessagePoller) WithError(err error) *MessagePoller {
	r.MockMessagePoller.EXPECT().PollMessage(gomock.Any()).Return(nil, err)

	return r
}

// WithNoResult sets the mock to return no results.
func (r *MessagePoller) WithNoResult() *MessagePoller {
	r.MockMessagePoller.EXPECT().PollMessage(gomock.Any()).Return(nil, nil)

	return r
}

// WithResult sets the mock to return the given messages.
func (r *MessagePoller) WithResult(messages *ingestor.DeviceMessage) *MessagePoller {
	return r.WithDelivery(&ingestor.Delivery{Message: messages})
}

// WithDeliveryAndAck sets the mock to return the given message with a specific ack handler.
func (r *MessagePoller) WithDeliveryAndAck(messages *ingestor.DeviceMessage, ack ingestor.AckHandler) *MessagePoller {
	return r.WithDelivery(&ingestor.Delivery{Message: messages, Ack: ack})
}

// WithResultAndAck sets the mock to return the given message with a specific ack handler.
func (r *MessagePoller) WithResultAndAck(messages *ingestor.DeviceMessage, ack ingestor.AckHandler) *MessagePoller {
	return r.WithDeliveryAndAck(messages, ack)
}

func (r *MessagePoller) WithDelivery(delivery *ingestor.Delivery) *MessagePoller {
	r.MockMessagePoller.EXPECT().PollMessage(gomock.Any()).Return(delivery, nil)
	return r
}

func (r *MessagePoller) WithTerminalDecodeFailure(delivery *ingestor.Delivery, err error) *MessagePoller {
	r.MockMessagePoller.EXPECT().
		PollMessage(gomock.Any()).
		Return(delivery, ingestor.NewTerminalError(err))
	return r
}

func (r *MessagePoller) WithTransientPollFailure(err error) *MessagePoller {
	r.MockMessagePoller.EXPECT().
		PollMessage(gomock.Any()).
		Return(nil, ingestor.NewTransientError(err))
	return r
}
