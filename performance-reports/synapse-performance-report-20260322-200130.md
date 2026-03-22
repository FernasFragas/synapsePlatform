# synapsePlatform Performance Test Report

**Test Date:** Sun Mar 22 20:01:31 WET 2026
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
Process Stats:   0.0  0.1  20672 437336032
Database: 330506|1|205.27734375
Events in DB: 330506
Kafka LAG: -
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 102s |
| **Messages Sent** | 600 |
| **Messages Processed** | 576 |
| **Failed Messages** | 1 |
| **Success Rate** | 96.00% |
| **Actual Throughput** | 5.6 msg/sec |
| **Peak Kafka LAG** | 38 |
| **Average Kafka LAG** | 17 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.3  0.1  39376 437475984 |

**Analysis:**
- ⚠️  Success rate < 100% - check failed_messages table

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 258s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 5976 |
| **Failed Messages** | 1 |
| **Success Rate** | 99.60% |
| **Actual Throughput** | 23.2 msg/sec |
| **Peak Kafka LAG** | 52 |
| **Average Kafka LAG** | 27 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.2  75200 437579984 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 945s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30024 |
| **Failed Messages** | 1 |
| **Success Rate** | 100.08% |
| **Actual Throughput** | 31.8 msg/sec |
| **Peak Kafka LAG** | 66 |
| **Average Kafka LAG** | 32 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.3 123840 437695184 |

**Analysis:**
- 📊 Observed throughput ceiling: ~31.8 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 367106 |
| **Total Failed** | 1 |
| **Database Stats** | 367106|1|227.99609375 |
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
| Test 1 | 10 msg/sec | 5.6 msg/sec | 96.00% | 38 | 17 |
| Test 2 | 100 msg/sec | 23.2 msg/sec | 99.60% | 52 | 27 |
| Test 3 | 500 msg/sec | 31.8 msg/sec | 100.08% | 66 | 32 |

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

- **Debug log:** ./performance-reports/debug-20260322-200130.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260322-200130.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260322-200130.log` for detailed timeline

---

**Test completed at:** Sun Mar 22 20:24:07 WET 2026
