# synapsePlatform Performance Test Report

**Test Date:** Sat Mar 21 22:48:28 WET 2026
**Git Commit:** db6ee13
**Machine:** arm64
**OS:** Darwin 25.3.0

---

## Pre-Flight Diagnostics

### SQLite Configuration
```
Journal Mode: [0;32m[INFO][0m Checking SQLite configuration... [0;32m[INFO][0m ✅ SQLite is in WAL mode [1;33m[WARN][0m ⚠️ SQLite busy_timeout is 0 - writes will fail immediately on contention wal
Busy Timeout: 0ms
Synchronous: 1
```

### Index Status
- Pagination Index (idx_ingested_event): **[0;32m[INFO][0m Checking database indexes...
[1;33m[WARN][0m ⚠️  Missing pagination index (idx_ingested_event) - queries will be slow
no**

### Baseline Metrics
```
Process Stats:   0.0  0.1  24832 437402816
Database: 130013|108|59.64453125
Events in DB: 130013
Kafka LAG: -
```

---

## Test Results

