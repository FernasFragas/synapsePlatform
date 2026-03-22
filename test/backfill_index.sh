#!/bin/bash

# Backfill INDEX.md from existing reports
backfill_index() {
    local REPORT_DIR="./performance-reports"
    local INDEX_FILE="${REPORT_DIR}/INDEX.md"
    local TEMP_FILE="${REPORT_DIR}/.index_backfill.tmp"

    echo "🔄 Backfilling INDEX.md from existing reports..."

    # Create header
    cat > "$TEMP_FILE" << 'EOF'
# Performance Test History

| Date | Report | Throughput (Test 2) | Success Rate | Peak LAG | Avg LAG | Query Latency |
|------|--------|---------------------|--------------|----------|---------|---------------|
EOF

    # Process each report in reverse chronological order (newest first)
    for report in $(ls -t "$REPORT_DIR"/synapse-performance-report-*.md 2>/dev/null); do
        local filename=$(basename "$report")
        local timestamp=$(echo "$filename" | sed 's/synapse-performance-report-//' | sed 's/\.md//')

        echo "  Processing: $filename"

        # Extract Test 2 metrics from the Summary table
        local test2_line=$(grep -A 3 "| Test | Target Rate |" "$report" | grep "Test 2" | head -n 1)

        if [ -n "$test2_line" ]; then
            # Parse the table row
            # Format: | Test 2 | 100 msg/sec | 23.8 msg/sec | 100.00% | 60 | 27 |
            local throughput=$(echo "$test2_line" | awk -F'|' '{print $4}' | xargs)
            local success=$(echo "$test2_line" | awk -F'|' '{print $5}' | xargs)
            local peak_lag=$(echo "$test2_line" | awk -F'|' '{print $6}' | xargs)
            local avg_lag=$(echo "$test2_line" | awk -F'|' '{print $7}' | xargs)

            # Extract query latency
            local query_latency=$(grep "List Query" "$report" | awk -F'|' '{print $3}' | xargs)

            # Add row to index
            echo "| $timestamp | [Report](./$filename) | $throughput | $success | $peak_lag | $avg_lag | $query_latency |" >> "$TEMP_FILE"

            echo "    ✓ Extracted: Throughput=$throughput, Success=$success, Peak LAG=$peak_lag"
        else
            echo "    ⚠️  Could not extract metrics (report may be incomplete)"
            # Add row with N/A values
            echo "| $timestamp | [Report](./$filename) | N/A | N/A | N/A | N/A | N/A |" >> "$TEMP_FILE"
        fi
    done

    # Replace old INDEX with new one
    mv "$TEMP_FILE" "$INDEX_FILE"

    echo "✅ INDEX.md backfilled successfully!"
    echo ""
    echo "Preview:"
    head -n 10 "$INDEX_FILE"
}

# Run the backfill
backfill_index