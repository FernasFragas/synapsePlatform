# synapsePlatform Performance Test Report

**Test Date:** Sun Jun 14 09:47:09 WEST 2026
**Git Commit:** f47629a
**Machine:** arm64
**OS:** Darwin 25.5.0

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
Process Stats:   0.2  0.1  27888 436667024
Database: 33630|0|19.6484375
Events in DB: 33630
Kafka LAG: 0
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 73s |
| **Messages Sent** | 600 |
| **Messages Processed** | 600 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 8.2 msg/sec |
| **Peak Kafka LAG** | 9 |
| **Average Kafka LAG** | 2 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  39952 436690528 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 145s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 41.4 msg/sec |
| **Peak Kafka LAG** | 46 |
| **Average Kafka LAG** | 12 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.1  0.1  52064 436716480 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 332s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 28539 |
| **Failed Messages** | 0 |
| **Success Rate** | 95.13% |
| **Actual Throughput** | 86.0 msg/sec |
| **Peak Kafka LAG** | 88 |
| **Average Kafka LAG** | 21 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.1  0.2  93456 436781536 |

**Analysis:**
- 📊 Observed throughput ceiling: ~86.0 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 68769 |
| **Total Failed** | 0 |
| **Database Stats** | 68769|0|40.640625 |
| **List Query (20 items)** | 15ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```

```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 8.2 msg/sec | 100.00% | 9 | 2 |
| Test 2 | 100 msg/sec | 41.4 msg/sec | 100.00% | 46 | 12 |
| Test 3 | 500 msg/sec | 86.0 msg/sec | 95.13% | 88 | 21 |

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

- **Debug log:** ./performance-reports/debug-20260614-094707.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260614-094707.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260614-094707.log` for detailed timeline

---

**Test completed at:** Sun Jun 14 09:57:10 WEST 2026
