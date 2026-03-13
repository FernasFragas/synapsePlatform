package ingestor

import "fmt"

type DomainDescriptor struct {
	EntityType string
	Source     string
	NewPayload func() NormalizedData
}

var registry = make(map[DataTypes]DomainDescriptor)

func RegisterDomain(dt DataTypes, desc DomainDescriptor) {
	if _, exists := registry[dt]; exists {
		panic(fmt.Sprintf("domain %q already registered", dt))
	}
	registry[dt] = desc
}

func LookupDomain(dt DataTypes) (DomainDescriptor, bool) {
	d, ok := registry[dt]
	return d, ok
}
