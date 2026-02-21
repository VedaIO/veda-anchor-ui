# Makefile for Veda Anchor UI

# Use git describe to get a version string.
# Example: v1.0.0-3-g1234567
# Fallback to 'dev' if not in a git repository.
VERSION ?= $(shell git describe --tags --always --dirty --first-parent 2>/dev/null || echo "dev")

.PHONY: all build build-debug fmt clean

all: build
build:
	@echo "Building Veda Anchor UI for windows..."
	CGO_ENABLED=0 wails build -platform windows/amd64 -ldflags="-H=windowsgui -X main.version=$(VERSION)"

build-debug:
	@echo "Building Veda Anchor UI for windows (debug)..."
	CGO_ENABLED=0 wails build -platform windows/amd64 -ldflags="-X main.version=$(VERSION)"

fmt:
	@echo "Formatting code..."
	go fmt ./...
	cd frontend && bun run format

lint:
	CGO_ENABLED=0 GOOS=windows golangci-lint run
	cd frontend && bun run lint

clean:
	@echo "Cleaning..."
	rm -rf build/bin
	rm -rf frontend/dist
	rm -rf frontend/wailsjs
	rm -rf frontend/package.json.md5
