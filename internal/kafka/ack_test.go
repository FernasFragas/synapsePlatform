package kafka

import (
	"context"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

type fakeMessageCommitter struct {
	commits []kafka.Message
}

func (f *fakeMessageCommitter) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	f.commits = append(f.commits, msgs...)
	return nil
}

func TestOffsetCommitter_OutOfOrderAckDoesNotCommitOverGap(t *testing.T) {
	reader := &fakeMessageCommitter{}
	committer := NewOffsetCommitterForTest(reader)

	offset0 := kafka.Message{Topic: "topic", Partition: 0, Offset: 0}
	offset1 := kafka.Message{Topic: "topic", Partition: 0, Offset: 1}

	ack0 := committer.Ack(offset0)
	ack1 := committer.Ack(offset1)

	require.NoError(t, ack1(context.Background()))
	require.Empty(t, reader.commits, "offset 1 must not commit while offset 0 is still incomplete")

	require.NoError(t, ack0(context.Background()))
	require.Len(t, reader.commits, 1)
	require.Equal(t, int64(1), reader.commits[0].Offset, "commit should advance to the contiguous high watermark")
}

func TestOffsetCommitter_ContiguousAckCommitsEachWatermark(t *testing.T) {
	reader := &fakeMessageCommitter{}
	committer := NewOffsetCommitterForTest(reader)

	offset0 := kafka.Message{Topic: "topic", Partition: 0, Offset: 0}
	offset1 := kafka.Message{Topic: "topic", Partition: 0, Offset: 1}

	ack0 := committer.Ack(offset0)
	ack1 := committer.Ack(offset1)

	require.NoError(t, ack0(context.Background()))
	require.NoError(t, ack1(context.Background()))

	require.Len(t, reader.commits, 2)
	require.Equal(t, int64(0), reader.commits[0].Offset)
	require.Equal(t, int64(1), reader.commits[1].Offset)
}

func TestOffsetCommitter_DuplicateAckDoesNotRegressOrPanic(t *testing.T) {
	reader := &fakeMessageCommitter{}
	committer := NewOffsetCommitterForTest(reader)

	offset0 := kafka.Message{Topic: "topic", Partition: 0, Offset: 0}
	ack0 := committer.Ack(offset0)

	require.NoError(t, ack0(context.Background()))
	require.NoError(t, ack0(context.Background()))

	require.Len(t, reader.commits, 1)
	require.Equal(t, int64(0), reader.commits[0].Offset)
}

func NewOffsetCommitterForTest(reader messageCommitter) *OffsetCommitter {
	return &OffsetCommitter{
		reader: reader,
		parts:  make(map[int]*partitionProgress),
	}
}
