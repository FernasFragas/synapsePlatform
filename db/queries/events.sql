-- name: CreateEvent :one
INSERT OR IGNORE INTO events (
    event_id,
    domain,
    event_type,
    entity_id,
    entity_type,
    occurred_at,
    ingested_at,
    source,
    schema_version,
    data,
    metadata
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING event_id, domain, event_type, entity_id, entity_type,
    occurred_at, ingested_at, source, schema_version, data, metadata;

-- name: GetEventsByDomain :many
SELECT * FROM events 
WHERE domain = ? 
ORDER BY occurred_at DESC 
LIMIT ?;

-- name: GetEventsBySource :many
SELECT * FROM events 
WHERE source = ? 
ORDER BY ingested_at DESC 
LIMIT ?;

-- name: DeleteEvent :exec
DELETE FROM events WHERE event_id = ?;

-- name: GetEvent :one
SELECT * FROM events WHERE event_id = ? LIMIT 1;

-- name: ListEventsFirstPage :many
SELECT * FROM events
ORDER BY ingested_at DESC, event_id DESC
LIMIT ?;

-- name: ListEventsAfterCursor :many
SELECT * FROM events
WHERE (ingested_at, event_id) < (?, ?)
ORDER BY ingested_at DESC, event_id DESC
LIMIT ?;

CREATE INDEX IF NOT EXISTS idx_ingested_event ON events(ingested_at DESC, event_id DESC);
CREATE UNIQUE INDEX idx_business_key ON events(entity_id, event_type, occurred_at);

-- name: SummarizeByDomain :many
SELECT domain, event_type,
    COUNT(*)        AS cnt,
    MIN(occurred_at) AS first_seen,
    MAX(occurred_at) AS last_seen
FROM events
WHERE occurred_at >= ?
GROUP BY domain, event_type
ORDER BY cnt DESC;

CREATE TABLE IF NOT EXISTS summaries (
                                         id          INTEGER PRIMARY KEY AUTOINCREMENT,
                                         domain      TEXT NOT NULL,
                                         window_from TIMESTAMP NOT NULL,
                                         model       TEXT NOT NULL,
                                         content     TEXT NOT NULL,
                                         created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_summary_lookup ON summaries(domain, window_from, created_at DESC);

-- name: GetLatestSummary :one
SELECT domain, window_from, model, content, created_at
FROM summaries
WHERE domain = ? AND window_from = ?
ORDER BY created_at DESC LIMIT 1;

-- name: InsertSummary :one
INSERT INTO summaries (domain, window_from, model, content, created_at)
VALUES (?, ?, ?, ?, ?)
    RETURNING id, domain, window_from, model, content, created_at;