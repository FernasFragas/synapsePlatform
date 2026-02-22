# Synapse Platform
![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Kafka](https://img.shields.io/badge/Apache_Kafka-231F20?style=for-the-badge&logo=apache-kafka&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-07405E?style=for-the-badge&logo=sqlite&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2CA5E0?style=for-the-badge&logo=docker&logoColor=white)
![Make](https://img.shields.io/badge/Make-427819?style=for-the-badge&logo=gnu&logoColor=white)

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

License: PolyForm Noncommercial 1.0.0 (PolyForm-Noncommercial-1.0.0). See LICENSE.
