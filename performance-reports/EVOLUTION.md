# Performance Test Evolution Report

This report compares every historical performance test report available in `performance-reports`.

Reports compared: 18
Oldest report: 20260320-212610
Latest report: 20260614-130006

## Test 1: Low Load (10 msg/sec)

| Run | Report | Target Rate | Duration | Messages Sent | Messages Consumed | Messages Inserted | Duplicate Messages Ignored | Failed Messages | Kafka Messages Committed | Insert Success Rate | Commit Rate | Actual Throughput | Peak Kafka LAG | Average Kafka LAG | Final Kafka LAG | Error Count | Process Stats (CPU% MEM% RSS VSZ) |
|-----|--------|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 20260320-212610 | [synapse-performance-report-20260320-212610.md](./synapse-performance-report-20260320-212610.md) | 10 msg/sec | 70s | 600 | N/A | 443 | N/A | 106 | N/A | 73.83% | N/A | 6.3 msg/sec | N/A | N/A | N/A | N/A | 0.0 0.1 35664 437602112 |
| 20260320-222724 | [synapse-performance-report-20260320-222724.md](./synapse-performance-report-20260320-222724.md) | 10 msg/sec | 95s | 600 | N/A | 600 | N/A | 106 | N/A | 100.00% | N/A | 6.3 msg/sec | 42 | 18 | 0 | 0 | 0.0 0.1 37776 437608256 |
| 20260321-213605 | [synapse-performance-report-20260321-213605.md](./synapse-performance-report-20260321-213605.md) | 10 msg/sec | 103s | 600 | N/A | 600 | N/A | 106 | N/A | 100.00% | N/A | 5.8 msg/sec | 40 | 21 | 0 | 0 | 0.0 0.1 37040 437538400 |
| 20260321-220421 | [synapse-performance-report-20260321-220421.md](./synapse-performance-report-20260321-220421.md) | 10 msg/sec | 96s | 600 | N/A | 600 | N/A | 106 | N/A | 100.00% | N/A | 6.2 msg/sec | 41 | 19 | 0 | 0 | 0.0 0.1 40448 437540400 |
| 20260321-225451 | [synapse-performance-report-20260321-225451.md](./synapse-performance-report-20260321-225451.md) | 10 msg/sec | 94s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 6.4 msg/sec | 39 | 18 | 0 | 0 | 0.0 0.1 35744 437600400 |
| 20260322-091418 | [synapse-performance-report-20260322-091418.md](./synapse-performance-report-20260322-091418.md) | 10 msg/sec | 92s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 6.5 msg/sec | 41 | 21 | 0 | 0 | 0.0 0.1 32816 437535104 |
| 20260322-113350 | [synapse-performance-report-20260322-113350.md](./synapse-performance-report-20260322-113350.md) | 10 msg/sec | 94s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 6.4 msg/sec | 42 | 16 | 0 | 0 | 0.0 0.1 33744 437534720 |
| 20260322-130117 | [synapse-performance-report-20260322-130117.md](./synapse-performance-report-20260322-130117.md) | 10 msg/sec | 97s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 6.2 msg/sec | 40 | 19 | 0 | 0 | 0.0 0.2 86800 437594608 |
| 20260322-133632 | [synapse-performance-report-20260322-133632.md](./synapse-performance-report-20260322-133632.md) | 10 msg/sec | 99s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 6.1 msg/sec | 39 | 19 | 0 | 0 | 0.0 0.4 142528 437716320 |
| 20260322-154523 | [synapse-performance-report-20260322-154523.md](./synapse-performance-report-20260322-154523.md) | 10 msg/sec | 101s | 600 | N/A | 577 | N/A | 0 | N/A | 96.17% | N/A | 5.7 msg/sec | 41 | 19 | 0 | 0 | 0.0 0.1 37952 437540448 |
| 20260322-232108 | [synapse-performance-report-20260322-232108.md](./synapse-performance-report-20260322-232108.md) | 10 msg/sec | 103s | 600 | N/A | 600 | N/A | 333 | N/A | 100.00% | N/A | 5.8 msg/sec | 8 | 3 | 0 | 0 | 0.1 0.1 39552 437476864 |
| 20260328-160232 | [synapse-performance-report-20260328-160232.md](./synapse-performance-report-20260328-160232.md) | 10 msg/sec | 93s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 6.5 msg/sec | 7 | 3 | 0 | 0 | 0.3 0.1 39600 444523360 |
| 20260328-183514 | [synapse-performance-report-20260328-183514.md](./synapse-performance-report-20260328-183514.md) | 10 msg/sec | 93s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 6.5 msg/sec | 7 | 4 | 0 | 0 | 0.6 0.1 42160 444529056 |
| 20260328-195357 | [synapse-performance-report-20260328-195357.md](./synapse-performance-report-20260328-195357.md) | 10 msg/sec | 93s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 6.5 msg/sec | 6 | 2 | 0 | 0 | 0.3 0.1 42912 444459856 |
| 20260328-202303 | [synapse-performance-report-20260328-202303.md](./synapse-performance-report-20260328-202303.md) | 10 msg/sec | 94s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 6.4 msg/sec | 7 | 3 | 0 | 0 | 0.3 0.1 41504 444460128 |
| 20260613-214432 | [synapse-performance-report-20260613-214432.md](./synapse-performance-report-20260613-214432.md) | 10 msg/sec | 73s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 8.2 msg/sec | 606 | 336 | 606 | 0 | 0.1 0.1 36224 436685776 |
| 20260614-094707 | [synapse-performance-report-20260614-094707.md](./synapse-performance-report-20260614-094707.md) | 10 msg/sec | 73s | 600 | N/A | 600 | N/A | 0 | N/A | 100.00% | N/A | 8.2 msg/sec | 9 | 2 | 0 | 0 | 0.0 0.1 39952 436690528 |
| 20260614-130006 | [synapse-performance-report-20260614-130006.md](./synapse-performance-report-20260614-130006.md) | 10 msg/sec | 63s | 600 | 600 | 600 | 0 | 0 | 600 | 100.00% | 100.00% | 9.5 msg/sec | 0 | 0 | 0 | 0 | 0.1 0.1 35472 436743088 |

