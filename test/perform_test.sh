#!/bin/bash
set -e  # Exit on error

# ============================================================================
# synapsePlatform Performance Test Suite with Real-Time Monitoring
# ============================================================================

# Configuration
REPORT_DIR="./performance-reports"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
REPORT_FILE="${REPORT_DIR}/synapse-performance-report-${TIMESTAMP}.md"
DEBUG_LOG="${REPORT_DIR}/debug-${TIMESTAMP}.log"
LAG_LOG="${REPORT_DIR}/kafka-lag-${TIMESTAMP}.log"
APP_PORT=8080
KAFKA_BROKER="localhost:9092"
KAFKA_TOPIC="ingestion.raw"
DB_PATH="data.db"

cleanup() {
    log_info "Cleaning up background processes..."
    if [ ! -z "$LAG_MONITOR_PID" ]; then
        kill $LAG_MONITOR_PID 2>/dev/null || true
    fi
    if [ ! -z "$PROCESS_MONITOR_PID" ]; then
        kill $PROCESS_MONITOR_PID 2>/dev/null || true
    fi
    # Kill any remaining monitor processes
    pkill -P $$ 2>/dev/null || true
}
trap cleanup EXIT INT TERM


# Create reports directory if it doesn't exist
mkdir -p "$REPORT_DIR"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================================================
# Helper Functions
# ============================================================================

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" | tee -a "$DEBUG_LOG"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$DEBUG_LOG"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$DEBUG_LOG"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1" >> "$DEBUG_LOG"
}

# Check if app is running
check_app() {
    if ! lsof -ti :$APP_PORT > /dev/null 2>&1; then
        log_error "Application not running on port $APP_PORT"
        exit 1
    fi
    APP_PID=$(lsof -ti :$APP_PORT)
    log_info "Application is running (PID: $APP_PID)"
}

# Get process stats
get_process_stats() {
    local pid=$(lsof -ti :$APP_PORT)
    ps -p $pid -o %cpu,%mem,rss,vsz | tail -1
}

# Get database stats
get_db_stats() {
    sqlite3 $DB_PATH << 'SQL'
SELECT
    (SELECT COUNT(*) FROM events) as total_events,
    (SELECT COUNT(*) FROM failed_messages) as failed_messages,
    (SELECT page_count * page_size / 1024.0 / 1024.0 FROM pragma_page_count(), pragma_page_size()) as db_size_mb;
SQL
}

# Get Kafka lag
get_kafka_lag() {
    docker exec synapseplatform-kafka-1 kafka-consumer-groups \
        --bootstrap-server localhost:9092 \
        --group synapse-platform-consumer \
        --describe 2>/dev/null | grep ingestion.raw | awk '{print $6}' || echo "N/A"
}

# Monitor Kafka lag in background
monitor_kafka_lag() {
    local test_name=$1
    > "$LAG_LOG"  # Clear log

    while true; do
        LAG=$(get_kafka_lag)
        TIMESTAMP=$(date +%H:%M:%S)
        echo "$TIMESTAMP $LAG" >> "$LAG_LOG"
        log_debug "[$test_name] Kafka LAG: $LAG"
        sleep 1
    done
}

# Stop lag monitoring and get stats
stop_lag_monitoring() {
    if [ ! -z "$LAG_MONITOR_PID" ]; then
        kill $LAG_MONITOR_PID 2>/dev/null || true
        wait $LAG_MONITOR_PID 2>/dev/null || true
    fi

    if [ -f "$LAG_LOG" ]; then
        PEAK_LAG=$(cat "$LAG_LOG" | awk '{print $2}' | grep -v "N/A" | sort -n | tail -1)
        AVG_LAG=$(cat "$LAG_LOG" | awk '{sum+=$2; count++} END {if(count>0) print int(sum/count); else print 0}')
        FINAL_LAG=$(tail -1 "$LAG_LOG" | awk '{print $2}')

        echo "$PEAK_LAG|$AVG_LAG|$FINAL_LAG"
    else
        echo "0|0|0"
    fi
}

# Monitor process stats in background
monitor_process_stats() {
    local test_name=$1
    local stats_log="${REPORT_DIR}/process-stats-${test_name}-${TIMESTAMP}.log"
    > "$stats_log"

    while true; do
        STATS=$(get_process_stats)
        TIMESTAMP=$(date +%H:%M:%S)
        echo "$TIMESTAMP $STATS" >> "$stats_log"
        sleep 2
    done
}

