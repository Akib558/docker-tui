BINARY  := docker-tui
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build run test lint clean install fmt fmt-check

build:
	go build $(LDFLAGS) -o $(BINARY) .

run:
	go run .

test:
	go test ./...

test-race:
	go test -race ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)"

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/ coverage.out coverage.html

install:
	go install $(LDFLAGS) .

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
