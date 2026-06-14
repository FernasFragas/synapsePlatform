# synapsePlatform Performance Test Report

**Test Date:** Sat Jun 13 21:44:33 WEST 2026
**Git Commit:** a27db6f
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
Process Stats:   0.1  0.1  31424 436676448
Database: 6|0|0.0546875
Events in DB: 6
Kafka LAG: 6
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
| **Peak Kafka LAG** | 606 |
| **Average Kafka LAG** | 336 |
| **Final Kafka LAG** | 606 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.1  0.1  36224 436685776 |

**Analysis:**
- ⚠️  Peak LAG exceeded 100 - consumer falling behind even at low load

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 144s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 0 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 41.7 msg/sec |
| **Peak Kafka LAG** | 6606 |
| **Average Kafka LAG** | 3761 |
| **Final Kafka LAG** | 6606 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.2  0.1  50096 436694016 |

**Analysis:**
- 🚨 Peak LAG exceeded 1000 - severe bottleneck detected
- ⚠️  Average LAG > 500 - consumer consistently falling behind

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 277s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 27024 |
| **Failed Messages** | 0 |
| **Success Rate** | 90.08% |
| **Actual Throughput** | 97.6 msg/sec |
| **Peak Kafka LAG** | 36606 |
| **Average Kafka LAG** | 23057 |
| **Final Kafka LAG** | 36606 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test3...
       2 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.2  0.2 109088 436756208 |

**Analysis:**
- 🚨 Peak LAG exceeded 10,000 - critical bottleneck
- 🚨 Final LAG still > 1000 after 30s wait - backlog not clearing
- 📊 Observed throughput ceiling: ~97.6 msg/sec

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 33630 |
| **Total Failed** | 0 |
| **Database Stats** | 33630|0|19.6484375 |
| **List Query (20 items)** | 16ms |
| **Final Kafka LAG** | 36606 |

### Failed Messages Breakdown
```

```

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | 8.2 msg/sec | 100.00% | 606 | 336 |
| Test 2 | 100 msg/sec | 41.7 msg/sec | 100.00% | 6606 | 3761 |
| Test 3 | 500 msg/sec | 97.6 msg/sec | 90.08% | 36606 | 23057 |

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

### 🔴 Severe Throughput Bottleneck
- **Peak LAG:** 36606 messages
- **Throughput ceiling:** ~97.6 msg/sec
- **Root cause:** Single-threaded processing + slow SQLite writes
- **Recommended fixes:**
  1. Enable WAL mode (immediate, 3-5x improvement)
  2. Implement batch inserts (4 hours, 10-20x improvement)
  3. Add worker pool (6 hours, 5-10x improvement)

---

## Debug Information

- **Debug log:** ./performance-reports/debug-20260613-214432.log
- **Kafka LAG log:** ./performance-reports/kafka-lag-20260613-214432.log
- **Full diagnostics:** Run `cat ./performance-reports/debug-20260613-214432.log` for detailed timeline

---

**Test completed at:** Sat Jun 13 21:53:38 WEST 2026
