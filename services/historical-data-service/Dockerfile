# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod .

# Explicitly set Go version
RUN go mod edit -go=1.21

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o historical-service ./cmd/server

# Final stage
FROM alpine:3.18

WORKDIR /app

# Install necessary packages
RUN apk --no-cache add ca-certificates tzdata curl

# Copy the binary from builder
COPY --from=builder /app/historical-service .

# Create config directory and copy configs
RUN mkdir -p /app/config
COPY config/config.yaml /app/config/

# Add healthcheck
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8081/health || exit 1

# Expose the service port
EXPOSE 8081

# Set environment variables with defaults that can be overridden
ENV DB_HOST=historical-db \
    DB_PORT=5432 \
    DB_USER=historical_service_user \
    DB_PASSWORD=historical_service_password \
    DB_NAME=historical_service \
    BACKTEST_SERVICE_URL=http://backtest-service:5000

# Run the service
CMD ["./historical-service"]