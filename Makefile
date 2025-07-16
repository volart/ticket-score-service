# Ticket Score Service Makefile

# Variables
PROTO_DIR = proto
GENERATED_DIR = proto/generated
GO_BIN = $(shell go env GOPATH)/bin

# Default target
.DEFAULT_GOAL := help

# Help target
.PHONY: help
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Install protobuf tools
.PHONY: install-proto-tools
install-proto-tools: ## Install protobuf code generation tools
	@echo "Installing protobuf tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Protobuf tools installed successfully!"

# Generate protobuf files
.PHONY: proto
proto: ## Generate Go code from proto files
	@echo "Creating generated directories..."
	mkdir -p $(GENERATED_DIR)/rating_analytics
	mkdir -p $(GENERATED_DIR)/ticket_scores
	mkdir -p $(GENERATED_DIR)/overall_quality
	mkdir -p $(GENERATED_DIR)/period_comparison
	@echo "Generating protobuf files..."
	export PATH=$(PATH):$(GO_BIN) && \
	protoc --go_out=. --go-grpc_out=. $(PROTO_DIR)/rating_analytics.proto && \
	protoc --go_out=. --go-grpc_out=. $(PROTO_DIR)/ticket_scores.proto && \
	protoc --go_out=. --go-grpc_out=. $(PROTO_DIR)/overall_quality.proto && \
	protoc --go_out=. --go-grpc_out=. $(PROTO_DIR)/period_comparison.proto
	@echo "Protobuf files generated successfully!"

# Clean generated files
.PHONY: clean-proto
clean-proto: ## Remove generated protobuf files
	@echo "Cleaning generated protobuf files..."
	rm -rf $(GENERATED_DIR)
	@echo "Generated files cleaned!"

# Build the application
.PHONY: build
build: ## Build the application
	@echo "Building application..."
	go build -o bin/server cmd/server/main.go
	@echo "Application built successfully!"

# Run the application
.PHONY: start
start: ## Start the server
	@echo "Starting server..."
	go run cmd/server/main.go

# Run the built binary
.PHONY: run
run: build ## Run the built server binary
	@echo "Running built server..."
	./bin/server

# Run tests
.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	go test -v ./...

# Docker commands
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t ticket-score-service .

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 50051:50051 ticket-score-service

.PHONY: docker-compose-up
docker-compose-up: ## Start with docker-compose
	@echo "Starting with docker-compose..."
	docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down: ## Stop docker-compose
	@echo "Stopping docker-compose..."
	docker-compose down

# Linting and formatting
.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting Go code..."
	go fmt ./...

