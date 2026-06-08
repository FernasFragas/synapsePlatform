package utilstest

import (
	"synapsePlatform/internal/api"
	"testing"

	mock_api "synapsePlatform/internal/utilstest/mocksgen/api"

	"go.uber.org/mock/gomock"
)

type Summarizer struct {
	*mock_api.MockSummarizer
	t *testing.T
}

func NewSummarizer(t *testing.T) *Summarizer {
	return &Summarizer{
		MockSummarizer: mock_api.NewMockSummarizer(gomock.NewController(t)),
		t:              t,
	}
}

func (sm *Summarizer) WithReport(report *api.Report) *Summarizer {
	sm.EXPECT().Summarize(gomock.Any(), gomock.Any()).Return(report, nil)
	return sm
}

func (sm *Summarizer) WithError(err error) *Summarizer {
	sm.EXPECT().Summarize(gomock.Any(), gomock.Any()).Return(nil, err)
	return sm
}
