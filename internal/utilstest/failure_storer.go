package utilstest

import (
	"synapsePlatform/internal/ingestor"
	mock_ingestor "synapsePlatform/internal/utilstest/mocksgen/ingestor"
	"testing"

	"go.uber.org/mock/gomock"
)

type FailureStorer struct {
	*mock_ingestor.MockFailureStorer

	t *testing.T
}

func NewFailureStorer(t *testing.T) *FailureStorer {
	return &FailureStorer{
		MockFailureStorer: mock_ingestor.NewMockFailureStorer(gomock.NewController(t)),
		t:                 t,
	}
}

func (f *FailureStorer) WithSuccess() *FailureStorer {
	f.MockFailureStorer.EXPECT().
		StoreFailure(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	return f
}

func (f *FailureStorer) WithError(err error) *FailureStorer {
	f.MockFailureStorer.EXPECT().
		StoreFailure(gomock.Any(), gomock.Any()).
		Return(err)

	return f
}

func (f *FailureStorer) ExpectStage(stage string) *FailureStorer {
	var s bool

	f.MockFailureStorer.EXPECT().
		StoreFailure(gomock.Any(), gomock.Cond(func(x any) bool {
			fm, ok := x.(ingestor.FailedMessage)

			s = ok && fm.Stage == stage

			return s
		})).
		Return(s)

	return f
}

func (f *FailureStorer) ExpectFailure(stage, errorType string) *FailureStorer {
	f.MockFailureStorer.EXPECT().
		StoreFailure(gomock.Any(), gomock.Cond(func(x any) bool {
			fm, ok := x.(ingestor.FailedMessage)
			return ok && fm.Stage == stage && fm.ErrorType == errorType
		})).
		Return(nil)
	return f
}

func (f *FailureStorer) ExpectFailureStoreError(err error) *FailureStorer {
	f.MockFailureStorer.EXPECT().
		StoreFailure(gomock.Any(), gomock.Any()).
		Return(err)
	return f
}

func (f *FailureStorer) ExpectNoFailure() *FailureStorer {
	f.MockFailureStorer.EXPECT().
		StoreFailure(gomock.Any(), gomock.Any()).
		Times(0)
	return f
}
