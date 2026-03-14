package ingestor

type Domain string

type TypeOfDevice string

const (
	EntityTypeDevice   = "device"
	EntityTypeSensor   = "sensor"
	EntityTypeActuator = "actuator"
	EntityTypeGateway  = "gateway"
	EntityTypeOther    = "other"
	EntityTypeUnknown  = "unknown"
)

type SchemaVersion string

const (
	SchemaVersion1 = "1.0.0" // currently only one
)

// DataTypes represents different types of device data.
type DataTypes string

// DataTypes enum values.
const (
	DataTypeEnvironmentalSensor DataTypes = "temperature_sensor"
	DataTypeClimateSensor       DataTypes = "climate_sensor"
	DataTypeEVCharger           DataTypes = "ev_charger"
	DataTypeGridMeter           DataTypes = "grid_meter"
	DataTypeMotionSensor        DataTypes = "motion_sensor"
	DataTypeFinancialStream     DataTypes = "financial_stream"
	DataTypeEnergyMeter         DataTypes = "energy_meter"
	DataTypeSolarPanel          DataTypes = "solar_panel"
	DataTypeBattery             DataTypes = "battery"
	DataTypeUnknown             DataTypes = "unknown"
)

// AllDataTypes returns all valid data types.
func AllDataTypes() []DataTypes {
	return []DataTypes{
		DataTypeEnvironmentalSensor,
		DataTypeClimateSensor,
		DataTypeEVCharger,
		DataTypeGridMeter,
		DataTypeMotionSensor,
		DataTypeFinancialStream,
		DataTypeEnergyMeter,
		DataTypeSolarPanel,
		DataTypeBattery,
	}
}

// IsValid checks if the DataType is valid.
func (d DataTypes) IsValid() bool {
	switch d { //nolint:exhaustive
	case DataTypeEnvironmentalSensor,
		DataTypeClimateSensor,
		DataTypeEVCharger,
		DataTypeGridMeter,
		DataTypeMotionSensor,
		DataTypeFinancialStream,
		DataTypeEnergyMeter,
		DataTypeSolarPanel,
		DataTypeBattery:
		return true
	default:
		return false
	}
}

// String returns the string representation.
func (d DataTypes) String() string {
	return string(d)
}

// ParseDataType converts a string to DataTypes.
func ParseDataType(s string) DataTypes {
	dt := DataTypes(s)
	if dt.IsValid() {
		return dt
	}

	return DataTypeUnknown
}

type PageRequest struct {
	Cursor string
	Limit  int
}
type PageResponse[T any] struct {
	Items      []T
	NextCursor string
	HasMore    bool
}