# Shelly Go Library Makefile
# Provides convenient commands for development and CI

.PHONY: all test test-ci test-local test-integration test-coverage lint build clean help

# Default target
all: lint test build

# =============================================================================
# Testing
# =============================================================================

# Run unit tests (skips hardware-dependent tests)
# Usage: make test
test:
	go test -v -race ./...

# Run tests in CI mode (no hardware, fast)
# Skips integration tests and any hardware-dependent functionality
# Usage: make test-ci
test-ci:
	SHELLY_CI=1 go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# Run tests locally with discovery support
# Attempts mDNS/CoIoT discovery if no device addresses provided
# Usage: make test-local
# Usage with devices: make test-local GEN2=10.23.47.220 GEN1=10.23.47.221
test-local:
ifdef GEN2
	$(eval export SHELLY_TEST_GEN2_ADDR=$(GEN2))
endif
ifdef GEN1
	$(eval export SHELLY_TEST_GEN1_ADDR=$(GEN1))
endif
	go test -v -race -coverprofile=coverage.out ./...

# Run integration tests against real devices
# Requires device addresses and SHELLY_INTEGRATION_TESTS=1
# Usage: make test-integration GEN2=10.23.47.220
# For device control tests: make test-integration GEN2=10.23.47.220 ACTUATE=1
test-integration:
ifndef GEN2
	$(error GEN2 is required. Usage: make test-integration GEN2=<ip-address>)
endif
	SHELLY_INTEGRATION_TESTS=1 \
	SHELLY_TEST_GEN2_ADDR=$(GEN2) \
	$(if $(GEN1),SHELLY_TEST_GEN1_ADDR=$(GEN1)) \
	$(if $(ACTUATE),SHELLY_TEST_ACTUATE=1) \
	go test -v -tags=integration ./internal/testutil/integration/...

# Run Cloud API integration tests
# Requires cloud credentials
# Usage: make test-cloud EMAIL=you@example.com PASSWORD=secret
test-cloud:
ifndef EMAIL
	$(error EMAIL is required for Cloud API tests)
endif
ifndef PASSWORD
	$(error PASSWORD is required for Cloud API tests)
endif
	SHELLY_INTEGRATION_TESTS=1 \
	SHELLY_TEST_CLOUD_EMAIL=$(EMAIL) \
	SHELLY_TEST_CLOUD_PASSWORD=$(PASSWORD) \
	go test -v -tags=integration ./internal/testutil/integration/cloud_test.go

# Run tests with coverage report
# Usage: make test-coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report: go tool cover -html=coverage.out"

# Run tests with HTML coverage report
# Usage: make test-coverage-html
test-coverage-html: test-coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run a specific package's tests
# Usage: make test-pkg PKG=gen2/components
test-pkg:
ifndef PKG
	$(error PKG is required. Usage: make test-pkg PKG=gen2/components)
endif
	go test -v -race -coverprofile=coverage.out ./$(PKG)/...
	go tool cover -func=coverage.out | grep -E '^total:|$(PKG)'

# =============================================================================
# Code Quality
# =============================================================================

# Run linter
# Usage: make lint
lint:
	golangci-lint run --timeout=5m

# Run linter with auto-fix
# Usage: make lint-fix
lint-fix:
	golangci-lint run --fix --timeout=5m

# Format code
# Usage: make fmt
fmt:
	go fmt ./...
	gofumpt -w .

# Vet code
# Usage: make vet
vet:
	go vet ./...

# =============================================================================
# Build
# =============================================================================

# Build all packages
# Usage: make build
build:
	go build -v ./...

# Build examples
# Usage: make build-examples
build-examples:
	@for dir in examples/basic/* examples/cloud/* examples/advanced/* examples/discovery/*; do \
		if [ -d "$$dir" ]; then \
			echo "Building $$dir..."; \
			go build -v ./$$dir || exit 1; \
		fi \
	done

# =============================================================================
# Dependencies
# =============================================================================

# Download and verify dependencies
# Usage: make deps
deps:
	go mod download
	go mod verify

# Tidy dependencies
# Usage: make tidy
tidy:
	go mod tidy

# =============================================================================
# Documentation
# =============================================================================

# Start local godoc server
# Usage: make docs
docs:
	@echo "Starting godoc server at http://localhost:6060"
	@echo "View package at: http://localhost:6060/pkg/github.com/tj-smith47/shelly-go/"
	godoc -http=:6060

# =============================================================================
# Cleanup
# =============================================================================

# Clean build artifacts
# Usage: make clean
clean:
	go clean
	rm -f coverage.out coverage.html

# =============================================================================
# Discovery (Development Helpers)
# =============================================================================

# Discover devices on local network (requires real network access)
# Usage: make discover
discover:
	@echo "Discovering Shelly devices on local network..."
	@go run -exec "env" examples/discovery/mdns/main.go 2>/dev/null || \
		echo "Run: go run examples/discovery/mdns/main.go"

# =============================================================================
# Help
# =============================================================================

# Show help
# Usage: make help
help:
	@echo "Shelly Go Library - Makefile Commands"
	@echo ""
	@echo "Testing:"
	@echo "  make test              - Run unit tests (no hardware needed)"
	@echo "  make test-ci           - Run tests in CI mode (SHELLY_CI=1)"
	@echo "  make test-local        - Run tests locally"
	@echo "  make test-local GEN2=<ip> GEN1=<ip> - Run with device addresses"
	@echo "  make test-integration GEN2=<ip>    - Run integration tests"
	@echo "  make test-cloud EMAIL=x PASSWORD=x - Run cloud API tests"
	@echo "  make test-coverage     - Run tests with coverage report"
	@echo "  make test-coverage-html- Generate HTML coverage report"
	@echo "  make test-pkg PKG=<pkg>- Test specific package"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint              - Run linter"
	@echo "  make lint-fix          - Run linter with auto-fix"
	@echo "  make fmt               - Format code"
	@echo "  make vet               - Run go vet"
	@echo ""
	@echo "Build:"
	@echo "  make build             - Build all packages"
	@echo "  make build-examples    - Build example programs"
	@echo ""
	@echo "Dependencies:"
	@echo "  make deps              - Download and verify dependencies"
	@echo "  make tidy              - Tidy go.mod"
	@echo ""
	@echo "Documentation:"
	@echo "  make docs              - Start local godoc server"
	@echo ""
	@echo "Other:"
	@echo "  make clean             - Clean build artifacts"
	@echo "  make discover          - Discover devices on network"
	@echo "  make help              - Show this help"
	@echo ""
	@echo "Environment Variables:"
	@echo "  SHELLY_CI=1                   - Skip hardware-dependent tests"
	@echo "  SHELLY_INTEGRATION_TESTS=1    - Enable integration tests"
	@echo "  SHELLY_TEST_GEN2_ADDR=<ip>    - Gen2 device IP for testing"
	@echo "  SHELLY_TEST_GEN1_ADDR=<ip>    - Gen1 device IP for testing"
	@echo "  SHELLY_TEST_ACTUATE=1         - Enable device control tests"
	@echo "  SHELLY_TEST_CLOUD_EMAIL=<email>    - Cloud API email"
	@echo "  SHELLY_TEST_CLOUD_PASSWORD=<pass>  - Cloud API password"
