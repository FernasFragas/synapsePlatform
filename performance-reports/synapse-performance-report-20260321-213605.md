# synapsePlatform Performance Test Report

**Test Date:** Sat Mar 21 21:36:07 WET 2026
**Git Commit:** db6ee13
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
Process Stats:   0.0  0.1  27616 437533728
Database: 56813|106|25.99609375
Events in DB: 56813
Kafka LAG: 0
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 103s |
| **Messages Sent** | 600 |
| **Messages Processed** | 600 |
| **Failed Messages** | 106 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 5.8 msg/sec |
| **Peak Kafka LAG** | 40 |
| **Average Kafka LAG** | 21 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  37040 437538400 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 249s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 106 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 24.1 msg/sec |
| **Peak Kafka LAG** | 58 |
| **Average Kafka LAG** | 30 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  39728 437608592 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 933s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30000 |
| **Failed Messages** | 106 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 32.2 msg/sec |
| **Peak Kafka LAG** | 66 |
| **Average Kafka LAG** | 32 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  40224 437608592 |

**Analysis:**
- 📊 Observed throughput ceiling: ~32.2 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 93413 |
| **Total Failed** | 106 |
| **Database Stats** | 93413|106|42.8046875 |
| **List Query (20 items)** | 121ms |
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
| Test 1 | 10 msg/sec | 5.8 msg/sec | 100.00% | 40 | 21 |
| Test 2 | 100 msg/sec | 24.1 msg/sec | 100.00% | 58 | 30 |
| Test 3 | 500 msg/sec | 32.2 msg/sec | 100.00% | 66 | 32 |

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

- **Debug log:** ./performance-reports/debug-20260321-213605.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260321-213605.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260321-213605.log` for detailed timeline

---

**Test completed at:** Sat Mar 21 21:58:24 WET 2026
