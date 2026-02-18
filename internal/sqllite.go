package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"synapsePlatform/internal/ingestor"

	_ "modernc.org/sqlite" // Import SQLite driver

	"synapsePlatform/internal/sqlc/generated"
)

type DB struct {
	Db      *sql.DB
	Queries *generated.Queries
}

func Open(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	var database = DB{
		Db: db,
	}

	if err := database.runMigrations(); err != nil {
		return nil, err
	}

	queries := generated.New(database.Db)

	database.Queries = queries

	return &database, nil
}

func (db *DB) runMigrations() error {
	schema, err := os.ReadFile("db/summary.sql")
	if err != nil {
		return err
	}

	_, err = db.Db.Exec(string(schema))
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) StoreData(ctx context.Context, data *ingestor.NormalizedData) (generated.Event, error) {
	// Serialize the data payload to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		// todo
	}

	value := *data

	savedEvent, err := db.Queries.CreateEvent(ctx, generated.CreateEventParams{
		EventID:       value.Value().EventID.String(),
		Domain:        value.Value().Domain,
		EventType:     value.Value().EventType,
		EntityID:      value.Value().EntityID,
		EntityType:    value.Value().EntityType,
		OccurredAt:    value.Value().OccurredAt,
		IngestedAt:    value.Value().IngestedAt,
		Source:        value.Value().Source,
		SchemaVersion: value.Value().SchemaVersion,
		Data:          string(dataJSON),
		Metadata:      sql.NullString{},
	})
	if err != nil {
		// todo
	}

	return savedEvent, nil

}
