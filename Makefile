.PHONY: help run build lint test clean docker-up docker-down docker-logs kafka-create-topic kafka-test-message sqlc-generate jaeger-up jaeger-down jaeger-logs

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := synapsePlatform
MAIN_PATH := cmd/main.go
DOCKER_COMPOSE := docker-compose

## help: Display this help message
help:
	@echo "Available targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""

## run: Run the application
run:
	go run $(MAIN_PATH)

## build: Build the application binary
build:
	go build -o bin/$(APP_NAME) $(MAIN_PATH)

dep-lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0; \
	fi

lint: dep-lint ## Lint with golangci-lint
	@golangci-lint run ./... --fix

## fmt: Format code
fmt:
	go fmt ./...

generate:
	rm -rf ./internal/utilstest/mocksgen
	go install go.uber.org/mock/mockgen@latest
	go generate -x ./internal/...

## test: Run tests
test:
	go test -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## mod-tidy: Clean up go.mod and go.sum
mod-tidy:
	@echo "🧹 Tidying Go modules..."
	go mod tidy
	@echo "✅ Modules tidied"

## mod-download: Download Go module dependencies
mod-download:
	@echo "📥 Downloading dependencies..."
	go mod download
	@echo "✅ Dependencies downloaded"

## local-resources: Start local Kafka infrastructure
local-resources:
	@echo "🚀 Starting local Kafka resources..."
	$(DOCKER_COMPOSE) up -d --remove-orphans
	@echo "✅ Kafka and Zookeeper started"


## docker-up: Start Docker Compose services (Kafka + Zookeeper)
docker-up:
	$(DOCKER_COMPOSE) up -d
	@echo "⏳ Waiting for Kafka to be ready..."
	@sleep 5
	@echo "✅ Docker services started"

## docker-down: Stop Docker Compose services
docker-down:
	@echo "🛑 Stopping Docker services..."
	$(DOCKER_COMPOSE) down
	@echo "✅ Docker services stopped"

## docker-restart: Restart Docker Compose services
docker-restart: docker-down docker-up

## docker-logs: Show Docker Compose logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## docker-logs-kafka: Show Kafka logs
docker-logs-kafka:
	$(DOCKER_COMPOSE) logs -f kafka

## kafka-topics: List all Kafka topics
kafka-topics:
	@echo "📋 Kafka topics:"
	@docker exec synapseplatform-kafka-1 kafka-topics --list --bootstrap-server localhost:9092

## kafka-create-topic: Create ingestion.raw topic (if not exists)
kafka-create-topic:
	@echo "📝 Creating Kafka topic: ingestion.raw"
	@docker exec synapseplatform-kafka-1 kafka-topics --create \
		--topic ingestion.raw \
		--partitions 1 \
		--replication-factor 1 \
		--if-not-exists \
		--bootstrap-server localhost:9092
	@echo "✅ Topic ready"

## kafka-test-message: Send a test message to Kafka
kafka-test-message:
	@echo "📤 Sending test message to Kafka..."
	@echo '{"device_id":"test-device-001","type":"temperature_sensor","timestamp":"'$$(date -u +"%Y-%m-%dT%H:%M:%SZ")'","metrics":{"temperature_c":22.5,"humidity":45.2}}' | \
		docker exec -i synapseplatform-kafka-1 kafka-console-producer \
		--broker-list localhost:9092 --topic ingestion.raw
	@echo "✅ Test message sent"

## kafka-test-file: Send a specific JSON file as a single message
kafka-test-file:
	@if [ -z "$(FILE)" ]; then \
		echo "❌ Usage: make kafka-test-file FILE=test/FinancialStreamEx.json"; \
		exit 1; \
	fi
	@echo "📤 Sending $(FILE) to Kafka..."
	@cat $(FILE) | jq -c . | docker exec -i synapseplatform-kafka-1 \
		kafka-console-producer --broker-list localhost:9092 --topic ingestion.raw
	@echo "✅ Message sent"

## kafka-send-sample: Send sample device messages from test directory
kafka-send-sample:
	@echo "📤 Sending sample messages..."
	@for file in test/*.json; do \
		echo "Sending $$file..."; \
		cat $$file | jq -c . | docker exec -i synapseplatform-kafka-1 kafka-console-producer \
			--broker-list localhost:9092 --topic ingestion.raw; \
		sleep 1; \
	done
	@echo "✅ All sample messages sent"

## kafka-console: Open Kafka console consumer (for debugging)
kafka-console:
	@echo "🎧 Starting Kafka console consumer (Ctrl+C to exit)..."
	@docker exec -it synapseplatform-kafka-1 kafka-console-consumer \
		--bootstrap-server localhost:9092 \
		--topic ingestion.raw \
		--from-beginning

## sqlc-generate: Generate sqlc code from SQL files
sqlc-generate:
	@echo "⚙️  Generating sqlc code..."
	@if command -v sqlc >/dev/null 2>&1; then \
		sqlc generate; \
		echo "✅ sqlc code generated"; \
	else \
		echo "❌ sqlc not installed. Install with: brew install sqlc"; \
		exit 1; \
	fi

## db-reset: Delete and recreate the database
db-reset:
	@echo "🗑️  Deleting database..."
	@rm -f data.db
	@echo "✅ Database will be recreated on next run"

## clean: Clean build artifacts and generated files
clean:
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@rm -f data.db
	@echo "✅ Cleaned"

## dev: Start development environment (Docker + Jaeger + App)
dev: docker-up jaeger-up
	@echo "Waiting for services to be ready..."
	@sleep 10
	@$(MAKE) run

## install-tools: Install development tools
install-tools:
	@echo "📦 Installing development tools..."
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing sqlc..."
	@command -v sqlc >/dev/null 2>&1 || echo "Please install sqlc see https://docs.sqlc.dev/en/latest/overview/install.html"
	@echo "✅ Tools installation complete (check messages above)"

## all: Run fmt, lint, test, and build
all: fmt lint test build
	@echo "✅ All tasks complete"

## jaeger-up: Start Jaeger for local trace collection (OTLP on :4318, UI on :16686)
jaeger-up:
	@echo "Starting Jaeger..."
	@docker run -d --name jaeger \
		-p 4318:4318 \
		-p 16686:16686 \
		jaegertracing/all-in-one:latest
	@echo "Jaeger UI: http://localhost:16686"

## jaeger-down: Stop and remove Jaeger container
jaeger-down:
	@echo "Stopping Jaeger..."
	@docker rm -f jaeger 2>/dev/null || true
	@echo "Jaeger stopped"

## jaeger-logs: Show Jaeger container logs
jaeger-logs:
	@docker logs -f jaeger

