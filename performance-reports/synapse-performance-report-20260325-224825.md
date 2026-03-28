# synapsePlatform Performance Test Report

**Test Date:** Wed Mar 25 22:48:26 WET 2026
**Git Commit:** 3b0fef0
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
Process Stats:   0.1  0.1  49248 444538080
Database: 4319|41017|17.09375
Events in DB: 4319
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
| **Messages Processed** | 283 |
| **Failed Messages** | 41604 |
| **Success Rate** | 47.17% |
| **Actual Throughput** | 2.9 msg/sec |
| **Peak Kafka LAG** | 5 |
| **Average Kafka LAG** | 2 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  50240 444538976 |

**Analysis:**
- ⚠️  Success rate < 100% - check failed_messages table

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 264s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 2954 |
| **Failed Messages** | 47617 |
| **Success Rate** | 49.23% |
| **Actual Throughput** | 11.2 msg/sec |
| **Peak Kafka LAG** | 32 |
| **Average Kafka LAG** | 15 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  46800 444548656 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 953s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 14891 |
| **Failed Messages** | 77617 |
| **Success Rate** | 49.64% |
| **Actual Throughput** | 15.6 msg/sec |
| **Peak Kafka LAG** | 43 |
| **Average Kafka LAG** | 21 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.3 107008 444594960 |

**Analysis:**
- 📊 Observed throughput ceiling: ~15.6 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 22447 |
| **Total Failed** | 77617 |
| **Database Stats** | 22447|77617|41.1640625 |
| **List Query (20 items)** | 19ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```
store_batch|77617
```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 2.9 msg/sec | 47.17% | 5 | 2 |
| Test 2 | 100 msg/sec | 11.2 msg/sec | 49.23% | 32 | 15 |
| Test 3 | 500 msg/sec | 15.6 msg/sec | 49.64% | 43 | 21 |

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
- **Count:** 77617
- **Check:** `sqlite3 data.db "SELECT stage, error FROM failed_messages LIMIT 5;"`
- **Likely cause:** Invalid test data or validation errors

---

## Debug Information

- **Debug log:** ./performance-reports/debug-20260325-224825.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260325-224825.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260325-224825.log` for detailed timeline

---

**Test completed at:** Wed Mar 25 23:11:11 WET 2026
