# Performance Comparison Chart

## Throughput Trend (Test 2: 100 msg/sec target)

| 2026-03-28T19:01:14+00:00 | ████ 20.0 msg/sec |
| 20260328-160232 | ████ 20.3 msg/sec |
| 20260322-232108 | █████ 22.6 msg/sec |
| 20260322-154523 | ████ 22.4 msg/sec |
| 20260322-133632 | ████ 22.0 msg/sec |
| 20260322-130117 | ████ 21.9 msg/sec |
| 20260322-113350 | █████ 23.5 msg/sec |
| 20260322-091418 | █████ 23.9 msg/sec |
| 20260321-225451 | █████ 23.8 msg/sec |

## Recent Performance Metrics

| Date | Test 2 Throughput | Success | Peak LAG | Avg LAG | Query Time | Status |
|------|-------------------|---------|----------|---------|------------|--------|

## Performance Insights

### Throughput Analysis
- **Target:** 100 msg/sec (Test 2)
- **Best Run:** See table above
- **Trend:** Check if throughput is improving over time

### Recommendations
- 🔴 **< 25 msg/sec:** Critical bottleneck - implement batching + worker pools
- 🟠 **25-50 msg/sec:** Moderate - add batching or increase workers
- 🟡 **50-100 msg/sec:** Good - fine-tune configuration
- 🟢 **> 100 msg/sec:** Excellent - meeting target!

