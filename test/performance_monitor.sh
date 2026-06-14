#!/bin/bash
set -e

# ============================================================================
# Enhanced Performance Test Monitor
# Wraps perform_test.sh with real-time bottleneck detection
# ============================================================================

# Configuration
REPORT_DIR="./performance-reports"
TIMESTAMP=${PERF_TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}
METRICS_DIR="${REPORT_DIR}/metrics-${TIMESTAMP}"
APP_PORT=8080
DB_PATH="data.db"
METRICS_ENDPOINT="http://localhost:${APP_PORT}/metrics"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Create directories
mkdir -p "$METRICS_DIR"

# ============================================================================
# Logging Functions
# ============================================================================

log_info() {
    echo -e "${GREEN}[MONITOR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[MONITOR]${NC} $1"
}

log_error() {
    echo -e "${RED}[MONITOR]${NC} $1"
}

log_metric() {
    echo -e "${CYAN}[METRIC]${NC} $1"
}

# ============================================================================
# Monitoring Functions
# ============================================================================

# Check if metrics endpoint is available
check_metrics_endpoint() {
    log_info "Checking metrics endpoint..."
    if curl -s --max-time 2 "$METRICS_ENDPOINT" > /dev/null 2>&1; then
        log_info "✅ Metrics endpoint available at $METRICS_ENDPOINT"
        return 0
    else
        log_warn "⚠️  Metrics endpoint not available at $METRICS_ENDPOINT"
        log_warn "    Add this to your API server routes:"
        log_warn "    mux.HandleFunc(\"GET /metrics\", providers.Metrics.ServeHTTP)"
        return 1
    fi
}

# Monitor Prometheus metrics in background
monitor_prometheus_metrics() {
    local test_phase=$1
    local output_file="${METRICS_DIR}/prometheus-${test_phase}.log"

    log_info "Starting Prometheus metrics monitoring for $test_phase..."

    while true; do
        TIMESTAMP=$(date +%H:%M:%S)

        # Fetch metrics
        METRICS=$(curl -s "$METRICS_ENDPOINT" 2>/dev/null || echo "")

        if [ -z "$METRICS" ]; then
            echo "$TIMESTAMP|ERROR|Failed to fetch metrics" >> "$output_file"
            sleep 2
            continue
        fi

        # Extract key metrics
        BATCH_DURATION_SUM=$(echo "$METRICS" | grep "ingestor_store_batch_duration_sum" | awk '{print $2}')
        BATCH_DURATION_COUNT=$(echo "$METRICS" | grep "ingestor_store_batch_duration_count" | awk '{print $2}')
        BATCH_SIZE_SUM=$(echo "$METRICS" | grep "ingestor_store_batch_size_sum" | awk '{print $2}')
        BATCH_SIZE_COUNT=$(echo "$METRICS" | grep "ingestor_store_batch_size_count" | awk '{print $2}')

        TRANSFORM_DURATION_SUM=$(echo "$METRICS" | grep "ingestor_transform_duration_sum" | awk '{print $2}')
        TRANSFORM_DURATION_COUNT=$(echo "$METRICS" | grep "ingestor_transform_duration_count" | awk '{print $2}')

        CHANNEL_UTIL=$(echo "$METRICS" | grep "ingestor_event_channel_utilization" | awk '{print $2}')

        SQLITE_WAL=$(echo "$METRICS" | grep "sqlite_wal_pages" | awk '{print $2}')
        SQLITE_DB=$(echo "$METRICS" | grep "sqlite_db_pages" | awk '{print $2}')

        # Calculate averages
        if [ ! -z "$BATCH_DURATION_COUNT" ] && [ "$BATCH_DURATION_COUNT" != "0" ]; then
            AVG_BATCH_DURATION=$(awk "BEGIN {printf \"%.2f\", $BATCH_DURATION_SUM / $BATCH_DURATION_COUNT * 1000}")
        else
            AVG_BATCH_DURATION="0"
        fi

        if [ ! -z "$BATCH_SIZE_COUNT" ] && [ "$BATCH_SIZE_COUNT" != "0" ]; then
            AVG_BATCH_SIZE=$(awk "BEGIN {printf \"%.1f\", $BATCH_SIZE_SUM / $BATCH_SIZE_COUNT}")
        else
            AVG_BATCH_SIZE="0"
        fi

        if [ ! -z "$TRANSFORM_DURATION_COUNT" ] && [ "$TRANSFORM_DURATION_COUNT" != "0" ]; then
            AVG_TRANSFORM_DURATION=$(awk "BEGIN {printf \"%.2f\", $TRANSFORM_DURATION_SUM / $TRANSFORM_DURATION_COUNT * 1000}")
        else
            AVG_TRANSFORM_DURATION="0"
        fi

        # Log metrics
        echo "$TIMESTAMP|batch_duration_ms:$AVG_BATCH_DURATION|batch_size:$AVG_BATCH_SIZE|transform_ms:$AVG_TRANSFORM_DURATION|channel_util:${CHANNEL_UTIL:-0}|wal_pages:${SQLITE_WAL:-0}" >> "$output_file"

        sleep 2
    done
}

