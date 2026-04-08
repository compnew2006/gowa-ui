.PHONY: all build build-prod run test clean docker-build docker-up docker-down migrate frontend-dev frontend-build backend-watch dev-watch air-install

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=whatomate
BINARY_PATH=./cmd/whatomate
LICENSE_VENDOR_BINARY=whatomate-license-vendor
LICENSE_VENDOR_PATH=./cmd/whatomate-license-vendor
LICENSE_ADMIN_BINARY=whatomate-license-admin
LICENSE_ADMIN_PATH=./cmd/whatomate-license-admin
LICENSE_ISSUE_BINARY=whatomate-license-issue
LICENSE_ISSUE_PATH=./cmd/whatomate-license-issue
LICENSE_STUDIO_BINARY=whatomate-license-studio
LICENSE_STUDIO_PATH=./cmd/whatomate-license-studio
VENDOR_TOOLS_DIR=bin/vendor-tools
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LICENSE_KEY_RING_JSON?=[]
GO_LDFLAGS=-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X github.com/compnew2006/whatomate/internal/license.EmbeddedPublicKeyRingJSON=$(LICENSE_KEY_RING_JSON)
LDFLAGS=-ldflags "$(GO_LDFLAGS)"
AIR_PACKAGE=github.com/air-verse/air@latest
AIR_BIN=$(shell sh -c 'if [ -n "$$GOBIN" ]; then printf "%s/air" "$$GOBIN"; else printf "%s/bin/air" "$$(go env GOPATH)"; fi')

# Docker parameters
DOCKER_COMPOSE=docker compose -f docker/docker-compose.yml

all: build

# Build the backend (development - without frontend)
build:
	$(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH)

build-license-tools:
	@mkdir -p $(VENDOR_TOOLS_DIR)
	$(GOBUILD) -o $(VENDOR_TOOLS_DIR)/$(LICENSE_VENDOR_BINARY) $(LICENSE_VENDOR_PATH)
	$(GOBUILD) -o $(VENDOR_TOOLS_DIR)/$(LICENSE_ADMIN_BINARY) $(LICENSE_ADMIN_PATH)
	$(GOBUILD) -o $(VENDOR_TOOLS_DIR)/$(LICENSE_ISSUE_BINARY) $(LICENSE_ISSUE_PATH)
	$(GOBUILD) -o $(VENDOR_TOOLS_DIR)/$(LICENSE_STUDIO_BINARY) $(LICENSE_STUDIO_PATH)

build-license-issue:
	@mkdir -p $(VENDOR_TOOLS_DIR)
	$(GOBUILD) -o $(VENDOR_TOOLS_DIR)/$(LICENSE_ISSUE_BINARY) $(LICENSE_ISSUE_PATH)

build-license-studio:
	@mkdir -p $(VENDOR_TOOLS_DIR)
	$(GOBUILD) -o $(VENDOR_TOOLS_DIR)/$(LICENSE_STUDIO_BINARY) $(LICENSE_STUDIO_PATH)

# Build production binary with embedded frontend
build-prod: frontend-build embed-frontend
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(BINARY_PATH)
	@echo "Production binary built: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@ls -lh $(BINARY_NAME)

