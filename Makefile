BINARY_NAME=claw-migrate
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0-dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build build-all install clean test lint

## build: Build for current platform
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

## build-all: Build for all platforms
build-all:
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 .
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 .

## install: Build and install to /usr/local/bin
install: build
	sudo cp bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

## test: Run tests
test:
	go test ./...

## lint: Run linter
lint:
	golangci-lint run

## clean: Remove build artifacts
clean:
	rm -rf bin/

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