# Monitor SQLite stats in background
monitor_sqlite_stats() {
    local test_phase=$1
    local output_file="${METRICS_DIR}/sqlite-${test_phase}.log"

    log_info "Starting SQLite monitoring for $test_phase..."

    while true; do
        TIMESTAMP=$(date +%H:%M:%S)

        # Get WAL checkpoint info
        WAL_INFO=$(sqlite3 "$DB_PATH" "PRAGMA wal_checkpoint(PASSIVE);" 2>/dev/null || echo "0|0|0")
        BUSY=$(echo "$WAL_INFO" | cut -d'|' -f1)
        LOG_PAGES=$(echo "$WAL_INFO" | cut -d'|' -f2)
        CHECKPOINTED=$(echo "$WAL_INFO" | cut -d'|' -f3)

        # Get page counts
        PAGE_COUNT=$(sqlite3 "$DB_PATH" "PRAGMA page_count;" 2>/dev/null || echo "0")
        PAGE_SIZE=$(sqlite3 "$DB_PATH" "PRAGMA page_size;" 2>/dev/null || echo "4096")
        FREELIST=$(sqlite3 "$DB_PATH" "PRAGMA freelist_count;" 2>/dev/null || echo "0")

        # Calculate sizes
        DB_SIZE_MB=$(awk "BEGIN {printf \"%.2f\", $PAGE_COUNT * $PAGE_SIZE / 1024 / 1024}")
        WAL_SIZE_MB=$(awk "BEGIN {printf \"%.2f\", $LOG_PAGES * $PAGE_SIZE / 1024 / 1024}")

        echo "$TIMESTAMP|busy:$BUSY|wal_pages:$LOG_PAGES|wal_mb:$WAL_SIZE_MB|db_pages:$PAGE_COUNT|db_mb:$DB_SIZE_MB|freelist:$FREELIST" >> "$output_file"

        sleep 5
    done
}

# Monitor system resources
monitor_system_resources() {
    local test_phase=$1
    local output_file="${METRICS_DIR}/system-${test_phase}.log"

    log_info "Starting system resource monitoring for $test_phase..."

    APP_PID=$(lsof -ti :$APP_PORT 2>/dev/null || echo "")

    if [ -z "$APP_PID" ]; then
        log_warn "Could not find application PID"
        return
    fi

    while true; do
        TIMESTAMP=$(date +%H:%M:%S)

        # Get process stats
        STATS=$(ps -p $APP_PID -o %cpu,%mem,rss,vsz 2>/dev/null | tail -1)

        if [ -z "$STATS" ]; then
            echo "$TIMESTAMP|ERROR|Process not found" >> "$output_file"
            sleep 2
            continue
        fi

        CPU=$(echo "$STATS" | awk '{print $1}')
        MEM=$(echo "$STATS" | awk '{print $2}')
        RSS=$(echo "$STATS" | awk '{print $3}')
        VSZ=$(echo "$STATS" | awk '{print $4}')

        # Get goroutine count (if pprof is available)
        GOROUTINES=$(curl -s "http://localhost:${APP_PORT}/debug/pprof/goroutine?debug=1" 2>/dev/null | grep "goroutine profile:" | awk '{print $3}' || echo "N/A")

        echo "$TIMESTAMP|cpu:$CPU|mem:$MEM|rss_kb:$RSS|vsz_kb:$VSZ|goroutines:$GOROUTINES" >> "$output_file"

        sleep 2
    done
}

