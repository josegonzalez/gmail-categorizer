.PHONY: build test coverage lint clean run

BINARY_NAME=gmail-categorizer
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gmail-categorizer

test:
	go test -v -race -coverprofile=coverage.out ./...

coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint:
	golangci-lint run

clean:
	rm -rf $(BUILD_DIR)/ coverage.out coverage.html

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

install:
	go install ./cmd/gmail-categorizer

# Development helpers
fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

all: fmt vet lint test build
