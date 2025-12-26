.PHONY: help build install test clean fmt vet lint dev-setup run-example

# Variables
BINARY_NAME=typconv
BUILD_DIR=./build
CMD_DIR=./cmd/typconv
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary created at $(BUILD_DIR)/$(BINARY_NAME)"

install: ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(CMD_DIR)
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report created at coverage.html"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	@echo "Format complete"

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet complete"

lint: ## Run golangci-lint (requires golangci-lint installed)
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install it from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run
	@echo "Lint complete"

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	go mod tidy
	@echo "Tidy complete"

dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	go mod download
	@which golangci-lint > /dev/null || echo "Consider installing golangci-lint: https://golangci-lint.run/usage/install/"
	@echo "Dev setup complete"

run-example: build ## Run example conversion (requires test file)
	@echo "Running example..."
	@if [ -f testdata/binary/sample.typ ]; then \
		$(BUILD_DIR)/$(BINARY_NAME) bin2txt testdata/binary/sample.typ; \
	else \
		echo "No sample file found at testdata/binary/sample.typ"; \
		echo "Add a TYP file there to test"; \
	fi

# Development helpers
watch-test: ## Watch and run tests on file changes (requires entr)
	@which entr > /dev/null || (echo "entr not found. Install it to use watch mode" && exit 1)
	@echo "Watching for changes..."
	@find . -name '*.go' | entr -c make test

all: clean fmt vet test build ## Run clean, fmt, vet, test, and build

release-build: ## Build release binaries for multiple platforms
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)/release
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "Release binaries created in $(BUILD_DIR)/release/"

.DEFAULT_GOAL := help