# Stop all monitors
stop_monitors() {
    log_info "Stopping monitors..."

    if [ ! -z "$PROMETHEUS_MONITOR_PID" ]; then
        kill $PROMETHEUS_MONITOR_PID 2>/dev/null || true
        wait $PROMETHEUS_MONITOR_PID 2>/dev/null || true
    fi

    if [ ! -z "$SQLITE_MONITOR_PID" ]; then
        kill $SQLITE_MONITOR_PID 2>/dev/null || true
        wait $SQLITE_MONITOR_PID 2>/dev/null || true
    fi

    if [ ! -z "$SYSTEM_MONITOR_PID" ]; then
        kill $SYSTEM_MONITOR_PID 2>/dev/null || true
        wait $SYSTEM_MONITOR_PID 2>/dev/null || true
    fi
}

# Analyze metrics for a test phase
analyze_metrics() {
    local test_phase=$1
    local prom_file="${METRICS_DIR}/prometheus-${test_phase}.log"
    local sqlite_file="${METRICS_DIR}/sqlite-${test_phase}.log"
    local system_file="${METRICS_DIR}/system-${test_phase}.log"
    local output_file="${METRICS_DIR}/analysis-${test_phase}.md"

    log_info "Analyzing metrics for $test_phase..."

    cat > "$output_file" << EOF
# Bottleneck Analysis: $test_phase

## Batch Write Performance

EOF

    # Analyze batch metrics
    if [ -f "$prom_file" ]; then
        AVG_BATCH_DUR=$(awk -F'|' '{
            if ($2 ~ /batch_duration_ms:/) {
                split($2, a, ":");
                sum += a[2];
                count++;
            }
        } END {
            if (count > 0) printf "%.2f", sum/count;
            else print "N/A";
        }' "$prom_file")

        MAX_BATCH_DUR=$(awk -F'|' '{
            if ($2 ~ /batch_duration_ms:/) {
                split($2, a, ":");
                if (a[2] > max) max = a[2];
            }
        } END {
            if (max > 0) printf "%.2f", max;
            else print "N/A";
        }' "$prom_file")

        AVG_BATCH_SIZE=$(awk -F'|' '{
            if ($3 ~ /batch_size:/) {
                split($3, a, ":");
                sum += a[2];
                count++;
            }
        } END {
            if (count > 0) printf "%.1f", sum/count;
            else print "N/A";
        }' "$prom_file")

        MAX_CHANNEL_UTIL=$(awk -F'|' '{
            if ($5 ~ /channel_util:/) {
                split($5, a, ":");
                if (a[2] > max) max = a[2];
            }
        } END {
            if (max > 0) printf "%.0f", max;
            else print "N/A";
        }' "$prom_file")

        cat >> "$output_file" << EOF
| Metric | Value | Status |
|--------|-------|--------|
| **Avg Batch Duration** | ${AVG_BATCH_DUR}ms | $([ "$AVG_BATCH_DUR" != "N/A" ] && (( $(echo "$AVG_BATCH_DUR > 100" | bc -l 2>/dev/null || echo 0) )) && echo "🔴 SLOW" || echo "🟢 OK") |
| **Max Batch Duration** | ${MAX_BATCH_DUR}ms | $([ "$MAX_BATCH_DUR" != "N/A" ] && (( $(echo "$MAX_BATCH_DUR > 200" | bc -l 2>/dev/null || echo 0) )) && echo "🔴 SLOW" || echo "🟢 OK") |
| **Avg Batch Size** | ${AVG_BATCH_SIZE} events | $([ "$AVG_BATCH_SIZE" != "N/A" ] && (( $(echo "$AVG_BATCH_SIZE < 40" | bc -l 2>/dev/null || echo 0) )) && echo "🟡 TIMEOUT FLUSHES" || echo "🟢 OK") |
| **Max Channel Utilization** | ${MAX_CHANNEL_UTIL} | $([ "$MAX_CHANNEL_UTIL" != "N/A" ] && [ "$MAX_CHANNEL_UTIL" -gt 90 ] && echo "🔴 BACKPRESSURE" || echo "🟢 OK") |

EOF
    fi

    # Analyze SQLite metrics
    if [ -f "$sqlite_file" ]; then
        cat >> "$output_file" << EOF

## SQLite Performance

EOF

        MAX_WAL_PAGES=$(awk -F'|' '{
            if ($2 ~ /wal_pages:/) {
                split($2, a, ":");
                if (a[2] > max) max = a[2];
            }
        } END {
            if (max > 0) printf "%.0f", max;
            else print "N/A";
        }' "$sqlite_file")

        AVG_WAL_PAGES=$(awk -F'|' '{
            if ($2 ~ /wal_pages:/) {
                split($2, a, ":");
                sum += a[2];
                count++;
            }
        } END {
            if (count > 0) printf "%.0f", sum/count;
            else print "N/A";
        }' "$sqlite_file")

        MAX_BUSY=$(awk -F'|' '{
            if ($1 ~ /busy:/) {
                split($1, a, ":");
                if (a[2] > max) max = a[2];
            }
        } END {
            print max + 0;
        }' "$sqlite_file")

        cat >> "$output_file" << EOF
| Metric | Value | Status |
|--------|-------|--------|
| **Avg WAL Pages** | ${AVG_WAL_PAGES} | $([ "$AVG_WAL_PAGES" != "N/A" ] && [ "$AVG_WAL_PAGES" -gt 1000 ] && echo "🟡 LARGE WAL" || echo "🟢 OK") |
| **Max WAL Pages** | ${MAX_WAL_PAGES} | $([ "$MAX_WAL_PAGES" != "N/A" ] && [ "$MAX_WAL_PAGES" -gt 2000 ] && echo "🔴 CHECKPOINT NEEDED" || echo "🟢 OK") |
| **Database Locks** | ${MAX_BUSY} | $([ "$MAX_BUSY" -gt 0 ] && echo "🔴 LOCKED" || echo "🟢 OK") |

EOF
    fi

    # Analyze system metrics
    if [ -f "$system_file" ]; then
        cat >> "$output_file" << EOF

## System Resources

EOF

        AVG_CPU=$(awk -F'|' '{
            if ($2 ~ /cpu:/) {
                split($2, a, ":");
                sum += a[2];
                count++;
            }
        } END {
            if (count > 0) printf "%.1f", sum/count;
            else print "N/A";
        }' "$system_file")

        MAX_CPU=$(awk -F'|' '{
            if ($2 ~ /cpu:/) {
                split($2, a, ":");
                if (a[2] > max) max = a[2];
            }
        } END {
            if (max > 0) printf "%.1f", max;
            else print "N/A";
        }' "$system_file")

        AVG_MEM=$(awk -F'|' '{
            if ($3 ~ /mem:/) {
                split($3, a, ":");
                sum += a[2];
                count++;
            }
        } END {
            if (count > 0) printf "%.1f", sum/count;
            else print "N/A";
        }' "$system_file")

        cat >> "$output_file" << EOF
| Metric | Value | Status |
|--------|-------|--------|
| **Avg CPU Usage** | ${AVG_CPU}% | $([ "$AVG_CPU" != "N/A" ] && (( $(echo "$AVG_CPU > 80" | bc -l 2>/dev/null || echo 0) )) && echo "🔴 CPU-BOUND" || echo "🟢 OK") |
| **Max CPU Usage** | ${MAX_CPU}% | $([ "$MAX_CPU" != "N/A" ] && (( $(echo "$MAX_CPU > 90" | bc -l 2>/dev/null || echo 0) )) && echo "🔴 CPU-BOUND" || echo "🟢 OK") |
| **Avg Memory** | ${AVG_MEM}% | $([ "$AVG_MEM" != "N/A" ] && (( $(echo "$AVG_MEM > 80" | bc -l 2>/dev/null || echo 0) )) && echo "🟡 HIGH" || echo "🟢 OK") |

EOF
    fi

    # Bottleneck determination
    cat >> "$output_file" << EOF

## Bottleneck Determination

EOF

    # Logic to determine bottleneck
    if [ "$AVG_BATCH_DUR" != "N/A" ] && (( $(echo "$AVG_BATCH_DUR > 100" | bc -l 2>/dev/null || echo 0) )); then
        cat >> "$output_file" << EOF
### 🔴 SQLite Write Bottleneck Detected

- **Evidence:** Average batch write duration > 100ms
- **Impact:** Transformers are blocked waiting for SQLite
- **Recommendation:**
  1. Verify WAL mode is enabled
  2. Consider PostgreSQL for higher write throughput
  3. Increase batch size to amortize write cost

EOF
    fi

    if [ "$MAX_CHANNEL_UTIL" != "N/A" ] && [ "$MAX_CHANNEL_UTIL" -gt 90 ]; then
        cat >> "$output_file" << EOF
### 🔴 Channel Backpressure Detected

- **Evidence:** Event channel utilization > 90%
- **Impact:** Workers blocked, cannot send transformed events
- **Recommendation:**
  1. Increase channel buffer size
  2. Add more worker threads
  3. Speed up batch writes

EOF
    fi

    if [ "$AVG_CPU" != "N/A" ] && (( $(echo "$AVG_CPU > 80" | bc -l 2>/dev/null || echo 0) )); then
        cat >> "$output_file" << EOF
### 🟡 CPU-Bound Processing

- **Evidence:** Average CPU usage > 80%
- **Impact:** Transformation or SQLite CPU-intensive
- **Recommendation:**
  1. Profile with pprof to find hot spots
  2. Optimize transformation logic
  3. Consider horizontal scaling

EOF
    fi

    if [ "$MAX_WAL_PAGES" != "N/A" ] && [ "$MAX_WAL_PAGES" -gt 2000 ]; then
        cat >> "$output_file" << EOF
### 🟡 Large WAL File

- **Evidence:** WAL file > 2000 pages
- **Impact:** Checkpoint pauses causing write spikes
- **Recommendation:**
  1. Tune checkpoint frequency
  2. Monitor checkpoint duration
  3. Consider smaller batch sizes

EOF
    fi

    log_info "Analysis saved to: $output_file"
}

