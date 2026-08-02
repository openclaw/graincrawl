.DEFAULT_GOAL := help

VERSION ?= 0.0.0-dev
TAG ?=
ARCHIVES ?=
GORELEASER := GOWORK=off go run github.com/goreleaser/goreleaser/v2@v2.17.1
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/openclaw/graincrawl/internal/buildinfo.Version=$(VERSION) \
	-X github.com/openclaw/graincrawl/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/openclaw/graincrawl/internal/buildinfo.Date=$(DATE)

.PHONY: help build test run fmt vet tidy deps lint check smoke snapshot release-snapshot release release-artifacts verify-release clean

help: ## Print available targets.
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the CLI into bin/graincrawl.
	GOWORK=off go build -trimpath -ldflags "$(LDFLAGS)" -o bin/graincrawl ./cmd/graincrawl

test: ## Run the full test suite.
	GOWORK=off go test -count=1 ./...

run: ## Run the CLI help surface.
	GOWORK=off go run ./cmd/graincrawl --help

fmt: ## Check Go source formatting.
	@set -e; changed="$$(gofmt -l .)"; \
	if [ -n "$$changed" ]; then printf 'gofmt wants changes in:\n%s\n' "$$changed"; exit 1; fi

vet: ## Run go vet.
	GOWORK=off go vet ./...

tidy: ## Tidy module metadata.
	GOWORK=off go mod tidy

deps: ## Verify module metadata and known vulnerabilities.
	GOWORK=off go mod verify
	$(MAKE) tidy
	git diff --exit-code -- go.mod go.sum
	GOWORK=off go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...

lint: vet ## Run static analysis enforced by CI.
	@output_file="$$(mktemp)"; trap 'rm -f "$$output_file"' EXIT; \
	if ! GOWORK=off go run golang.org/x/tools/cmd/deadcode@v0.48.0 -test ./... > "$$output_file"; then \
		cat "$$output_file"; exit 1; \
	fi; \
	if [ -s "$$output_file" ]; then cat "$$output_file"; exit 1; fi

check: ## Run every local gate enforced by CI.
	$(MAKE) deps
	$(MAKE) fmt
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) smoke
	$(MAKE) snapshot

smoke: build ## Smoke-test the CLI control surface in isolated directories.
	set -eu; \
	tmp="$$(mktemp -d)"; \
	trap 'rm -rf "$$tmp"' EXIT; \
	cfg="$$tmp/config.toml"; \
	db="$$tmp/graincrawl.db"; \
	mkdir -p "$$tmp/home" "$$tmp/xdg-config" "$$tmp/xdg-cache" "$$tmp/xdg-state"; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" GRAINCRAWL_DB_PATH="$$db" ./bin/graincrawl --config "$$cfg" init --json; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" ./bin/graincrawl --config "$$cfg" metadata --json; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" ./bin/graincrawl --config "$$cfg" status --json; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" ./bin/graincrawl --config "$$cfg" tui --json; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" ./bin/graincrawl --config "$$cfg" snapshot create --out "$$tmp/snapshot" --json

snapshot: ## Build credential-free snapshot artifacts without publishing.
	$(GORELEASER) release --snapshot --clean --skip=publish

release-snapshot: snapshot ## Alias for snapshot.

release: ## Refuse local publication and print the official CI command.
	@echo "release is CI-only; run:" >&2
	@echo "gh workflow run release-unified.yml --repo openclaw/graincrawl -f version=X.Y.Z" >&2
	@false

release-artifacts: release ## Alias for release.

verify-release: ## Verify legacy macOS archives for TAG=vX.Y.Z ARCHIVES='path ...'.
	@test -n "$(TAG)" && test -n "$(ARCHIVES)" || (echo "usage: make verify-release TAG=v0.3.5 ARCHIVES='dist/graincrawl_0.3.5_darwin_*.tar.gz'" >&2; exit 2)
	./scripts/verify-graincrawl-release.sh "$(TAG)" $(ARCHIVES)

clean: ## Remove local build and snapshot artifacts.
	rm -rf bin dist
