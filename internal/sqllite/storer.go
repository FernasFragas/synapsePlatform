//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/sqllite/mocked_$GOFILE
package sqllite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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

	// Configure SQLite for optimal performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",           // Enable Write-Ahead Logging
		"PRAGMA busy_timeout=5000",          // Wait 5s on lock contention
		"PRAGMA synchronous=NORMAL",         // Balance durability vs speed
		"PRAGMA cache_size=-64000",          // 64MB cache (negative = KB)
		"PRAGMA foreign_keys=ON",            // Enable foreign key constraints
		"PRAGMA temp_store=MEMORY",          // Store temp tables in memory
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	// Set connection pool limits (SQLite works best with limited concurrency)
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0) // Connections never expire

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

func (db *Repo) ListEvents(ctx context.Context, page ingestor.PageRequest) (*ingestor.PageResponse[*ingestor.BaseEvent], error) {
	limit := clamp(page.Limit, 1, maxPageSize, defaultPageSize)
	fetchLimit := int64(limit + 1) // fetch one extra to detect "has more"

	var rows []generated.Event
	var err error

	if page.Cursor == "" {
		rows, err = db.Queries.ListEventsFirstPage(ctx, fetchLimit)
	} else {
		c, cErr := decodeCursor(page.Cursor)
		if cErr != nil {
			return nil, cErr
		}

		rows, err = db.Queries.ListEventsAfterCursor(ctx, generated.ListEventsAfterCursorParams{
			IngestedAt: c.IngestedAt,
			Limit:      fetchLimit,
		})
	}
	if err != nil {
		return nil, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	events := make([]*ingestor.BaseEvent, len(rows))
	for i, row := range rows {
		events[i], err = toBaseEvent(row)
		if err != nil {
			return nil, err
		}
	}

	var nextCursor string
	if hasMore {
		last := rows[len(rows)-1]
		nextCursor = encodeCursor(last.IngestedAt, last.EventID)
	}

	return &ingestor.PageResponse[*ingestor.BaseEvent]{
		Items:      events,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (db *Repo) StoreFailure(ctx context.Context, failed ingestor.FailedMessage) error {
	var msgJSON []byte
	if failed.Message != nil {
		msgJSON, _ = json.Marshal(failed.Message)
	}

	var errText string
	if failed.Err != nil {
		errText = failed.Err.Error()
	}

	_, err := db.Db.ExecContext(ctx,
		`INSERT INTO failed_messages (stage, message, error, created_at) VALUES (?, ?, ?, datetime('now'))`,
		failed.Stage, string(msgJSON), errText,
	)

	return err
}

func (db *Repo) Close() error {
	return db.Db.Close()
}

func (db *Repo) Name() string { return "db" }

func (db *Repo) Check(ctx context.Context) error {
	return db.Db.PingContext(ctx)
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

	eventID, err := uuid.Parse(row.EventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event UUID %q: %w", row.EventID, err)
	}

	return &ingestor.BaseEvent{
		EventID:       eventID,
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
	desc, ok := ingestor.LookupDomain(eventType)
	if !ok {
		desc, _ = ingestor.LookupDomain(ingestor.DataTypeUnknown)
	}

	payload := desc.NewPayload()

	err := json.Unmarshal([]byte(data), payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

const maxPageSize = 100
const defaultPageSize = 20

type cursor struct {
	IngestedAt time.Time `json:"t"`
	EventID    string    `json:"id"`
}

func encodeCursor(t time.Time, id string) string {
	b, _ := json.Marshal(cursor{IngestedAt: t, EventID: id})

	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return cursor{}, fmt.Errorf("invalid cursor: %w", err)
	}

	var c cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return cursor{}, fmt.Errorf("invalid cursor: %w", err)
	}

	return c, nil
}

func clamp(v, min, max, fallback int) int {
	if v <= 0 {
		return fallback
	}

	if v < min {
		return min
	}

	if v > max {
		return max
	}

	return v
}
