#!/bin/bash
set -euo pipefail

# synapsePlatform performance test suite.

REPORT_DIR="./performance-reports"
TIMESTAMP=${PERF_TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}
REPORT_FILE="${REPORT_DIR}/synapse-performance-report-${TIMESTAMP}.md"
METRICS_FILE="${REPORT_DIR}/metrics-${TIMESTAMP}.json"
DEBUG_LOG="${REPORT_DIR}/debug-${TIMESTAMP}.log"
LAG_LOG="${REPORT_DIR}/kafka-lag-${TIMESTAMP}.log"
APP_PORT=${APP_PORT:-8080}
KAFKA_BROKER=${KAFKA_BROKER:-"localhost:9092"}
KAFKA_TOPIC=${KAFKA_TOPIC:-"ingestion.raw"}
KAFKA_GROUP=${KAFKA_GROUP:-"synapse-platform-consumer"}
KAFKA_SERVICE=${KAFKA_SERVICE:-kafka}
DB_PATH=${DB_PATH:-"data.db"}
DOCKER_COMPOSE=${DOCKER_COMPOSE:-$(command -v docker-compose >/dev/null 2>&1 && echo docker-compose || echo "docker compose")}
REQUIRE_CLEAN_LAG=${REQUIRE_CLEAN_LAG:-1}

RUN_FAILED=0
TEST_JSON=""

mkdir -p "$REPORT_DIR"

log_info() {
    echo "[INFO] $1" | tee -a "$DEBUG_LOG"
}

log_warn() {
    echo "[WARN] $1" | tee -a "$DEBUG_LOG"
}

log_error() {
    echo "[ERROR] $1" | tee -a "$DEBUG_LOG"
}

log_debug() {
    echo "[DEBUG] $1" >> "$DEBUG_LOG"
}

mark_run_failed() {
    RUN_FAILED=1
    log_error "$1"
}

