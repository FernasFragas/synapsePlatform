# Synapse Platform

A high-performance event ingestion and processing platform built in Go. Synapse consumes heterogeneous device and service data from Kafka, normalizes it into strongly-typed domain events, and persists them for analysis and querying.

## What It Does

Synapse Platform ingests heterogeneous event streams from multiple sources, transforms them into normalized domain events, and stores them for querying and analysis. Perfect for:

- **Energy Monitoring** - Track power consumption, voltage, and current from IoT sensors
- **Financial Tracking** - Process transaction streams with merchant and currency data
- **Service Monitoring** - Collect latency metrics and status codes from microservices


## Features

- **Real-time Event Processing** - Kafka-based message consumption with type-safe event handling
- **Domain-Driven Design** - Normalized events across Energy, Finance, and Monitoring domains
- **Persistent Storage** - SQLite with sqlc for type-safe database operations
- **Generic Architecture** - Leverages Go 1.18+ generics for type-safe event structures
- **Multi-Domain Support** - Handles energy readings, financial transactions, and latency samples
- **Pluggable Components** - Interface-based design for message pollers and storage backends

**Stack**: Go • Kafka • SQLite • sqlc • Docker
