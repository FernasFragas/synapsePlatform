#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPORT_DIR="${REPORT_DIR:-${REPO_ROOT}/performance-reports}"
OUTPUT="${OUTPUT:-${REPORT_DIR}/EVOLUTION.md}"

mkdir -p "$(dirname "$OUTPUT")"

awk -v output="$OUTPUT" '
function clean(value,    esc) {
    esc = sprintf("%c", 27)
    gsub(esc "\\[[0-9;]*m", "", value)
    gsub(/\*\*/, "", value)
    gsub(/\r/, "", value)
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
    gsub(/[[:space:]]+/, " ", value)
    return value
}

function alias_metric(metric) {
    metric = clean(metric)
    if (metric == "Messages Processed") return "Messages Inserted"
    if (metric == "Success Rate") return "Insert Success Rate"
    if (metric == "Avg LAG") return "Average Kafka LAG"
    if (metric == "Process Stats") return "Process Stats (CPU% MEM% RSS VSZ)"
    return metric
}

function markdown_cell(value) {
    value = clean(value)
    gsub(/\|/, "\\|", value)
    return value
}

function number(value,    match_value) {
    value = clean(value)
    gsub(/,/, "", value)
    if (match(value, /-?[0-9]+(\.[0-9]+)?/)) {
        match_value = substr(value, RSTART, RLENGTH)
        return match_value + 0
    }
    return ""
}

function better(metric, delta) {
    if (metric == "Duration" || metric == "Failed Messages" || metric == "Peak Kafka LAG" || metric == "Average Kafka LAG" || metric == "Final Kafka LAG" || metric == "Error Count") {
        return delta < 0 ? "improved" : "got worse"
    }
    if (metric == "Messages Consumed" || metric == "Messages Inserted" || metric == "Kafka Messages Committed" || metric == "Insert Success Rate" || metric == "Commit Rate" || metric == "Actual Throughput") {
        return delta > 0 ? "improved" : "got worse"
    }
    return "changed"
}

function remember_run(timestamp) {
    if (!(timestamp in run_seen)) {
        run_seen[timestamp] = 1
        run_count++
        runs[run_count] = timestamp
    }
}

function remember_metric(test_id, metric) {
    metric_seen[test_id SUBSEP metric] = 1
}

function update_numeric(test_id, metric, timestamp, value,    n, key) {
    n = number(value)
    if (n == "") return

    key = test_id SUBSEP metric
    if (!(key in first_value)) {
        first_value[key] = n
        first_ts[key] = timestamp
    }
    last_value[key] = n
    last_ts[key] = timestamp
}

function outcome(test_id, metric,    key, delta, direction) {
    key = test_id SUBSEP metric
    if (!(key in first_value) || first_ts[key] == last_ts[key]) return ""

    delta = last_value[key] - first_value[key]
    if (delta == 0) {
        return metric " stayed stable at " last_value[key]
    }

    direction = delta > 0 ? "up" : "down"
    return metric " " better(metric, delta) ", moving " direction " from " first_value[key] " (" first_ts[key] ") to " last_value[key] " (" last_ts[key] ")"
}

function conclusion(test_id,    metrics, count, i, text, item) {
    split("Actual Throughput|Insert Success Rate|Failed Messages|Peak Kafka LAG|Final Kafka LAG", metrics, "|")
    text = ""
    count = 0

    for (i = 1; i <= 5; i++) {
        item = outcome(test_id, metrics[i])
        if (item == "") continue
        count++
        text = text (text == "" ? "" : "; ") item
    }

    if (count == 0) {
        return "Conclusion: not enough historical data to determine this test evolution."
    }
    return "Conclusion: " text "."
}

