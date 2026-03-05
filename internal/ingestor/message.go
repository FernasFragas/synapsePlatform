package ingestor

import (
	"log/slog"
	"time"
)

type DeviceMessage struct {
	DeviceID  string         `json:"device_id"`
	Type      string         `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Metrics   map[string]any `json:"metrics"`
}

func (m *DeviceMessage) ValidateRawMessage() error {
	if m.DeviceID == "" {
		return ErrMissingFieldDeviceID
	}

	if m.Type == "" {
		return ErrMissingFieldType
	}

	if m.Timestamp.IsZero() {
		return ErrMissingFieldTimestamp
	}

	return nil
}

func (m *DeviceMessage) LogValue() slog.Value {
	if m == nil {
		return slog.StringValue("<nil>")
	}

	return slog.GroupValue(
		slog.String("device_id", m.DeviceID),
		slog.String("type", m.Type),
		slog.Time("timestamp", m.Timestamp),
	)
}