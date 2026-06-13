//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
)

// MessagePoller is the port interface for consuming messages
// Any message broker (Kafka, RabbitMQ, NATS) must implement this.
type MessagePoller interface {
	PollMessage(ctx context.Context) (*Delivery, error)

	Close(ctx context.Context) error
}

// AckHandler marks a delivery as durably handled by the inbound adapter.
// For Kafka this commits an offset; other adapters can map it to their own
// acknowledgment mechanism.
type AckHandler func(ctx context.Context) error

type Processor struct {
	poller MessagePoller
}

type MessageMetadata struct {
	Source  string
	Headers map[string]string
	Labels  map[string]string
}

type Delivery struct {
	Message  *DeviceMessage
	Metadata MessageMetadata
	Ack      AckHandler
}

func NewProcessor(poller MessagePoller) *Processor {
	return &Processor{
		poller: poller,
	}
}

func (p *Processor) ProcessData(ctx context.Context) (*Delivery, error) {
	delivery, err := p.poller.PollMessage(ctx)
	if err != nil {
		return nil, ProcessorError{
			TypeOfError:            ErrPollingMsg,
			ErrorOccurredBecauseOf: ErrFailedToPollMsg,
			Field:                  "delivery",
			Expected:               "Delivery",
			Got:                    delivery,
			Err:                    err,
		}
	}

	if delivery == nil || delivery.Message == nil {
		return delivery, ProcessorError{
			TypeOfError:            ErrProcessingMsg,
			ErrorOccurredBecauseOf: ErrFailedToProcessMsg,
			Field:                  "delivery.message",
			Expected:               "DeviceMessage",
			Got:                    delivery,
			Err:                    ErrNilMessage,
		}
	}

	if err := delivery.Message.ValidateRawMessage(); err != nil {
		return delivery, ProcessorError{
			TypeOfError:            ErrValidatingMsg,
			ErrorOccurredBecauseOf: ErrFailedToValidateMsg,
			Field:                  "delivery.message",
			Expected:               "valid DeviceMessage",
			Got:                    delivery.Message,
			Err:                    err,
		}
	}

	return delivery, nil
}
