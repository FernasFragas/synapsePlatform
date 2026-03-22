# synapsePlatform Performance Test Report

**Test Date:** Sun Mar 22 17:29:14 WET 2026
**Git Commit:** 01cbe1c
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
[0;32m[INFO][0m ✅ Pagination index (idx_ingested_event) exists
yes**

### Baseline Metrics
```
Process Stats:   0.0  0.1  27136 437402384
Database: 182989|1|113.546875
Events in DB: 182989
Kafka LAG: -
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 101s |
| **Messages Sent** | 600 |
| **Messages Processed** | 600 |
| **Failed Messages** | 1 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 5.9 msg/sec |
| **Peak Kafka LAG** | 38 |
| **Average Kafka LAG** | 18 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  39120 437672368 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 258s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 1 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 23.3 msg/sec |
| **Peak Kafka LAG** | 56 |
| **Average Kafka LAG** | 28 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.2  64272 437699488 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 935s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30000 |
| **Failed Messages** | 1 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 32.1 msg/sec |
| **Peak Kafka LAG** | 66 |
| **Average Kafka LAG** | 32 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.3 108880 437744256 |

**Analysis:**
- 📊 Observed throughput ceiling: ~32.1 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 219589 |
| **Total Failed** | 1 |
| **Database Stats** | 219589|1|136.31640625 |
| **List Query (20 items)** | 19ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```
process|1
```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 5.9 msg/sec | 100.00% | 38 | 18 |
| Test 2 | 100 msg/sec | 23.3 msg/sec | 100.00% | 56 | 28 |
| Test 3 | 500 msg/sec | 32.1 msg/sec | 100.00% | 66 | 32 |

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
- **Count:** 1
- **Check:** `sqlite3 data.db "SELECT stage, error FROM failed_messages LIMIT 5;"`
- **Likely cause:** Invalid test data or validation errors

---

## Debug Information

- **Debug log:** ./performance-reports/debug-20260322-172913.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260322-172913.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260322-172913.log` for detailed timeline

---

**Test completed at:** Sun Mar 22 17:51:40 WET 2026
