//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
)

// MessagePoller is the port interface for consuming messages
// Any message broker (Kafka, RabbitMQ, NATS) must implement this.
type MessagePoller interface {

	// PollMessage returns the device message and a receipt handle for acknowledgment
	PollMessage(ctx context.Context) (*DeviceMessage, error)

	Close(ctx context.Context) error
}

type Processor struct {
	poller            MessagePoller
}

func NewProcessor(poller MessagePoller) *Processor {
	return &Processor{
		poller: poller,
	}
}

func (p *Processor) ProcessData(ctx context.Context) (*DeviceMessage, error) {
	msg, err := p.poller.PollMessage(ctx)
	if err != nil {
		return nil, ProcessorError{
			TypeOfError:            ErrPollingMsg,
			ErrorOccurredBecauseOf: ErrFailedToPollMsg,
			Field:                  "msg",
			Expected:               "DeviceMessage",
			Got:                    msg,
			Err:                    err,
		}
	}

	if msg == nil {
		return nil, ProcessorError{
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
		return nil, ProcessorError{
			TypeOfError:            ErrValidatingMsg,
			ErrorOccurredBecauseOf: ErrFailedToValidateMsg,
			Field:                  "msg",
			Expected:               "DeviceMessage",
			Got:                    msg,
			Err:                    err,
		}
	}

	return msg, nil
}
