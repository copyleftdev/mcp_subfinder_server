# MCP Subfinder Server Makefile

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod
GOLINT = golangci-lint
BINARY_NAME = mcp-subfinder-server
BINARY_UNIX = $(BINARY_NAME)_unix
MAIN_PATH = .

# Build flags
LDFLAGS = -ldflags "-s -w"

# Default port for the server
PORT ?= 8080

# Default provider config location
PROVIDER_CONFIG ?= provider-config.yaml

.PHONY: all build clean test coverage lint deps fmt help run tidy check-config integration-test live-test docker docker-run

# Default target
all: test build

# Build the project
build:
	@echo "Building..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

# Build for Unix/Linux
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) $(MAIN_PATH)

# Clean the project
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out
	rm -f coverage.html

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Run integration tests
integration-test:
	@echo "Running integration tests..."
	ENABLE_INTEGRATION_TESTS=1 $(GOTEST) -v ./...

# Run live subfinder tests
live-test:
	@echo "Running live subfinder tests..."
	ENABLE_LIVE_TESTS=1 $(GOTEST) -v ./internal/subfinder/...

# Run linter
lint:
	@echo "Running linter..."
	$(GOLINT) run

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOGET) -v ./...

# Format the code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Update go.mod and go.sum
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Check if provider config exists
check-config:
	@if [ ! -f $(PROVIDER_CONFIG) ]; then \
		echo "Provider config not found at $(PROVIDER_CONFIG)"; \
		echo "Using default sources only. For optimal results, add API keys."; \
	else \
		echo "Provider config found at $(PROVIDER_CONFIG)"; \
	fi

# Run the server
run: check-config
	@echo "Running server on port $(PORT)..."
	$(GOCMD) run $(MAIN_PATH) -port $(PORT) -provider-config $(PROVIDER_CONFIG)

# Build a Docker image
docker:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME) .

# Run the server in Docker
docker-run:
	@echo "Running server in Docker on port $(PORT)..."
	docker run -p $(PORT):$(PORT) -v $(PWD)/$(PROVIDER_CONFIG):/app/$(PROVIDER_CONFIG) $(BINARY_NAME) -port $(PORT) -provider-config /app/$(PROVIDER_CONFIG)

# Show help
help:
	@echo "Make targets:"
	@echo "  all              - Run tests and build"
	@echo "  build            - Build the binary"
	@echo "  build-linux      - Build for Linux"
	@echo "  clean            - Remove binaries and coverage files"
	@echo "  test             - Run tests"
	@echo "  integration-test - Run integration tests"
	@echo "  live-test        - Run live subfinder tests"
	@echo "  coverage         - Run tests with coverage report" 
	@echo "  lint             - Run linter"
	@echo "  deps             - Download dependencies"
	@echo "  fmt              - Format the code"
	@echo "  tidy             - Update go.mod and go.sum"
	@echo "  check-config     - Check if provider config exists"
	@echo "  run              - Run the server (PORT=8080 by default)"
	@echo "  docker           - Build Docker image"
	@echo "  docker-run       - Run in Docker"
	@echo "  help             - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  PORT             - Server port (default: 8080)"
	@echo "  PROVIDER_CONFIG  - Path to provider config file (default: provider-config.yaml)"
