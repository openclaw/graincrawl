<img src="docs/graincrawl_banner.jpg" alt="graincrawl banner"/>

# 🌾 graincrawl

`graincrawl` archives Granola notes, transcripts, panels, people, workspaces,
and sync metadata into a private local SQLite store.

It is local-first, read-only against Granola, and shaped like the other crawl
apps: stable JSON output, Markdown export, crawlkit snapshots, and a terminal
browser over the archived SQLite data.

## Current Scope

- sync notes from Granola's private desktop API session
- import plaintext `cache-v6.json` as an offline fallback
- retain transcripts, panels, people, workspaces, and raw source payloads
- export notes to Markdown
- browse archived notes with the shared crawlkit TUI
- create/import portable crawlkit snapshots
- keep encrypted JSON, OPFS, Keychain, and helper paths behind explicit unlock
  surfaces

## Install

```bash
brew install openclaw/tap/graincrawl
```

From source:

```bash
go install github.com/openclaw/graincrawl/cmd/graincrawl@latest
```

## Quick Start

```bash
graincrawl init
graincrawl doctor
graincrawl sync --source private-api
graincrawl status
graincrawl notes
graincrawl tui
```

Use JSON for automation:

```bash
graincrawl doctor --json
graincrawl status --json
graincrawl notes --json
graincrawl tui --json
```

## Commands

```bash
graincrawl version
graincrawl init
graincrawl doctor
graincrawl check-update
graincrawl metadata
graincrawl status
graincrawl sync --source private-api
graincrawl sync --source desktop-cache
graincrawl runs
graincrawl notes
graincrawl search "decision"
graincrawl --json sql "select count(*) as notes from notes;"
graincrawl note get <id>
graincrawl transcripts get <id>
graincrawl panels get <id>
graincrawl people
graincrawl workspaces
graincrawl sources
graincrawl unlock
graincrawl secrets
graincrawl export markdown --out ./granola-notes
graincrawl snapshot create --out ./graincrawl-snapshot
graincrawl import ./graincrawl-snapshot
graincrawl tui
graincrawl completion zsh
```

## Shared crawlkit surfaces

`graincrawl metadata` exposes crawlkit control metadata for scripts and status
dashboards.

`graincrawl snapshot create` and `graincrawl import` use `crawlkit/snapshot` so
archives can move between machines without touching the live Granola profile.

`graincrawl tui` uses `crawlkit/tui` over archived notes. The detail pane is
fed from SQLite, including note text, transcript chunks, panels, and retained
source metadata.

## Distribution

Releases use GoReleaser for GitHub release assets and Linux packages, plus a
source-built Homebrew formula in `openclaw/tap`.

`graincrawl` checks for new GitHub releases at most once every 24 hours during
interactive CLI use and prints a short upgrade hint when a newer version is
available. The check is skipped for JSON output, CI, non-terminal stderr, and
development builds. Run `graincrawl check-update` to check explicitly, or
disable passive checks with `CRAWLKIT_NO_UPDATE_CHECK=1` or
`GRAINCRAWL_NO_UPDATE_CHECK=1`.

See [docs/distribution.md](docs/distribution.md).

## Safety Model

`graincrawl` never writes to Granola app data. It reads from Granola's private
read endpoints or local files and stores its own archive under the configured
graincrawl paths.

Encrypted JSON, OPFS, Electron safeStorage, and macOS Keychain paths require
explicit unlock flow. Ordinary `doctor`, `status`, `notes`, `export`, and `tui`
commands must not surprise-prompt Keychain.

See [docs/security.md](docs/security.md).
