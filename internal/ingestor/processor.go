//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
	"fmt"
)

// MessagePoller is the port interface for consuming messages
// Any message broker (Kafka, RabbitMQ, NATS) must implement this.
type MessagePoller interface {

	// PollMessage returns the device message and a receipt handle for acknowledgment
	PollMessage(ctx context.Context) (*DeviceMessage, string, error)

	// AckMessageSuccess acknowledges a message using its receipt handle
	AckMessageSuccess(ctx context.Context, receiptHandle string) error

	Close(ctx context.Context) error
}

type Processor struct {
	poller            MessagePoller
	lastReceiptHandle string
}

func NewProcessor(poller MessagePoller) *Processor {
	return &Processor{
		poller: poller,
	}
}

func (p *Processor) ProcessData(ctx context.Context) (*DeviceMessage, error) {
	msg, receiptHandle, err := p.poller.PollMessage(ctx)
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

	p.lastReceiptHandle = receiptHandle

	return msg, nil
}

func (p *Processor) AckDataSuccess(ctx context.Context) error {
	if p.lastReceiptHandle == "" {
		return fmt.Errorf("no message to acknowledge")
	}
	return p.poller.AckMessageSuccess(ctx, p.lastReceiptHandle)
}
