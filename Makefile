.PHONY: test vet tidy check build run smoke release-snapshot clean

VERSION ?= 0.0.0-dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/openclaw/graincrawl/internal/buildinfo.Version=$(VERSION) \
	-X github.com/openclaw/graincrawl/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/openclaw/graincrawl/internal/buildinfo.Date=$(DATE)

test:
	GOWORK=off go test -count=1 ./...

vet:
	GOWORK=off go vet ./...

tidy:
	GOWORK=off go mod tidy

check: tidy vet test
	git diff --exit-code -- go.mod go.sum

build:
	GOWORK=off go build -trimpath -ldflags "$(LDFLAGS)" -o bin/graincrawl ./cmd/graincrawl

run:
	GOWORK=off go run ./cmd/graincrawl --help

smoke: build
	tmp="$$(mktemp -d)"; \
	cfg="$$tmp/config.toml"; \
	db="$$tmp/graincrawl.db"; \
	mkdir -p "$$tmp/home" "$$tmp/xdg-config" "$$tmp/xdg-cache" "$$tmp/xdg-state"; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" GRAINCRAWL_DB_PATH="$$db" ./bin/graincrawl --config "$$cfg" init --json; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" ./bin/graincrawl --config "$$cfg" metadata --json; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" ./bin/graincrawl --config "$$cfg" status --json; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" ./bin/graincrawl --config "$$cfg" tui --json; \
	env HOME="$$tmp/home" XDG_CONFIG_HOME="$$tmp/xdg-config" XDG_CACHE_HOME="$$tmp/xdg-cache" XDG_STATE_HOME="$$tmp/xdg-state" ./bin/graincrawl --config "$$cfg" snapshot create --out "$$tmp/snapshot" --json

release-snapshot:
	GOWORK=off goreleaser release --snapshot --clean --skip=publish

clean:
	rm -rf bin dist
