//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
)

// MessagePoller is the port interface for consuming messages
// Any message broker (Kafka, RabbitMQ, NATS) must implement this.
type MessagePoller interface {
	// Subscribe registers topics/queues to consume from
	Subscribe(topics string) error

	// PollMessage begins consuming messages, calling handler for each
	PollMessage(ctx context.Context) (*DeviceMessage, error)

	// Close gracefully shuts down the consumer
	Close() error
}

type Processor struct {
	poller MessagePoller
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
