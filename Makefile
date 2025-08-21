.PHONY: build run dev clean check-commands register-commands delete-commands setup-commands

# Suppress gopus warnings (where applicable)
export CGO_CFLAGS=-Wno-stringop-overread -Wno-format -Wno-unused-parameter -Wno-pragma-messages

# Binary name
BIN_NAME=tarumae

# Source entry
ENTRY=cmd/main.go

# Build the bot (optimized)
build:
	@echo "🔧 Building $(BIN_NAME) with CGO optimizations and warning suppression..."
	CGO_CFLAGS="-O2 -Wno-stringop-overread -Wno-unused-parameter" go build -ldflags="-X 'github.com/latoulicious/HKTM/internal/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'dev')' -X 'github.com/latoulicious/HKTM/internal/version.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'" -o $(BIN_NAME) $(ENTRY)

# Run the optimized binary
run: build
	@echo "🚀 Running $(BIN_NAME)..."
	./$(BIN_NAME)

# Fast dev run using `go run` (may show gopus warning)
dev:
	@echo "⚡ Dev running (no build, fast iteration)..."
	go run -ldflags="-X 'github.com/latoulicious/HKTM/internal/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'dev')' -X 'github.com/latoulicious/HKTM/internal/version.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'" $(ENTRY)

# Check registered commands
check-commands:
	@echo "🔍 Checking registered slash commands..."
	go run tools/slash-manager.go -action check

# Register slash commands
register-commands:
	@echo "📝 Registering slash commands..."
	go run tools/slash-manager.go -action register

# Delete all slash commands
delete-commands:
	@echo "🗑️ Deleting all slash commands..."
	go run tools/slash-manager.go -action delete-all

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f $(BIN_NAME)
	go clean

# Setup: delete old commands and register new ones
setup-commands: delete-commands register-commands check-commands
	@echo "✅ Slash commands setup complete!"

# Create a new release tag
release:
	@echo "Current version: $(shell git describe --tags --abbrev=0 2>/dev/null || echo 'No tags found')"
	@echo "Creating new release..."
	@read -p "Enter version (e.g., v0.1.0): " version; \
	git tag -a $$version -m "Release $$version"; \
	git push origin $$version; \
	echo "Released $$version"

# Show version info
version:
	@echo "Version info:"
	@go run -ldflags="-X 'github.com/latoulicious/HKTM/internal/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'dev')' -X 'github.com/latoulicious/HKTM/internal/version.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'" tools/version.go
