# synapsePlatform Performance Test Report

**Test Date:** Sat Jun 13 21:40:54 WEST 2026
**Git Commit:** a27db6f
**Machine:** arm64
**OS:** Darwin 25.5.0

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
[0;32m[INFO][0m ✅ Pagination index (idx_ingested_event) exists
yes**

### Baseline Metrics
```
Process Stats:   0.0  0.1  31408 436676448
Database: 6|0|0.0546875
Events in DB: 6
Kafka LAG: 6
```

---

## Test Results

