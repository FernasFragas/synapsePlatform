//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
)

// MessagePoller is the port interface for consuming messages
// Any message broker (Kafka, RabbitMQ, NATS) must implement this.
type MessagePoller interface {
	// PollMessage returns the device message and an AckHandler for committing
	// the offset. The caller MUST call ack(ctx) after successful store.
	PollMessage(ctx context.Context) (*DeviceMessage, AckHandler, error)

	Close(ctx context.Context) error
}

// AckHandler is a function that commits the Kafka offset for a consumed message.
// Analogous to PostHog's producer::AckFuture.
type AckHandler func(ctx context.Context) error

type Processor struct {
	poller MessagePoller
}

func NewProcessor(poller MessagePoller) *Processor {
	return &Processor{
		poller: poller,
	}
}

func (p *Processor) ProcessData(ctx context.Context) (*DeviceMessage, AckHandler, error) {
	msg, ack, err := p.poller.PollMessage(ctx)
	if err != nil {
		return nil, nil, ProcessorError{
			TypeOfError:            ErrPollingMsg,
			ErrorOccurredBecauseOf: ErrFailedToPollMsg,
			Field:                  "msg",
			Expected:               "DeviceMessage",
			Got:                    msg,
			Err:                    err,
		}
	}

	if msg == nil {
		return nil, ack, ProcessorError{
			TypeOfError:            ErrProcessingMsg,
			ErrorOccurredBecauseOf: ErrFailedToProcessMsg,
			Field:                  "msg",
			Expected:               "DeviceMessage",
			Got:                    msg,
			Err:                    ErrNilMessage,
		}
	}

	err = msg.ValidateRawMessage()
	if err != nil {
		return nil, ack, ProcessorError{
			TypeOfError:            ErrValidatingMsg,
			ErrorOccurredBecauseOf: ErrFailedToValidateMsg,
			Field:                  "msg",
			Expected:               "DeviceMessage",
			Got:                    msg,
			Err:                    err,
		}
	}

	return msg, ack, nil
}

func CommitOffset(ctx context.Context, ack AckHandler, stage string) error {
	if ack == nil {
		return nil
	}
	return ack(ctx)
}