# Stop process monitoring and get stats
stop_process_monitoring() {
    if [ ! -z "$PROCESS_MONITOR_PID" ]; then
        kill $PROCESS_MONITOR_PID 2>/dev/null || true
        wait $PROCESS_MONITOR_PID 2>/dev/null || true
    fi
}

# Check SQLite configuration
check_sqlite_config() {
    log_info "Checking SQLite configuration..."

    JOURNAL_MODE=$(sqlite3 $DB_PATH "PRAGMA journal_mode;")
    BUSY_TIMEOUT=$(sqlite3 $DB_PATH "PRAGMA busy_timeout;")
    SYNCHRONOUS=$(sqlite3 $DB_PATH "PRAGMA synchronous;")

    log_debug "SQLite journal_mode: $JOURNAL_MODE"
    log_debug "SQLite busy_timeout: $BUSY_TIMEOUT"
    log_debug "SQLite synchronous: $SYNCHRONOUS"

    if [ "$JOURNAL_MODE" != "wal" ]; then
        log_warn "⚠️  SQLite is NOT in WAL mode (current: $JOURNAL_MODE) - this limits write performance"
    else
        log_info "✅ SQLite is in WAL mode"
    fi

    if [ "$BUSY_TIMEOUT" = "0" ]; then
        log_warn "⚠️  SQLite busy_timeout is 0 - writes will fail immediately on contention"
    fi

    echo "$JOURNAL_MODE|$BUSY_TIMEOUT|$SYNCHRONOUS"
}

# Check for missing indexes
check_indexes() {
    log_info "Checking database indexes..."

    INDEXES=$(sqlite3 $DB_PATH "SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='events';")

    if echo "$INDEXES" | grep -q "idx_ingested_event"; then
        log_info "✅ Pagination index (idx_ingested_event) exists"
        HAS_PAGINATION_INDEX="yes"
    else
        log_warn "⚠️  Missing pagination index (idx_ingested_event) - queries will be slow"
        HAS_PAGINATION_INDEX="no"
    fi

    echo "$HAS_PAGINATION_INDEX"
}

# Update INDEX.md with test results
update_index() {
    local timestamp=$1
    local test2_throughput=$2
    local test2_success=$3
    local peak_lag=$4
    local avg_lag=$5

    local INDEX_FILE="${REPORT_DIR}/INDEX.md"
    local TEMP_FILE="${REPORT_DIR}/.index.tmp"

    log_info "Updating INDEX.md..."

    # Create INDEX.md if it doesn't exist
    if [ ! -f "$INDEX_FILE" ]; then
        cat > "$INDEX_FILE" << 'EOF'
# Performance Test History

| Date | Report | Test 2 Throughput | Success Rate | Peak LAG | Avg LAG | Query Latency |
|------|--------|-------------------|--------------|----------|---------|---------------|
EOF
    fi

    # Extract the header
    head -n 3 "$INDEX_FILE" > "$TEMP_FILE"

    # Add new entry at the top (most recent first)
    echo "| $timestamp | [Report](./synapse-performance-report-${timestamp}.md) | ${test2_throughput} msg/sec | ${test2_success}% | $peak_lag | $avg_lag | ${QUERY_LATENCY}ms |" >> "$TEMP_FILE"

    # Append existing entries (skip header)
    if [ $(wc -l < "$INDEX_FILE") -gt 3 ]; then
        tail -n +4 "$INDEX_FILE" >> "$TEMP_FILE"
    fi

    # Replace old INDEX with new one
    mv "$TEMP_FILE" "$INDEX_FILE"

    log_info "✅ INDEX.md updated"
}

