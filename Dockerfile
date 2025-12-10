# Production Dockerfile - Multi-stage build
# Usage: docker build --build-arg SERVICE=backend-api-gateway -t booking-rush-gateway .

# ================================
# Stage 1: Build
# ================================
FROM golang:1.25-alpine AS builder

ARG SERVICE=backend-api-gateway

WORKDIR /app

# Install git (required for go modules)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go workspace files
COPY go.work go.work.sum ./

# Copy all modules
COPY pkg/ pkg/
COPY backend-api-gateway/ backend-api-gateway/
COPY backend-auth/ backend-auth/
COPY backend-ticket/ backend-ticket/
COPY backend-booking/ backend-booking/
COPY backend-payment/ backend-payment/
COPY scripts/ scripts/
COPY tests/ tests/

# Download dependencies
RUN go work sync

# Build the service
WORKDIR /app/${SERVICE}
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/main .

# ================================
# Stage 2: Runtime
# ================================
FROM alpine:3.19

# Add ca-certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Copy Lua scripts (for booking service)
COPY --from=builder /app/scripts/lua ./scripts/lua

# Use non-root user
USER appuser

# Expose port (override in docker-compose)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${SERVER_PORT:-8080}/health || exit 1

# Run the binary
CMD ["./main"]
