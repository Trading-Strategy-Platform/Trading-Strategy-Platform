.PHONY: all build clean test run help start stop lint setup-lint

all: build

build:
	@echo "Building all services..."
	@cd services/user-service && go build -o bin/user-service cmd/server/main.go
	@cd services/strategy-service && go build -o bin/strategy-service cmd/server/main.go
	@cd services/historical-data-service && go build -o bin/historical-service cmd/server/main.go
	@cd services/api-gateway && go build -o bin/api-gateway cmd/server/main.go

test:
	@echo "Running tests..."
	@cd services/user-service && go test ./...
	@cd services/strategy-service && go test ./...
	@cd services/historical-data-service && go test ./...
	@cd services/api-gateway && go test ./...

clean:
	@echo "Cleaning..."
	@rm -rf services/user-service/bin
	@rm -rf services/strategy-service/bin
	@rm -rf services/historical-data-service/bin
	@rm -rf services/api-gateway/bin

start:
	@echo "Starting services..."
	@docker-compose up -d

stop:
	@echo "Stopping services..."
	@docker-compose down

infra-up:
	@echo "Starting infrastructure only..."
	@docker-compose up -d user-db strategy-db historical-db kafka zookeeper redis

setup-lint:
	@echo "Setting up golangci-lint v1.55.2..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.55.2

lint: setup-lint
	@echo "Linting Go code..."
	@cd services/user-service && $(shell go env GOPATH)/bin/golangci-lint run
	@cd services/strategy-service && $(shell go env GOPATH)/bin/golangci-lint run
	@cd services/historical-data-service && $(shell go env GOPATH)/bin/golangci-lint run
	@cd services/api-gateway && $(shell go env GOPATH)/bin/golangci-lint run

help:
	@echo "Available commands:"
	@echo "  make build       - Build all services"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make start       - Start all services with Docker Compose"
	@echo "  make stop        - Stop all services"
	@echo "  make infra-up    - Start only infrastructure services"
	@echo "  make setup-lint  - Install golangci-lint v1.55.2"
	@echo "  make lint        - Run linters"