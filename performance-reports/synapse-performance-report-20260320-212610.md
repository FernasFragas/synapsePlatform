# synapsePlatform Performance Test Report

**Test Date:** Fri Mar 20 21:26:10 WET 2026
**Git Commit:** 2c5889d
**Machine:** arm64
**OS:** Darwin 25.3.0

---

## Baseline Metrics

```
Process Stats:   0.0  0.1  26672 437139360
Database: 15754|106|7.24609375
Events in DB: 15754
```

---

## Test Results

### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | 70s |
| **Messages Sent** | 600 |
| **Messages Processed** | 443 |
| **Failed Messages** | 106 |
| **Success Rate** | 73.83% |
| **Actual Throughput** | 6.3 msg/sec |
| **Process Stats** |   0.0  0.1  35664 437602112 |

### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | 70s |
| **Messages Sent** | 6000 |
| **Messages Processed** | 1639 |
| **Failed Messages** | 106 |
| **Success Rate** | 27.32% |
| **Actual Throughput** | 23.4 msg/sec |
| **Process Stats** |   0.0  0.1  40128 437603968 |

### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | 70s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | 2377 |
| **Failed Messages** | 106 |
| **Success Rate** | 7.92% |
| **Actual Throughput** | 34.0 msg/sec |
| **Process Stats** |   0.0  0.1  40944 437604160 |

---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | 20213 |
| **Total Failed** | 106 |
| **Database Stats** | 20213|106|9.26171875 |
| **List Query (20 items)** | 28ms |

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate |
|------|-------------|-------------------|--------------|
| Test 1 | 10 msg/sec | 6.3 msg/sec | 73.83% |
| Test 2 | 100 msg/sec | 23.4 msg/sec | 27.32% |
| Test 3 | 500 msg/sec | 34.0 msg/sec | 7.92% |

---

## Observations

- ⚠️  **106 failed messages** - check validation or transformation logic

---

**Test completed at:** Fri Mar 20 21:30:30 WET 2026