cleanup() {
    if [ "${LAG_MONITOR_PID:-}" != "" ]; then
        kill "$LAG_MONITOR_PID" 2>/dev/null || true
    fi
    if [ "${PROCESS_MONITOR_PID:-}" != "" ]; then
        kill "$PROCESS_MONITOR_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

check_app() {
    if ! lsof -ti ":$APP_PORT" >/dev/null 2>&1; then
        log_error "Application not running on port $APP_PORT"
        exit 1
    fi

    APP_PID=$(lsof -ti ":$APP_PORT" | head -1)
    log_info "Application is running (PID: $APP_PID)"
}

get_process_stats() {
    local pid
    pid=$(lsof -ti ":$APP_PORT" | head -1)
    ps -p "$pid" -o %cpu,%mem,rss,vsz | tail -1 | xargs
}

get_db_stats() {
    sqlite3 "$DB_PATH" <<'SQL'
SELECT
    (SELECT COUNT(*) FROM events) || '|' ||
    (SELECT COUNT(*) FROM failed_messages) || '|' ||
    (SELECT page_count * page_size / 1024.0 / 1024.0 FROM pragma_page_count(), pragma_page_size());
SQL
}

get_event_count() {
    sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM events;"
}

get_failed_count() {
    sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM failed_messages;"
}

get_store_accounting() {
    local value
    value=$(sqlite3 "$DB_PATH" \
        "SELECT attempted_events || '|' || inserted_events || '|' || duplicate_events FROM store_accounting WHERE id = 1;" \
        2>/dev/null || true)

    if echo "$value" | grep -Eq '^[0-9]+\|[0-9]+\|[0-9]+$'; then
        echo "$value"
    else
        echo "0|0|0"
    fi
}

get_metric_part() {
    local value=$1
    local field=$2
    echo "$value" | cut -d'|' -f"$field"
}

get_kafka_lag() {
    local lag
    lag=$($DOCKER_COMPOSE exec -T "$KAFKA_SERVICE" kafka-consumer-groups \
        --bootstrap-server localhost:9092 \
        --group "$KAFKA_GROUP" \
        --describe 2>/dev/null | awk -v topic="$KAFKA_TOPIC" '
            $1 == topic && $6 ~ /^[0-9]+$/ {sum += $6; found = 1}
            END {if (found) print sum; else print 0}
        ' || echo "0")

    if echo "$lag" | grep -Eq '^[0-9]+$'; then
        echo "$lag"
    else
        echo "0"
    fi
}

send_kafka_stream() {
    $DOCKER_COMPOSE exec -T "$KAFKA_SERVICE" kafka-console-producer \
        --bootstrap-server "$KAFKA_BROKER" \
        --topic "$KAFKA_TOPIC"
}

generate_messages() {
    local count=$1
    local sleep_seconds=$2

    python3 - "$count" "$sleep_seconds" <<'PY'
import json
import sys
import time
from datetime import datetime, timedelta, timezone

count = int(sys.argv[1])
sleep_seconds = float(sys.argv[2])
base = datetime.now(timezone.utc)

for i in range(1, count + 1):
    ts = base + timedelta(microseconds=i)
    timestamp = ts.strftime("%Y-%m-%dT%H:%M:%S") + f".{ts.microsecond:06d}Z"
    msg = {
        "device_id": f"sensor-{i % 100:03d}",
        "type": "temperature_sensor",
        "timestamp": timestamp,
        "metrics": {
            "temperature_c": round(20 + ((i % 100) / 10.0), 1),
            "humidity_percent": 45.0,
            "air_quality_index": 35,
            "sequence": i,
        },
    }
    print(json.dumps(msg, separators=(",", ":")), flush=True)
    if sleep_seconds > 0:
        time.sleep(sleep_seconds)
PY
}

monitor_kafka_lag() {
    local test_name=$1
    > "$LAG_LOG"

    while true; do
        local lag now
        lag=$(get_kafka_lag)
        now=$(date +%H:%M:%S)
        echo "$now $lag" >> "$LAG_LOG"
        log_debug "[$test_name] Kafka LAG: $lag"
        sleep 1
    done
}

stop_lag_monitoring() {
    if [ "${LAG_MONITOR_PID:-}" != "" ]; then
        kill "$LAG_MONITOR_PID" 2>/dev/null || true
        wait "$LAG_MONITOR_PID" 2>/dev/null || true
        LAG_MONITOR_PID=""
    fi

    if [ ! -s "$LAG_LOG" ]; then
        echo "0|0|0"
        return
    fi

    awk '
        $2 ~ /^[0-9]+$/ {
            if ($2 > peak) peak = $2
            sum += $2
            count++
            final = $2
        }
        END {
            if (count == 0) {
                print "0|0|0"
            } else {
                printf "%d|%d|%d\n", peak, int(sum / count), final
            }
        }
    ' "$LAG_LOG"
}

monitor_process_stats() {
    local test_name=$1
    local stats_log="${REPORT_DIR}/process-stats-${test_name}-${TIMESTAMP}.log"
    > "$stats_log"

    while true; do
        echo "$(date +%H:%M:%S) $(get_process_stats)" >> "$stats_log"
        sleep 2
    done
}

stop_process_monitoring() {
    if [ "${PROCESS_MONITOR_PID:-}" != "" ]; then
        kill "$PROCESS_MONITOR_PID" 2>/dev/null || true
        wait "$PROCESS_MONITOR_PID" 2>/dev/null || true
        PROCESS_MONITOR_PID=""
    fi
}

check_sqlite_config() {
    local journal_mode busy_timeout synchronous
    journal_mode=$(sqlite3 "$DB_PATH" "PRAGMA journal_mode;")
    busy_timeout=$(sqlite3 "$DB_PATH" "PRAGMA busy_timeout;")
    synchronous=$(sqlite3 "$DB_PATH" "PRAGMA synchronous;")

    log_debug "SQLite journal_mode: $journal_mode"
    log_debug "SQLite busy_timeout on CLI connection: $busy_timeout"
    log_debug "SQLite synchronous on CLI connection: $synchronous"

    if [ "$journal_mode" != "wal" ]; then
        log_warn "SQLite is not in WAL mode (current: $journal_mode)"
    else
        log_info "SQLite is in WAL mode"
    fi

    echo "$journal_mode|$busy_timeout|$synchronous"
}

check_indexes() {
    if sqlite3 "$DB_PATH" "SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='events';" | grep -q "idx_ingested_event"; then
        echo "yes"
    else
        echo "no"
    fi
}

append_test_json() {
    local json=$1
    if [ "$TEST_JSON" = "" ]; then
        TEST_JSON="$json"
    else
        TEST_JSON="${TEST_JSON},${json}"
    fi
}

run_test() {
    local test_id=$1
    local title=$2
    local target_rate=$3
    local sent=$4
    local sleep_seconds=$5
    local wait_seconds=$6

    log_info "Starting Test $test_id: $target_rate msg/sec target ($sent messages)"

    local start_time end_time duration
    local start_events end_events inserted
    local start_failed end_failed failed_delta
    local start_accounting end_accounting
    local start_attempted start_inserted start_duplicates
    local end_attempted end_inserted end_duplicates
    local consumed duplicates accounting_inserted
    local start_lag lag_stats peak_lag avg_lag final_lag committed
    local throughput success_rate commit_rate process_stats error_count

    start_time=$(date +%s)
    start_events=$(get_event_count)
    start_failed=$(get_failed_count)
    start_accounting=$(get_store_accounting)
    start_attempted=$(get_metric_part "$start_accounting" 1)
    start_inserted=$(get_metric_part "$start_accounting" 2)
    start_duplicates=$(get_metric_part "$start_accounting" 3)
    start_lag=$(get_kafka_lag)

    monitor_kafka_lag "Test$test_id" &
    LAG_MONITOR_PID=$!
    monitor_process_stats "test${test_id}" &
    PROCESS_MONITOR_PID=$!

    generate_messages "$sent" "$sleep_seconds" | send_kafka_stream >/dev/null 2>&1

    end_time=$(date +%s)
    duration=$((end_time - start_time))
    if [ "$duration" -le 0 ]; then
        duration=1
    fi

    log_info "Messages sent for Test $test_id. Waiting ${wait_seconds}s for processing..."
    sleep "$wait_seconds"

    lag_stats=$(stop_lag_monitoring)
    stop_process_monitoring

    peak_lag=$(get_metric_part "$lag_stats" 1)
    avg_lag=$(get_metric_part "$lag_stats" 2)
    final_lag=$(get_metric_part "$lag_stats" 3)

    end_events=$(get_event_count)
    end_failed=$(get_failed_count)
    end_accounting=$(get_store_accounting)
    end_attempted=$(get_metric_part "$end_accounting" 1)
    end_inserted=$(get_metric_part "$end_accounting" 2)
    end_duplicates=$(get_metric_part "$end_accounting" 3)

    inserted=$((end_events - start_events))
    failed_delta=$((end_failed - start_failed))
    consumed=$((end_attempted - start_attempted))
    accounting_inserted=$((end_inserted - start_inserted))
    duplicates=$((end_duplicates - start_duplicates))
    committed=$((sent + start_lag - final_lag))
    if [ "$committed" -lt 0 ]; then
        committed=0
    fi

    if [ "$consumed" -eq 0 ]; then
        consumed=$((inserted + duplicates + failed_delta))
    fi

    throughput=$(awk "BEGIN {printf \"%.1f\", $inserted / $duration}")
    success_rate=$(awk "BEGIN {printf \"%.2f\", ($inserted / $sent) * 100}")
    commit_rate=$(awk "BEGIN {printf \"%.2f\", ($committed / $sent) * 100}")
    process_stats=$(get_process_stats)
    error_count=$(grep -i "error\\|failed\\|panic" "$DEBUG_LOG" 2>/dev/null | wc -l | xargs || echo "0")

    log_info "Test $test_id complete: inserted=$inserted consumed=$consumed duplicates=$duplicates failed=$failed_delta committed=$committed final_lag=$final_lag"

    cat >> "$REPORT_FILE" <<EOF
### Test $test_id: $title

| Metric | Value |
|--------|-------|
| **Target Rate** | $target_rate msg/sec |
| **Duration** | ${duration}s |
| **Messages Sent** | $sent |
| **Messages Consumed** | $consumed |
| **Messages Inserted** | $inserted |
| **Duplicate Messages Ignored** | $duplicates |
| **Failed Messages** | $failed_delta |
| **Kafka Messages Committed** | $committed |
| **Insert Success Rate** | ${success_rate}% |
| **Commit Rate** | ${commit_rate}% |
| **Actual Throughput** | ${throughput} msg/sec |
| **Peak Kafka LAG** | $peak_lag |
| **Average Kafka LAG** | $avg_lag |
| **Final Kafka LAG** | $final_lag |
| **Error Count** | $error_count |
| **Process Stats (CPU% MEM% RSS VSZ)** | $process_stats |

**Analysis:**
EOF

    if [ "$duplicates" -gt 0 ]; then
        echo "- Duplicate messages were ignored by storage. Check producer uniqueness and deterministic event IDs." >> "$REPORT_FILE"
    fi
    if [ "$final_lag" -ne 0 ]; then
        echo "- Final Kafka lag is nonzero; this run is failed." >> "$REPORT_FILE"
        mark_run_failed "Test $test_id finished with nonzero Kafka lag: $final_lag"
    fi
    if [ "$inserted" -lt "$sent" ] && [ "$duplicates" -eq 0 ] && [ "$failed_delta" -eq 0 ]; then
        echo "- Inserted count is below sent count without duplicate/failure accounting; investigate in-flight or commit accounting." >> "$REPORT_FILE"
    fi
    if [ "$accounting_inserted" -ne "$inserted" ]; then
        echo "- Store accounting inserted delta ($accounting_inserted) differs from event table delta ($inserted)." >> "$REPORT_FILE"
    fi

    echo "" >> "$REPORT_FILE"

    append_test_json "$(cat <<EOF
{"id":$test_id,"target_rate":$target_rate,"sent":$sent,"duration_seconds":$duration,"consumed":$consumed,"inserted":$inserted,"duplicates":$duplicates,"failed":$failed_delta,"committed":$committed,"insert_success_rate":$success_rate,"commit_rate":$commit_rate,"throughput":$throughput,"peak_lag":$peak_lag,"avg_lag":$avg_lag,"final_lag":$final_lag}
EOF
)"

    eval "TEST${test_id}_THROUGHPUT=$throughput"
    eval "TEST${test_id}_SUCCESS=$success_rate"
    eval "TEST${test_id}_PEAK_LAG=$peak_lag"
    eval "TEST${test_id}_AVG_LAG=$avg_lag"
    eval "TEST${test_id}_FINAL_LAG=$final_lag"
    eval "TEST${test_id}_INSERTED=$inserted"
    eval "TEST${test_id}_DUPLICATES=$duplicates"
    eval "TEST${test_id}_FAILED=$failed_delta"
    eval "TEST${test_id}_COMMITTED=$committed"
}

write_metrics_json() {
    local final_lag=$1
    local total_events=$2
    local total_failed=$3
    local query_latency=$4
    local journal_mode=$5
    local busy_timeout=$6
    local synchronous=$7

    cat > "$METRICS_FILE" <<EOF
{
  "timestamp": "$TIMESTAMP",
  "report": "$(basename "$REPORT_FILE")",
  "git_commit": "$(git rev-parse --short HEAD 2>/dev/null || echo "N/A")",
  "sqlite": {
    "journal_mode": "$journal_mode",
    "busy_timeout_cli_connection_ms": $busy_timeout,
    "synchronous_cli_connection": $synchronous
  },
  "totals": {
    "events": $total_events,
    "failed_messages": $total_failed,
    "final_kafka_lag": $final_lag,
    "query_latency_ms": $query_latency
  },
  "tests": [$TEST_JSON],
  "status": "$([ "$RUN_FAILED" -eq 0 ] && echo "passed" || echo "failed")"
}
EOF
}

rebuild_index() {
    python3 - "$REPORT_DIR" <<'PY'
import re
import sys
from pathlib import Path

report_dir = Path(sys.argv[1])
reports = sorted(report_dir.glob("synapse-performance-report-*.md"), reverse=True)

def cell(text):
    return text.strip().replace("|", "\\|")

rows = []
for report in reports:
    text = report.read_text(errors="ignore")
    timestamp = report.stem.replace("synapse-performance-report-", "")
    test2 = re.search(r"\| Test 2 \| [^|]+ \| ([0-9.]+) msg/sec \| ([0-9.]+)% \| ([0-9,]+) \| ([0-9,]+) \|", text)
    final_lags = re.findall(r"\*\*Final Kafka LAG\*\* \| ([0-9,]+)", text)
    query = re.search(r"\*\*List Query \(20 items\)\*\* \| ([0-9]+)ms", text)
    complete = "## Summary" in text and test2 is not None
    if test2:
        throughput, success, peak, avg = test2.groups()
    else:
        throughput = success = peak = avg = ""
    final_lag = final_lags[-1] if final_lags else ""
    query_ms = f"{query.group(1)}ms" if query else ""
    status = "complete" if complete and final_lag in ("", "0") else ("failed-lag" if final_lag not in ("", "0") else "incomplete")
    rows.append([timestamp, f"[Report](./{report.name})", throughput, success, peak, avg, final_lag, query_ms, status])

out = [
    "# Performance Test History",
    "",
    "| Date | Report | Test 2 Throughput | Success Rate | Peak LAG | Avg LAG | Final LAG | Query Latency | Status |",
    "|------|--------|-------------------|--------------|----------|---------|-----------|---------------|--------|",
]
for row in rows:
    throughput = f"{row[2]} msg/sec" if row[2] else ""
    success = f"{row[3]}%" if row[3] else ""
    out.append(f"| {cell(row[0])} | {row[1]} | {throughput} | {success} | {row[4]} | {row[5]} | {row[6]} | {row[7]} | {row[8]} |")

(report_dir / "INDEX.md").write_text("\n".join(out) + "\n")
PY
}

generate_comparison_chart() {
    python3 - "$REPORT_DIR" <<'PY'
import re
import sys
from pathlib import Path

report_dir = Path(sys.argv[1])
entries = []
for report in sorted(report_dir.glob("synapse-performance-report-*.md"), reverse=True):
    text = report.read_text(errors="ignore")
    timestamp = report.stem.replace("synapse-performance-report-", "")
    m = re.search(r"\| Test 2 \| [^|]+ \| ([0-9.]+) msg/sec \| ([0-9.]+)% \| ([0-9,]+) \| ([0-9,]+) \|", text)
    if not m:
        continue
    final_lags = re.findall(r"\*\*Final Kafka LAG\*\* \| ([0-9,]+)", text)
    query = re.search(r"\*\*List Query \(20 items\)\*\* \| ([0-9]+)ms", text)
    throughput = float(m.group(1))
    entries.append({
        "timestamp": timestamp,
        "throughput": throughput,
        "success": m.group(2),
        "peak": m.group(3),
        "avg": m.group(4),
        "final": final_lags[-1] if final_lags else "",
        "query": f"{query.group(1)}ms" if query else "",
    })

out = [
    "# Performance Comparison Chart",
    "",
    "## Throughput Trend (Test 2: 100 msg/sec target)",
    "",
]
for entry in entries[:10]:
    bars = "#" * max(1, int(round(entry["throughput"] / 5)))
    out.append(f"| {entry['timestamp']} | {bars} {entry['throughput']:.1f} msg/sec |")

out.extend([
    "",
    "## Recent Performance Metrics",
    "",
    "| Date | Test 2 Throughput | Success | Peak LAG | Avg LAG | Final LAG | Query Time | Status |",
    "|------|-------------------|---------|----------|---------|-----------|------------|--------|",
])
for entry in entries[:10]:
    t = entry["throughput"]
    if t >= 100:
        status = "excellent"
    elif t >= 50:
        status = "good"
    elif t >= 25:
        status = "moderate"
    else:
        status = "poor"
    if entry["final"] not in ("", "0"):
        status = "failed-lag"
    out.append(f"| {entry['timestamp']} | {t:.1f} msg/sec | {entry['success']}% | {entry['peak']} | {entry['avg']} | {entry['final']} | {entry['query']} | {status} |")

if entries:
    complete_entries = [entry for entry in entries if entry["final"] in ("", "0")]
    best = max(complete_entries or entries, key=lambda e: e["throughput"])
    latest = entries[0]
    out.extend([
        "",
        "## Performance Insights",
        "",
        f"- Target: 100 msg/sec for Test 2.",
        f"- Best complete Test 2 throughput in this history: {best['throughput']:.1f} msg/sec ({best['timestamp']}).",
        f"- Latest Test 2 throughput: {latest['throughput']:.1f} msg/sec ({latest['timestamp']}).",
    ])

(report_dir / "COMPARISON.md").write_text("\n".join(out) + "\n")
PY
}

verify_report_complete() {
    for pattern in "### Test 1:" "### Test 2:" "### Test 3:" "## Summary" "## Database Performance"; do
        if ! grep -q "$pattern" "$REPORT_FILE"; then
            mark_run_failed "Report is incomplete: missing $pattern"
        fi
    done
}

log_info "Starting Performance Test Suite at $(date)"
log_info "Report will be saved to: $REPORT_FILE"
log_info "Debug log: $DEBUG_LOG"

check_app

SQLITE_CONFIG=$(check_sqlite_config)
SQLITE_JOURNAL=$(get_metric_part "$SQLITE_CONFIG" 1)
SQLITE_BUSY_TIMEOUT=$(get_metric_part "$SQLITE_CONFIG" 2)
SQLITE_SYNCHRONOUS=$(get_metric_part "$SQLITE_CONFIG" 3)
INDEX_STATUS=$(check_indexes)

BASELINE_STATS=$(get_process_stats)
BASELINE_DB=$(get_db_stats)
BASELINE_EVENTS=$(get_event_count)
BASELINE_FAILED=$(get_failed_count)
BASELINE_ACCOUNTING=$(get_store_accounting)
BASELINE_LAG=$(get_kafka_lag)

if [ "$REQUIRE_CLEAN_LAG" = "1" ] && [ "$BASELINE_LAG" -ne 0 ]; then
    log_error "Baseline Kafka lag is $BASELINE_LAG. Clear lag or run with REQUIRE_CLEAN_LAG=0."
    exit 1
fi

cat > "$REPORT_FILE" <<EOF
# synapsePlatform Performance Test Report

**Test Date:** $(date)
**Git Commit:** $(git rev-parse --short HEAD 2>/dev/null || echo "N/A")
**Machine:** $(uname -m)
**OS:** $(uname -s) $(uname -r)

---

## Pre-Flight Diagnostics

### SQLite Configuration
\`\`\`
Journal Mode: $SQLITE_JOURNAL
Busy Timeout (CLI connection): ${SQLITE_BUSY_TIMEOUT}ms
Synchronous (CLI connection): $SQLITE_SYNCHRONOUS
\`\`\`

Note: busy_timeout and synchronous are connection-local when read through sqlite3 CLI. The application sets its own pragmas at startup.

### Index Status
- Pagination Index (idx_ingested_event): **$INDEX_STATUS**

### Baseline Metrics
\`\`\`
Process Stats: $BASELINE_STATS
Database: $BASELINE_DB
Events in DB: $BASELINE_EVENTS
Failed Messages: $BASELINE_FAILED
Store Accounting: $BASELINE_ACCOUNTING
Kafka LAG: $BASELINE_LAG
\`\`\`

---

## Test Results

EOF

run_test 1 "Low Load (10 msg/sec)" 10 600 0.1 10
run_test 2 "Medium Load (100 msg/sec)" 100 6000 0.01 10
run_test 3 "High Load (500 msg/sec)" 500 30000 0.001 30

FINAL_DB_STATS=$(get_db_stats)
TOTAL_EVENTS=$(get_event_count)
TOTAL_FAILED=$(get_failed_count)
FINAL_ACCOUNTING=$(get_store_accounting)
FINAL_LAG=$(get_kafka_lag)

QUERY_LATENCY=$(python3 - "$DB_PATH" <<'PY'
import sqlite3
import sys
import time

start = time.perf_counter()
conn = sqlite3.connect(sys.argv[1])
try:
    list(conn.execute("SELECT * FROM events ORDER BY ingested_at DESC LIMIT 20"))
finally:
    conn.close()
print(round((time.perf_counter() - start) * 1000))
PY
)

FAILED_BY_STAGE=$(sqlite3 "$DB_PATH" "SELECT stage, COUNT(*) FROM failed_messages GROUP BY stage;" 2>/dev/null || echo "N/A")

cat >> "$REPORT_FILE" <<EOF
---

## Database Performance

| Metric | Value |
|--------|-------|
| **Total Events** | $TOTAL_EVENTS |
| **Total Failed** | $TOTAL_FAILED |
| **Database Stats** | $FINAL_DB_STATS |
| **Store Accounting (attempted|inserted|duplicates)** | $FINAL_ACCOUNTING |
| **List Query (20 items)** | ${QUERY_LATENCY}ms |
| **Final Kafka LAG** | $FINAL_LAG |

### Failed Messages Breakdown
\`\`\`
$FAILED_BY_STAGE
\`\`\`

---

## Summary

| Test | Target Rate | Actual Throughput | Insert Success Rate | Peak LAG | Avg LAG | Final LAG |
|------|-------------|-------------------|---------------------|----------|---------|-----------|
| Test 1 | 10 msg/sec | ${TEST1_THROUGHPUT} msg/sec | ${TEST1_SUCCESS}% | $TEST1_PEAK_LAG | $TEST1_AVG_LAG | $TEST1_FINAL_LAG |
| Test 2 | 100 msg/sec | ${TEST2_THROUGHPUT} msg/sec | ${TEST2_SUCCESS}% | $TEST2_PEAK_LAG | $TEST2_AVG_LAG | $TEST2_FINAL_LAG |
| Test 3 | 500 msg/sec | ${TEST3_THROUGHPUT} msg/sec | ${TEST3_SUCCESS}% | $TEST3_PEAK_LAG | $TEST3_AVG_LAG | $TEST3_FINAL_LAG |

---

## Bottleneck Analysis

EOF

if [ "$SQLITE_JOURNAL" != "wal" ]; then
    cat >> "$REPORT_FILE" <<EOF
### Critical: SQLite Not in WAL Mode
- Current mode: $SQLITE_JOURNAL
- Fix: enable PRAGMA journal_mode=WAL on the application connection.

EOF
fi

if [ "$INDEX_STATUS" = "no" ]; then
    cat >> "$REPORT_FILE" <<EOF
### Missing Pagination Index
- Impact: slow list queries (${QUERY_LATENCY}ms for 20 items).
- Fix: create idx_ingested_event on events(ingested_at DESC, event_id DESC).

EOF
fi

if [ "$FINAL_LAG" -ne 0 ]; then
    cat >> "$REPORT_FILE" <<EOF
### Run Failed: Nonzero Final Kafka Lag
- Final Kafka lag: $FINAL_LAG

EOF
    mark_run_failed "Final Kafka lag is nonzero: $FINAL_LAG"
fi

cat >> "$REPORT_FILE" <<EOF
---

## Monitoring Artifacts

- Debug log: $DEBUG_LOG
- Kafka lag log: $LAG_LOG
- Process stats logs: ${REPORT_DIR}/process-stats-test*-${TIMESTAMP}.log
- Structured metrics: $METRICS_FILE

For batch duration, batch size, channel utilization, WAL pages, CPU, memory, and goroutines, run:
\`\`\`bash
test/monitored_perf_test.sh
\`\`\`

---

**Test completed at:** $(date)
EOF

verify_report_complete
write_metrics_json "$FINAL_LAG" "$TOTAL_EVENTS" "$TOTAL_FAILED" "$QUERY_LATENCY" "$SQLITE_JOURNAL" "$SQLITE_BUSY_TIMEOUT" "$SQLITE_SYNCHRONOUS"

ln -sf "$(basename "$REPORT_FILE")" "$REPORT_DIR/latest.md"
rebuild_index
generate_comparison_chart

log_info "Performance test complete"
log_info "Report saved to: $REPORT_FILE"
log_info "Structured metrics saved to: $METRICS_FILE"
log_info "Index rebuilt: $REPORT_DIR/INDEX.md"
log_info "Comparison chart rebuilt: $REPORT_DIR/COMPARISON.md"

if [ "$RUN_FAILED" -ne 0 ]; then
    exit 1
fi
