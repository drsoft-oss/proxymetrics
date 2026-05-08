SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help

GO ?= go
PNPM ?= pnpm
BIN := bin/proxymetrics

UI_SRC := ui
UI_DIST_SRC := ui/dist
UI_DIST_DST := internal/ui/ui-dist

LDFLAGS := -X github.com/drsoft-oss/proxymetrics/internal/cli.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) \
           -X github.com/drsoft-oss/proxymetrics/internal/cli.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo none) \
           -X github.com/drsoft-oss/proxymetrics/internal/cli.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  %-22s %s\n", $$1, $$2}'

build-ui: ## Build the Vite SPA into internal/ui/ui-dist/
	cd $(UI_SRC) && $(PNPM) install --frozen-lockfile
	cd $(UI_SRC) && $(PNPM) run build
	rm -rf $(UI_DIST_DST)
	mkdir -p $(UI_DIST_DST)
	cp -R $(UI_DIST_SRC)/. $(UI_DIST_DST)/

build: build-ui ## Build the binary with embedded UI
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/proxymetrics

dev: ## Hot-reload dev (Vite + air on Go); requires make parallelism
	@echo "Starting Vite dev (:5173) and air (Go)..."
	@$(MAKE) -j 2 dev-go dev-ui

dev-go:
	air -c .air.toml

dev-ui:
	cd $(UI_SRC) && $(PNPM) run dev

test: ## Run Go unit tests
	$(GO) test ./...

test-integration: ## Run Go integration tests
	$(GO) test -tags=integration ./test/integration/...

test-ui: ## Run frontend tests
	cd $(UI_SRC) && $(PNPM) run test

bench: ## Run benchmarks on the proxy hot path
	$(GO) test -tags=integration -bench=. -benchmem ./test/integration/...

lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format
	gofmt -w .

tidy: ## Tidy modules
	$(GO) mod tidy

clean: ## Remove build artifacts (./data is left alone)
	rm -rf bin/
	rm -rf $(UI_SRC)/dist
	rm -rf $(UI_SRC)/.vite
	@find $(UI_DIST_DST) -mindepth 1 ! -name 'index.html' -delete

.PHONY: help build-ui build test test-integration test-ui bench lint fmt tidy clean dev dev-go dev-ui
