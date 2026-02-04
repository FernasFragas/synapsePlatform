-- name: CreateEvent :one
INSERT INTO events (
    event_id, domain, event_type, 
    entity_id, entity_type, 
    occurred_at, ingested_at, 
    source, schema_version, data, metadata
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

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
DELETE FROM events WHERE id = ?;