Conclusion: Actual Throughput improved, moving up from 6.3 (20260320-212610) to 9.5 (20260614-130006); Insert Success Rate improved, moving up from 73.83 (20260320-212610) to 100 (20260614-130006); Failed Messages improved, moving down from 106 (20260320-212610) to 0 (20260614-130006); Peak Kafka LAG improved, moving down from 42 (20260320-222724) to 0 (20260614-130006); Final Kafka LAG stayed stable at 0.

## Test 2: Medium Load (100 msg/sec)

| Run | Report | Target Rate | Duration | Messages Sent | Messages Consumed | Messages Inserted | Duplicate Messages Ignored | Failed Messages | Kafka Messages Committed | Insert Success Rate | Commit Rate | Actual Throughput | Peak Kafka LAG | Average Kafka LAG | Final Kafka LAG | Error Count | Process Stats (CPU% MEM% RSS VSZ) |
|-----|--------|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 20260320-212610 | [synapse-performance-report-20260320-212610.md](./synapse-performance-report-20260320-212610.md) | 100 msg/sec | 70s | 6000 | N/A | 1639 | N/A | 106 | N/A | 27.32% | N/A | 23.4 msg/sec | N/A | N/A | N/A | N/A | 0.0 0.1 40128 437603968 |
| 20260320-222724 | [synapse-performance-report-20260320-222724.md](./synapse-performance-report-20260320-222724.md) | 100 msg/sec | 254s | 6000 | N/A | 6000 | N/A | 106 | N/A | 100.00% | N/A | 23.6 msg/sec | 57 | 29 | 0 | 1 | 0.0 0.1 39488 437608256 |
| 20260321-213605 | [synapse-performance-report-20260321-213605.md](./synapse-performance-report-20260321-213605.md) | 100 msg/sec | 249s | 6000 | N/A | 6000 | N/A | 106 | N/A | 100.00% | N/A | 24.1 msg/sec | 58 | 30 | 0 | 1 | 0.0 0.1 39728 437608592 |
| 20260321-220421 | [synapse-performance-report-20260321-220421.md](./synapse-performance-report-20260321-220421.md) | 100 msg/sec | 263s | 6000 | N/A | 6000 | N/A | 106 | N/A | 100.00% | N/A | 22.8 msg/sec | 57 | 30 | 0 | 1 | 0.0 0.1 53776 437622832 |
| 20260321-225451 | [synapse-performance-report-20260321-225451.md](./synapse-performance-report-20260321-225451.md) | 100 msg/sec | 252s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 23.8 msg/sec | 60 | 27 | 0 | 1 | 0.0 0.1 44128 437611216 |
| 20260322-091418 | [synapse-performance-report-20260322-091418.md](./synapse-performance-report-20260322-091418.md) | 100 msg/sec | 251s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 23.9 msg/sec | 57 | 28 | 0 | 1 | 0.0 0.1 45664 437548448 |
| 20260322-113350 | [synapse-performance-report-20260322-113350.md](./synapse-performance-report-20260322-113350.md) | 100 msg/sec | 255s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 23.5 msg/sec | 56 | 29 | 0 | 1 | 0.0 0.1 44448 437548320 |
| 20260322-130117 | [synapse-performance-report-20260322-130117.md](./synapse-performance-report-20260322-130117.md) | 100 msg/sec | 274s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 21.9 msg/sec | 56 | 28 | 0 | 1 | 0.0 0.3 95136 437669344 |
| 20260322-133632 | [synapse-performance-report-20260322-133632.md](./synapse-performance-report-20260322-133632.md) | 100 msg/sec | 273s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 22.0 msg/sec | 56 | 27 | 0 | 1 | 0.0 0.4 152192 437725408 |
| 20260322-154523 | [synapse-performance-report-20260322-154523.md](./synapse-performance-report-20260322-154523.md) | 100 msg/sec | 268s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 22.4 msg/sec | 56 | 29 | 0 | 1 | 0.0 0.1 55520 437560464 |
| 20260322-232108 | [synapse-performance-report-20260322-232108.md](./synapse-performance-report-20260322-232108.md) | 100 msg/sec | 266s | 6000 | N/A | 6000 | N/A | 333 | N/A | 100.00% | N/A | 22.6 msg/sec | 31 | 16 | 0 | 1 | 0.0 0.2 82128 437653088 |
| 20260328-160232 | [synapse-performance-report-20260328-160232.md](./synapse-performance-report-20260328-160232.md) | 100 msg/sec | 296s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 20.3 msg/sec | 18 | 9 | 0 | 1 | 0.3 0.1 52080 444543424 |
| 20260328-183514 | [synapse-performance-report-20260328-183514.md](./synapse-performance-report-20260328-183514.md) | 100 msg/sec | 300s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 20.0 msg/sec | 10 | 4 | 0 | 1 | 0.5 0.2 57392 444547024 |
| 20260328-195357 | [synapse-performance-report-20260328-195357.md](./synapse-performance-report-20260328-195357.md) | 100 msg/sec | 297s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 20.2 msg/sec | 9 | 5 | 0 | 1 | 0.5 0.2 61280 444551664 |
| 20260328-202303 | [synapse-performance-report-20260328-202303.md](./synapse-performance-report-20260328-202303.md) | 100 msg/sec | 296s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 20.3 msg/sec | 22 | 7 | 0 | 1 | 0.3 0.2 65120 444489744 |
| 20260613-214432 | [synapse-performance-report-20260613-214432.md](./synapse-performance-report-20260613-214432.md) | 100 msg/sec | 144s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 41.7 msg/sec | 6606 | 3761 | 6606 | 1 | 0.2 0.1 50096 436694016 |
| 20260614-094707 | [synapse-performance-report-20260614-094707.md](./synapse-performance-report-20260614-094707.md) | 100 msg/sec | 145s | 6000 | N/A | 6000 | N/A | 0 | N/A | 100.00% | N/A | 41.4 msg/sec | 46 | 12 | 0 | 1 | 0.1 0.1 52064 436716480 |
| 20260614-130006 | [synapse-performance-report-20260614-130006.md](./synapse-performance-report-20260614-130006.md) | 100 msg/sec | 73s | 6000 | 6000 | 6000 | 0 | 0 | 6000 | 100.00% | 100.00% | 82.2 msg/sec | 0 | 0 | 0 | 1 | 0.1 0.1 46816 436792912 |

