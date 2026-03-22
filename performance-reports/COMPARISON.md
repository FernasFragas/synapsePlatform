# Performance Comparison Chart

## Throughput Trend (Test 2: 100 msg/sec target)

| 20260322-232108 | █████ 22.6 msg/sec |
| 20260322-212219 | ████ 22.3 msg/sec |
| 20260322-204900 | ████ 22.4 msg/sec |
| 20260322-200130 | █████ 23.2 msg/sec |
| 20260322-185921 | ████ 22.5 msg/sec |
| 20260322-180931 | █████ 23.5 msg/sec |
| 20260322-172913 | █████ 23.3 msg/sec |

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

