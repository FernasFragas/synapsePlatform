# synapsePlatform Performance Test Report

**Test Date:** Sun Mar 22 18:37:30 WET 2026
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
Process Stats:   0.0  0.1  27280 437402512
Database: 256189|1|159.0078125
Events in DB: 256189
Kafka LAG: -
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 97s |
| **Messages Sent** | 600 |
| **Messages Processed** | 600 |
| **Failed Messages** | 1 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 6.2 msg/sec |
| **Peak Kafka LAG** | 35 |
| **Average Kafka LAG** | 16 |
| **Final Kafka LAG** | 22 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  39056 437542272 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 253s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 1 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 23.7 msg/sec |
| **Peak Kafka LAG** | 54 |
| **Average Kafka LAG** | 29 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.2  70496 437641136 |

**Analysis:**

