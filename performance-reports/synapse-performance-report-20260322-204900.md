# synapsePlatform Performance Test Report

**Test Date:** Sun Mar 22 20:49:02 WET 2026
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
Process Stats:   0.0  0.1  26992 437270960
Database: 367106|3|227.99609375
Events in DB: 367106
Kafka LAG: -
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 98s |
| **Messages Sent** | 600 |
| **Messages Processed** | 576 |
| **Failed Messages** | 3 |
| **Success Rate** | 96.00% |
| **Actual Throughput** | 5.9 msg/sec |
| **Peak Kafka LAG** | 40 |
| **Average Kafka LAG** | 18 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  34816 437541776 |

**Analysis:**
- ⚠️  Success rate < 100% - check failed_messages table

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 268s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 3 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 22.4 msg/sec |
| **Peak Kafka LAG** | 54 |
| **Average Kafka LAG** | 27 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.2  77008 437648064 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 996s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30024 |
| **Failed Messages** | 3 |
| **Success Rate** | 100.08% |
| **Actual Throughput** | 30.1 msg/sec |
| **Peak Kafka LAG** | 65 |
| **Average Kafka LAG** | 33 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.2  86240 437699456 |

**Analysis:**
- 📊 Observed throughput ceiling: ~30.1 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 403706 |
| **Total Failed** | 3 |
| **Database Stats** | 403706|3|250.828125 |
| **List Query (20 items)** | 15ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```
process|3
```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 5.9 msg/sec | 96.00% | 40 | 18 |
| Test 2 | 100 msg/sec | 22.4 msg/sec | 100.00% | 54 | 27 |
| Test 3 | 500 msg/sec | 30.1 msg/sec | 100.08% | 65 | 33 |

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
- **Count:** 3
- **Check:** `sqlite3 data.db "SELECT stage, error FROM failed_messages LIMIT 5;"`
- **Likely cause:** Invalid test data or validation errors

---

## Debug Information

- **Debug log:** ./performance-reports/debug-20260322-204900.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260322-204900.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260322-204900.log` for detailed timeline

---

**Test completed at:** Sun Mar 22 21:12:36 WET 2026
