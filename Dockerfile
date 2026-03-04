# syntax=docker/dockerfile:1

# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy manifests first so this layer is cached until dependencies change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# modernc.org/sqlite is pure Go — CGO is not needed.
RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags="-w -s" \
      -o bin/synapsePlatform \
      ./cmd/main.go

# ── Runtime stage ─────────────────────────────────────────────────────────────
# distroless/static has no shell, no package manager, minimal attack surface.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/bin/synapsePlatform .

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/synapsePlatform"]