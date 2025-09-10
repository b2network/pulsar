BINARY_NAME=pulsard
CLI_NAME=pulsarcli
BUILD_DIR=build
HOME_DIR=$(HOME)/.pulsar
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse HEAD)
LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)

.PHONY: all build clean install init start reset deps test lint fmt run fresh build-all

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/pulsard
	@echo "Building $(CLI_NAME)..."
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(CLI_NAME) ./cmd/pulsarcli

build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/pulsard
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/pulsard
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/pulsard
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/pulsard
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/pulsard

install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/pulsard
	@echo "Installing $(CLI_NAME)..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/pulsarcli

init: build
	@echo "Initializing node..."
	@./$(BUILD_DIR)/$(BINARY_NAME) init "test-node" --home $(HOME_DIR)

start: build
	@echo "Starting node..."
	@./$(BUILD_DIR)/$(BINARY_NAME) start --home $(HOME_DIR)

reset:
	@echo "Resetting node data..."
	@rm -rf $(HOME_DIR)

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

test-integration:
	@echo "Running integration tests..."
	@go test -v -tags integration ./tests/...

lint:
	@echo "Running linters..."
	@golangci-lint run

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then goimports -w .; else echo "goimports not found, skipping import organization"; fi

check: fmt lint test

run: build init start

fresh: reset run

help:
	@echo "Available commands:"
	@echo "  build          - Build the binary"
	@echo "  build-all      - Build for all platforms"
	@echo "  install        - Install the binary"
	@echo "  init           - Initialize node"
	@echo "  start          - Start node"
	@echo "  reset          - Reset node data"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  test           - Run tests"
	@echo "  test-integration - Run integration tests"
	@echo "  lint           - Run linters"
	@echo "  fmt            - Format code"
	@echo "  check          - Run fmt, lint and test"
	@echo "  run            - Build, init and start"
	@echo "  fresh          - Reset, init and start"
	@echo "  help           - Show this help"