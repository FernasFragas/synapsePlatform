# Synapse Platform — Grafana / Prometheus Metric Queries

Reference for verifying every metric the platform emits. Run queries in **Grafana Cloud → Explore** against the Prometheus data source (`grafanacloud-fernasfragas-prom`).

All series carry these labels: `job="synapse"`, `instance`, `otel_scope_name="synapse-platform"`, plus per-metric attributes (`status`, `operation`, `domain`, etc.).

## Naming Conventions (Read First)

OTel → Prometheus conversion applies:

- Dots become underscores: `ingestor.poller.duration` → `ingestor_poller_duration`.
- Timing histograms (unit `s`) get `_seconds`: → `ingestor_poller_duration_seconds`.
- Histograms expand into `_bucket`, `_sum`, `_count`.
- Counters get `_total`. Counters already ending in `.total` become **`_total_total`** (e.g. `ingestor_poller_total_total`). Prefer the histogram `_count` series instead.

Confirm the exact names emitted by your build:

```bash
curl -s localhost:8080/metrics | grep -v '#' | cut -d'{' -f1 | sort -u
```

## 0. Pipeline Health (Meta)

```promql
up{job="synapse"}
```

Whether Alloy is scraping the target. **Expect `1`.** `0` = scrape failing.

```promql
target_info{job="synapse"}
```

Resource metadata (service name). **Expect one series** with `service_name="synapse-platform"`.

```promql
scrape_duration_seconds{job="synapse"}
```

How long each scrape takes. **Expect a small value** (< 0.1s typically).

## 1. Poller — Kafka Read

Reads raw messages off Kafka. Runs continuously.

```promql
rate(ingestor_poller_duration_seconds_count{job="synapse", status="success"}[5m])
```

Successful poll throughput (polls/sec). **Expect > 0 while consuming.** Note: this includes empty-poll cycles, so it stays non-zero even with no traffic.

```promql
histogram_quantile(0.95, sum(rate(ingestor_poller_duration_seconds_bucket{job="synapse"}[5m])) by (le))
```

p95 poll latency in seconds. **Expect low values**; large spikes mean broker slowness.

```promql
rate(ingestor_poller_errors_total{job="synapse"}[5m])
```

Poll error rate. **Expect 0.** Sustained > 0 = broker/connectivity problems.

```promql
rate(ingestor_poller_ack_duration_seconds_count{job="synapse"}[5m])
```

Acknowledgment throughput. **Expect > 0 when messages commit.**

## 2. Processor — Decode

Decodes raw bytes into `DeviceMessage`.

```promql
rate(ingestor_process_data_duration_seconds_count{job="synapse", status="success"}[5m])
```

Successful decode rate. **Expect to track ingestion volume after `make kafka-send-sample`.**

```promql
rate(ingestor_process_data_errors_total{job="synapse"}[5m])
```

Decode errors. **Expect 0** for well-formed messages; > 0 indicates malformed payloads (these go to the DLQ).

```promql
histogram_quantile(0.95, sum(rate(ingestor_process_data_duration_seconds_bucket{job="synapse"}[5m])) by (le))
```

p95 decode latency.

## 3. Transformer — Map to Domain Event

Maps device messages to typed `BaseEvent`s and computes the deterministic ID.

```promql
sum(rate(ingestor_transform_duration_seconds_count{job="synapse", status="success"}[5m])) by (domain, event_type)
```

Transform rate **broken down by domain and event type** (Energy / Finance / Monitoring). **Expect one line per domain** you sent samples for.

```promql
histogram_quantile(0.95, sum(rate(ingestor_transform_duration_seconds_bucket{job="synapse"}[5m])) by (le))
```

p95 transform latency.

```promql
rate(ingestor_transform_errors_total{job="synapse"}[5m])
```

Transform errors by device type. **Expect 0** for known device types.

## 4. Storer — SQLite Writes

Two paths: per-event (`store_data`) and batched (`store_batch`).

```promql
sum(rate(ingestor_store_data_duration_seconds_count{job="synapse", status="success"}[5m])) by (domain)
```

Per-event store rate by domain. **Expect > 0 during ingestion.**

```promql
ingestor_store_batch_size_sum{job="synapse"}
```

Cumulative count of events written across all batches. **Expect to roughly equal the number of sample messages sent.**

