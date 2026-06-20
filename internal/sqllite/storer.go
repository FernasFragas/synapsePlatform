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

	result, err := db.Db.ExecContext(ctx, `
    INSERT OR IGNORE INTO events (
        event_id, domain, event_type, entity_id, entity_type,
        occurred_at, ingested_at, source, schema_version, data, metadata
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		value.EventID.String(),
		value.Domain,
		value.EventType,
		value.EntityID,
		value.EntityType,
		value.OccurredAt,
		value.IngestedAt,
		value.Source,
		value.SchemaVersion,
		string(dataJSON),
		sql.NullString{},
	)
	if err != nil {
		return classifyStoreError(fmt.Errorf("create event: %w", err))
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		return classifyStoreError(fmt.Errorf("read rows affected: %w", err))
	}

	if err := db.recordStoreAccounting(ctx, 1, inserted); err != nil {
		return err
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

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return classifyStoreError(fmt.Errorf("execute batch insert: %w", err))
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return classifyStoreError(fmt.Errorf("read rows affected: %w", err))
	}
	if err := db.recordStoreAccountingTx(ctx, tx, int64(len(events)), inserted); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return classifyStoreError(fmt.Errorf("commit transaction: %w", err))
	}

	return nil
}

func (db *Repo) recordStoreAccounting(ctx context.Context, attempted, inserted int64) error {
	_, err := db.Db.ExecContext(ctx, storeAccountingQuery, attempted, inserted, attempted-inserted)
	if err != nil {
		return classifyStoreError(fmt.Errorf("record store accounting: %w", err))
	}

	return nil
}

func (db *Repo) recordStoreAccountingTx(ctx context.Context, tx *sql.Tx, attempted, inserted int64) error {
	_, err := tx.ExecContext(ctx, storeAccountingQuery, attempted, inserted, attempted-inserted)
	if err != nil {
		return classifyStoreError(fmt.Errorf("record store accounting: %w", err))
	}

	return nil
}

const storeAccountingQuery = `
UPDATE store_accounting
SET attempted_events = attempted_events + ?,
    inserted_events = inserted_events + ?,
    duplicate_events = duplicate_events + ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
`

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

// ListSummaryEvidence selects a bounded, curated set of events for the summary
// prompt. It returns stable event fields only (event_id, domain, event_type,
// entity_id, occurred_at) — no raw payloads or secrets.
//
// Selection strategy: the most recent events within the requested window,
// optionally filtered by domain. This prioritizes "recent failures and unusual
// activity" by ordering on occurred_at DESC, which surfaces the latest events
// the model is most likely to need during incident troubleshooting. When the
// domain is empty, events from all domains are included so a cross-domain
// summary still has evidence.
//
// The result is bounded by req.MaxResults (defaulting to 30 when <= 0) so the
// model never sees the full raw event stream even when the window contains
// thousands of events.
func (db *Repo) ListSummaryEvidence(ctx context.Context, req api.EvidenceRequest) ([]api.SummaryEvidenceEvent, error) {
	limit := req.MaxResults
	if limit <= 0 {
		limit = api.DefaultMaxEvidenceEvents
	}

	until := req.Until
	if until.IsZero() {
		until = time.Now().UTC()
	}

	var (
		rows *sql.Rows
		err  error
	)

	if req.Domain != "" {
		rows, err = db.Db.QueryContext(ctx,
			`SELECT event_id, domain, event_type, entity_id, occurred_at
			 FROM events
			 WHERE domain = ? AND occurred_at >= ? AND occurred_at <= ?
			 ORDER BY occurred_at DESC
			 LIMIT ?`,
			req.Domain, req.Since, until, limit,
		)
	} else {
		rows, err = db.Db.QueryContext(ctx,
			`SELECT event_id, domain, event_type, entity_id, occurred_at
			 FROM events
			 WHERE occurred_at >= ? AND occurred_at <= ?
			 ORDER BY occurred_at DESC
			 LIMIT ?`,
			req.Since, until, limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("query summary evidence: %w", err)
	}
	defer rows.Close()

	var evidence []api.SummaryEvidenceEvent
	for rows.Next() {
		var e api.SummaryEvidenceEvent
		if err := rows.Scan(&e.EventID, &e.Domain, &e.EventType, &e.EntityID, &e.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan summary evidence row: %w", err)
		}
		evidence = append(evidence, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate summary evidence rows: %w", err)
	}

	return evidence, nil
}

func (db *Repo) LatestSummary(ctx context.Context, lookup api.SummaryLookup) (*api.Report, bool, error) {
	row := db.Db.QueryRowContext(ctx,
		`SELECT domain, window_from, model, content,
		        structured_content, provider, prompt_version, input_hash,
		        created_at
		 FROM summaries
		 WHERE domain = ? AND window_from = ?
		   AND (provider IS ? OR provider = ?)
		   AND model = ?
		   AND (prompt_version IS ? OR prompt_version = ?)
		   AND (input_hash IS ? OR input_hash = ?)
		 ORDER BY created_at DESC LIMIT 1`,
		lookup.Domain, lookup.WindowFrom,
		nullableString(lookup.Provider), lookup.Provider,
		lookup.Model,
		nullableString(lookup.PromptVersion), lookup.PromptVersion,
		nullableString(lookup.InputHash), lookup.InputHash,
	)

	var r api.Report
	var structuredContent, provider, promptVersion, inputHash sql.NullString
	err := row.Scan(
		&r.Domain, &r.WindowFrom, &r.Model, &r.Content,
		&structuredContent, &provider, &promptVersion, &inputHash,
		&r.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	r.StructuredContent = structuredContent.String
	r.Provider = provider.String
	r.PromptVersion = promptVersion.String
	r.InputHash = inputHash.String

	return &r, true, nil
}

func (db *Repo) SaveSummary(ctx context.Context, r *api.Report) (int64, error) {
	result, err := db.Db.ExecContext(ctx,
		`INSERT INTO summaries
		   (domain, window_from, model, content,
		    structured_content, provider, prompt_version, input_hash,
		    created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Domain, r.WindowFrom, r.Model, r.Content,
		nullableString(r.StructuredContent),
		nullableString(r.Provider),
		nullableString(r.PromptVersion),
		nullableString(r.InputHash),
		r.CreatedAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get summary last insert id: %w", err)
	}
	return id, nil
}

