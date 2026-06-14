#!/bin/bash
# test/monitored_perf_test.sh
# Runs performance test with real-time monitoring

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="./performance-reports"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=========================================="
echo "Monitored Performance Test Suite"
echo -e "==========================================${NC}"
echo ""

# Check if app is running
if ! lsof -ti :8080 > /dev/null 2>&1; then
    echo -e "${GREEN}[INFO]${NC} Application not running. Start it first:"
    echo "  make run-with-logs"
    echo ""
    echo "Or run in background:"
    echo "  nohup make run > performance-reports/app-${TIMESTAMP}.log 2>&1 &"
    exit 1
fi

# Create monitoring script if it doesn't exist
MONITOR_SCRIPT="${SCRIPT_DIR}/performance_monitor.sh"

# Run performance test in background
echo -e "${GREEN}[INFO]${NC} Starting performance test..."
PERF_TIMESTAMP="$TIMESTAMP" "${SCRIPT_DIR}/perform_test.sh" > "${REPORT_DIR}/test-output-${TIMESTAMP}.log" 2>&1 &
TEST_PID=$!

# Run monitoring in parallel
echo -e "${GREEN}[INFO]${NC} Starting monitoring..."
PERF_TIMESTAMP="$TIMESTAMP" "${MONITOR_SCRIPT}" &
MONITOR_PID=$!

# Wait for test to complete
set +e
wait $TEST_PID
TEST_EXIT=$?
set -e

# Wait for monitoring to complete
wait $MONITOR_PID || true

echo ""
echo -e "${GREEN}=========================================="
echo "Test Complete!"
echo -e "==========================================${NC}"
echo ""

if [ $TEST_EXIT -eq 0 ]; then
    echo "✅ Performance test completed successfully"
else
    echo "❌ Performance test failed with exit code: $TEST_EXIT"
fi

echo ""
echo "📊 View results:"
echo "  - Performance report: cat ${REPORT_DIR}/latest.md"
echo "  - Monitoring summary: cat ${REPORT_DIR}/metrics-${TIMESTAMP}/MONITORING_SUMMARY.md"
echo "  - Test output: cat ${REPORT_DIR}/test-output-${TIMESTAMP}.log"
echo ""

exit $TEST_EXIT
