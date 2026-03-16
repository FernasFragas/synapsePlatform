#!/bin/bash
# test-ci.sh - Simulate CI locally

set -e  # Exit on error

echo "Running CI pipeline locally..."

# 1. Lint
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 1: Linting"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
golangci-lint run --timeout=5m

# 2. Test with coverage
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 2: Testing with Coverage"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# 3. Coverage report
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 3: Coverage Report"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
go tool cover -func=coverage.out | grep total

# Extract percentage
total=$(go tool cover -func=coverage.out | grep total | grep -Eo '[0-9]+\.[0-9]+')
echo "Total coverage: ${total}%"

# 4. Build
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 4: Building"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
go build -v -o bin/synapsePlatform ./cmd/main.go

echo ""
echo "All CI steps passed!"