package utilstest

import (
	"synapsePlatform/internal/ingestor"
	mock_ingestor "synapsePlatform/internal/utilstest/mocksgen/ingestor"
	"testing"

	"go.uber.org/mock/gomock"
)

type Transformer struct {
	*mock_ingestor.MockTransformer

	t *testing.T
}

func NewTransformer(t *testing.T) *Transformer {
	return &Transformer{
		MockTransformer: mock_ingestor.NewMockTransformer(gomock.NewController(t)),
		t:               t,
	}
}

// WithError sets the mock to return an error.
func (r *Transformer) WithError(err error) *Transformer {
	r.MockTransformer.EXPECT().Transform(gomock.Any(), gomock.Any()).Return(nil, err)

	return r
}

// WithNoResults sets the mock to return no results.
func (r *Transformer) WithNoResults() *Transformer {
	r.MockTransformer.EXPECT().Transform(gomock.Any(), gomock.Any()).Return(nil, nil)

	return r
}

// WithResult sets the mock to return the given messages.
func (r *Transformer) WithResult(message *ingestor.BaseEvent) *Transformer {
	r.MockTransformer.EXPECT().Transform(gomock.Any(), gomock.Any()).Return(message, nil)

	return r
}
