BINARY     := netsoryn
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-X main.version=$(VERSION) -s -w"
BUILD_DIR  := dist
GO         := go

.PHONY: all build run clean test lint tidy install help

all: build

## build: compile the binary
build:
	@echo "  Building $(BINARY) $(VERSION)…"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/$(BINARY)

## run: build and run immediately
run: build
	./$(BUILD_DIR)/$(BINARY)

## install: install to $GOPATH/bin
install:
	$(GO) install $(LDFLAGS) ./cmd/$(BINARY)

## test: run all tests
test:
	$(GO) test ./... -v -race -timeout 60s

## test-cover: run tests with coverage report
test-cover:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "  Coverage report: coverage.html"

## lint: run golangci-lint (must be installed separately)
lint:
	golangci-lint run ./...

## tidy: tidy go modules
tidy:
	$(GO) mod tidy

## clean: remove build artefacts
clean:
	@rm -rf $(BUILD_DIR) coverage.out coverage.html

## cross: build for Linux amd64, macOS arm64, and Windows amd64
cross:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/$(BINARY)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/$(BINARY)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/$(BINARY)

## help: show this help
help:
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^##' Makefile | sed 's/## /  /'
	@echo ""
