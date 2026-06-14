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
	dataTypesSupported []DataTypes
	domainsSupported   []Domains
}

// NewMessageTransformer creates a transformer.
func NewMessageTransformer(dataTypes []DataTypes) *MessageTransformer {
	// Derive the unique set of domains from the supported data types
	domainSet := make(map[Domains]struct{})
	for _, dt := range dataTypes {
		if domain, ok := typeToDomain[dt]; ok {
			domainSet[domain] = struct{}{}
		}
	}
	domains := make([]Domains, 0, len(domainSet))
	for d := range domainSet {
		domains = append(domains, d)
	}

	return &MessageTransformer{
		dataTypesSupported: dataTypes,
		domainsSupported:   domains,
	}
}

// Transform converts a device message to a domain event.
func (t *MessageTransformer) Transform(ctx context.Context, msg *DeviceMessage) (*BaseEvent, error) {
	domain := ParseDomainType(msg.Type)

	if !t.isDomainSupported(domain) {
		return nil, ProcessorError{
			TypeOfError:            ErrValidatingData,
			ErrorOccurredBecauseOf: ErrFailedToValidateData,
			Field:                  "domain",
			Expected:               "DataTypes",
			Got:                    domain,
			Err:                    fmt.Errorf("%w: unsupported domain %s (supported: %v)", ErrUnknownDataType, domain, t.dataTypesSupported),
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

	businessKey := fmt.Sprintf("%s|%s|%s",
		msg.DeviceID, msg.Type, msg.Timestamp.Format(time.RFC3339Nano))
	eventID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(businessKey))

	return &BaseEvent{
		EventID:       eventID,
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
func (t *MessageTransformer) isDomainSupported(domain Domains) bool {
	if len(t.dataTypesSupported) == 0 {
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
