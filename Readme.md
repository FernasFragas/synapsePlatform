# Synapse Platform
![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Kafka](https://img.shields.io/badge/Apache_Kafka-231F20?style=for-the-badge&logo=apache-kafka&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-07405E?style=for-the-badge&logo=sqlite&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2CA5E0?style=for-the-badge&logo=docker&logoColor=white)
![Make](https://img.shields.io/badge/Make-427819?style=for-the-badge&logo=gnu&logoColor=white)

A high-performance event ingestion and processing platform built in Go. Synapse consumes device and service data from Kafka, normalizes it into strongly-typed domain events, persists them to SQLite, and exposes them via a JWT-authenticated HTTP API.

## What It Does

Synapse Platform ingests heterogeneous event streams from multiple sources, transforms them into normalized domain events, and stores them for querying and analysis. Perfect for:

- **Energy Monitoring** - Track power consumption, voltage, and current from IoT sensors
- **Financial Tracking** - Process transaction streams with merchant and currency data
- **Service Monitoring** - Collect latency metrics and status codes from microservices

## Features

- **Real-time Event Processing** - Kafka-based message consumption with type-safe event handling
- **Domain-Driven Design** - Normalized events across Energy, Finance, and Monitoring domains
- **Persistent Storage** - SQLite with sqlc for type-safe, generated database operations
- **REST API** - HTTP API for querying ingested events with JWT Bearer authentication
- **Structured Logging** - Decorator-based logging on every component using `log/slog`
- **Pluggable Components** - Interface-based design; every dependency is injected, never global

## Prerequisites

| Tool | Version | Install |
|---|---|---|
| Go | 1.25+ | [go.dev/doc/install](https://go.dev/doc/install) |
| Docker + Compose | any recent | [docs.docker.com](https://docs.docker.com/get-docker/) |
| sqlc | latest | `brew install sqlc` |
| golangci-lint | v2.4.0+ | `make install-tools` |
| mockgen | latest | `make install-tools` |
| jq | any | `brew install jq` |

## Quick Start
### 1. Install development tools
make install-tools
### 2. Start Kafka and Zookeeper
make local-resources
### 3. Run the application
make run
### 4. Send sample messages to verify ingestion
make kafka-send-sample

---

## API Usage

### Authentication

All API endpoints require a valid JWT Bearer token. The token must be signed with **HS256** and include the following claims:

| Claim       | Description                          | Example                         |
|-------------|--------------------------------------|---------------------------------|
| `iss`       | Token issuer                         | `https://auth.example.com`      |
| `aud`       | Target audience                      | `synapse-platform-api`          |
| `sub`       | Subject (user identifier)            | `user-123`                      |
| `client_id` | Client application identifier        | `my-client`                     |
| `scope`     | Space-separated list of permissions  | `read:events`                   |
| `exp`       | Expiration time (Unix timestamp)     | `1899999999`                    |

### Generating a Token

Using [jwt.io](https://jwt.io):

1. Set the algorithm to **HS256**
2. Use the following payload:

```json
{
  "iss": "https://auth.example.com",
  "aud": "synapse-platform-api",
  "sub": "user-123",
  "client_id": "my-client",
  "scope": "read:events",
  "exp": 1899999999
}
```

3. Set the signing secret to your configured JWT secret
4. Copy the encoded token

### Endpoints

#### List Events

```
GET /events
```

Returns all stored events.

```bash
curl http://localhost:8080/events \
  -H "Authorization: Bearer <token>"
```

#### Get Event by ID

```
GET /events/{id}
```

Returns a single event by its ID.

```bash
curl http://localhost:8080/events/some-event-id \
  -H "Authorization: Bearer <token>"
```

### Response Format

```json
[
  {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "domain": "iot",
    "event_type": "temperature_reading",
    "entity_id": "sensor-42",
    "occurred_at": "2025-01-15T10:30:00Z",
    "source": "env-sensor",
    "schema_version": "1.0",
    "data": {}
  }
]
```

### Error Responses

| Status | Cause                                      |
|--------|--------------------------------------------|
| `401`  | Missing or invalid token                   |
| `403`  | Token valid but missing `read:events` scope|
| `404`  | Event not found (GET by ID only)           |
| `500`  | Internal server error                      |

---
