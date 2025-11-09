# Makefile for Bank Service
# Provides convenient commands for development tasks

.PHONY: help migrate-up migrate-down migrate-version migrate-create db-reset build run test clean

# Default database URL
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable

help: ## Show this help message
	@echo "Bank Service - Available Commands"
	@echo "=================================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Database Migration Commands
migrate-up: ## Apply all pending migrations
	@echo "Applying migrations..."
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down: ## Rollback last migration
	@echo "Rolling back last migration..."
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-version: ## Show current migration version
	@echo "Current migration version:"
	migrate -path ./migrations -database "$(DATABASE_URL)" version

migrate-create: ## Create new migration (use NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required"; \
		echo "Usage: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	@./run-migrations.sh create

db-reset: ## Reset database (rollback all and reapply)
	@echo "Resetting database..."
	-migrate -path ./migrations -database "$(DATABASE_URL)" down -all
	migrate -path ./migrations -database "$(DATABASE_URL)" up
	@echo "Database reset complete!"

# Build Commands
build: ## Build the service
	@echo "Building bank-service..."
	go build -o bin/bank-service ./cmd/server

run: ## Run the service
	@echo "Starting bank-service..."
	go run ./cmd/server/main.go

# Testing Commands
test: ## Run tests
	@echo "Running tests..."
	go test ./... -v

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code Quality
fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

# Dependency Management
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy

# Clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Development
dev: ## Run in development mode with auto-reload (requires air)
	@echo "Starting development server..."
	air

# Proto Generation
proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@if [ -f "../common/scripts/generate_protos.sh" ]; then \
		cd ../common && ./scripts/generate_protos.sh; \
	else \
		echo "Warning: Proto generation script not found"; \
	fi

# Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t bank-service:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 50051:50051 bank-service:latest

# Database Setup (for local development)
db-start: ## Start PostgreSQL in Docker
	@echo "Starting PostgreSQL..."
	docker run -d \
		--name bank-postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=bank_db \
		-p 5432:5432 \
		postgres:15

db-stop: ## Stop PostgreSQL Docker container
	@echo "Stopping PostgreSQL..."
	docker stop bank-postgres
	docker rm bank-postgres

db-logs: ## Show PostgreSQL logs
	docker logs -f bank-postgres

# All-in-one setup
setup: deps migrate-up ## Setup project (install deps and run migrations)
	@echo "Setup complete!"
