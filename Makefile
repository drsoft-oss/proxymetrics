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

release: ## Tag the next semver from commits and push to GitHub (triggers release CI). Use VERSION=vX.Y.Z to override.
	@if [ -n "$$(git status --porcelain)" ]; then \
	  echo "error: working tree is dirty. Commit or stash before releasing." >&2; \
	  exit 1; \
	fi; \
	branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$branch" != "main" ]; then \
	  echo "error: must be on main branch (currently on $$branch)" >&2; \
	  exit 1; \
	fi; \
	git fetch --quiet origin main; \
	if [ -n "$$(git rev-list HEAD..origin/main)" ]; then \
	  echo "error: local main is behind origin/main. Pull first." >&2; \
	  exit 1; \
	fi; \
	current=$$(git tag --list 'v*.*.*' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | sort -V | tail -n 1); \
	if [ -n "$${VERSION:-}" ]; then \
	  next="$$VERSION"; \
	  bump_kind="forced"; \
	  if ! echo "$$next" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
	    echo "error: VERSION=$$next must match v<major>.<minor>.<patch>" >&2; \
	    exit 1; \
	  fi; \
	else \
	  if ! command -v svu >/dev/null 2>&1; then \
	    echo "error: svu not found — install with: brew install caarlos0/tap/svu" >&2; \
	    exit 1; \
	  fi; \
	  if [ -z "$$current" ]; then \
	    next="v0.1.0"; \
	    bump_kind="first release"; \
	  else \
	    next=$$(svu next); \
	    if [ "$$next" = "$$current" ]; then \
	      echo "error: no release-worthy commits since $$current (only chore/docs/ci/...). Use VERSION= to force." >&2; \
	      exit 1; \
	    fi; \
	    bump_kind="bumped from $$current"; \
	  fi; \
	fi; \
	if git rev-parse --verify --quiet "$$next" >/dev/null; then \
	  echo "error: tag $$next already exists locally. Delete it or pass VERSION=" >&2; \
	  exit 1; \
	fi; \
	current_display=$${current:-none}; \
	echo "Current tag : $$current_display"; \
	echo "Next tag    : $$next  ($$bump_kind)"; \
	echo "Commits in this release:"; \
	if [ -z "$$current" ]; then \
	  git log --oneline; \
	else \
	  git log --oneline "$$current"..HEAD; \
	fi; \
	printf "Proceed? [y/N] "; \
	read -r reply </dev/tty || { echo "error: no interactive terminal — aborting." >&2; exit 1; }; \
	case "$$reply" in y|Y) ;; *) echo "aborted."; exit 1;; esac; \
	git tag -a "$$next" -m "Release $$next"; \
	git push origin main; \
	git push origin "$$next"; \
	echo "released $$next"

.PHONY: help build-ui build test test-integration test-ui bench lint fmt tidy clean dev dev-go dev-ui release
