# Variables
APP_NAME := branchlore
CLI_NAME := branchlore-cli
DAEMON_NAME := branchlore-daemon
CMD_DIR := ./cmd
CLI_DIR := $(CMD_DIR)/cli
DAEMON_DIR := $(CMD_DIR)/daemon
INTERNAL_DIR := ./internal
TEST_DIR := ./...
BRANCHLORE_REPO := branchlore-repo
BIN_DIR := ./bin
CONFIG_DIR := ~/.branchlore

# Build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)

# Default target
.PHONY: all
all: build

# Build all binaries
.PHONY: build
build: build-cli build-daemon

# Build CLI
.PHONY: build-cli
build-cli:
	@echo "Building CLI..."
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(CLI_NAME) $(CLI_DIR)

# Build daemon
.PHONY: build-daemon
build-daemon:
	@echo "Building daemon..."
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(DAEMON_NAME) $(DAEMON_DIR)

# Build for multiple platforms
.PHONY: build-all
build-all: clean-bin
	@echo "Building for multiple platforms..."
	@mkdir -p $(BIN_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(CLI_NAME)-linux-amd64 $(CLI_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(DAEMON_NAME)-linux-amd64 $(DAEMON_DIR)
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(CLI_NAME)-darwin-amd64 $(CLI_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(DAEMON_NAME)-darwin-amd64 $(DAEMON_DIR)
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(CLI_NAME)-darwin-arm64 $(CLI_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(DAEMON_NAME)-darwin-arm64 $(DAEMON_DIR)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(CLI_NAME)-windows-amd64.exe $(CLI_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(DAEMON_NAME)-windows-amd64.exe $(DAEMON_DIR)

# Install binaries to system PATH
.PHONY: install
install: build
	@echo "Installing binaries..."
	sudo cp $(BIN_DIR)/$(CLI_NAME) /usr/local/bin/$(APP_NAME)
	sudo cp $(BIN_DIR)/$(DAEMON_NAME) /usr/local/bin/$(DAEMON_NAME)
	@echo "Installed $(APP_NAME) and $(DAEMON_NAME) to /usr/local/bin/"

# Uninstall binaries from system PATH
.PHONY: uninstall
uninstall:
	@echo "Uninstalling binaries..."
	sudo rm -f /usr/local/bin/$(APP_NAME)
	sudo rm -f /usr/local/bin/$(DAEMON_NAME)
	@echo "Uninstalled binaries from /usr/local/bin/"

# Run CLI
.PHONY: run-cli
run-cli: build-cli
	@echo "Running CLI..."
	$(BIN_DIR)/$(CLI_NAME) $(ARGS)

# Run daemon
.PHONY: run-daemon
run-daemon: build-daemon
	@echo "Running daemon..."
	$(BIN_DIR)/$(DAEMON_NAME) $(ARGS)

# Run daemon in development mode
.PHONY: dev-daemon
dev-daemon: build-daemon
	@echo "Running daemon in development mode..."
	$(BIN_DIR)/$(DAEMON_NAME) -verbose -addr :8080

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test $(TEST_DIR) -v

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test $(TEST_DIR) -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test $(TEST_DIR) -bench=. -benchmem

# Run simulation tests
.PHONY: simulate
simulate: build-cli
	@echo "Running simulation tests..."
	$(BIN_DIR)/$(CLI_NAME) simulate basic
	$(BIN_DIR)/$(CLI_NAME) simulate stress
	$(BIN_DIR)/$(CLI_NAME) simulate concurrent

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt $(TEST_DIR)

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	go vet $(TEST_DIR)

# Check code quality
.PHONY: check
check: fmt vet lint test

# Generate Go modules
.PHONY: mod
mod:
	@echo "Updating Go modules..."
	go mod download
	go mod tidy
	go mod verify

# Clean up build artifacts
.PHONY: clean-bin
clean-bin:
	@echo "Cleaning up build artifacts..."
	rm -rf $(BIN_DIR)

# Clean up repository data
.PHONY: clean-repo
clean-repo:
	@echo "Cleaning up repository data..."
	rm -rf $(BRANCHLORE_REPO)
	@echo "Cleaned up $(BRANCHLORE_REPO) folder."

# Clean up configuration
.PHONY: clean-config
clean-config:
	@echo "Cleaning up configuration..."
	rm -rf $(CONFIG_DIR)
	@echo "Cleaned up $(CONFIG_DIR) folder."

# Full clean
.PHONY: clean
clean: clean-bin clean-repo
	@echo "Cleaning up generated files..."
	rm -f coverage.out coverage.html

# Initialize development environment
.PHONY: init
init:
	@echo "Initializing development environment..."
	go mod download
	@mkdir -p $(BIN_DIR)
	$(MAKE) build
	$(BIN_DIR)/$(CLI_NAME) init
	@echo "Development environment ready!"

# Create release archive
.PHONY: release
release: build-all
	@echo "Creating release archives..."
	@mkdir -p dist
	# Linux
	tar -czf dist/$(APP_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BIN_DIR) $(CLI_NAME)-linux-amd64 $(DAEMON_NAME)-linux-amd64
	# macOS AMD64
	tar -czf dist/$(APP_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(BIN_DIR) $(CLI_NAME)-darwin-amd64 $(DAEMON_NAME)-darwin-amd64
	# macOS ARM64
	tar -czf dist/$(APP_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(BIN_DIR) $(CLI_NAME)-darwin-arm64 $(DAEMON_NAME)-darwin-arm64
	# Windows
	zip -j dist/$(APP_NAME)-$(VERSION)-windows-amd64.zip $(BIN_DIR)/$(CLI_NAME)-windows-amd64.exe $(BIN_DIR)/$(DAEMON_NAME)-windows-amd64.exe
	@echo "Release archives created in dist/"

# Start a development server
.PHONY: dev
dev: init
	@echo "Starting development server..."
	$(BIN_DIR)/$(DAEMON_NAME) -verbose -addr :8080 &
	@echo "Development server started on :8080"
	@echo "Use 'make stop-dev' to stop the server"

# Stop development server
.PHONY: stop-dev
stop-dev:
	@echo "Stopping development server..."
	pkill -f "$(DAEMON_NAME)" || true

# Show project status
.PHONY: status
status: build-cli
	@echo "Project Status:"
	@echo "==============="
	$(BIN_DIR)/$(CLI_NAME) status || echo "No repository initialized"

# Run integration tests
.PHONY: integration
integration: build
	@echo "Running integration tests..."
	@echo "Starting daemon..."
	$(BIN_DIR)/$(DAEMON_NAME) -addr :8081 &
	sleep 2
	@echo "Running CLI integration tests..."
	$(BIN_DIR)/$(CLI_NAME) init test-repo
	$(BIN_DIR)/$(CLI_NAME) branch create test-branch
	$(BIN_DIR)/$(CLI_NAME) query test-branch "INSERT INTO demo (msg) VALUES ('integration test')"
	$(BIN_DIR)/$(CLI_NAME) query test-branch "SELECT * FROM demo"
	$(BIN_DIR)/$(CLI_NAME) status
	@echo "Stopping daemon..."
	pkill -f "$(DAEMON_NAME)" || true
	@echo "Integration tests completed"

# Generate documentation
.PHONY: docs
docs: build-cli
	@echo "Generating documentation..."
	@mkdir -p docs
	$(BIN_DIR)/$(CLI_NAME) --help > docs/cli-help.txt
	$(BIN_DIR)/$(CLI_NAME) branch --help > docs/branch-help.txt
	$(BIN_DIR)/$(CLI_NAME) query --help > docs/query-help.txt
	$(BIN_DIR)/$(CLI_NAME) simulate --help > docs/simulate-help.txt
	@echo "Documentation generated in docs/"

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker build -t $(APP_NAME):latest .

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 $(APP_NAME):latest

# Help menu
.PHONY: help
help:
	@echo "Makefile for $(APP_NAME)"
	@echo ""
	@echo "Build Targets:"
	@echo "  build          Build CLI and daemon binaries"
	@echo "  build-cli      Build CLI binary only"
	@echo "  build-daemon   Build daemon binary only"
	@echo "  build-all      Build for multiple platforms"
	@echo "  install        Install binaries to system PATH"
	@echo "  uninstall      Remove binaries from system PATH"
	@echo ""
	@echo "Development Targets:"
	@echo "  init           Initialize development environment"
	@echo "  dev            Start development server"
	@echo "  stop-dev       Stop development server"
	@echo "  run-cli        Run CLI with ARGS='your-args'"
	@echo "  run-daemon     Run daemon with ARGS='your-args'"
	@echo "  dev-daemon     Run daemon in development mode"
	@echo ""
	@echo "Testing Targets:"
	@echo "  test           Run unit tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  bench          Run benchmarks"
	@echo "  simulate       Run simulation tests"
	@echo "  integration    Run integration tests"
	@echo ""
	@echo "Code Quality Targets:"
	@echo "  fmt            Format code"
	@echo "  lint           Lint code"
	@echo "  vet            Vet code"
	@echo "  check          Run fmt, vet, lint, and test"
	@echo ""
	@echo "Utility Targets:"
	@echo "  mod            Update Go modules"
	@echo "  status         Show project status"
	@echo "  docs           Generate documentation"
	@echo "  clean          Clean build artifacts and repo data"
	@echo "  clean-bin      Clean build artifacts only"
	@echo "  clean-repo     Clean repository data only"
	@echo "  clean-config   Clean configuration data"
	@echo ""
	@echo "Release Targets:"
	@echo "  release        Create release archives"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-run     Run Docker container"
	@echo ""
	@echo "Examples:"
	@echo "  make run-cli ARGS='status'"
	@echo "  make run-cli ARGS='query main \"SELECT * FROM demo\"'"
	@echo "  make run-daemon ARGS='-addr :9090 -verbose'"