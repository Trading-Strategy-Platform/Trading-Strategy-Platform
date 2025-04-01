# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy source code first to ensure proper initialization
COPY . .

# Explicitly set Go version
RUN go mod edit -go=1.21

# Initialize the module and download dependencies
RUN go mod tidy
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o media-service ./cmd/server || true

# If the build failed, add missing dependencies and try again
RUN go get ./... || true
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o media-service ./cmd/server

# Final stage
FROM alpine:3.18

WORKDIR /app

# Install necessary packages
RUN apk --no-cache add ca-certificates tzdata

# Create data directory for local storage
RUN mkdir -p /data/images && chmod 777 /data/images

# Copy the binary from builder
COPY --from=builder /app/media-service .

# Create config directory and copy configs
RUN mkdir -p /app/config
COPY --from=builder /app/config/config.yaml /app/config/

# Expose the service port
EXPOSE 8085

# Set volume for local storage mode
VOLUME ["/data/images"]

# Run the service
CMD ["./media-service"]