```promql
rate(ingestor_store_batch_size_sum{job="synapse"}[5m])
/
rate(ingestor_store_batch_size_count{job="synapse"}[5m])
```

Average events per batch. **Expect between 1 and 50** (your batcher caps at `BatchSize=50`). Near 50 = healthy batching; near 1 = flushing on timeout (low traffic).

```promql
histogram_quantile(0.95, sum(rate(ingestor_store_batch_duration_seconds_bucket{job="synapse"}[5m])) by (le))
```

p95 batch write latency. **Watch this** — the storage path is the known throughput bottleneck (single SQLite connection).

```promql
rate(ingestor_store_data_errors_total{job="synapse"}[5m])
```

Store errors by domain/entity/source. **Expect 0.**

## 5. API — HTTP Layer

Records only when you hit `/v1/events`.

```promql
sum(rate(api_request_duration_seconds_count{job="synapse"}[5m])) by (operation, status)
```

Request rate by operation (`get_event`, `list_events`) and status. **Empty until you call the API.**

```promql
histogram_quantile(0.95, sum(rate(api_request_duration_seconds_bucket{job="synapse"}[5m])) by (le, operation))
```

p95 API latency per operation.

```promql
rate(api_request_errors_total{job="synapse"}[5m])
```

API error rate. **Expect 0** under normal queries.

## 6. Auth — JWT Validation

Records on each authenticated request.

```promql
sum(rate(auth_validate_duration_seconds_count{job="synapse"}[5m])) by (status)
```

Validation rate split by `success`/`error`. **Empty until authenticated requests arrive.**

```promql
rate(auth_validate_errors_total{job="synapse"}[5m])
```

Failed token validations. **Spikes here = bad/expired tokens or attacks.**

## 7. Health Probes

Records each time `/readyz` runs its probes.

```promql
sum(rate(health_check_total_total{job="synapse"}[5m])) by (probe, status)
```

Probe invocation rate per probe (`db`, kafka consumers) and status. Note the `_total_total` quirk here. **Expect `status="ok"`** for healthy probes.

```promql
histogram_quantile(0.95, sum(rate(health_check_duration_seconds_bucket{job="synapse"}[5m])) by (le, probe))
```

p95 probe latency. **Watch the DB/Kafka probes** for slowness.

## Quick Smoke Test

After `make kafka-send-sample`, these should all move:

```promql
up{job="synapse"}                                                  # 1
rate(ingestor_transform_duration_seconds_count{job="synapse"}[5m]) # > 0
ingestor_store_batch_size_sum{job="synapse"}                       # ~= messages sent
rate(ingestor_transform_errors_total{job="synapse"}[5m])           # 0
```

## Metric Inventory

| Component | Histograms (`_seconds`) | Counters |
|---|---|---|
| Poller | `ingestor_poller_duration`, `ingestor_poller_ack_duration` | `ingestor_poller_total_total`, `ingestor_poller_errors_total`, `ingestor_poller_ack_total_total`, `ingestor_poller_ack_errors_total` |
| Processor | `ingestor_process_data_duration`, `ingestor_ack_data_duration` | `ingestor_process_data_total_total`, `ingestor_process_data_errors_total`, `ingestor_ack_data_total_total`, `ingestor_ack_data_errors_total` |
| Transformer | `ingestor_transform_duration` | `ingestor_transform_total_total`, `ingestor_transform_errors_total` |
| Storer | `ingestor_store_data_duration`, `ingestor_store_batch_duration`, `ingestor_store_batch_size` (no `_seconds`) | `ingestor_store_data_total_total`, `ingestor_store_data_errors_total` |
| API | `api_request_duration` | `api_request_total_total`, `api_request_errors_total` |
| Auth | `auth_validate_duration` | `auth_validate_total_total`, `auth_validate_errors_total` |
| Health | `health_check_duration_seconds` | `health_check_total_total` |

---

A few things worth flagging:

- The `_total_total` names in the inventory are the literal exporter output for the `.total` counters. They're awkward but correct. I led the section queries with histogram `_count` series so you mostly avoid them — confirm against your `/metrics` dump before committing the file.
- Several sections (API, Auth) stay empty until you actually call `/v1/events` with a JWT — that's expected, not a failure.
- If you want, in Agent mode I can write this to `docs/METRICS_QUERIES.md` and run the `curl ... /metrics` command to replace any `_total_total` guesses with the exact names your build emits.