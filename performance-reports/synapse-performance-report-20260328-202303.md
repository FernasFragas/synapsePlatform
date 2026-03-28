# synapsePlatform Performance Test Report

**Test Date:** Sat Mar 28 20:23:04 WET 2026
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
Process Stats:   0.3  0.1  27488 444250192
Database: 153248|0|94.99609375
Events in DB: 153248
Kafka LAG: 0
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 94s |
| **Messages Sent** | 600 |
| **Messages Processed** | 600 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 6.4 msg/sec |
| **Peak Kafka LAG** | 7 |
| **Average Kafka LAG** | 3 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.3  0.1  41504 444460128 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 296s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 20.3 msg/sec |
| **Peak Kafka LAG** | 22 |
| **Average Kafka LAG** | 7 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.3  0.2  65120 444489744 |

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
| **Peak Kafka LAG** | 27 |
| **Average Kafka LAG** | 16 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.2  0.3 109344 444599264 |

**Analysis:**
- 📊 Observed throughput ceiling: ~26.1 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 189848 |
| **Total Failed** | 0 |
| **Database Stats** | 189848|0|117.6796875 |
| **List Query (20 items)** | 15ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```

```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 6.4 msg/sec | 100.00% | 7 | 3 |
| Test 2 | 100 msg/sec | 20.3 msg/sec | 100.00% | 22 | 7 |
| Test 3 | 500 msg/sec | 26.1 msg/sec | 100.00% | 27 | 16 |

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

- **Debug log:** ./performance-reports/debug-20260328-202303.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260328-202303.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260328-202303.log` for detailed timeline

---

**Test completed at:** Sat Mar 28 20:49:36 WET 2026
