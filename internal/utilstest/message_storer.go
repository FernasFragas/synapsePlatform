package utilstest

import (
	"testing"

	mock_ingestor "synapsePlatform/internal/utilstest/mocksgen/ingestor"

	"go.uber.org/mock/gomock"
)

type MessageStorer struct {
	*mock_ingestor.MockMessageStorer

	t *testing.T
}

func NewMessageStorerMock(t *testing.T) *MessageStorer {
	return &MessageStorer{
		MockMessageStorer: mock_ingestor.NewMockMessageStorer(gomock.NewController(t)),
		t:                 t,
	}
}

// WithSuccess sets the mock to return the given messages.
func (r *MessageStorer) WithSuccess() *MessageStorer {
	r.MockMessageStorer.EXPECT().StoreData(gomock.Any(), gomock.Any()).Return(nil)

	return r
}

func (r *MessageStorer) WithError(err error) *MessageStorer {
	r.MockMessageStorer.EXPECT().StoreData(gomock.Any(), gomock.Any()).Return(err)

	return r
}
