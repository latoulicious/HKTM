# -------- Global config --------
.SILENT:
SHELL := /bin/sh
.ONESHELL:
.DEFAULT_GOAL := build

# Binary & entry
BIN_NAME ?= HKTM
ENTRY    ?= cmd/main.go

# Module path (used by -ldflags)
MODULE := github.com/latoulicious/HKTM

# Version info
VERSION   ?= $(shell git describe --tags --match 'v[0-9]*' --abbrev=0 2>/dev/null || echo v0.0.0-dev)
SHORT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
IS_DIRTY  := $(shell git diff --quiet || echo -dirty)
COMMIT    := $(shell git rev-parse HEAD 2>/dev/null || echo 0000000)
BUILDTIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

ON_TAG := $(shell git describe --tags --exact-match >/dev/null 2>&1 && echo yes || echo no)
ifeq ($(ON_TAG),no)
  EFFECTIVE_VERSION := $(VERSION)+$(SHORT_SHA)$(IS_DIRTY)
else
  EFFECTIVE_VERSION := $(VERSION)
endif

# CGO (opus/ffmpeg)
export CGO_CFLAGS := -O2 -Wno-stringop-overread -Wno-unused-parameter -Wno-format -Wno-pragma-messages

# Build flags
GO_BUILD_FLAGS := -trimpath
LDFLAGS := -s -w \
  -X '$(MODULE)/internal/version.Version=$(EFFECTIVE_VERSION)' \
  -X '$(MODULE)/internal/version.GitCommit=$(COMMIT)' \
  -X '$(MODULE)/internal/version.BuildTime=$(BUILDTIME)'

# -------- Targets --------
.PHONY: build run dev clean version check-commands register-commands delete-commands setup-commands build-all release tag verify-tag changelog

build:
	echo "🔧 Building $(BIN_NAME) ($(EFFECTIVE_VERSION))..."
	go build $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_NAME) $(ENTRY)

run: build
	echo "🚀 Running $(BIN_NAME)..."
	./$(BIN_NAME)

dev:
	echo "⚡ go run (no binary)…"
	go run $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" $(ENTRY)

# Cross-compilation (common triples). CGO on linux only to avoid cross C toolchains.
GOOS_LIST   ?= linux darwin windows
GOARCH_LIST ?= amd64 arm64

build-all:
	set -eu; \
	echo "🧱 Building matrix for $(EFFECTIVE_VERSION)…"; \
	mkdir -p dist; \
	host_goos="$$(go env GOOS)"; \
	host_goarch="$$(go env GOARCH)"; \
	for GOOS in $(GOOS_LIST); do \
	  for GOARCH in $(GOARCH_LIST); do \
	    OUT="dist/$(BIN_NAME)_$${GOOS}_$${GOARCH}"; \
	    if [ "$${GOOS}" = "windows" ]; then \
	      OUT="$${OUT}.exe"; \
	    fi; \
	    if [ "$${GOOS}" = "linux" ]; then \
	      cgo=1; \
	    else \
	      cgo=0; \
	    fi; \
	    if [ "$$cgo" = "1" ] && [ "$${GOOS}" != "$$host_goos" -o "$${GOARCH}" != "$$host_goarch" ]; then \
	      echo "  ↷ skip $${GOOS}/$${GOARCH} (CGO cross-compile not configured)"; \
	      continue; \
	    fi; \
	    echo "  → $${OUT} (CGO_ENABLED=$$cgo)"; \
	    CGO_ENABLED=$$cgo GOOS=$${GOOS} GOARCH=$${GOARCH} \
	      go build $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o "$${OUT}" $(ENTRY); \
	  done; \
	done

# Human-friendly version print
version:
	echo "Version:      $(EFFECTIVE_VERSION)"
	echo "Commit:       $(COMMIT)"
	echo "Build Time:   $(BUILDTIME)"
	go run $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" tools/version.go || true

# Slash commands
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
verify-tag:
	@if [ -z "$$tag" ]; then echo "Usage: make tag tag=vX.Y.Z"; exit 2; fi
	@echo "$$tag" | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' >/dev/null || { echo "❌ Invalid tag: $$tag (want vX.Y.Z)"; exit 2; }

tag: verify-tag
	git tag -a "$$tag" -m "Release $$tag"
	git push origin "$$tag"
	echo "🏷️  Tagged $$tag"

changelog:
	@if [ ! -x scripts/changelog.sh ]; then echo "Missing scripts/changelog.sh"; exit 2; fi
	prev="$$(git describe --tags --abbrev=0 2>/dev/null || true)"; \
	next="$(EFFECTIVE_VERSION)"; \
	echo "📝 Updating CHANGELOG.md for range $$prev..$$next"; \
	./scripts/changelog.sh "$$next" "$$prev" HEAD