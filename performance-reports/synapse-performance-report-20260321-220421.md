# synapsePlatform Performance Test Report

**Test Date:** Sat Mar 21 22:04:22 WET 2026
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
Process Stats:   0.0  0.1  27056 437271792
Database: 93413|106|42.8046875
Events in DB: 93413
Kafka LAG: 0
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 96s |
| **Messages Sent** | 600 |
| **Messages Processed** | 600 |
| **Failed Messages** | 106 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 6.2 msg/sec |
| **Peak Kafka LAG** | 41 |
| **Average Kafka LAG** | 19 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  40448 437540400 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 263s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 106 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 22.8 msg/sec |
| **Peak Kafka LAG** | 57 |
| **Average Kafka LAG** | 30 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  53776 437622832 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 949s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30000 |
| **Failed Messages** | 106 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 31.6 msg/sec |
| **Peak Kafka LAG** | 66 |
| **Average Kafka LAG** | 33 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.2  89264 437657024 |

**Analysis:**
- 📊 Observed throughput ceiling: ~31.6 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 130013 |
| **Total Failed** | 106 |
| **Database Stats** | 130013|106|59.64453125 |
| **List Query (20 items)** | 427ms |
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
| Test 1 | 10 msg/sec | 6.2 msg/sec | 100.00% | 41 | 19 |
| Test 2 | 100 msg/sec | 22.8 msg/sec | 100.00% | 57 | 30 |
| Test 3 | 500 msg/sec | 31.6 msg/sec | 100.00% | 66 | 33 |

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
- **Count:** 106
- **Check:** `sqlite3 data.db "SELECT stage, error FROM failed_messages LIMIT 5;"`
- **Likely cause:** Invalid test data or validation errors

---

## Debug Information

- **Debug log:** ./performance-reports/debug-20260321-220421.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260321-220421.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260321-220421.log` for detailed timeline

---

**Test completed at:** Sat Mar 21 22:27:03 WET 2026
