# synapsePlatform Performance Test Report

**Test Date:** Sun Jun 14 13:00:07 WEST 2026
**Git Commit:** f47629a
**Machine:** arm64
**OS:** Darwin 25.5.0

---

## Pre-Flight Diagnostics

### SQLite Configuration
```
Journal Mode: [INFO] SQLite is in WAL mode
wal
Busy Timeout (CLI connection): [INFO] SQLite is in WAL mode
0ms
Synchronous (CLI connection): [INFO] SQLite is in WAL mode
1
```

Note: busy_timeout and synchronous are connection-local when read through sqlite3 CLI. The application sets its own pragmas at startup.

### Index Status
- Pagination Index (idx_ingested_event): **yes**

### Baseline Metrics
```
Process Stats: 0.1 0.1 28208 436733152
Database: 0|0|0.05859375
Events in DB: 0
Failed Messages: 0
Store Accounting: 0|0|0
Kafka LAG: 0
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 63s |
| **Messages Sent** | 600 |
| **Messages Consumed** | 600 |
| **Messages Inserted** | 600 |
| **Duplicate Messages Ignored** | 0 |
| **Failed Messages** | 0 |
| **Kafka Messages Committed** | 600 |
| **Insert Success Rate** | 100.00% |
| **Commit Rate** | 100.00% |
| **Actual Throughput** | 9.5 msg/sec |
| **Peak Kafka LAG** | 0 |
| **Average Kafka LAG** | 0 |
| **Final Kafka LAG** | 0 |
| **Error Count** | 0
0 |
| **Process Stats (CPU% MEM% RSS VSZ)** | 0.1 0.1 35472 436743088 |

**Analysis:**

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 73s |
| **Messages Sent** | 6000 |
| **Messages Consumed** | 6000 |
| **Messages Inserted** | 6000 |
| **Duplicate Messages Ignored** | 0 |
| **Failed Messages** | 0 |
| **Kafka Messages Committed** | 6000 |
| **Insert Success Rate** | 100.00% |
| **Commit Rate** | 100.00% |
| **Actual Throughput** | 82.2 msg/sec |
| **Peak Kafka LAG** | 0 |
| **Average Kafka LAG** | 0 |
| **Final Kafka LAG** | 0 |
| **Error Count** | 1 |
| **Process Stats (CPU% MEM% RSS VSZ)** | 0.1 0.1 46816 436792912 |

**Analysis:**

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 40s |
| **Messages Sent** | 30000 |
| **Messages Consumed** | 30000 |
| **Messages Inserted** | 30000 |
| **Duplicate Messages Ignored** | 0 |
| **Failed Messages** | 0 |
| **Kafka Messages Committed** | 30000 |
| **Insert Success Rate** | 100.00% |
| **Commit Rate** | 100.00% |
| **Actual Throughput** | 750.0 msg/sec |
| **Peak Kafka LAG** | 0 |
| **Average Kafka LAG** | 0 |
| **Final Kafka LAG** | 0 |
| **Error Count** | 2 |
| **Process Stats (CPU% MEM% RSS VSZ)** | 0.0 0.2 90992 436835760 |

**Analysis:**

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 36600 |
| **Total Failed** | 0 |
| **Database Stats** | 36600|0|22.1875 |
| **Store Accounting (attempted|inserted|duplicates)** | 36600|36600|0 |
| **List Query (20 items)** | 5ms |
| **Final Kafka LAG** | 0 |

### Failed Messages Breakdown
```

```

---

## Summary

| Test | Target Rate | Actual Throughput | Insert Success Rate | Peak LAG | Avg LAG | Final LAG |
|------|-------------|-------------------|---------------------|----------|---------|-----------|
| Test 1 | 10 msg/sec | 9.5 msg/sec | 100.00% | 0 | 0 | 0 |
| Test 2 | 100 msg/sec | 82.2 msg/sec | 100.00% | 0 | 0 | 0 |
| Test 3 | 500 msg/sec | 750.0 msg/sec | 100.00% | 0 | 0 | 0 |

---

## Bottleneck Analysis

### Critical: SQLite Not in WAL Mode
- Current mode: [INFO] SQLite is in WAL mode
wal
- Fix: enable PRAGMA journal_mode=WAL on the application connection.

---

## Monitoring Artifacts

- Debug log: ./performance-reports/debug-20260614-130006.log
- Kafka lag log: ./performance-reports/kafka-lag-20260614-130006.log
- Process stats logs: ./performance-reports/process-stats-test*-20260614-130006.log
- Structured metrics: ./performance-reports/metrics-20260614-130006.json

For batch duration, batch size, channel utilization, WAL pages, CPU, memory, and goroutines, run:
```bash
test/monitored_perf_test.sh
```

---

**Test completed at:** Sun Jun 14 13:03:54 WEST 2026
