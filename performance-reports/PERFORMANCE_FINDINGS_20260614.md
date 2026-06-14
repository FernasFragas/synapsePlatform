# Performance Findings - 2026-06-14

## Executive Summary

The latest complete run, `synapse-performance-report-20260614-094707.md`, is materially faster than the stable March baseline, but it still does not meet the 100 msg/sec medium-load target and has a correctness/measurement issue under high load.

Key findings:

- Medium-load throughput improved from the late-March baseline of about 20.2 msg/sec to 41.4 msg/sec, a 104% improvement, while keeping 100% success and final Kafka lag at 0.
- High-load throughput improved from about 26.1 msg/sec to 86.0 msg/sec, a 229% improvement, but only 28,539 of 30,000 sent messages were counted as inserted events.
- The June 13 run was not healthy: Kafka lag climbed monotonically to 36,606 and stayed there after the wait period.
- The June 14 run fixed the lag behavior, but the high-load missing rows remain unexplained by `failed_messages`, which stayed at 0.
- The report generator has several bugs that make `INDEX.md`, `COMPARISON.md`, regression checks, and the WAL/busy-timeout bottleneck section unreliable.

## Data Set Reviewed

Reviewed report files:

- March baseline reports: `synapse-performance-report-20260320-*.md` through `synapse-performance-report-20260328-*.md`
- June reports: `synapse-performance-report-20260613-214053.md`, `20260613-214123.md`, `20260613-214432.md`, and `20260614-094707.md`
- Debug and support logs: `debug-20260613-214432.log`, `debug-20260614-094707.log`, `kafka-lag-*.log`, `process-stats-*.log`, and `app-logs-20260328-124146.log`
- Runtime DB snapshot: `data.db`

Two June reports, `20260613-214053` and `20260613-214123`, are incomplete: they contain preflight diagnostics but no test results.

## Performance Comparison

### Stable March Baseline

Excluding the first failing March 20 run, the complete March runs averaged:

| Test | Target | Avg Throughput | Min-Max Throughput | Avg Success | Avg Peak Lag | Avg Lag |
|------|--------|----------------|--------------------|-------------|--------------|---------|
| Test 1 | 10 msg/sec | 6.2 msg/sec | 5.7-6.5 | 99.73% | 28.6 | 13.2 |
| Test 2 | 100 msg/sec | 22.2 msg/sec | 20.0-24.1 | 100.00% | 43.1 | 21.3 |
| Test 3 | 500 msg/sec | 29.5 msg/sec | 26.1-32.6 | 100.00% | 52.2 | 26.1 |

The late-March `20260328` runs are the cleanest baseline because they had no failed messages and stable list-query latency:

| Test | Avg Throughput | Avg Peak Lag | Avg Lag |
|------|----------------|--------------|---------|
| Test 1 | 6.5 msg/sec | 6.8 | 3.0 |
| Test 2 | 20.2 msg/sec | 14.8 | 6.2 |
| Test 3 | 26.1 msg/sec | 27.8 | 15.5 |

### Latest Run vs Late-March Baseline

Using `20260328-202303` as the direct baseline:

| Test | March 28 Throughput | June 14 Throughput | Change | Success Change | Peak Lag Change |
|------|---------------------|--------------------|--------|----------------|-----------------|
| Test 1 | 6.4 msg/sec | 8.2 msg/sec | +28.1% | 100.00% -> 100.00% | 7 -> 9 |
| Test 2 | 20.3 msg/sec | 41.4 msg/sec | +103.9% | 100.00% -> 100.00% | 22 -> 46 |
| Test 3 | 26.1 msg/sec | 86.0 msg/sec | +229.5% | 100.00% -> 95.13% | 27 -> 88 |

List-query latency is stable: 15 ms on March 28 and 15 ms on June 14.

## June Run Analysis

### June 13 Complete Run

`synapse-performance-report-20260613-214432.md` showed high throughput but severe Kafka lag:

| Test | Throughput | Success | Peak Lag | Avg Lag | Final Lag |
|------|------------|---------|----------|---------|-----------|
| Test 1 | 8.2 msg/sec | 100.00% | 606 | 336 | 606 |
| Test 2 | 41.7 msg/sec | 100.00% | 6,606 | 3,761 | 6,606 |
| Test 3 | 97.6 msg/sec | 90.08% | 36,606 | 23,057 | 36,606 |

