# synapsePlatform Performance Test Report

**Test Date:** Sun Mar 22 21:22:20 WET 2026
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
Process Stats:   0.0  0.1  26848 437336272
Database: 403706|4|250.828125
Events in DB: 403706
Kafka LAG: 
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 99s |
| **Messages Sent** | 600 |
| **Messages Processed** | 576 |
| **Failed Messages** | 4 |
| **Success Rate** | 96.00% |
| **Actual Throughput** | 5.8 msg/sec |
| **Peak Kafka LAG** | 37 |
| **Average Kafka LAG** | 19 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  40176 437542032 |

**Analysis:**
- ⚠️  Success rate < 100% - check failed_messages table

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 269s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 4 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 22.3 msg/sec |
| **Peak Kafka LAG** | 56 |
| **Average Kafka LAG** | 29 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.2  0.2  79472 437650320 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 982s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30024 |
| **Failed Messages** | 4 |
| **Success Rate** | 100.08% |
| **Actual Throughput** | 30.6 msg/sec |
| **Peak Kafka LAG** | 65 |
| **Average Kafka LAG** | 32 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.4 133792 437703760 |

**Analysis:**
- 📊 Observed throughput ceiling: ~30.6 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 440306 |
| **Total Failed** | 4 |
| **Database Stats** | 440306|4|273.703125 |
| **List Query (20 items)** | 24ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```
process|4
```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 5.8 msg/sec | 96.00% | 37 | 19 |
| Test 2 | 100 msg/sec | 22.3 msg/sec | 100.00% | 56 | 29 |
| Test 3 | 500 msg/sec | 30.6 msg/sec | 100.08% | 65 | 32 |

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
- **Count:** 4
- **Check:** `sqlite3 data.db "SELECT stage, error FROM failed_messages LIMIT 5;"`
- **Likely cause:** Invalid test data or validation errors

---

## Debug Information

- **Debug log:** ./performance-reports/debug-20260322-212219.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260322-212219.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260322-212219.log` for detailed timeline

---

**Test completed at:** Sun Mar 22 21:45:42 WET 2026
