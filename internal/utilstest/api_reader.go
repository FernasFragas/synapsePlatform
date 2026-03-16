package utilstest

import (
	"testing"

	"synapsePlatform/internal/ingestor"
	mock_api "synapsePlatform/internal/utilstest/mocksgen/api"

	"go.uber.org/mock/gomock"
)

type EventReader struct {
	*mock_api.MockEventReader
	t *testing.T
}

func NewEventReader(t *testing.T) *EventReader {
	return &EventReader{
		MockEventReader: mock_api.NewMockEventReader(gomock.NewController(t)),
		t:               t,
	}
}

func (er *EventReader) WithEvents(events []*ingestor.BaseEvent) *EventReader {
	er.EXPECT().ListEvents(gomock.Any(), gomock.Any()).Return(&ingestor.PageResponse[*ingestor.BaseEvent]{
		Items: events,
	}, nil)

	return er
}

func (er *EventReader) WithEvent(event *ingestor.BaseEvent) *EventReader {
	er.EXPECT().GetEvent(gomock.Any(), gomock.Any()).Return(event, nil)

	return er
}

func (er *EventReader) WithListError(err error) *EventReader {
	er.EXPECT().ListEvents(gomock.Any(), gomock.Any()).Return(nil, err)

	return er
}

func (er *EventReader) WithGetError(err error) *EventReader {
	er.EXPECT().GetEvent(gomock.Any(), gomock.Any()).Return(nil, err)

	return er
}
