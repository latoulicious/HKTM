# -------- Global config --------
.SILENT:
SHELL := /usr/bin/env bash
.DEFAULT_GOAL := build

# Binary & entry
BIN_NAME ?= HTKM
ENTRY    ?= cmd/main.go

# Module path (used by -ldflags)
MODULE := github.com/latoulicious/HKTM

# Version info (one place)
# Prefer annotated SemVer tags (vX.Y.Z). Fallback to dev+sha, keep dirty marker.
VERSION   ?= $(shell git describe --tags --match 'v[0-9]*' --abbrev=0 2>/dev/null || echo v0.0.0-dev)
SHORT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
IS_DIRTY  := $(shell git diff --quiet || echo -dirty)
COMMIT    := $(shell git rev-parse HEAD 2>/dev/null || echo 0000000)
BUILDTIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# If we are NOT on a tag checkout, expose dev+sha
ON_TAG := $(shell git describe --tags --exact-match >/dev/null 2>&1 && echo yes || echo no)
ifeq ($(ON_TAG),no)
  EFFECTIVE_VERSION := $(VERSION)+$(SHORT_SHA)$(IS_DIRTY)
else
  EFFECTIVE_VERSION := $(VERSION)
endif

# CGO (opus/ffmpeg). Keep in one place.
export CGO_CFLAGS := -O2 -Wno-stringop-overread -Wno-unused-parameter -Wno-format -Wno-pragma-messages

# Build flags
LDFLAGS := -s -w -trimpath \
  -X '$(MODULE)/internal/version.Version=$(EFFECTIVE_VERSION)' \
  -X '$(MODULE)/internal/version.GitCommit=$(COMMIT)' \
  -X '$(MODULE)/internal/version.BuildTime=$(BUILDTIME)'

# -------- Targets --------
.PHONY: build run dev clean version check-commands register-commands delete-commands setup-commands build-all release tag verify-tag changelog

build:
	echo "🔧 Building $(BIN_NAME) ($(EFFECTIVE_VERSION))..."
	go build -ldflags="$(LDFLAGS)" -o $(BIN_NAME) $(ENTRY)

run: build
	echo "🚀 Running $(BIN_NAME)..."
	./$(BIN_NAME)

dev:
	echo "⚡ go run (no binary)…"
	go run -ldflags="$(LDFLAGS)" $(ENTRY)

# Cross-compilation (common triples). Extend if you need more.
build-all:
	set -euo pipefail; \
	echo "🧱 Building matrix for $(EFFECTIVE_VERSION)…"; \
	mkdir -p dist; \
	for GOOS in linux darwin windows; do \
	  for GOARCH in amd64 arm64; do \
	    OUT="dist/$(BIN_NAME)_$${GOOS}_$${GOARCH}"; \
	    [[ "$${GOOS}" == "windows" ]] && OUT="$${OUT}.exe"; \
	    echo "  → $${OUT}"; \
	    GOOS=$${GOOS} GOARCH=$${GOARCH} go build -ldflags="$(LDFLAGS)" -o "$${OUT}" $(ENTRY); \
	  done; \
	done

# Human-friendly version print (same source of truth)
version:
	echo "Version:      $(EFFECTIVE_VERSION)"
	echo "Commit:       $(COMMIT)"
	echo "Build Time:   $(BUILDTIME)"
	go run -ldflags="$(LDFLAGS)" tools/version.go || true

# Slash commands (fix path to your actual file)
check-commands:
	echo "🔍 Checking registered slash commands…"
	go run tools/slash_sync.go -action check

register-commands:
	echo "📝 Registering slash commands…"
	go run tools/slash_sync.go -action register

delete-commands:
	echo "🗑️ Deleting all slash commands…"
	go run tools/slash_sync.go -action delete-all

setup-commands: delete-commands register-commands check-commands
	echo "✅ Slash commands setup complete!"

clean:
	echo "🧹 Cleaning…"
	rm -f $(BIN_NAME)
	rm -rf dist
	go clean

# -------- Release helpers --------

# Guard that the provided tag is proper SemVer (vX.Y.Z).
verify-tag:
	@if [[ -z "$$tag" ]]; then echo "Usage: make tag tag=vX.Y.Z"; exit 2; fi
	@if [[ ! "$$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$$ ]]; then \
	  echo "❌ Invalid tag: $$tag (want vX.Y.Z)"; exit 2; fi

# Create and push an annotated tag (safe, validated)
tag: verify-tag
	git tag -a "$$tag" -m "Release $$tag"
	git push origin "$$tag"
	echo "🏷️  Tagged $$tag"

# Optional: generate/append changelog section from last tag
# Requires scripts/changelog.sh from earlier message.
changelog:
	@if [[ ! -x scripts/changelog.sh ]]; then echo "Missing scripts/changelog.sh"; exit 2; fi
	# Detect previous and next tagish values
	prev="$$(git describe --tags --abbrev=0 2>/dev/null || true)"; \
	next="$(EFFECTIVE_VERSION)"; \
	echo "📝 Updating CHANGELOG.md for range $$prev..$$next"; \
	./scripts/changelog.sh "$$next" "$$prev" HEAD
