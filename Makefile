VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY  := NetBird-TUI

.PHONY: build install clean vet vet-cross fmt lint test help

## build: compile binary with version ldflags
build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/netbird-tui

## install: build and install to ~/.local/bin
install: build
	install -m 755 $(BINARY) $(HOME)/.local/bin/$(BINARY)

## clean: remove compiled binary
clean:
	rm -f $(BINARY)

## vet: run go vet
vet:
	go vet ./...

## vet-cross: run go vet for both Linux and Darwin
vet-cross:
	GOOS=linux  go vet ./...
	GOOS=darwin go vet ./...

## fmt: format source files with gofmt
fmt:
	gofmt -l -w .

## lint: run golangci-lint (install separately)
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found — install from https://golangci-lint.run/usage/install/"; exit 0; }
	golangci-lint run ./...

## test: run all tests
test:
	go test ./...

## help: list available targets
help:
	@grep -E '^## ' Makefile | awk 'BEGIN{FS=": "} {printf "  %-14s %s\n", $$1, $$2}' | sed 's/^  ## //'
