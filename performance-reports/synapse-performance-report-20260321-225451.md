# synapsePlatform Performance Test Report

**Test Date:** Sat Mar 21 22:54:52 WET 2026
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
Process Stats:   0.0  0.1  27360 437270688
Database: 0|0|0.04296875
Events in DB: 0
Kafka LAG: -
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
| **Peak Kafka LAG** | 39 |
| **Average Kafka LAG** | 18 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  35744 437600400 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 252s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 23.8 msg/sec |
| **Peak Kafka LAG** | 60 |
| **Average Kafka LAG** | 27 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  44128 437611216 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 945s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 30000 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 31.7 msg/sec |
| **Peak Kafka LAG** | 66 |
| **Average Kafka LAG** | 31 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.2  76352 437644864 |

**Analysis:**
- 📊 Observed throughput ceiling: ~31.7 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 36600 |
| **Total Failed** | 0 |
| **Database Stats** | 36600|0|16.671875 |
| **List Query (20 items)** | 273ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```

```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 6.4 msg/sec | 100.00% | 39 | 18 |
| Test 2 | 100 msg/sec | 23.8 msg/sec | 100.00% | 60 | 27 |
| Test 3 | 500 msg/sec | 31.7 msg/sec | 100.00% | 66 | 31 |

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

- **Debug log:** ./performance-reports/debug-20260321-225451.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260321-225451.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260321-225451.log` for detailed timeline

---

**Test completed at:** Sat Mar 21 23:17:15 WET 2026