# Generate comparison chart for last N runs
generate_comparison_chart() {
    local INDEX_FILE="${REPORT_DIR}/INDEX.md"
    local CHART_FILE="${REPORT_DIR}/COMPARISON.md"

    log_info "Generating comparison chart..."

    # Check if we have enough data
    local entry_count=$(tail -n +4 "$INDEX_FILE" 2>/dev/null | wc -l)
    if [ "$entry_count" -lt 2 ]; then
        log_debug "Not enough data for comparison (need at least 2 runs)"
        return
    fi

    # Extract last 10 runs for comparison
    cat > "$CHART_FILE" << 'EOF'
# Performance Comparison Chart

## Throughput Trend (Test 2: 100 msg/sec target)

EOF

    # Parse INDEX.md and create ASCII chart
    tail -n +4 "$INDEX_FILE" | head -n 10 | while IFS='|' read -r _ date _ throughput _ success _ peak _ avg _ latency _; do
        # Clean up whitespace
        date=$(echo "$date" | xargs)
        throughput=$(echo "$throughput" | sed 's/ msg\/sec//' | xargs)
        success=$(echo "$success" | sed 's/%//' | xargs)
        peak=$(echo "$peak" | xargs)
        avg=$(echo "$avg" | xargs)

        # Create bar chart (1 block = 5 msg/sec)
        bars=$(awk "BEGIN {printf \"%.0f\", $throughput / 5}")
        bar_str=$(printf '█%.0s' $(seq 1 $bars))

        echo "| $date | $bar_str $throughput msg/sec |" >> "$CHART_FILE"
    done

    cat >> "$CHART_FILE" << 'EOF'

## Recent Performance Metrics

EOF

    # Add detailed comparison table
    echo '| Date | Test 2 Throughput | Success | Peak LAG | Avg LAG | Query Time | Status |' >> "$CHART_FILE"
    echo '|------|-------------------|---------|----------|---------|------------|--------|' >> "$CHART_FILE"

    tail -n +4 "$INDEX_FILE" | head -n 10 | while IFS='|' read -r _ date _ throughput _ success _ peak _ avg _ latency _; do
        date=$(echo "$date" | xargs)
        throughput=$(echo "$throughput" | xargs)
        success=$(echo "$success" | xargs)
        peak=$(echo "$peak" | xargs)
        avg=$(echo "$avg" | xargs)
        latency=$(echo "$latency" | xargs)

        # Determine status emoji
        throughput_num=$(echo "$throughput" | sed 's/ msg\/sec//')
        if (( $(echo "$throughput_num >= 100" | bc -l) )); then
            status="🟢 Excellent"
        elif (( $(echo "$throughput_num >= 50" | bc -l) )); then
            status="🟡 Good"
        elif (( $(echo "$throughput_num >= 25" | bc -l) )); then
            status="🟠 Moderate"
        else
            status="🔴 Poor"
        fi

        echo "| $date | $throughput | $success | $peak | $avg | $latency | $status |" >> "$CHART_FILE"
    done

    cat >> "$CHART_FILE" << 'EOF'

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

EOF

    log_info "✅ Comparison chart generated: $CHART_FILE"
}

# Sample application logs
sample_app_logs() {
    local test_name=$1
    log_info "Sampling application logs for $test_name..."

    # Try to get logs from running process (if logs are being written)
    # This assumes logs go to stdout/stderr

    # Count error messages
    ERROR_COUNT=$(grep -i "error\|failed\|panic" "$DEBUG_LOG" 2>/dev/null | wc -l || echo "0")

    log_debug "[$test_name] Error count in logs: $ERROR_COUNT"
    echo "$ERROR_COUNT"
}

# ============================================================================
# Test Setup
# ============================================================================

log_info "Starting Performance Test Suite at $(date)"
log_info "Report will be saved to: $REPORT_FILE"
log_info "Debug log: $DEBUG_LOG"

# Check prerequisites
check_app

# Create test message
log_info "Creating test message..."
cat > /tmp/test-event.json << 'EOF'
{
  "device_id": "sensor-001",
  "type": "temperature_sensor",
  "timestamp": "2026-03-20T21:00:00Z",
  "metrics": {
    "temperature_c": 22.5,
    "humidity_percent": 45.0,
    "air_quality_index": 35
  }
}
EOF

# Pre-flight checks
log_info "Running pre-flight checks..."
SQLITE_CONFIG=$(check_sqlite_config)
INDEX_STATUS=$(check_indexes)

# Get baseline stats
log_info "Collecting baseline metrics..."
BASELINE_STATS=$(get_process_stats)
BASELINE_DB=$(get_db_stats)
BASELINE_EVENTS=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM events;")
BASELINE_LAG=$(get_kafka_lag)

log_debug "Baseline - Process: $BASELINE_STATS"
log_debug "Baseline - DB: $BASELINE_DB"
log_debug "Baseline - Events: $BASELINE_EVENTS"
log_debug "Baseline - Kafka LAG: $BASELINE_LAG"

# ============================================================================
# Initialize Report
# ============================================================================

