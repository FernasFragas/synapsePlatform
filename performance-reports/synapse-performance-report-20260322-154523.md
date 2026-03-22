# synapsePlatform Performance Test Report

**Test Date:** Sun Mar 22 15:45:24 WET 2026
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
Process Stats:   0.1  0.1  26464 437336400
Database: 109800|0|67.9609375
Events in DB: 109800
Kafka LAG: 
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 101s |
| **Messages Sent** | 600 |
| **Messages Processed** | 577 |
| **Failed Messages** | 0 |
| **Success Rate** | 96.17% |
| **Actual Throughput** | 5.7 msg/sec |
| **Peak Kafka LAG** | 41 |
| **Average Kafka LAG** | 19 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  37952 437540448 |

**Analysis:**
- ⚠️  Success rate < 100% - check failed_messages table

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 268s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 22.4 msg/sec |
| **Peak Kafka LAG** | 56 |
| **Average Kafka LAG** | 29 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  55520 437560464 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 987s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30012 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.04% |
| **Actual Throughput** | 30.4 msg/sec |
| **Peak Kafka LAG** | 64 |
| **Average Kafka LAG** | 30 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.2  94272 437669936 |

**Analysis:**
- 📊 Observed throughput ceiling: ~30.4 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 146389 |
| **Total Failed** | 0 |
| **Database Stats** | 146389|0|90.74609375 |
| **List Query (20 items)** | 14ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```

```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 5.7 msg/sec | 96.17% | 41 | 19 |
| Test 2 | 100 msg/sec | 22.4 msg/sec | 100.00% | 56 | 29 |
| Test 3 | 500 msg/sec | 30.4 msg/sec | 100.04% | 64 | 30 |

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

---

## Debug Information

- **Debug log:** ./performance-reports/debug-20260322-154523.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260322-154523.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260322-154523.log` for detailed timeline

---

**Test completed at:** Sun Mar 22 16:08:51 WET 2026
