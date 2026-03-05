package ingestor

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

//nolint:gochecknoglobals
var validate = validator.New(validator.WithRequiredStructEnabled())

type BaseEventValue = NormalizedData
type BaseEvent struct {
	EventID   uuid.UUID `json:"event_id" validate:"required"`
	Domain    string    `json:"domain" validate:"required"`
	EventType string    `json:"event_type" validate:"required"`

	EntityID   string `json:"entity_id" validate:"required,min=1,max=255"`
	EntityType string `json:"entity_type" validate:"required"`

	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	IngestedAt time.Time `json:"ingested_at" validate:"required"`

	Source string `json:"source" validate:"required"`

	SchemaVersion string `json:"schema_version" validate:"required,semver"`

	Data BaseEventValue `json:"data" validate:"required"`
}

type EnergyReading struct {
	PowerW    int64 `json:"power_w" validate:"required,gte=0"`
	EnergyWh  int64 `json:"energy_wh" validate:"required,gte=0"`
	VoltageV  int32 `json:"voltage_v" validate:"required,gte=0,lte=500"`
	CurrentMA int32 `json:"current_ma" validate:"required,gte=0"`
}

type FinancialTransaction struct {
	AmountMinor int64  `json:"amount_minor" validate:"required"`
	Currency    string `json:"currency" validate:"required"`
	Merchant    string `json:"merchant" validate:"required"`
	Status      string `json:"status" validate:"required,oneof=completed failed pending"`
}

type EnvironmentalSensor struct {
	TemperatureC    int64 `json:"temperature_c" validate:"required,gte=-50,lte=100"`
	HumidityPercent int64 `json:"humidity_percent" validate:"required,gte=0,lte=100"`
	AirQualityIndex int64 `json:"air_quality_index" validate:"required,gte=0,lte=500"`
}

type UnknownEvent struct {
	Data map[string]any `json:"data"`
}

func (e *BaseEvent) Validate() error {
	err := validate.Struct(e)
	if err != nil {
		return ProcessorError{
			TypeOfError:            ErrValidatingData,
			ErrorOccurredBecauseOf: ErrFailedToValidateData,
			Field:                  "e.Data",
			Expected:               "NormalizedData",
			Got:                    e.Data,
			Err:                    err,
		}
	}

	return e.Data.Validate()
}

func (e *BaseEvent) Normalize() error {
	// Standardize domain (lowercase)
	e.Domain = strings.ToLower(strings.TrimSpace(e.Domain))

	// Standardize event type (lowercase, underscores)
	e.EventType = strings.ToLower(strings.ReplaceAll(
		strings.TrimSpace(e.EventType), " ", "_",
	))

	// Standardize entity type (lowercase)
	e.EntityType = strings.ToLower(strings.TrimSpace(e.EntityType))

	// Ensure UTC timezone
	if e.OccurredAt.Location() != time.UTC {
		e.OccurredAt = e.OccurredAt.UTC()
	}

	if e.IngestedAt.Location() != time.UTC {
		e.IngestedAt = e.IngestedAt.UTC()
	}

	// Set defaults if missing
	if e.Source == "" {
		e.Source = "unknown"
	}

	if e.SchemaVersion == "" {
		e.SchemaVersion = "1.0"
	}

	return e.Data.Normalize()
}

// In internal/ingestor/normalized_message.go

func (e *BaseEvent) LogValue() slog.Value {
	if e == nil {
		return slog.StringValue("<nil>")
	}

	return slog.GroupValue(
		slog.String("event_id", e.EventID.String()),
		slog.String("domain", e.Domain),
		slog.String("event_type", e.EventType),
		slog.String("entity_type", e.EntityType),
		slog.String("schema_version", e.SchemaVersion),
	)
}

func (er *EnergyReading) Validate() error {
	err := validate.Struct(er)
	if err != nil {
		return ProcessorError{
			TypeOfError:            ErrValidatingData,
			ErrorOccurredBecauseOf: ErrFailedToValidateData,
			Field:                  "er",
			Expected:               "EnergyReading",
			Got:                    er,
			Err:                    err,
		}
	}

	return nil
}