# Capture metrics snapshot
capture_metrics_snapshot() {
    local phase=$1
    local snapshot_file="${METRICS_DIR}/snapshot-${phase}.txt"

    log_info "Capturing metrics snapshot for $phase..."
    curl -s "$METRICS_ENDPOINT" > "$snapshot_file" 2>/dev/null || true
}

# Generate final report
generate_monitoring_report() {
    local final_report="${METRICS_DIR}/MONITORING_SUMMARY.md"

    log_info "Generating monitoring summary..."

    cat > "$final_report" << EOF
# Performance Monitoring Summary
**Timestamp:** $(date)
**Metrics Directory:** $METRICS_DIR

---

EOF

    # Combine all analysis files
    for phase in test1 test2 test3; do
        if [ -f "${METRICS_DIR}/analysis-${phase}.md" ]; then
            cat "${METRICS_DIR}/analysis-${phase}.md" >> "$final_report"
            echo "" >> "$final_report"
            echo "---" >> "$final_report"
            echo "" >> "$final_report"
        fi
    done

    cat >> "$final_report" << EOF

## Raw Data Files

- Prometheus metrics: \`prometheus-*.log\`
- SQLite stats: \`sqlite-*.log\`
- System resources: \`system-*.log\`
- Metrics snapshots: \`snapshot-*.txt\`

## How to Analyze Further

\`\`\`bash
# View Prometheus metrics over time
cat ${METRICS_DIR}/prometheus-test2.log | column -t -s'|'

# View SQLite stats
cat ${METRICS_DIR}/sqlite-test2.log | column -t -s'|'

# Extract specific metric
grep "batch_duration_ms" ${METRICS_DIR}/prometheus-test2.log | awk -F'|' '{print \$1, \$2}'
\`\`\`

EOF

    log_info "Monitoring summary saved to: $final_report"
}

# ============================================================================
# Main Monitoring Orchestration
# ============================================================================

cleanup_monitors() {
    log_info "Cleaning up monitors..."
    stop_monitors
}

trap cleanup_monitors EXIT INT TERM

main() {
    log_info "=========================================="
    log_info "Enhanced Performance Test Monitor"
    log_info "=========================================="
    log_info ""

    # Pre-flight checks
    if ! check_metrics_endpoint; then
        log_error "Metrics endpoint not available. Tests will run but monitoring will be limited."
        METRICS_AVAILABLE=false
    else
        METRICS_AVAILABLE=true
    fi

    # Check if app is running
    if ! lsof -ti :$APP_PORT > /dev/null 2>&1; then
        log_error "Application not running on port $APP_PORT"
        log_error "Start the application first: make run-with-logs"
        exit 1
    fi

    log_info "Starting performance test with monitoring..."
    log_info "Metrics will be saved to: $METRICS_DIR"
    log_info ""

    # Hook into test phases
    # We'll monitor during the actual test execution

    # Start baseline monitoring
    log_info "Capturing baseline metrics..."
    if [ "$METRICS_AVAILABLE" = true ]; then
        capture_metrics_snapshot "baseline"
    fi

    # Start monitors for Test 1
    log_info ""
    log_info "=========================================="
    log_info "Starting monitors for Test 1 (10 msg/sec)"
    log_info "=========================================="

    if [ "$METRICS_AVAILABLE" = true ]; then
        monitor_prometheus_metrics "test1" &
        PROMETHEUS_MONITOR_PID=$!
    fi

    monitor_sqlite_stats "test1" &
    SQLITE_MONITOR_PID=$!

    monitor_system_resources "test1" &
    SYSTEM_MONITOR_PID=$!

    # Wait for test 1 duration (approximately 70 seconds)
    log_info "Monitoring Test 1... (waiting 75 seconds)"
    sleep 75

    stop_monitors
    if [ "$METRICS_AVAILABLE" = true ]; then
        capture_metrics_snapshot "test1"
        analyze_metrics "test1"
    fi

    # Start monitors for Test 2
    log_info ""
    log_info "=========================================="
    log_info "Starting monitors for Test 2 (100 msg/sec)"
    log_info "=========================================="

    if [ "$METRICS_AVAILABLE" = true ]; then
        monitor_prometheus_metrics "test2" &
        PROMETHEUS_MONITOR_PID=$!
    fi

    monitor_sqlite_stats "test2" &
    SQLITE_MONITOR_PID=$!

    monitor_system_resources "test2" &
    SYSTEM_MONITOR_PID=$!

    log_info "Monitoring Test 2... (waiting 75 seconds)"
    sleep 75

    stop_monitors
    if [ "$METRICS_AVAILABLE" = true ]; then
        capture_metrics_snapshot "test2"
        analyze_metrics "test2"
    fi

    # Start monitors for Test 3
    log_info ""
    log_info "=========================================="
    log_info "Starting monitors for Test 3 (500 msg/sec)"
    log_info "=========================================="

    if [ "$METRICS_AVAILABLE" = true ]; then
        monitor_prometheus_metrics "test3" &
        PROMETHEUS_MONITOR_PID=$!
    fi

    monitor_sqlite_stats "test3" &
    SQLITE_MONITOR_PID=$!

    monitor_system_resources "test3" &
    SYSTEM_MONITOR_PID=$!

    log_info "Monitoring Test 3... (waiting 95 seconds)"
    sleep 95

    stop_monitors
    if [ "$METRICS_AVAILABLE" = true ]; then
        capture_metrics_snapshot "test3"
        analyze_metrics "test3"
    fi

    # Generate final report
    log_info ""
    log_info "=========================================="
    log_info "Generating Monitoring Report"
    log_info "=========================================="

    if [ "$METRICS_AVAILABLE" = true ]; then
        generate_monitoring_report
    fi

    log_info ""
    log_info "✅ Monitoring complete!"
    log_info ""
    log_info "📊 Results:"
    log_info "  - Metrics directory: $METRICS_DIR"
    if [ "$METRICS_AVAILABLE" = true ]; then
        log_info "  - Summary report: ${METRICS_DIR}/MONITORING_SUMMARY.md"
        log_info "  - Test 1 analysis: ${METRICS_DIR}/analysis-test1.md"
        log_info "  - Test 2 analysis: ${METRICS_DIR}/analysis-test2.md"
        log_info "  - Test 3 analysis: ${METRICS_DIR}/analysis-test3.md"
    fi
    log_info ""
    log_info "View summary:"
    if [ "$METRICS_AVAILABLE" = true ]; then
        log_info "  cat ${METRICS_DIR}/MONITORING_SUMMARY.md"
    fi
}

# Run if executed directly (not sourced)
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