The Kafka lag log is a monotonic climb from 6,606 at 21:48:31 to 36,606 at 21:53:07, then a flat 36,606 through the end of the wait period. That means the consumer group did not commit through the backlog during this run.

### June 14 Complete Run

`synapse-performance-report-20260614-094707.md` fixed the backlog profile:

| Test | Throughput | Success | Peak Lag | Avg Lag | Final Lag |
|------|------------|---------|----------|---------|-----------|
| Test 1 | 8.2 msg/sec | 100.00% | 9 | 2 | 0 |
| Test 2 | 41.4 msg/sec | 100.00% | 46 | 12 | 0 |
| Test 3 | 86.0 msg/sec | 95.13% | 88 | 21 | 0 |

The high-load lag log oscillates mostly between 0 and 38, with occasional peaks up to 88, then stays at 0 during the final wait. Operationally, this is much healthier than June 13.

## Anomalies and Likely Causes

### 1. High-Load "Success" Is Under-Counted Because Test Data Collides

Evidence:

- June 14 Test 3 sent 30,000 messages but only 28,539 new rows appeared.
- `failed_messages` stayed at 0.
- Kafka final lag was 0.
- `internal/ingestor/transformer.go` derives `event_id` deterministically from `device_id|type|timestamp`.
- `test/perform_test.sh` generates timestamps with `date -u -Iseconds`, which has second precision, and rotates through only 100 sensor IDs.
- At the improved high-load rate, the same `sensor-NNN` can appear multiple times in the same second, producing the same deterministic event ID.
- Storage uses `INSERT OR IGNORE`, so duplicates can be silently ignored in batch inserts.

Interpretation:

The missing 1,461 June 14 high-load rows are likely duplicate business keys created by the test generator after throughput improved enough to reuse the same sensor and second. This is a test harness/data-shape problem unless production is expected to ingest multiple readings per device per second.

Potential fixes:

- Change the performance generator timestamp to nanosecond precision, for example `date -u +"%Y-%m-%dT%H:%M:%S.%NZ"`.
- Add a sequence or sample ID to the business key if multiple same-device readings per second are valid production data.
- Stop using `INSERT OR IGNORE` without accounting. Return rows affected from batch inserts and record ignored duplicates as a separate metric.
- Split report metrics into `messages consumed`, `events inserted`, `duplicates ignored`, and `failed messages`.

### 2. June 13 Lag Was a Real Consumer Commit/Backlog Problem

Evidence:

- Kafka lag increased continuously through Test 1, Test 2, and Test 3.
- Final lag stayed at 36,606 after the 30 second high-load wait.
- The report still counted many messages as processed because it measures inserted DB rows, not committed Kafka messages.

Interpretation:

June 13 was not simply slow SQLite. The consumer group did not advance offsets for the backlog during the run. Since June 14 did not reproduce this, likely causes include stale consumer group state, app instance not consuming the group used by the lag command, topic/group mismatch, a stuck/old process, or the test starting with uncleared lag.

Potential fixes:

- Before each test, assert initial Kafka lag is 0 or explicitly reset/clear the test topic and consumer group.
- Log topic, partition, current offset, end offset, and committed offset in the report, not only aggregate lag.
- Fail the run if final lag remains nonzero after the wait period.
- Include app PID and consumer group in the lag log header.

### 3. Report Generator Mislabels WAL Status

Evidence:

- Reports often say `Critical: SQLite Not in WAL Mode` even when preflight says `SQLite is in WAL mode`.
- Current `internal/sqllite/storer.go` sets:
  - `PRAGMA journal_mode=WAL`
  - `PRAGMA busy_timeout=5000`
  - `PRAGMA synchronous=NORMAL`
- The `sqlite3 data.db` CLI connection reports `journal_mode=wal`, `busy_timeout=0`, `synchronous=1`.

Interpretation:

`busy_timeout` is connection-local, so checking it through a separate `sqlite3` process does not prove the app connection is missing it. The report should not recommend enabling WAL when WAL is already enabled. The current report section is generated from contaminated shell output, not a clean machine-readable value.

Potential fixes:

- Capture SQLite pragmas in machine-readable variables only, with logs redirected away from command substitution output.
- Add an app health/debug endpoint or startup log that reports the app connection pragmas.
- Change the bottleneck section to distinguish "WAL disabled" from "busy timeout could not be verified externally".

### 4. INDEX.md and COMPARISON.md Are Broken

Evidence:

