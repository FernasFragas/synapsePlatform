# synapsePlatform Performance Test Report

**Test Date:** Fri Mar 27 23:14:37 WET 2026
**Git Commit:** 70f88b8
**Machine:** arm64
**OS:** Darwin 25.4.0

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
Process Stats:   0.0  0.1  28000 444307472
Database: 18112|36668|23.9609375
Events in DB: 18112
Kafka LAG: 0
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 97s |
| **Messages Sent** | 600 |
| **Messages Processed** | 285 |
| **Failed Messages** | 37267 |
| **Success Rate** | 47.50% |
| **Actual Throughput** | 2.9 msg/sec |
| **Peak Kafka LAG** | 132 |
| **Average Kafka LAG** | 63 |
| **Final Kafka LAG** | 68 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  36224 444454464 |

**Analysis:**
- ⚠️  Peak LAG exceeded 100 - consumer falling behind even at low load
- ⚠️  Success rate < 100% - check failed_messages table

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 260s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 2992 |
| **Failed Messages** | 43268 |
| **Success Rate** | 49.87% |
| **Actual Throughput** | 11.5 msg/sec |
| **Peak Kafka LAG** | 494 |
| **Average Kafka LAG** | 233 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.3  0.1  53152 444542352 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 947s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 15089 |
| **Failed Messages** | 73268 |
| **Success Rate** | 50.30% |
| **Actual Throughput** | 15.9 msg/sec |
| **Peak Kafka LAG** | 672 |
| **Average Kafka LAG** | 330 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.3 118528 444614208 |

**Analysis:**
- 📊 Observed throughput ceiling: ~15.9 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 36478 |
| **Total Failed** | 73268 |
| **Database Stats** | 36478|73268|48.1640625 |
| **List Query (20 items)** | 15ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```
ack|68
store_batch|73200
```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 2.9 msg/sec | 47.50% | 132 | 63 |
| Test 2 | 100 msg/sec | 11.5 msg/sec | 49.87% | 494 | 233 |
| Test 3 | 500 msg/sec | 15.9 msg/sec | 50.30% | 672 | 330 |

---

## Bottleneck Analysis

### 🔴 Critical: SQLite Not in WAL Mode
- **Current mode:** [0;32m[INFO][0m Checking SQLite configuration... [0;32m[INFO][0m ✅ SQLite is in WAL mode [1;33m[WARN][0m ⚠️ SQLite busy_timeout is 0 - writes will fail immediately on contention wal
- **Impact:** 3-5x slower writes, blocks reads during writes
- **Fix:** Add pragmas to `internal/sqllite/storer.go`:
```go
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```
- **Expected improvement:** 30-35 msg/sec → 100-150 msg/sec

### ⚠️  Failed Messages Detected
- **Count:** 73268
- **Check:** `sqlite3 data.db "SELECT stage, error FROM failed_messages LIMIT 5;"`
- **Likely cause:** Invalid test data or validation errors

---

## Debug Information

- **Debug log:** ./performance-reports/debug-20260327-231436.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260327-231436.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260327-231436.log` for detailed timeline

---

**Test completed at:** Fri Mar 27 23:37:13 WET 2026
