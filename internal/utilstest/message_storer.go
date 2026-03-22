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

// WithBatchSuccess sets the mock to return success for StoreBatch.
func (r *MessageStorer) WithBatchSuccess() *MessageStorer {
	r.MockMessageStorer.EXPECT().StoreBatch(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	return r
}

// WithBatchError sets the mock to return an error for StoreBatch.
func (r *MessageStorer) WithBatchError(err error) *MessageStorer {
	r.MockMessageStorer.EXPECT().StoreBatch(gomock.Any(), gomock.Any()).Return(err).AnyTimes()
	return r
}

// WithBatchSuccessOnce sets the mock to return success for StoreBatch once.
func (r *MessageStorer) WithBatchSuccessOnce() *MessageStorer {
	r.MockMessageStorer.EXPECT().StoreBatch(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	return r
}
