BINARY_NAME=acb
BINARY_DIR=dist

.PHONY: all build clean test

all: clean build

build:
	@echo "Building..."
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/acb

clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -rf tmp

test:
	@echo "Running tests..."
	@go test -v ./...

lint:
	@echo "Running linter..."
	@go vet ./...
	@test -z "$$(gofmt -l .)"

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build    - Build the binary"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make test     - Run tests"
	@echo "  make lint     - Run linter"
	@echo "  make all      - Clean and build"
