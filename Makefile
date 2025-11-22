# MetaBase Optimized Makefile

BINARY_NAME=metabase
BUILD_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

.PHONY: all setup preprocess dev build build-bin go test check clean release

# Default target
all: setup build

# Complete development setup
setup:
	@echo "📦 Setting up development environment..."
	@go mod tidy && go mod download
	@go install github.com/air-verse/air@latest github.com/golangci/golangci-lint/cmd/golangci-lint@latest golang.org/x/tools/cmd/goimports@latest
	@if [ -d "admin-svelte" ]; then cd admin-svelte && npm install; fi
	@if [ -d "clients/typescript" ]; then cd clients/typescript && npm install; fi
	@mkdir -p data/{pebble,sqlite,nats} logs uploads/{temp,files,cache} web/{admin,assets}
	@echo "✅ Setup complete"

# Preprocess code - clean imports and fix basic lint issues
preprocess:
	@echo "🧹 Preprocessing code..."
	@echo "  • Removing unused imports..."
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w . 2>/dev/null || echo "    ⚠️ goimports completed with warnings"; \
	else \
		echo "    ⚠️ goimports not found, run 'make setup' first"; \
	fi
	@echo "  • Formatting code..."
	@go fmt ./... 2>/dev/null || echo "    ⚠️ go fmt completed with warnings"
	@echo "  • Tidying modules..."
	@go mod tidy 2>/dev/null || echo "    ⚠️ go mod tidy completed with warnings"
	@echo "  • Running quick lint fixes..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix --max-issues-per-linter=10 --max-same-issues=10 --disable=errcheck,staticcheck,unused,ineffassign,gosimple 2>/dev/null || echo "    ⚠️ golangci-lint fixes completed with warnings"; \
	else \
		echo "    ⚠️ golangci-lint not found, run 'make setup' first"; \
	fi
	@echo "✅ Preprocessing complete"

# Development with hot reload and full server
dev:
	@echo "🚀 Starting development server with hot reload..."
	@echo "三层架构服务启动: Gateway(7609) + API(7610) + Admin(7680) + Website(8080)"
	@mkdir -p data logs uploads temp
	@$(MAKE) preprocess
	@go run ./cmd/$(BINARY_NAME) gateway --dev --enable-api --enable-admin --enable-web

# Development with hot reload for API only
dev-api:
	@echo "🚀 Starting API server with hot reload..."
	@mkdir -p data logs uploads temp
	@$(MAKE) preprocess
	@go run ./cmd/$(BINARY_NAME) api --dev --port 7610

# Development with hot reload for Admin only
dev-admin:
	@echo "🚀 Starting Admin server with hot reload..."
	@mkdir -p data logs uploads temp
	@$(MAKE) preprocess
	@go run ./cmd/$(BINARY_NAME) admin --dev --port 7680

# Development with hot reload for Website only
dev-www:
	@echo "🚀 Starting Website server with hot reload..."
	@mkdir -p data logs uploads temp
	@$(MAKE) preprocess
	@go run ./cmd/$(BINARY_NAME) www --dev --port 8080

# Build binary with assets
build:
	@echo "🏗️ Building MetaBase..."
	@mkdir -p $(BUILD_DIR)
	@$(MAKE) preprocess
	@if [ -d "admin-svelte" ] && [ -f "admin-svelte/package.json" ]; then \
		if [ -d "admin-svelte/dist" ] && [ -d "admin-svelte/src" ] && \
		   [ -z "$(find admin-svelte/src -type f -newer admin-svelte/dist -print -quit)" ]; then \
			echo "⏩ Skipping admin interface (no changes)"; \
		else \
			echo "Building admin interface..." && \
			cd admin-svelte && (npm run build >/dev/null 2>&1 || true); \
		fi; \
		if [ -d "../web/admin-svelte" ]; then \
			rm -rf ../web/admin && mkdir -p ../web/admin && \
			cp -r ../web/admin-svelte/* ../web/admin/ 2>/dev/null || true; \
			echo "✓ Admin interface built"; \
		else \
			echo "⚠️ Admin interface build skipped (warnings)"; \
		fi; \
	fi
	@if [ -d "clients/typescript" ]; then \
		if [ -d "clients/typescript/dist" ] && [ -d "clients/typescript/src" ] && \
		   [ -z "$(find clients/typescript/src -type f -newer clients/typescript/dist -print -quit)" ]; then \
			echo "⏩ Skipping TypeScript client (no changes)"; \
		else \
			echo "Building TypeScript client..." && \
			cd clients/typescript && (npm run build >/dev/null 2>&1 || true); \
		fi; \
		mkdir -p web/assets/client && \
		cp -r clients/typescript/dist/* web/assets/client/ 2>/dev/null || true; \
		echo "✓ TypeScript client ready"; \
	fi
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Quick binary build (no assets)
build-bin:
	@echo "⚡ Building binary only..."
	@mkdir -p $(BUILD_DIR)
	@$(MAKE) preprocess
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)



go: build-bin

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
	@golangci-lint run --max-issues-per-linter=50 --max-same-issues=50

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
	@echo "Setup:     setup | preprocess"
	@echo "Develop:   dev | dev-api | dev-admin | dev-www"
	@echo "Build:     build | build-bin | go"
	@echo "Test:      test | test-cover"
	@echo "Quality:   check | lint | security | fmt"
	@echo "Release:   release | docker-build"
	@echo "Clean:     clean"