//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/sqllite/mocked_$GOFILE
package sqllite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"

	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/sqlc/generated"

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
	// Serialize the data payload to JSON
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

func (db *Repo) runMigrations() error {
	_, err := db.Db.Exec(schema)
	if err != nil {
		return err
	}

	return nil
}

func (db *Repo) Close() error {
	return db.Db.Close()
}
