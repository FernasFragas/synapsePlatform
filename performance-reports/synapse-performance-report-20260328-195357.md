# synapsePlatform Performance Test Report

**Test Date:** Sat Mar 28 19:53:58 WET 2026
**Git Commit:** 737622e
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
Process Stats:   0.5  0.1  25600 444245440
Database: 116648|0|72.1484375
Events in DB: 116648
Kafka LAG: 0
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 93s |
| **Messages Sent** | 600 |
| **Messages Processed** | 600 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 6.5 msg/sec |
| **Peak Kafka LAG** | 6 |
| **Average Kafka LAG** | 2 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.3  0.1  42912 444459856 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 297s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 20.2 msg/sec |
| **Peak Kafka LAG** | 9 |
| **Average Kafka LAG** | 5 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.5  0.2  61280 444551664 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 1151s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30000 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 26.1 msg/sec |
| **Peak Kafka LAG** | 28 |
| **Average Kafka LAG** | 16 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.3  0.3 105680 444599376 |

**Analysis:**
- 📊 Observed throughput ceiling: ~26.1 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 153248 |
| **Total Failed** | 0 |
| **Database Stats** | 153248|0|94.99609375 |
| **List Query (20 items)** | 19ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```

```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 6.5 msg/sec | 100.00% | 6 | 2 |
| Test 2 | 100 msg/sec | 20.2 msg/sec | 100.00% | 9 | 5 |
| Test 3 | 500 msg/sec | 26.1 msg/sec | 100.00% | 28 | 16 |

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

- **Debug log:** ./performance-reports/debug-20260328-195357.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260328-195357.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260328-195357.log` for detailed timeline

---

**Test completed at:** Sat Mar 28 20:20:31 WET 2026
