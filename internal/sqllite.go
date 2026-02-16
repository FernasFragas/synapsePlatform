package internal

import (
	"context"
	"database/sql"
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

func (db *DB) StoreData(ctx context.Context, messages []ingestor.DeviceMessage) error {

}
