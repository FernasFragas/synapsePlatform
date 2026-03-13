//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/sqllite/mocked_$GOFILE
package sqllite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"

	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/sqlc/generated"

	"github.com/google/uuid"
	_ "modernc.org/sqlite" //nolint:depguard
)

//go:embed summary.sql
var schema string

type Repo struct {
	Db      *sql.DB
	Queries *generated.Queries
}

func NewRepo(dbPath string) (*Repo, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	var database = Repo{
		Db: db,
	}

	if err := database.runMigrations(); err != nil {
		return nil, err
	}

	queries := generated.New(database.Db)

	database.Queries = queries

	return &database, nil
}

func (db *Repo) StoreData(ctx context.Context, data *ingestor.BaseEvent) error {
	dataJSON, err := json.Marshal(data.Data)
	if err != nil {
		return ingestor.ProcessorError{
			TypeOfError:            ingestor.ErrStoringMsg,
			ErrorOccurredBecauseOf: ingestor.ErrFailedToStoreMsg,
			Field:                  "msg",
			Expected:               "DeviceMessage",
			Got:                    dataJSON,
			Err:                    err,
		}
	}

	value := *data

	_, err = db.Queries.CreateEvent(ctx, generated.CreateEventParams{
		EventID:       value.EventID.String(),
		Domain:        value.Domain,
		EventType:     value.EventType,
		EntityID:      value.EntityID,
		EntityType:    value.EntityType,
		OccurredAt:    value.OccurredAt,
		IngestedAt:    value.IngestedAt,
		Source:        value.Source,
		SchemaVersion: value.SchemaVersion,
		Data:          string(dataJSON),
		Metadata:      sql.NullString{},
	})
	if err != nil {
		return ingestor.ProcessorError{
			TypeOfError:            ingestor.ErrStoringMsg,
			ErrorOccurredBecauseOf: ingestor.ErrFailedToStoreMsg,
			Field:                  "msg",
			Expected:               "DeviceMessage",
			Got:                    dataJSON,
			Err:                    err,
		}
	}

	return nil

}

func (db *Repo) GetEvent(ctx context.Context, eventID string) (*ingestor.BaseEvent, error) {
	row, err := db.Queries.GetEvent(ctx, eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ingestor.ErrEventNotFound
	}

	if err != nil {
		return nil, err
	}

	return toBaseEvent(row)
}

func (db *Repo) ListEvents(ctx context.Context) ([]*ingestor.BaseEvent, error) {
	rows, err := db.Queries.ListEvents(ctx)
	if err != nil {
		return nil, err
	}

	events := make([]*ingestor.BaseEvent, len(rows))
	for i, row := range rows {
		events[i], err = toBaseEvent(row)
		if err != nil {   // ← check error, don't blindly return after first iteration
			return nil, err
		}
	}

	return events, nil
}

func (db *Repo) Close() error {
	return db.Db.Close()
}

func (db *Repo) runMigrations() error {
	_, err := db.Db.Exec(schema)
	if err != nil {
		return err
	}

	return nil
}

func toBaseEvent(row generated.Event) (*ingestor.BaseEvent, error) {
	dataValue, err := toBaseEventValue(row.Data, ingestor.ParseDataType(row.EventType))
	if err != nil {
		return nil, err
	}

	return &ingestor.BaseEvent{
		EventID:       uuid.MustParse(row.EventID),
		Domain:        row.Domain,
		EventType:     row.EventType,
		EntityID:      row.EntityID,
		EntityType:    row.EntityType,
		OccurredAt:    row.OccurredAt,
		IngestedAt:    row.IngestedAt,
		Source:        row.Source,
		SchemaVersion: row.SchemaVersion,
		Data:          dataValue,
	}, nil
}

func toBaseEventValue(data string, eventType ingestor.DataTypes) (ingestor.BaseEventValue, error) {
	switch eventType {
	case ingestor.DataTypeFinancialStream:
		var financialTransaction ingestor.FinancialTransaction

		err := json.Unmarshal([]byte(data), &financialTransaction)
		if err != nil {
			return nil, err
		}

		return &financialTransaction, nil
	case ingestor.DataTypeEnergyMeter:
		var energyReading ingestor.EnergyReading

		err := json.Unmarshal([]byte(data), &energyReading)
		if err != nil {
			return nil, err
		}

		return &energyReading, nil
	case ingestor.DataTypeEnvironmentalSensor:
		var environmentalSensor ingestor.EnvironmentalSensor

		err := json.Unmarshal([]byte(data), &environmentalSensor)
		if err != nil {
			return nil, err
		}

		return &environmentalSensor, nil
	default:
		var unknownEvent ingestor.UnknownEvent

		err := json.Unmarshal([]byte(data), &unknownEvent)
		if err != nil {
			return nil, err
		}

		return &unknownEvent, nil
	}
}
