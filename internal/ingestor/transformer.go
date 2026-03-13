//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/ingestor/mocked_$GOFILE
package ingestor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageTransformer converts DeviceMessage to BaseEvent.
type MessageTransformer struct {
	domainsSupported []DataTypes
}

// NewMessageTransformer creates a transformer.
func NewMessageTransformer(domains []DataTypes) *MessageTransformer {
	return &MessageTransformer{
		domainsSupported: domains,
	}
}

// Transform converts a device message to a domain event.
func (t *MessageTransformer) Transform(ctx context.Context, msg *DeviceMessage) (*BaseEvent, error) {
	domain := ParseDataType(msg.Type)

	if !t.isDomainSupported(domain) {
		return nil, ProcessorError{ // todo fix fields and
			TypeOfError:            ErrValidatingData,
			ErrorOccurredBecauseOf: ErrFailedToValidateData,
			Field:                  "domain",
			Expected:               "DataTypes",
			Got:                    domain,
			Err:                    fmt.Errorf("unsupported domain: %s (supported: %v)", domain, t.domainsSupported),
		}
	}

	mapping, err := t.filterPerDomain(msg)
	if err != nil {
		return nil, err
	}

	if err = unmarshalEvent(msg.Metrics, &mapping.payload); err != nil {
		return nil, err
	}

	err = mapping.payload.Normalize()
	if err != nil {
		return nil, err
	}

	err = mapping.payload.Validate()
	if err != nil {
		return nil, err
	}

	return &BaseEvent{
		EventID:       uuid.New(),
		Domain:        domain.String(),
		EventType:     msg.Type,
		EntityID:      msg.DeviceID,
		EntityType:    mapping.entityType,
		OccurredAt:    msg.Timestamp,
		IngestedAt:    time.Now().UTC(),
		Source:        mapping.source,
		SchemaVersion: SchemaVersion1,
		Data:          mapping.payload,
	}, nil
}

// isDomainSupported checks if the domain is in the supported list.
func (t *MessageTransformer) isDomainSupported(domain DataTypes) bool {
	if len(t.domainsSupported) == 0 {
		return true
	}

	for _, supported := range t.domainsSupported {
		if domain == supported {
			return true
		}
	}

	return false
}

type domainMapping struct {
	payload    NormalizedData
	source     string
	entityType string
}

func (t *MessageTransformer) filterPerDomain(msg *DeviceMessage) (domainMapping, error) {
	domain := ParseDataType(msg.Type)

	desc, ok := LookupDomain(domain)
	if !ok {
		return domainMapping{}, ErrUnknownDataType
	}

	return domainMapping{
		payload:    desc.NewPayload(),
		source:     desc.Source,
		entityType: desc.EntityType,
	}, nil
}
