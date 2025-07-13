.PHONY: build test lint fmt vet clean install-tools setup

# Build the application
build:
	go build -o osoba main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -cover ./...

# Run linter
lint:
	@if command -v golangci-lint >/dev/null 2>&1 || [ -f "$$(go env GOPATH)/bin/golangci-lint" ]; then \
		export PATH=$$PATH:$$(go env GOPATH)/bin; \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Run 'make install-tools' to install it."; \
		exit 1; \
	fi

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -f osoba
	go clean

# Install development tools
install-tools:
	@echo "Installing golangci-lint v2..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin latest
	@echo "golangci-lint installed to $$(go env GOPATH)/bin"
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