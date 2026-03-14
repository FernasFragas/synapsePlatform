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

CREATE TABLE IF NOT EXISTS failed_messages (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    stage      TEXT NOT NULL,
    message    TEXT,
    error      TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_failed_stage ON failed_messages(stage);
CREATE INDEX IF NOT EXISTS idx_failed_created_at ON failed_messages(created_at);