- `INDEX.md` has empty metric columns for most rows.
- `COMPARISON.md` includes only one throughput trend entry and an empty recent metrics table.
- Debug logs show `Non-numeric throughput values - skipping regression check` and `Skipping invalid throughput value`.

Likely causes:

- The index header shape changed over time, but the parser still assumes fixed pipe-field positions.
- `generate_comparison_chart` strips `msg/sec` in one path but validates an unstripped throughput value in another path.
- `check_regression` reads the latest row before/after updates in a way that can compare against invalid or current data.

Potential fixes:

- Rebuild `INDEX.md` from the report files using a robust parser.
- Stop parsing markdown tables for regression logic; emit a JSON or CSV metrics artifact per run.
- Normalize timestamp format to `YYYYMMDD-HHMMSS`, not mixed `YYYYMMDD HHMMSS`.
- Add tests for the shell parser or replace this part with a small Go command.

### 5. Application Log Shows Shutdown Context Cancellation

Evidence from `app-logs-20260328-124146.log`:

- 605 `ERROR` entries.
- 601 are `failed to store failure`.
- 600 have stage `store_batch`.
- Main cause: `batch store failed: failed to insert chunk 0-90: failed to begin transaction: context canceled`.
- One `failed to poll message` also reports `context canceled`.
- One process error reports `message is nil`.
- The error burst happened around shutdown and ended with `shutdown`, reason `received signal: interrupt`.

Interpretation:

Most of this log is shutdown noise, but it exposes a real failure-mode gap: when a batch fails because the context is canceled, the code then tries to store failure records with the same canceled context, so failure recording also fails. This can leave consumed or in-flight messages hard to account for during shutdown.

Potential fixes:

- On shutdown, stop polling first, then drain/flush in-flight batches with a bounded grace context.
- Use a fresh short-lived context for failure persistence during shutdown, or intentionally skip failure persistence for context cancellation and log it as shutdown drain loss.
- Add shutdown metrics: in-flight deliveries, flushed events, skipped events, failed failure-store attempts.

### 6. CPU Is Not the Bottleneck

Evidence:

- Process stats during June runs show CPU around 0.1%-1.1% and memory under about 110 MB RSS.
- Throughput limit appears while CPU is mostly idle.

Interpretation:

The bottleneck is likely not raw compute. It is more likely single-writer SQLite serialization, Kafka producer/test harness pacing, batch flush/ack behavior, logging/metrics overhead, or consumer group measurement.

Potential fixes:

- Measure batch write latency through the existing Prometheus metrics in `test/performance_monitor.sh`.
- Capture poll duration, transform duration, store batch duration, ack duration, and channel utilization during each run.
- Tune one dimension at a time: batch size, batch timeout, worker count, Kafka reader bytes/wait settings, and logging level.

## Prioritized Action Plan

1. Fix the performance harness correctness first.
   - Use nanosecond timestamps or unique message IDs.
   - Report consumed, inserted, duplicate, failed, and committed counts separately.
   - Fail runs with nonzero final Kafka lag or incomplete report files.

2. Repair the history/comparison generator.
   - Regenerate `INDEX.md` from existing reports.
   - Fix the markdown parsing bugs or replace markdown parsing with structured metrics.
   - Remove the false "SQLite Not in WAL Mode" section when `journal_mode=wal`.

3. Add a monitored run.
   - Use `test/performance_monitor.sh` or `test/monitored_perf_test.sh` to collect batch duration, batch size, channel utilization, WAL pages, CPU, memory, and goroutines.
   - Compare June 14 against a new clean run after clearing Kafka lag and using collision-free messages.

4. Improve store accounting.
   - Replace silent batch `INSERT OR IGNORE` accounting with explicit row-count/duplicate metrics.
   - Consider returning an error or metric when a whole batch stores fewer rows than events.

5. Improve shutdown behavior.
   - Stop polling, drain in-flight work, then close.
   - Avoid using an already-canceled context for final failure persistence.

## Bottom Line

Performance has improved substantially since March. The latest medium-load result is roughly twice as fast as the late-March baseline, and high-load throughput is more than three times higher. The current system still does not meet the 100 msg/sec target in Test 2, and the latest high-load success rate is not trustworthy until the test generator stops producing duplicate deterministic event IDs.

The next reliable performance conclusion should come from a clean monitored run with unique high-load messages, verified zero starting lag, fixed report parsing, and explicit duplicate/insert accounting.
