-- name: CreateEvent :one
INSERT INTO events (
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