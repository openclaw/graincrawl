# AGENTS.md

## Purpose

`graincrawl` is a local-first Granola archive CLI. Preserve the read-only
archive model: inspect local caches, databases, notes, transcripts, and
snapshots without mutating Granola app data.

Reusable archive mechanics belong in `crawlkit`. Keep Granola-specific parsing,
metadata, auth discovery, and CLI behavior in this repository.

## Development Rules

- Do not add Granola mutation endpoints.
- Do not write to Granola app data or real user archive stores.
- Use temp directories and temp SQLite databases in tests.
- Do not print tokens, refresh tokens, note bodies, transcript text, emails, or
  decrypted key material from diagnostics.
- Keep CLI output explicit about partial coverage, missing caches, and
  unavailable local state.

## Validation

Run before handoff:

```bash
GOWORK=off go mod tidy
git diff --exit-code -- go.mod go.sum
GOWORK=off go vet ./...
GOWORK=off go test -count=1 ./...
```
