package log

const (
	// Kafka provenance — where a message came from
	AttrTopic     = "topic"
	AttrPartition = "partition"
	AttrOffset    = "offset"
	// Domain identity
	AttrDeviceID  = "device_id"
	AttrEventID   = "event_id"
	AttrEventType = "event_type"
	AttrDomain    = "domain"
	AttrEntityID  = "entity_id"
	// Auth identity
	AttrSubject  = "subject"
	AttrClientID = "client_id"
)