// SaveSummaryEvidenceLinks persists the evidence links for a summary. Each
// link connects a summary row to an event it cited, with a relationship
// describing why the event is linked (evidence, notable, recommendation).
//
// Links are inserted with INSERT OR IGNORE so re-saving the same links is
// idempotent. The primary key (summary_id, event_id, relationship) prevents
// duplicates.
func (db *Repo) SaveSummaryEvidenceLinks(ctx context.Context, links []api.SummaryEvidenceLink) error {
	if len(links) == 0 {
		return nil
	}
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin evidence links transaction: %w", err)
	}
	defer tx.Rollback()

	for _, link := range links {
		_, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO intelligence_summary_events
			   (summary_id, event_id, relationship, created_at)
			 VALUES (?, ?, ?, ?)`,
			link.SummaryID, link.EventID, link.Relationship, time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert evidence link (summary=%d, event=%s, rel=%s): %w",
				link.SummaryID, link.EventID, link.Relationship, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit evidence links: %w", err)
	}
	return nil
}

// nullableString converts an empty string to a NULL-friendly sql.NullString so
// the new structured-summary columns store NULL instead of "" for legacy and
// free-form summaries. This keeps the schema's nullable semantics explicit.
func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}

	return sql.NullString{String: s, Valid: true}
}

func (db *Repo) Close() error {
	return db.Db.Close()
}

func (db *Repo) Name() string { return "db" }

func (db *Repo) Check(ctx context.Context) error {
	return db.Db.PingContext(ctx)
}

func (db *Repo) runMigrations() error {
	if _, err := db.Db.Exec(schema); err != nil {
		return err
	}

	summaryMigrations := []struct {
		column string
		ddl    string
	}{
		{"structured_content", "ALTER TABLE summaries ADD COLUMN structured_content TEXT"},
		{"provider", "ALTER TABLE summaries ADD COLUMN provider TEXT"},
		{"prompt_version", "ALTER TABLE summaries ADD COLUMN prompt_version TEXT"},
		{"input_hash", "ALTER TABLE summaries ADD COLUMN input_hash TEXT"},
	}

	for _, m := range summaryMigrations {
		missing, err := db.columnMissing("summaries", m.column)
		if err != nil {
			return fmt.Errorf("check summaries column %s: %w", m.column, err)
		}
		if !missing {
			continue
		}
		if _, err := db.Db.Exec(m.ddl); err != nil {
			return fmt.Errorf("migrate summaries column %s: %w", m.column, err)
		}
	}

	return nil
}

// columnMissing reports whether a column is absent from the named table. Used
// to guard ALTER TABLE ADD COLUMN statements so migrations are idempotent.
func (db *Repo) columnMissing(table, column string) (bool, error) {
	var count int
	err := db.Db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`,
		table, column,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	return count == 0, nil
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