# Copy frontend build to embed directory
embed-frontend:
	@echo "Copying frontend build to embed directory..."
	@rm -rf internal/frontend/dist/*
	@cp -r frontend/dist/* internal/frontend/dist/
	@echo "Frontend embedded successfully"

# Run the backend locally
run:
	$(GOCMD) run $(BINARY_PATH)/main.go server -config config.toml

# Run with migrations
run-migrate:
	$(GOCMD) run $(BINARY_PATH)/main.go server -config config.toml -migrate

air-install:
	$(GOCMD) install $(AIR_PACKAGE)

backend-watch:
	@AIR_CMD=$$(command -v air 2>/dev/null || true); \
	if [ -z "$$AIR_CMD" ] && [ -x "$(AIR_BIN)" ]; then \
		AIR_CMD="$(AIR_BIN)"; \
	fi; \
	if [ -z "$$AIR_CMD" ]; then \
		echo "Installing air watcher..."; \
		$(GOCMD) install $(AIR_PACKAGE); \
		AIR_CMD="$(AIR_BIN)"; \
	fi; \
	echo "Starting backend watcher with $$AIR_CMD"; \
	"$$AIR_CMD" -c .air.toml

# Run tests
test:
	$(GOTEST) -v ./...

# Run database tests with an ephemeral container
test-db:
	@echo "Starting ephemeral test database..."
	docker run --rm --name whatomate-test-db -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=test -p 5433:5432 -d postgres:17-alpine
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@TEST_DATABASE_URL="postgres://test:test@127.0.0.1:5433/test?sslmode=disable" $(GOTEST) -v -p 1 -coverprofile=coverage-db.out ./internal/database ./internal/contactutil || (docker stop whatomate-test-db && exit 1)
	@echo "Stopping test database..."
	docker stop whatomate-test-db
	$(GOCMD) tool cover -func=coverage-db.out || true

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf $(VENDOR_TOOLS_DIR)

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
deps-update:
	$(GOMOD) tidy
	$(GOGET) -u ./...

# Docker commands
docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-restart:
	$(DOCKER_COMPOSE) restart

# Database migrations
migrate:
	$(GOCMD) run $(BINARY_PATH)/main.go server -config config.toml -migrate

# Frontend commands
frontend-install:
	cd frontend && npm install

frontend-dev:
	@if [ ! -d "frontend/node_modules" ]; then \
		echo "Installing frontend dependencies..."; \
		cd frontend && npm install; \
	fi
	cd frontend && npm run dev

frontend-build:
	@if [ ! -d "frontend/node_modules" ] || [ ! -f "frontend/node_modules/.package-lock.json" ] || [ "frontend/package.json" -nt "frontend/node_modules/.package-lock.json" ] || [ "frontend/package-lock.json" -nt "frontend/node_modules/.package-lock.json" ]; then \
		echo "Installing frontend dependencies..."; \
		if [ -f "frontend/package-lock.json" ]; then \
			cd frontend && npm ci; \
		else \
			cd frontend && npm install; \
		fi; \
	fi
	cd frontend && npm run build

frontend-preview:
	cd frontend && npm run preview

# Development - run both backend and frontend
dev:
	@echo "Starting backend and frontend in development mode..."
	@make run-migrate &
	@make frontend-dev

dev-watch:
	@echo "Starting backend watcher and frontend dev server..."
	@trap 'kill 0' INT TERM EXIT; \
	$(MAKE) backend-watch & \
	$(MAKE) frontend-dev & \
	wait

# Lint
lint:
	golangci-lint run ./...

# Format code
fmt:
	$(GOCMD) fmt ./...

# Generate swagger docs (if using)
swagger:
	swag init -g cmd/whatomate/main.go -o api/docs

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Production:"
	@echo "  build-prod     - Build single binary with embedded frontend"
	@echo "  build-license-tools - Build private vendor license utilities into bin/vendor-tools"
	@echo "  build-license-issue - Build the dedicated private issuer into bin/vendor-tools"
	@echo "  build-license-studio - Build the private localhost GUI studio into bin/vendor-tools"
	@echo ""
	@echo "Development:"
	@echo "  build          - Build the backend binary (without frontend)"
	@echo "  run            - Run the backend locally"
	@echo "  run-migrate    - Run the backend with database migrations"
	@echo "  dev            - Run both backend and frontend in development mode"
	@echo "  backend-watch  - Run backend with Go hot reload and migrations"
	@echo "  dev-watch      - Run backend hot reload + frontend dev server"
	@echo ""
	@echo "Frontend:"
	@echo "  frontend-install - Install frontend dependencies"
	@echo "  frontend-dev   - Run frontend in development mode"
	@echo "  frontend-build - Build frontend for production"
	@echo "  air-install    - Install the Go air watcher locally"
	@echo ""
	@echo "Testing:"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-up      - Start Docker containers"
	@echo "  docker-down    - Stop Docker containers"
	@echo "  docker-logs    - View Docker logs"
	@echo ""
	@echo "Other:"
	@echo "  clean          - Remove build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
