.PHONY: build test lint fmt vet clean install-tools setup install run help

# 変数定義
BINARY_NAME := osoba
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date +%Y-%m-%d\ %H:%M:%S)
LDFLAGS := -ldflags "-X 'github.com/douhashi/osoba/internal/version.Version=$(VERSION)' -X 'github.com/douhashi/osoba/internal/version.Commit=$(COMMIT)' -X 'github.com/douhashi/osoba/internal/version.Date=$(DATE)'"

# Build the application
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -cover ./...

# Run linter (removed - using standard Go tools instead)
lint:
	@echo "Linting with standard Go tools..."
	@go vet ./...
	@echo "Linting complete!"

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean

# Install development tools
install-tools:
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Make sure to add $$(go env GOPATH)/bin to your PATH"

# Setup development environment
setup: install-tools
	@echo "Setting up git hooks..."
	@git config core.hooksPath .githooks
	@echo "Development environment setup complete!"

# Run all checks (format, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

# Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $$(go env GOPATH)/bin..."
	@cp $(BINARY_NAME) $$(go env GOPATH)/bin/
	@echo "Installation complete! Make sure $$(go env GOPATH)/bin is in your PATH."

# Run the application with default arguments
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Display help information about available targets
help:
	@echo "Makefile for $(BINARY_NAME) v$(VERSION)"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build           Build the application binary"
	@echo "  test            Run all tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  lint            Run go vet"
	@echo "  fmt             Format code using go fmt"
	@echo "  vet             Run go vet"
	@echo "  clean           Remove build artifacts"
	@echo "  install         Build and install to GOPATH/bin"
	@echo "  run             Build and run the application"
	@echo "  check           Run all checks (fmt, vet, lint, test)"
	@echo "  install-tools   Install required development tools"
	@echo "  setup           Setup development environment"
	@echo "  help            Display this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  COMMIT=$(COMMIT)"
	@echo "  GOPATH=$$(go env GOPATH)"

# Default target
.DEFAULT_GOAL := help