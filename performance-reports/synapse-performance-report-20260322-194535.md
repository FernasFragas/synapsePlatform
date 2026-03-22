# synapsePlatform Performance Test Report

**Test Date:** Sun Mar 22 19:45:37 WET 2026
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
Process Stats:   0.0  0.1  27632 437337840
Database: 312962|1|194.30859375
Events in DB: 312962
Kafka LAG: 0
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
| **Failed Messages** | 1 |
| **Success Rate** | 96.00% |
| **Actual Throughput** | 5.8 msg/sec |
| **Peak Kafka LAG** | 39 |
| **Average Kafka LAG** | 20 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test1...
       0 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.1  39344 437411232 |

**Analysis:**
- ⚠️  Success rate < 100% - check failed_messages table

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 259s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 6000 |
| **Failed Messages** | 1 |
| **Success Rate** | 100.00% |
| **Actual Throughput** | 23.2 msg/sec |
| **Peak Kafka LAG** | 56 |
| **Average Kafka LAG** | 28 |
| **Final Kafka LAG** | 0 |
| **Error Count** | [0;32m[INFO][0m Sampling application logs for Test2...
       1 |
| **Process Stats (CPU% MEM% RSS VSZ)** |   0.0  0.2  74544 437645456 |

**Analysis:**

