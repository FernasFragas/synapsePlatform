# synapsePlatform Performance Test Report

**Test Date:** Fri Mar 20 22:27:25 WET 2026
**Git Commit:** 2c5889d
**Machine:** arm64
**OS:** Darwin 25.3.0

---

## Pre-Flight Diagnostics

### SQLite Configuration
```
Journal Mode: [0;32m[INFO][0m Checking SQLite configuration... [1;33m[WARN][0m ⚠️ SQLite is NOT in WAL mode (current: delete) - this limits write performance [1;33m[WARN][0m ⚠️ SQLite busy_timeout is 0 - writes will fail immediately on contention delete
Busy Timeout: 0ms
Synchronous: 2
```

### Index Status
- Pagination Index (idx_ingested_event): **[0;32m[INFO][0m Checking database indexes...
[1;33m[WARN][0m ⚠️  Missing pagination index (idx_ingested_event) - queries will be slow
no**

### Baseline Metrics
```
Process Stats:   0.0  0.1  28400 437604160
Database: 20213|106|9.26171875
Events in DB: 20213
Kafka LAG: 0
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 95s |
| **Messages Sent** | 600 |
| **Messages Processed** | 600 |
| **Failed Messages** | 106 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 6.3 msg/sec |
| **Peak Kafka LAG** | 42 |
| **Average Kafka LAG** | 18 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  37776 437608256 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 254s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 106 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 23.6 msg/sec |
| **Peak Kafka LAG** | 57 |
| **Average Kafka LAG** | 29 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  39488 437608256 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 921s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30000 |
| **Failed Messages** | 106 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 32.6 msg/sec |
| **Peak Kafka LAG** | 64 |
| **Average Kafka LAG** | 33 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  41168 437608256 |

**Analysis:**
- 📊 Observed throughput ceiling: ~32.6 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 56813 |
| **Total Failed** | 106 |
| **Database Stats** | 56813|106|25.99609375 |
| **List Query (20 items)** | 51ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```
process|1
transform|105
```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 6.3 msg/sec | 100.00% | 42 | 18 |
| Test 2 | 100 msg/sec | 23.6 msg/sec | 100.00% | 57 | 29 |
| Test 3 | 500 msg/sec | 32.6 msg/sec | 100.00% | 64 | 33 |

---

## Bottleneck Analysis

### 🔴 Critical: SQLite Not in WAL Mode
- **Current mode:** [0;32m[INFO][0m Checking SQLite configuration... [1;33m[WARN][0m ⚠️ SQLite is NOT in WAL mode (current: delete) - this limits write performance [1;33m[WARN][0m ⚠️ SQLite busy_timeout is 0 - writes will fail immediately on contention delete
- **Impact:** 3-5x slower writes, blocks reads during writes
- **Fix:** Add pragmas to `internal/sqllite/storer.go`:
```go
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```
- **Expected improvement:** 30-35 msg/sec → 100-150 msg/sec

### ⚠️  Failed Messages Detected
- **Count:** 106
- **Check:** `sqlite3 data.db "SELECT stage, error FROM failed_messages LIMIT 5;"`
- **Likely cause:** Invalid test data or validation errors

---

## Debug Information

- **Debug log:** ./performance-reports/debug-20260320-222724.log
- **Kafka LAG log:** /tmp/kafka-lag-monitor.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260320-222724.log` for detailed timeline

---

**Test completed at:** Fri Mar 20 22:49:28 WET 2026
