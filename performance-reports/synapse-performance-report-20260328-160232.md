# synapsePlatform Performance Test Report

**Test Date:** Sat Mar 28 16:02:33 WET 2026
**Git Commit:** e9ad3d6
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
Process Stats:   0.5  0.1  27440 444245520
Database: 43448|0|26.68359375
Events in DB: 43448
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
| **Peak Kafka LAG** | 7 |
| **Average Kafka LAG** | 3 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.3  0.1  39600 444523360 |

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
| **Peak Kafka LAG** | 18 |
| **Average Kafka LAG** | 9 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.3  0.1  52080 444543424 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 1145s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30000 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 26.2 msg/sec |
| **Peak Kafka LAG** | 28 |
| **Average Kafka LAG** | 15 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.1  0.2  75280 444587440 |

**Analysis:**
- 📊 Observed throughput ceiling: ~26.2 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 80048 |
| **Total Failed** | 0 |
| **Database Stats** | 80048|0|49.52734375 |
| **List Query (20 items)** | 16ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```

```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 6.5 msg/sec | 100.00% | 7 | 3 |
| Test 2 | 100 msg/sec | 20.3 msg/sec | 100.00% | 18 | 9 |
| Test 3 | 500 msg/sec | 26.2 msg/sec | 100.00% | 28 | 15 |

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

- **Debug log:** ./performance-reports/debug-20260328-160232.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260328-160232.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260328-160232.log` for detailed timeline

---

**Test completed at:** Sat Mar 28 16:29:00 WET 2026
