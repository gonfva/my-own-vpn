.PHONY: all build build-windows test lint fmt clean tools check security tools-security

# Binary name
BINARY_NAME=my-own-vpn
VERSION?=dev

# Build flags
LDFLAGS=-s -w -X main.Version=$(VERSION)
# Windows-specific LDFLAGS to hide console window
LDFLAGS_WINDOWS=$(LDFLAGS) -H windowsgui

all: lint test build

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/my-own-vpn

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS_WINDOWS)" -o $(BINARY_NAME).exe ./cmd/my-own-vpn

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

# Security scanning
security:
	govulncheck ./...

# Install security tools
tools-security:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
