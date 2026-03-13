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
	r.MockMessagePoller.EXPECT().PollMessage(gomock.Any()).Return(messages, nil)

	return r
}

// WithSubscriptionSuccessful successfully subscribe.
func (r *MessagePoller) WithSubscriptionSuccessful(topic string) *MessagePoller {
	r.MockMessagePoller.EXPECT().Subscribe(gomock.Any(), topic).Return(nil)

	return r
}

// WithSubscriptionUnSuccessful unsuccessfully tries to subscribing to a topic returning the given error.
func (r *MessagePoller) WithSubscriptionUnSuccessful(topic string, err error) *MessagePoller {
	r.MockMessagePoller.EXPECT().Subscribe(gomock.Any(), topic).Return(err)

	return r
}