func (er *EnergyReading) Normalize() error {
	// Convert negative values to zero (sensor errors)
	if er.PowerW < 0 {
		er.PowerW = 0
	}
	if er.EnergyWh < 0 {
		er.EnergyWh = 0
	}
	if er.VoltageV < 0 {
		er.VoltageV = 0
	}
	if er.CurrentMA < 0 {
		er.CurrentMA = 0
	}

	// Calculate missing values if possible
	// Power = Voltage × Current (if one is missing)
	if er.PowerW == 0 && er.VoltageV > 0 && er.CurrentMA > 0 {
		// P = V × I, convert mA to A
		er.PowerW = int64(er.VoltageV) * int64(er.CurrentMA) / 1000
	}

	// Round power to nearest watt
	er.PowerW = (er.PowerW / 10) * 10

	return nil
}

func (ft *FinancialTransaction) Validate() error {
	err := validate.Struct(ft)
	if err != nil {
		return ProcessorError{
			TypeOfError:            ErrValidatingData,
			ErrorOccurredBecauseOf: ErrFailedToValidateData,
			Field:                  "ft",
			Expected:               "FinancialTransaction",
			Got:                    ft,
			Err:                    err,
		}
	}

	return nil
}

func (ft *FinancialTransaction) Normalize() error {
	ft.Currency = strings.ToUpper(strings.TrimSpace(ft.Currency))

	ft.Status = strings.ToLower(strings.TrimSpace(ft.Status))

	ft.Merchant = strings.TrimSpace(ft.Merchant)

	return nil
}

func (ft *FinancialTransaction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("currency", ft.Currency),
		slog.String("status", ft.Status),
		slog.String("amount", strconv.FormatInt(ft.AmountMinor, 10)),
	)
}

func (es *EnvironmentalSensor) Validate() error {
	err := validate.Struct(es)
	if err != nil {
		return ProcessorError{
			TypeOfError:            ErrValidatingData,
			ErrorOccurredBecauseOf: ErrFailedToValidateData,
			Field:                  "e",
			Expected:               "EnvironmentalSensor",
			Got:                    es,
			Err:                    err,
		}
	}

	return nil
}

func (es *EnvironmentalSensor) Normalize() error {
	// Clamp temperature to realistic range
	if es.TemperatureC < -50 {
		es.TemperatureC = -50
	}
	if es.TemperatureC > 100 {
		es.TemperatureC = 100
	}

	// Clamp humidity to 0-100%
	if es.HumidityPercent < 0 {
		es.HumidityPercent = 0
	}
	if es.HumidityPercent > 100 {
		es.HumidityPercent = 100
	}

	// Clamp air quality index to valid range
	if es.AirQualityIndex < 0 {
		es.AirQualityIndex = 0
	}
	if es.AirQualityIndex > 500 {
		es.AirQualityIndex = 500
	}

	return nil
}

func (e *UnknownEvent) Validate() error {
	err := validate.Struct(e)
	if err != nil {
		return ProcessorError{
			TypeOfError:            ErrValidatingData,
			ErrorOccurredBecauseOf: ErrFailedToValidateData,
			Field:                  "e",
			Expected:               "UnknownEvent",
			Got:                    e,
			Err:                    err,
		}
	}

	return nil
}

func (e *UnknownEvent) Normalize() error {
	return nil
}

func unmarshalEvent(msg map[string]any, target *NormalizedData) error {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return ProcessorError{
			TypeOfError:            ErrMarshalingMsg,
			ErrorOccurredBecauseOf: ErrFailedToMarshalMsg,
			Field:                  "msg",
			Expected:               "jsonBytes",
			Got:                    msg,
			Err:                    err,
		}
	}

	err = json.Unmarshal(jsonBytes, target)
	if err != nil {
		return ProcessorError{
			TypeOfError:            ErrUnmarshallingMsg,
			ErrorOccurredBecauseOf: ErrFailedToUnmarshalMsg,
			Field:                  "target",
			Expected:               "NormalizedData",
			Got:                    target,
			Err:                    err,
		}
	}

	return nil
}
