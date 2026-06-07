.PHONY: help run build lint test clean docker-up docker-down docker-logs kafka-create-topic kafka-test-message sqlc-generate jaeger-up jaeger-down jaeger-logs

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := synapsePlatform
MAIN_PATH := ./cmd/
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

test-int:
	go test -v -tags integration ./internal/ingestor/...

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

## perf-test: Run performance test suite
perf-test:
	@echo "🚀 Starting performance test suite..."
	@chmod +x test/perform_test.sh
	@./test/perform_test.sh
## perf-test-quick: Run quick performance test (Test 1 only)
perf-test-quick:
	@echo "🚀 Running quick performance test..."
	@chmod +x test/perform_test.sh
	@./test/perform_test.sh quick
## perf-report: Show the latest performance report
perf-report:
	@echo "📊 Latest performance report:"
	@LATEST=$$(ls -t ./performance-reports/synapse-performance-report-*.md 2>/dev/null | head -1);
## perf-clean: Remove all performance reports
## perf-clean: Remove all performance reports
perf-clean:
	@echo "🗑️  Removing performance reports..."
	@rm -f ./performance-reports/synapse-performance-report-*.md

## perf-history: Show performance test history
perf-history:
	@echo "📊 Performance Test History"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@cat performance-reports/INDEX.md 2>/dev/null || echo "No performance tests run yet. Run 'make perf-test' first."

## perf-compare: Show performance comparison chart
perf-compare:
	@cat performance-reports/COMPARISON.md 2>/dev/null || echo "No comparison data yet. Run at least 2 performance tests."

## perf-trend: Show throughput trend (last 10 runs)
perf-trend:
	@echo "📈 Throughput Trend (Test 2: 100 msg/sec target)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@tail -n +4 performance-reports/INDEX.md 2>/dev/null | head -n 10 | \
		awk -F'|' '{gsub(/^[ \t]+|[ \t]+$$/, "", $$3); gsub(/^[ \t]+|[ \t]+$$/, "", $$4); \
		printf "%-20s %s (Success: %s)\n", $$2, $$3, $$4}' || \
		echo "No performance data available yet."

## run-with-logs: Run the application and save logs to file
run-with-logs:
	@mkdir -p performance-reports
	@echo "🚀 Starting application with log capture..."
	@echo "📝 Logs will be saved to: performance-reports/app-logs-$$(date +%Y%m%d-%H%M%S).log"
	@go run $(MAIN_PATH) > performance-reports/app-logs-$$(date +%Y%m%d-%H%M%S).log 2>&1

## grafana-up: Start full stack with Grafana Cloud metrics
grafana-up:
	$(DOCKER_COMPOSE) --profile monitoring up -d --build

## grafana-down: Stop full stack
grafana-down:
	$(DOCKER_COMPOSE) --profile monitoring down

## infra-only: Start infra + Alloy for GoLand-run app (no synapse container)
infra-only:
	@echo "🚀 Starting infra (Kafka, Zookeeper) for GoLand..."
	$(DOCKER_COMPOSE) up -d kafka zookeeper init-kafka
	@echo "📡 Starting Alloy without its synapse dependency..."
	$(DOCKER_COMPOSE) up -d --no-deps alloy
	@echo "✅ Infra + Alloy up. Run the app from GoLand on :8080"

## infra-down: Stop infra + Alloy
infra-down:
	@echo "🛑 Stopping infra + Alloy..."
	$(DOCKER_COMPOSE) stop kafka zookeeper init-kafka alloy
	@echo "✅ Stopped"

## e2e-test: Run end-to-end tests against Docker Compose stack
## e2e-test: Full end-to-end test with model readiness check
e2e-test:
	@echo "🚀 Starting e2e test environment..."
	docker compose down -v
	docker compose up -d --build
	@echo "⏳ Waiting for Ollama model (port 11435) — this takes ~90s on first pull..."
	@for i in $$(seq 1 120); do \
		if curl -s http://localhost:11435/api/tags | grep -q "llama3.2:1b"; then \
			echo "✅ Model llama3.2:1b ready"; \
			break; \
		fi; \
		if [ $$i -eq 120 ]; then \
			echo "❌ Timeout: model never appeared after 10 minutes"; \
			echo "init-ollama logs:"; \
			docker compose logs init-ollama --tail 20; \
			exit 1; \
		fi; \
		echo "   attempt $$i/120 — still waiting..."; \
		sleep 5; \
	done
	@echo "⏳ Giving synapse a few seconds to finish startup..."
	@sleep 5
	JWT_SECRET="${JWT_SECRET:-your-256-bit-secret-replace-me!!}" \
	E2E_API_URL=http://localhost:8080 \
	E2E_OLLAMA_URL=http://localhost:11435 \
	go test -v -timeout 300s -tags e2e ./test/e2e/...
	docker compose down