function print_test(test_id,    title, metrics, metric_count, i, run_idx, timestamp, metric, value, row) {
    title = test_title[test_id]
    print "## Test " test_id ": " title >> output
    print "" >> output

    metric_count = 0
    for (i = 1; i <= ordered_metric_count; i++) {
        metric = ordered_metrics[i]
        if ((test_id SUBSEP metric) in metric_seen) {
            metric_count++
            metrics[metric_count] = metric
        }
    }

    if (metric_count == 0) {
        print "No historical data found for this test." >> output
        print "" >> output
        return
    }

    row = "| Run | Report |"
    for (i = 1; i <= metric_count; i++) row = row " " markdown_cell(metrics[i]) " |"
    print row >> output

    row = "|-----|--------|"
    for (i = 1; i <= metric_count; i++) row = row "---|"
    print row >> output

    for (run_idx = 1; run_idx <= run_count; run_idx++) {
        timestamp = runs[run_idx]
        row = "| " markdown_cell(timestamp) " | [" markdown_cell(report_name[timestamp]) "](./" markdown_cell(report_name[timestamp]) ") |"
        for (i = 1; i <= metric_count; i++) {
            metric = metrics[i]
            value = values[timestamp, test_id, metric]
            if (value == "") value = "N/A"
            row = row " " markdown_cell(value) " |"
        }
        print row >> output
    }

    print "" >> output
    print conclusion(test_id) >> output
    print "" >> output
}

BEGIN {
    ordered_metric_count = split("Target Rate|Duration|Messages Sent|Messages Consumed|Messages Inserted|Duplicate Messages Ignored|Failed Messages|Kafka Messages Committed|Insert Success Rate|Commit Rate|Actual Throughput|Peak Kafka LAG|Average Kafka LAG|Final Kafka LAG|Error Count|Process Stats (CPU% MEM% RSS VSZ)", ordered_metrics, "|")
}

FNR == 1 {
    timestamp = FILENAME
    sub(/^.*synapse-performance-report-/, "", timestamp)
    sub(/\.md$/, "", timestamp)
    report = FILENAME
    sub(/^.*\//, "", report)
    report_name[timestamp] = report
    in_test = 0
}

/^### Test [123]:/ {
    in_test = 1
    test_id = $3
    sub(/:/, "", test_id)
    remember_run(timestamp)
    test_title[test_id] = clean(substr($0, index($0, ":") + 1))
    next
}

in_test && /^##[[:space:]]/ {
    in_test = 0
    pending_error = 0
    next
}

in_test && pending_error && /^[[:space:]]*[0-9]+[[:space:]]*\|/ {
    value = clean($0)
    sub(/\|.*$/, "", value)
    values[pending_ts, pending_test_id, pending_metric] = value
    update_numeric(pending_test_id, pending_metric, pending_ts, value)
    pending_error = 0
    next
}

in_test && /^\|/ {
    split($0, parts, "|")
    metric = alias_metric(parts[2])
    value = clean(parts[3])

    if (metric == "Metric" || metric == "--------" || value == "Value" || value == "-------") next
    if (metric == "" || value == "") next

    values[timestamp, test_id, metric] = value
    remember_metric(test_id, metric)

    if (metric == "Error Count" && value !~ /^-?[0-9]+(\.[0-9]+)?$/) {
        pending_error = 1
        pending_ts = timestamp
        pending_test_id = test_id
        pending_metric = metric
        next
    }

    update_numeric(test_id, metric, timestamp, value)
}

END {
    print "# Performance Test Evolution Report" > output
    print "" >> output
    print "This report compares every historical performance test report available in `performance-reports`." >> output
    print "" >> output
    print "Reports compared: " run_count >> output
    print "Oldest report: " (run_count ? runs[1] : "N/A") >> output
    print "Latest report: " (run_count ? runs[run_count] : "N/A") >> output
    print "" >> output

    if (run_count == 0) {
        print "No performance reports were found." >> output
        exit
    }

    print_test(1)
    print_test(2)
    print_test(3)
}
' "${REPORT_DIR}"/synapse-performance-report-*.md

COMPARED_COUNT="$(awk -F': ' '/^Reports compared:/ {print $2; exit}' "$OUTPUT")"
echo "Generated ${OUTPUT} from ${COMPARED_COUNT:-0} reports"
