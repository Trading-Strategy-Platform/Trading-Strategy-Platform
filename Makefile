.PHONY: all build clean test run help start stop lint setup-lint infra infra-up apply-k8s

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
	@docker compose up -d

stop:
	@echo "Stopping services..."
	@docker compose down

infra-up:
	@echo "Starting infrastructure only..."
	@docker compose up -d user-db strategy-db historical-db kafka zookeeper redis

setup-lint:
	@echo "Setting up golangci-lint v1.55.2..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.55.2

lint: setup-lint
	@echo "Linting Go code..."
	@cd services/user-service && $(shell go env GOPATH)/bin/golangci-lint run
	@cd services/strategy-service && $(shell go env GOPATH)/bin/golangci-lint run
	@cd services/historical-data-service && $(shell go env GOPATH)/bin/golangci-lint run
	@cd services/api-gateway && $(shell go env GOPATH)/bin/golangci-lint run

fix-dependencies:
	@echo "Fixing dependencies..."
	@chmod +x ./fix-dependencies.sh
	@./fix-dependencies.sh

setup-infra:
	@echo "Setting up infrastructure directories..."
	@mkdir -p infrastructure/kafka/config
	@mkdir -p infrastructure/postgres
	@mkdir -p infrastructure/redis
	@mkdir -p infrastructure/timescaledb
	@cp -f infrastructure-configs/server.properties infrastructure/kafka/config/
	@cp -f infrastructure-topic-script/topics.sh infrastructure/kafka/config/
	@chmod +x infrastructure/kafka/config/topics.sh
	@cp -f postgres-config/postgres.conf infrastructure/postgres/
	@cp -f redis-config/redis.conf infrastructure/redis/
	@cp -f timescaledb-config/timescaledb.conf infrastructure/timescaledb/

apply-k8s:
	@echo "Applying Kubernetes configurations..."
	@kubectl apply -f k8s/kafka/
	@kubectl apply -f k8s/postgres/
	@kubectl apply -f k8s/redis/
	@kubectl apply -f k8s/timescaledb/
	@kubectl apply -f k8s/api-gateway/
	@kubectl apply -f k8s/historical-data-service/
	@kubectl apply -f k8s/strategy-service/
	@kubectl apply -f k8s/user-service/

help:
	@echo "Available commands:"
	@echo "  make build            - Build all services"
	@echo "  make test             - Run tests"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make start            - Start all services with Docker Compose"
	@echo "  make stop             - Stop all services"
	@echo "  make infra-up         - Start only infrastructure services"
	@echo "  make setup-lint       - Install golangci-lint v1.55.2"
	@echo "  make lint             - Run linters"
	@echo "  make fix-dependencies - Fix dependencies using the script"
	@echo "  make setup-infra      - Set up infrastructure configuration files"
	@echo "  make apply-k8s        - Apply Kubernetes configurations"