cat > $REPORT_FILE << EOF
# synapsePlatform Performance Test Report

**Test Date:** $(date)
**Git Commit:** $(git rev-parse --short HEAD 2>/dev/null || echo "N/A")
**Machine:** $(uname -m)
**OS:** $(uname -s) $(uname -r)

---

## Pre-Flight Diagnostics

### SQLite Configuration
\`\`\`
Journal Mode: $(echo $SQLITE_CONFIG | cut -d'|' -f1)
Busy Timeout: $(echo $SQLITE_CONFIG | cut -d'|' -f2)ms
Synchronous: $(echo $SQLITE_CONFIG | cut -d'|' -f3)
\`\`\`

### Index Status
- Pagination Index (idx_ingested_event): **$INDEX_STATUS**

### Baseline Metrics
\`\`\`
Process Stats: $BASELINE_STATS
Database: $BASELINE_DB
Events in DB: $BASELINE_EVENTS
Kafka LAG: $BASELINE_LAG
\`\`\`

---

## Test Results

EOF

# ============================================================================
# Test 1: 10 msg/sec for 60 seconds (600 messages)
# ============================================================================

log_info "Starting Test 1: 10 msg/sec for 60 seconds (600 messages)"

TEST1_START=$(date +%s)
TEST1_START_EVENTS=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM events;")

# Start monitoring
monitor_kafka_lag "Test1" &
LAG_MONITOR_PID=$!

monitor_process_stats "test1" &
PROCESS_MONITOR_PID=$!

# Run test
log_info "Sending messages..."
for i in {1..600}; do
  kcat -b localhost:9092 -t ingestion.raw -P /tmp/test-event.json 2>/dev/null
  sleep 0.1

  # Log progress every 100 messages
  if [ $((i % 100)) -eq 0 ]; then
    CURRENT_LAG=$(get_kafka_lag)
    log_debug "[Test1] Progress: $i/600, LAG: $CURRENT_LAG"
  fi
done

TEST1_END=$(date +%s)
TEST1_DURATION=$((TEST1_END - TEST1_START))

log_info "Messages sent. Waiting 10 seconds for processing..."
sleep 10

# Stop monitoring
LAG_STATS=$(stop_lag_monitoring)
stop_process_monitoring

TEST1_PEAK_LAG=$(echo $LAG_STATS | cut -d'|' -f1)
TEST1_AVG_LAG=$(echo $LAG_STATS | cut -d'|' -f2)
TEST1_FINAL_LAG=$(echo $LAG_STATS | cut -d'|' -f3)

TEST1_END_EVENTS=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM events;")
TEST1_PROCESSED=$((TEST1_END_EVENTS - TEST1_START_EVENTS))
TEST1_FAILED=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM failed_messages;")
TEST1_STATS=$(get_process_stats)
TEST1_SUCCESS_RATE=$(awk "BEGIN {printf \"%.2f\", ($TEST1_PROCESSED / 600.0) * 100}")
TEST1_ERRORS=$(sample_app_logs "Test1")

log_info "Test 1 Complete: $TEST1_PROCESSED/600 messages processed ($TEST1_SUCCESS_RATE%)"
log_info "Test 1 Peak LAG: $TEST1_PEAK_LAG, Avg LAG: $TEST1_AVG_LAG, Final LAG: $TEST1_FINAL_LAG"

cat >> $REPORT_FILE << EOF
### Test 1: Low Load (10 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 10 msg/sec |
| **Duration** | ${TEST1_DURATION}s |
| **Messages Sent** | 600 |
| **Messages Processed** | $TEST1_PROCESSED |
| **Failed Messages** | $TEST1_FAILED |
| **Success Rate** | ${TEST1_SUCCESS_RATE}% |
| **Actual Throughput** | $(awk "BEGIN {printf \"%.1f\", $TEST1_PROCESSED / $TEST1_DURATION}") msg/sec |
| **Peak Kafka LAG** | $TEST1_PEAK_LAG |
| **Average Kafka LAG** | $TEST1_AVG_LAG |
| **Final Kafka LAG** | $TEST1_FINAL_LAG |
| **Error Count** | $TEST1_ERRORS |
| **Process Stats (CPU% MEM% RSS VSZ)** | $TEST1_STATS |

**Analysis:**
EOF

if [ "$TEST1_PEAK_LAG" -gt 100 ]; then
    echo "- ⚠️  Peak LAG exceeded 100 - consumer falling behind even at low load" >> $REPORT_FILE
fi

if [ "$TEST1_SUCCESS_RATE" != "100.00" ]; then
    echo "- ⚠️  Success rate < 100% - check failed_messages table" >> $REPORT_FILE
fi

echo "" >> $REPORT_FILE

# ============================================================================
# Test 2: 100 msg/sec for 60 seconds (6000 messages)
# ============================================================================

log_info "Starting Test 2: 100 msg/sec for 60 seconds (6000 messages)"

TEST2_START=$(date +%s)
TEST2_START_EVENTS=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM events;")

# Start monitoring
monitor_kafka_lag "Test2" &
LAG_MONITOR_PID=$!

monitor_process_stats "test2" &
PROCESS_MONITOR_PID=$!

# Run test
log_info "Sending messages..."
for i in {1..6000}; do
  kcat -b localhost:9092 -t ingestion.raw -P /tmp/test-event.json 2>/dev/null
  sleep 0.01

  if [ $((i % 1000)) -eq 0 ]; then
    CURRENT_LAG=$(get_kafka_lag)
    log_debug "[Test2] Progress: $i/6000, LAG: $CURRENT_LAG"
  fi
done

TEST2_END=$(date +%s)
TEST2_DURATION=$((TEST2_END - TEST2_START))

log_info "Messages sent. Waiting 10 seconds for processing..."
sleep 10

# Stop monitoring
LAG_STATS=$(stop_lag_monitoring)
stop_process_monitoring

TEST2_PEAK_LAG=$(echo $LAG_STATS | cut -d'|' -f1)
TEST2_AVG_LAG=$(echo $LAG_STATS | cut -d'|' -f2)
TEST2_FINAL_LAG=$(echo $LAG_STATS | cut -d'|' -f3)

TEST2_END_EVENTS=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM events;")
TEST2_PROCESSED=$((TEST2_END_EVENTS - TEST2_START_EVENTS))
TEST2_FAILED=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM failed_messages;")
TEST2_STATS=$(get_process_stats)
TEST2_SUCCESS_RATE=$(awk "BEGIN {printf \"%.2f\", ($TEST2_PROCESSED / 6000.0) * 100}")
TEST2_ERRORS=$(sample_app_logs "Test2")

log_info "Test 2 Complete: $TEST2_PROCESSED/6000 messages processed ($TEST2_SUCCESS_RATE%)"
log_info "Test 2 Peak LAG: $TEST2_PEAK_LAG, Avg LAG: $TEST2_AVG_LAG, Final LAG: $TEST2_FINAL_LAG"

cat >> $REPORT_FILE << EOF
### Test 2: Medium Load (100 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 100 msg/sec |
| **Duration** | ${TEST2_DURATION}s |
| **Messages Sent** | 6000 |
| **Messages Processed** | $TEST2_PROCESSED |
| **Failed Messages** | $TEST2_FAILED |
| **Success Rate** | ${TEST2_SUCCESS_RATE}% |
| **Actual Throughput** | $(awk "BEGIN {printf \"%.1f\", $TEST2_PROCESSED / $TEST2_DURATION}") msg/sec |
| **Peak Kafka LAG** | $TEST2_PEAK_LAG |
| **Average Kafka LAG** | $TEST2_AVG_LAG |
| **Final Kafka LAG** | $TEST2_FINAL_LAG |
| **Error Count** | $TEST2_ERRORS |
| **Process Stats (CPU% MEM% RSS VSZ)** | $TEST2_STATS |

**Analysis:**
EOF

if [ "$TEST2_PEAK_LAG" -gt 1000 ]; then
    echo "- 🚨 Peak LAG exceeded 1000 - severe bottleneck detected" >> $REPORT_FILE
fi

if [ "$TEST2_AVG_LAG" -gt 500 ]; then
    echo "- ⚠️  Average LAG > 500 - consumer consistently falling behind" >> $REPORT_FILE
fi

echo "" >> $REPORT_FILE

# ============================================================================
# Test 3: 500 msg/sec for 60 seconds (30,000 messages)
# ============================================================================

log_info "Starting Test 3: 500 msg/sec for 60 seconds (30,000 messages)"

TEST3_START=$(date +%s)
TEST3_START_EVENTS=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM events;")

# Start monitoring
monitor_kafka_lag "Test3" &
LAG_MONITOR_PID=$!

monitor_process_stats "test3" &
PROCESS_MONITOR_PID=$!

# Run test
log_info "Sending messages..."
for i in {1..30000}; do
  kcat -b localhost:9092 -t ingestion.raw -P /tmp/test-event.json 2>/dev/null
  sleep 0.002

  if [ $((i % 5000)) -eq 0 ]; then
    CURRENT_LAG=$(get_kafka_lag)
    log_debug "[Test3] Progress: $i/30000, LAG: $CURRENT_LAG"
  fi
done

TEST3_END=$(date +%s)
TEST3_DURATION=$((TEST3_END - TEST3_START))

log_info "Messages sent. Waiting 30 seconds for processing (larger backlog)..."
sleep 30

# Stop monitoring
LAG_STATS=$(stop_lag_monitoring)
stop_process_monitoring

TEST3_PEAK_LAG=$(echo $LAG_STATS | cut -d'|' -f1)
TEST3_AVG_LAG=$(echo $LAG_STATS | cut -d'|' -f2)
TEST3_FINAL_LAG=$(echo $LAG_STATS | cut -d'|' -f3)

TEST3_END_EVENTS=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM events;")
TEST3_PROCESSED=$((TEST3_END_EVENTS - TEST3_START_EVENTS))
TEST3_FAILED=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM failed_messages;")
TEST3_STATS=$(get_process_stats)
TEST3_SUCCESS_RATE=$(awk "BEGIN {printf \"%.2f\", ($TEST3_PROCESSED / 30000.0) * 100}")
TEST3_ERRORS=$(sample_app_logs "Test3")

log_info "Test 3 Complete: $TEST3_PROCESSED/30,000 messages processed ($TEST3_SUCCESS_RATE%)"
log_info "Test 3 Peak LAG: $TEST3_PEAK_LAG, Avg LAG: $TEST3_AVG_LAG, Final LAG: $TEST3_FINAL_LAG"

cat >> $REPORT_FILE << EOF
### Test 3: High Load (500 msg/sec)

| Metric | Value |
|--------|-------|
| **Target Rate** | 500 msg/sec |
| **Duration** | ${TEST3_DURATION}s |
| **Messages Sent** | 30,000 |
| **Messages Processed** | $TEST3_PROCESSED |
| **Failed Messages** | $TEST3_FAILED |
| **Success Rate** | ${TEST3_SUCCESS_RATE}% |
| **Actual Throughput** | $(awk "BEGIN {printf \"%.1f\", $TEST3_PROCESSED / $TEST3_DURATION}") msg/sec |
| **Peak Kafka LAG** | $TEST3_PEAK_LAG |
| **Average Kafka LAG** | $TEST3_AVG_LAG |
| **Final Kafka LAG** | $TEST3_FINAL_LAG |
| **Error Count** | $TEST3_ERRORS |
| **Process Stats (CPU% MEM% RSS VSZ)** | $TEST3_STATS |

**Analysis:**
EOF

if [ "$TEST3_PEAK_LAG" -gt 10000 ]; then
    echo "- 🚨 Peak LAG exceeded 10,000 - critical bottleneck" >> $REPORT_FILE
fi

if [ "$TEST3_FINAL_LAG" -gt 1000 ]; then
    echo "- 🚨 Final LAG still > 1000 after 30s wait - backlog not clearing" >> $REPORT_FILE
fi

# Calculate throughput ceiling
THROUGHPUT_CEILING=$(awk "BEGIN {printf \"%.1f\", $TEST3_PROCESSED / $TEST3_DURATION}")
echo "- 📊 Observed throughput ceiling: ~${THROUGHPUT_CEILING} msg/sec" >> $REPORT_FILE

echo "" >> $REPORT_FILE

# ============================================================================
# Final Database Analysis
# ============================================================================

log_info "Collecting final database metrics..."

FINAL_DB_STATS=$(get_db_stats)
TOTAL_EVENTS=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM events;")
TOTAL_FAILED=$(sqlite3 $DB_PATH "SELECT COUNT(*) FROM failed_messages;")
FINAL_LAG=$(get_kafka_lag)

# Query performance test
QUERY_START=$(date +%s.%N)
sqlite3 $DB_PATH "SELECT * FROM events ORDER BY ingested_at DESC LIMIT 20;" > /dev/null
QUERY_END=$(date +%s.%N)
QUERY_LATENCY=$(awk "BEGIN {printf \"%.0f\", ($QUERY_END - $QUERY_START) * 1000}")

# Check for failed messages breakdown
FAILED_BY_STAGE=$(sqlite3 $DB_PATH "SELECT stage, COUNT(*) FROM failed_messages GROUP BY stage;" 2>/dev/null || echo "N/A")

cat >> $REPORT_FILE << EOF
---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | $TOTAL_EVENTS |
| **Total Failed** | $TOTAL_FAILED |
| **Database Stats** | $FINAL_DB_STATS |
| **List Query (20 items)** | ${QUERY_LATENCY}ms |
| **Final Kafka LAG** | $FINAL_LAG |

### Failed Messages Breakdown
\`\`\`
$FAILED_BY_STAGE
\`\`\`

---

## Summary

| Test | Target Rate | Actual Throughput | Success Rate | Peak LAG | Avg LAG |
|------|-------------|-------------------|--------------|----------|---------|
| Test 1 | 10 msg/sec | $(awk "BEGIN {printf \"%.1f\", $TEST1_PROCESSED / $TEST1_DURATION}") msg/sec | ${TEST1_SUCCESS_RATE}% | $TEST1_PEAK_LAG | $TEST1_AVG_LAG |
| Test 2 | 100 msg/sec | $(awk "BEGIN {printf \"%.1f\", $TEST2_PROCESSED / $TEST2_DURATION}") msg/sec | ${TEST2_SUCCESS_RATE}% | $TEST2_PEAK_LAG | $TEST2_AVG_LAG |
| Test 3 | 500 msg/sec | $(awk "BEGIN {printf \"%.1f\", $TEST3_PROCESSED / $TEST3_DURATION}") msg/sec | ${TEST3_SUCCESS_RATE}% | $TEST3_PEAK_LAG | $TEST3_AVG_LAG |

---

## Bottleneck Analysis

EOF

# Automated bottleneck detection
SQLITE_JOURNAL=$(echo $SQLITE_CONFIG | cut -d'|' -f1)
if [ "$SQLITE_JOURNAL" != "wal" ]; then
    cat >> $REPORT_FILE << EOF
### 🔴 Critical: SQLite Not in WAL Mode
- **Current mode:** $SQLITE_JOURNAL
- **Impact:** 3-5x slower writes, blocks reads during writes
- **Fix:** Add pragmas to \`internal/sqllite/storer.go\`:
\`\`\`go
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
\`\`\`
- **Expected improvement:** 30-35 msg/sec → 100-150 msg/sec

EOF
fi

if [ "$INDEX_STATUS" = "no" ]; then
    cat >> $REPORT_FILE << EOF
### 🟡 Missing Pagination Index
- **Impact:** Slow list queries (${QUERY_LATENCY}ms for 20 items)
- **Fix:** Add to \`internal/sqllite/summary.sql\`:
\`\`\`sql
CREATE INDEX IF NOT EXISTS idx_ingested_event ON events(ingested_at DESC, event_id DESC);
\`\`\`
- **Expected improvement:** ${QUERY_LATENCY}ms → 2-5ms

EOF
fi

if [ "$TEST3_PEAK_LAG" -gt 10000 ]; then
    cat >> $REPORT_FILE << EOF
### 🔴 Severe Throughput Bottleneck
- **Peak LAG:** $TEST3_PEAK_LAG messages
- **Throughput ceiling:** ~${THROUGHPUT_CEILING} msg/sec
- **Root cause:** Single-threaded processing + slow SQLite writes
- **Recommended fixes:**
  1. Enable WAL mode (immediate, 3-5x improvement)
  2. Implement batch inserts (4 hours, 10-20x improvement)
  3. Add worker pool (6 hours, 5-10x improvement)

EOF
fi

if [ $TOTAL_FAILED -gt 0 ]; then
    cat >> $REPORT_FILE << EOF
### ⚠️  Failed Messages Detected
- **Count:** $TOTAL_FAILED
- **Check:** \`sqlite3 data.db "SELECT stage, error FROM failed_messages LIMIT 5;"\`
- **Likely cause:** Invalid test data or validation errors

EOF
fi

cat >> $REPORT_FILE << EOF
---

## Debug Information

- **Debug log:** $DEBUG_LOG
- **Kafka LAG log:** $LAG_LOG
- **Full diagnostics:** Run \`cat $DEBUG_LOG\` for detailed timeline

---

**Test completed at:** $(date)
EOF

# Detect performance regression
check_regression() {
    local current_throughput=$1
    local INDEX_FILE="${REPORT_DIR}/INDEX.md"

    # Get previous run's throughput
    local prev_throughput=$(tail -n +4 "$INDEX_FILE" 2>/dev/null | head -n 1 | \
        awk -F'|' '{print $3}' | sed 's/ msg\/sec//' | xargs)

    if [ -z "$prev_throughput" ]; then
        log_info "First run - no regression check"
        return
    fi

    # Calculate percentage change
    local change=$(awk "BEGIN {printf \"%.1f\", (($current_throughput - $prev_throughput) / $prev_throughput) * 100}")

    if (( $(echo "$change < -10" | bc -l) )); then
        log_warn "⚠️  REGRESSION DETECTED: Throughput dropped by ${change}% (was: ${prev_throughput} msg/sec, now: ${current_throughput} msg/sec)"
        echo "## 🚨 Regression Alert" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        echo "Performance decreased by **${change}%** compared to previous run." >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
    elif (( $(echo "$change > 10" | bc -l) )); then
        log_info "🎉 IMPROVEMENT: Throughput increased by ${change}% (was: ${prev_throughput} msg/sec, now: ${current_throughput} msg/sec)"
        echo "## 🎉 Performance Improvement" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        echo "Performance increased by **${change}%** compared to previous run!" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
    else
        log_info "Performance stable (${change}% change)"
    fi
}

# Create symlink to latest
ln -sf "$(basename $REPORT_FILE)" "$REPORT_DIR/latest.md"

# ============================================================================
# Update Index and Generate Comparison
# ============================================================================

# Calculate Test 2 metrics for index
TEST2_THROUGHPUT=$(awk "BEGIN {printf \"%.1f\", $TEST2_PROCESSED / $TEST2_DURATION}")

# Check for regression
check_regression "$TEST2_THROUGHPUT"

# Update INDEX.md with this run's results
update_index "$TIMESTAMP" "$TEST2_THROUGHPUT" "$TEST2_SUCCESS_RATE" "$TEST2_PEAK_LAG" "$TEST2_AVG_LAG"

# Generate comparison chart
generate_comparison_chart

# Create symlink to latest report
ln -sf "$(basename $REPORT_FILE)" "$REPORT_DIR/latest.md"

# ============================================================================
# Finish
# ============================================================================

log_info "Performance test complete!"
log_info "Report saved to: $REPORT_FILE"
log_info "Index updated: $REPORT_DIR/INDEX.md"
log_info "Comparison chart: $REPORT_DIR/COMPARISON.md"
log_info "Debug log saved to: $DEBUG_LOG"
echo ""
echo "📊 Latest Results:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
tail -n 1 "$REPORT_DIR/INDEX.md"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
log_info "View full report: cat $REPORT_FILE"
log_info "View index: cat $REPORT_DIR/INDEX.md"
log_info "View comparison: cat $REPORT_DIR/COMPARISON.md"
log_info "View debug log: cat $DEBUG_LOG"

# Create an index of all test runs
create_index() {
    local index_file="${REPORT_DIR}/INDEX.md"

    cat > "$index_file" << 'EOF'
# Performance Test History

| Date | Report | Peak LAG | Throughput (Test 2) | Success Rate |
|------|--------|----------|---------------------|--------------|
EOF

    for report in $(ls -t ${REPORT_DIR}/synapse-performance-report-*.md); do
        TIMESTAMP=$(basename "$report" | sed 's/synapse-performance-report-\(.*\)\.md/\1/')
        DATE=$(echo "$TIMESTAMP" | sed 's/\([0-9]\{8\}\)-\([0-9]\{6\}\)/\1 \2/')

        # Extract metrics from report
        PEAK_LAG=$(grep "Peak Kafka LAG" "$report" | head -1 | awk '{print $5}')
        THROUGHPUT=$(grep "Test 2.*Actual Throughput" "$report" | awk '{print $5}')
        SUCCESS=$(grep "Test 2.*Success Rate" "$report" | awk '{print $5}')

        echo "| $DATE | [Report](./${report##*/}) | $PEAK_LAG | $THROUGHPUT | $SUCCESS |" >> "$index_file"
    done

    log_info "Index created: $index_file"
}

# Call at the end of the script
create_index