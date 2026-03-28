# Performance Comparison Chart

## Throughput Trend (Test 2: 100 msg/sec target)

| 2026-03-28T20:49:05+00:00 | ████ 20.3 msg/sec |

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

