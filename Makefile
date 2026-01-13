.PHONY: all build test lint fmt clean tools check

# Binary name
BINARY_NAME=my-own-vpn
VERSION?=dev

# Build flags
LDFLAGS=-s -w -X main.Version=$(VERSION)

all: lint test build

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/my-own-vpn

test:
	go test -v -race ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w .

clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Run all checks (useful before committing)
check: fmt lint test
