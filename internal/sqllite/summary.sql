CREATE TABLE IF NOT EXISTS events (
    -- Primary identification
    event_id TEXT PRIMARY KEY,
    
    -- Event classification
    domain TEXT NOT NULL,           -- NEW: e.g., "energy", "finance", "monitoring"
    event_type TEXT NOT NULL,       -- e.g., "reading", "transaction", "latency_sample"
    
    -- Entity information
    entity_id TEXT NOT NULL,
    entity_type TEXT NOT NULL,      -- e.g., "sensor", "account", "service"
    
    -- Timestamps
    occurred_at TIMESTAMP NOT NULL,      -- ISO8601: when event happened
    ingested_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,      -- ISO8601: when was received it
    
    -- Provenance & versioning
    source TEXT NOT NULL,           -- NEW: e.g., "iot-gateway", "payment-processor"
    schema_version TEXT NOT NULL,   -- NEW: e.g., "1.0", "2.1"
    
    -- Payload
    data TEXT NOT NULL,             -- JSON: the generic T data (EnergyReading, etc.)
    
    -- Optional metadata (keep if you need it for additional info)
    metadata TEXT
);
-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_domain_type ON events(domain, event_type);
CREATE INDEX IF NOT EXISTS idx_entity ON events(entity_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_occurred_at ON events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_source ON events(source);
CREATE INDEX IF NOT EXISTS idx_ingested_event ON events(ingested_at DESC, event_id DESC);

CREATE TABLE IF NOT EXISTS failed_messages (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    stage      TEXT NOT NULL,
    message    TEXT,
    error      TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_failed_stage ON failed_messages(stage);
CREATE INDEX IF NOT EXISTS idx_failed_created_at ON failed_messages(created_at);

CREATE TABLE IF NOT EXISTS store_accounting (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    attempted_events INTEGER NOT NULL DEFAULT 0,
    inserted_events INTEGER NOT NULL DEFAULT 0,
    duplicate_events INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO store_accounting (
    id, attempted_events, inserted_events, duplicate_events, updated_at
) VALUES (1, 0, 0, 0, CURRENT_TIMESTAMP);

CREATE TABLE IF NOT EXISTS summaries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    domain      TEXT NOT NULL,
    window_from TIMESTAMP NOT NULL,
    model       TEXT NOT NULL,
    content     TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_summaries_lookup ON summaries(domain, window_from, created_at DESC);
