---
name: graincrawl
description: Use for local Granola archive search, sync freshness, notes/transcripts/panels, Markdown export, snapshots, Keychain-safe source debugging, and Graincrawl repo/release work.
---

# Graincrawl

Use local archive data first for Granola questions. Browse or hit Granola
private/API surfaces only when the archive is stale, missing the requested
scope, or the user asks for current external context.

## Sources

- DB: configured `paths.db_path`; override with `GRAINCRAWL_DB_PATH`
- Config: default crawlkit config path; override with `GRAINCRAWL_CONFIG`
- Granola profile: `GRAINCRAWL_GRANOLA_PROFILE` or the configured profile path
- Repo: `~/GIT/_Perso/graincrawl`
- Preferred CLI: `graincrawl`; fallback to `go run ./cmd/graincrawl` from the repo if the installed binary is stale

## Freshness

For recent/current questions, check freshness before analysis:

```bash
sqlite3 "$(graincrawl status --json | jq -r '.database_path')" \
  "select coalesce(max(completed_at), '') from sync_runs where status = 'ok';"
```

Routine refresh:

```bash
graincrawl doctor
graincrawl sync --source private-api
```

Desktop-cache refresh:

```bash
graincrawl sync --source desktop-cache
```

Use encrypted JSON, OPFS, Electron safeStorage, or Keychain-backed paths only
after an explicit unlock check.

## Query Workflow

1. Resolve scope: note, transcript, panel, person, workspace, keyword, or date range.
2. Check freshness for recent/current requests.
3. Use CLI for normal reads; use read-only SQL for precise counts/rankings.
4. Report absolute date spans, note titles, source gaps, and transcript/panel availability.

Common commands:

```bash
graincrawl search "query"
graincrawl notes --json
graincrawl note get <id>
graincrawl transcripts get <id>
graincrawl panels get <id>
```

## SQL

`graincrawl` does not currently expose a first-class `sql` command. For exact
local archive counts or rankings, discover the configured DB from status and
open it read-only with SQLite.

Useful examples:

```bash
sqlite3 -readonly "$(graincrawl status --json | jq -r '.database_path')" \
  "select count(*) as notes from notes;"
sqlite3 -readonly "$(graincrawl status --json | jq -r '.database_path')" \
  "select source, count(*) as notes from notes group by source order by notes desc;"
sqlite3 -readonly "$(graincrawl status --json | jq -r '.database_path')" \
  "select title, updated_at from notes order by updated_at desc limit 20;"
```

Do not run mutating SQL against the archive.

When the installed CLI lacks a new feature, build or run from
`~/GIT/_Perso/graincrawl` before concluding the feature is missing.

## Granola Boundaries

Ordinary `doctor`, `status`, `notes`, `search`, `export`, and `tui` commands
must not surprise-prompt Keychain. Prefer `graincrawl secrets --json` before
debugging unlock issues and `graincrawl unlock --json` before enabling
encrypted sources.

## Verification

For repo edits, prefer existing Go gates:

```bash
GOWORK=off go test ./...
```

Then run targeted CLI smoke for the touched surface, for example:

```bash
graincrawl doctor --json
graincrawl status --json
graincrawl search "test"
```
