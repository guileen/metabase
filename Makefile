# MetaBase Optimized Makefile

BINARY_NAME=metabase
BUILD_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

.PHONY: all setup dev build test check clean release

# Default target
all: setup build

# Complete development setup
setup:
	@echo "📦 Setting up development environment..."
	@go mod tidy && go mod download
	@go install github.com/air-verse/air@latest github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@if [ -d "admin-svelte" ]; then cd admin-svelte && npm install; fi
	@if [ -d "clients/typescript" ]; then cd clients/typescript && npm install; fi
	@mkdir -p data/{pebble,sqlite,nats} logs uploads/{temp,files,cache} web/{admin,assets}
	@echo "✅ Setup complete"

# Development with hot reload and full server
dev:
	@echo "🚀 Starting development server with hot reload..."
	@mkdir -p data logs uploads temp
	@go run ./cmd/$(BINARY_NAME) server --dev --enable-all

# Build binary with assets
build:
	@echo "🏗️ Building MetaBase..."
	@mkdir -p $(BUILD_DIR)
	@if [ -d "admin-svelte" ]; then \
		cd admin-svelte && npm run build && \
		rm -rf ../web/admin && mkdir -p ../web/admin && \
		cp -r build/* ../web/admin/; \
	fi
	@if [ -d "clients/typescript" ]; then \
		cd clients/typescript && npm run build && \
		mkdir -p ../../web/assets/client && \
		cp -r dist/* ../../web/assets/client/; \
	fi
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Quick binary build (no assets)
build-bin:
	@echo "⚡ Building binary only..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v -race ./...

# Test with coverage
test-cover:
	@echo "📊 Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Code quality checks
check: lint security
	@echo "✅ All quality checks passed"

# Lint code
lint:
	@echo "🔍 Linting code..."
	@golangci-lint run

# Security scan
security:
	@echo "🔒 Security scan..."
	@gosec ./...

# Clean artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html data logs uploads temp tmp
	@rm -rf web/admin web/assets/client
	@go clean -cache -testcache

# Production build for all platforms
release: clean test
	@echo "📦 Building release..."
	@mkdir -p $(BUILD_DIR) release
	# Build assets first
	@if [ -d "admin-svelte" ]; then cd admin-svelte && npm run build; fi
	@if [ -d "clients/typescript" ]; then cd clients/typescript && npm run build; fi
	# Cross-platform builds
	@CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/$(BINARY_NAME)
	@CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/$(BINARY_NAME)
	@CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/$(BINARY_NAME)
	@CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/$(BINARY_NAME)
	@CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/$(BINARY_NAME)
	# Package
	@cd $(BUILD_DIR) && tar -czf ../release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf ../release/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(BUILD_DIR) && tar -czf ../release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf ../release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && zip ../release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "✅ Release artifacts created in release/"

# Docker
docker-build:
	@docker build -t metabase:$(VERSION) . && docker tag metabase:$(VERSION) metabase:latest

# Format code
fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./... && goimports -w .
	@if [ -d "admin-svelte" ]; then cd admin-svelte && npm run format; fi

# Help
help:
	@echo "🚀 MetaBase Commands"
	@echo "Setup:     setup"
	@echo "Develop:   dev"
	@echo "Build:     build | build-bin"
	@echo "Test:      test | test-cover"
	@echo "Quality:   check | lint | security | fmt"
	@echo "Release:   release | docker-build"
	@echo "Clean:     clean"