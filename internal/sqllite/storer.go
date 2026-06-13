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
	"strings"
	"synapsePlatform/internal/api"
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
		"PRAGMA journal_mode=WAL",   // Enable Write-Ahead Logging
		"PRAGMA busy_timeout=5000",  // Wait 5s on lock contention
		"PRAGMA synchronous=NORMAL", // Balance durability vs speed
		"PRAGMA cache_size=-64000",  // 64MB cache (negative = KB)
		"PRAGMA foreign_keys=ON",    // Enable foreign key constraints
		"PRAGMA temp_store=MEMORY",  // Store temp tables in memory
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	// Set connection pool limits (SQLite works best with limited concurrency)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
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
		return ingestor.NewTerminalError(fmt.Errorf("marshal event data: %w", err))
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
		return classifyStoreError(fmt.Errorf("create event: %w", err))
	}

	return nil

}

func (db *Repo) StoreBatch(ctx context.Context, events []*ingestor.BaseEvent) error {
	if len(events) == 0 {
		return nil
	}
	const (
		varsPerEvent      = 11
		maxVars           = 999
		maxEventsPerChunk = maxVars / varsPerEvent // = 90 events per chunk
	)
	// Process events in chunks
	for i := 0; i < len(events); i += maxEventsPerChunk {
		end := i + maxEventsPerChunk
		if end > len(events) {
			end = len(events)
		}

		chunk := events[i:end]
		if err := db.insertChunk(ctx, chunk); err != nil {
			return fmt.Errorf("failed to insert chunk %d-%d: %w", i, end, err)
		}
	}

	return nil
}
func (db *Repo) insertChunk(ctx context.Context, events []*ingestor.BaseEvent) error {
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		return classifyStoreError(fmt.Errorf("begin transaction: %w", err))
	}
	defer tx.Rollback()

	placeholders := make([]string, 0, len(events))
	args := make([]interface{}, 0, len(events)*11)

	for _, event := range events {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

		dataJSON, err := json.Marshal(event.Data)
		if err != nil {
			return ingestor.NewTerminalError(fmt.Errorf("marshal event data: %w", err))
		}

		args = append(args,
			event.EventID.String(),
			event.Domain,
			event.EventType,
			event.EntityID,
			event.EntityType,
			event.OccurredAt,
			event.IngestedAt,
			event.Source,
			event.SchemaVersion,
			string(dataJSON),
			sql.NullString{},
		)
	}

	query := fmt.Sprintf(`
    INSERT OR IGNORE INTO events (
        event_id, domain, event_type, entity_id, entity_type,
        occurred_at, ingested_at, source, schema_version, data, metadata
    ) VALUES %s
`, strings.Join(placeholders, ", "))

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return classifyStoreError(fmt.Errorf("execute batch insert: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return classifyStoreError(fmt.Errorf("commit transaction: %w", err))
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

	errText := failed.ErrorMessage

	_, err := db.Db.ExecContext(ctx,
		`INSERT INTO failed_messages (stage, message, error, created_at) VALUES (?, ?, ?, datetime('now'))`,
		failed.Stage, string(msgJSON), errText,
	)

	return err
}

func (db *Repo) AggregateByDomain(ctx context.Context, since time.Time) ([]api.DomainStat, error) {
	rows, err := db.Queries.SummarizeByDomain(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("summarize by domain: %w", err)
	}

	stats := make([]api.DomainStat, len(rows))
	for i, r := range rows {
		stats[i] = api.DomainStat{
			Domain:    r.Domain,
			EventType: r.EventType,
			Count:     r.Cnt,
			FirstSeen: toTime(r.FirstSeen),
			LastSeen:  toTime(r.LastSeen),
		}
	}
	return stats, nil
}

func (db *Repo) LatestSummary(ctx context.Context, domain string, since time.Time) (*api.Report, bool, error) {
	// You need a sqlc query first; placeholder using raw SQL until then:
	row := db.Db.QueryRowContext(ctx,
		`SELECT domain, window_from, model, content, created_at
		 FROM summaries
		 WHERE domain = ? AND window_from = ?
		 ORDER BY created_at DESC LIMIT 1`,
		domain, since,
	)

	var r api.Report
	err := row.Scan(&r.Domain, &r.WindowFrom, &r.Model, &r.Content, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &r, true, nil
}

func (db *Repo) SaveSummary(ctx context.Context, r *api.Report) error {
	_, err := db.Db.ExecContext(ctx,
		`INSERT INTO summaries (domain, window_from, model, content, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		r.Domain, r.WindowFrom, r.Model, r.Content, r.CreatedAt,
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

func toTime(v interface{}) time.Time {
	if t, ok := v.(time.Time); ok {
		return t
	}
	if s, ok := v.(string); ok {
		return parseSQLiteTime(s)
	}
	if b, ok := v.([]byte); ok {
		return parseSQLiteTime(string(b))
	}
	return time.Time{}
}

func parseSQLiteTime(value string) time.Time {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}