Conclusion: Actual Throughput improved, moving up from 23.4 (20260320-212610) to 82.2 (20260614-130006); Insert Success Rate improved, moving up from 27.32 (20260320-212610) to 100 (20260614-130006); Failed Messages improved, moving down from 106 (20260320-212610) to 0 (20260614-130006); Peak Kafka LAG improved, moving down from 57 (20260320-222724) to 0 (20260614-130006); Final Kafka LAG stayed stable at 0.

## Test 3: High Load (500 msg/sec)

| Run | Report | Target Rate | Duration | Messages Sent | Messages Consumed | Messages Inserted | Duplicate Messages Ignored | Failed Messages | Kafka Messages Committed | Insert Success Rate | Commit Rate | Actual Throughput | Peak Kafka LAG | Average Kafka LAG | Final Kafka LAG | Error Count | Process Stats (CPU% MEM% RSS VSZ) |
|-----|--------|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 20260320-212610 | [synapse-performance-report-20260320-212610.md](./synapse-performance-report-20260320-212610.md) | 500 msg/sec | 70s | 30,000 | N/A | 2377 | N/A | 106 | N/A | 7.92% | N/A | 34.0 msg/sec | N/A | N/A | N/A | N/A | 0.0 0.1 40944 437604160 |
| 20260320-222724 | [synapse-performance-report-20260320-222724.md](./synapse-performance-report-20260320-222724.md) | 500 msg/sec | 921s | 30,000 | N/A | 30000 | N/A | 106 | N/A | 100.00% | N/A | 32.6 msg/sec | 64 | 33 | 0 | 2 | 0.0 0.1 41168 437608256 |
| 20260321-213605 | [synapse-performance-report-20260321-213605.md](./synapse-performance-report-20260321-213605.md) | 500 msg/sec | 933s | 30,000 | N/A | 30000 | N/A | 106 | N/A | 100.00% | N/A | 32.2 msg/sec | 66 | 32 | 0 | 2 | 0.0 0.1 40224 437608592 |
| 20260321-220421 | [synapse-performance-report-20260321-220421.md](./synapse-performance-report-20260321-220421.md) | 500 msg/sec | 949s | 30,000 | N/A | 30000 | N/A | 106 | N/A | 100.00% | N/A | 31.6 msg/sec | 66 | 33 | 0 | 2 | 0.0 0.2 89264 437657024 |
| 20260321-225451 | [synapse-performance-report-20260321-225451.md](./synapse-performance-report-20260321-225451.md) | 500 msg/sec | 945s | 30,000 | N/A | 30000 | N/A | 0 | N/A | 100.00% | N/A | 31.7 msg/sec | 66 | 31 | 0 | 2 | 0.0 0.2 76352 437644864 |
| 20260322-091418 | [synapse-performance-report-20260322-091418.md](./synapse-performance-report-20260322-091418.md) | 500 msg/sec | 965s | 30,000 | N/A | 30000 | N/A | 0 | N/A | 100.00% | N/A | 31.1 msg/sec | 65 | 32 | 0 | 2 | 0.0 0.2 91040 437659344 |
| 20260322-113350 | [synapse-performance-report-20260322-113350.md](./synapse-performance-report-20260322-113350.md) | 500 msg/sec | 1000s | 30,000 | N/A | 30000 | N/A | 0 | N/A | 100.00% | N/A | 30.0 msg/sec | 63 | 31 | 0 | 2 | 0.0 0.2 90224 437593648 |
| 20260322-130117 | [synapse-performance-report-20260322-130117.md](./synapse-performance-report-20260322-130117.md) | 500 msg/sec | 1039s | 30,000 | N/A | 30000 | N/A | 0 | N/A | 100.00% | N/A | 28.9 msg/sec | 60 | 31 | 0 | 2 | 0.0 0.4 143200 437715360 |
| 20260322-133632 | [synapse-performance-report-20260322-133632.md](./synapse-performance-report-20260322-133632.md) | 500 msg/sec | 998s | 30,000 | N/A | 30000 | N/A | 0 | N/A | 100.00% | N/A | 30.1 msg/sec | 64 | 31 | 0 | 2 | 0.0 0.5 181664 437753824 |
| 20260322-154523 | [synapse-performance-report-20260322-154523.md](./synapse-performance-report-20260322-154523.md) | 500 msg/sec | 987s | 30,000 | N/A | 30012 | N/A | 0 | N/A | 100.04% | N/A | 30.4 msg/sec | 64 | 30 | 0 | 2 | 0.0 0.2 94272 437669936 |
| 20260322-232108 | [synapse-performance-report-20260322-232108.md](./synapse-performance-report-20260322-232108.md) | 500 msg/sec | 987s | 30,000 | N/A | 30000 | N/A | 333 | N/A | 100.00% | N/A | 30.4 msg/sec | 42 | 20 | 0 | 2 | 0.0 0.4 138512 437708128 |
| 20260328-160232 | [synapse-performance-report-20260328-160232.md](./synapse-performance-report-20260328-160232.md) | 500 msg/sec | 1145s | 30,000 | N/A | 30000 | N/A | 0 | N/A | 100.00% | N/A | 26.2 msg/sec | 28 | 15 | 0 | 2 | 0.1 0.2 75280 444587440 |
| 20260328-183514 | [synapse-performance-report-20260328-183514.md](./synapse-performance-report-20260328-183514.md) | 500 msg/sec | 1144s | 30,000 | N/A | 30000 | N/A | 0 | N/A | 100.00% | N/A | 26.2 msg/sec | 28 | 15 | 0 | 2 | 0.5 0.3 102256 444590768 |
| 20260328-195357 | [synapse-performance-report-20260328-195357.md](./synapse-performance-report-20260328-195357.md) | 500 msg/sec | 1151s | 30,000 | N/A | 30000 | N/A | 0 | N/A | 100.00% | N/A | 26.1 msg/sec | 28 | 16 | 0 | 2 | 0.3 0.3 105680 444599376 |
| 20260328-202303 | [synapse-performance-report-20260328-202303.md](./synapse-performance-report-20260328-202303.md) | 500 msg/sec | 1151s | 30,000 | N/A | 30000 | N/A | 0 | N/A | 100.00% | N/A | 26.1 msg/sec | 27 | 16 | 0 | 2 | 0.2 0.3 109344 444599264 |
| 20260613-214432 | [synapse-performance-report-20260613-214432.md](./synapse-performance-report-20260613-214432.md) | 500 msg/sec | 277s | 30,000 | N/A | 27024 | N/A | 0 | N/A | 90.08% | N/A | 97.6 msg/sec | 36606 | 23057 | 36606 | 2 | 0.2 0.2 109088 436756208 |
| 20260614-094707 | [synapse-performance-report-20260614-094707.md](./synapse-performance-report-20260614-094707.md) | 500 msg/sec | 332s | 30,000 | N/A | 28539 | N/A | 0 | N/A | 95.13% | N/A | 86.0 msg/sec | 88 | 21 | 0 | 2 | 0.1 0.2 93456 436781536 |
| 20260614-130006 | [synapse-performance-report-20260614-130006.md](./synapse-performance-report-20260614-130006.md) | 500 msg/sec | 40s | 30000 | 30000 | 30000 | 0 | 0 | 30000 | 100.00% | 100.00% | 750.0 msg/sec | 0 | 0 | 0 | 2 | 0.0 0.2 90992 436835760 |

Conclusion: Actual Throughput improved, moving up from 34 (20260320-212610) to 750 (20260614-130006); Insert Success Rate improved, moving up from 7.92 (20260320-212610) to 100 (20260614-130006); Failed Messages improved, moving down from 106 (20260320-212610) to 0 (20260614-130006); Peak Kafka LAG improved, moving down from 64 (20260320-222724) to 0 (20260614-130006); Final Kafka LAG stayed stable at 0.

