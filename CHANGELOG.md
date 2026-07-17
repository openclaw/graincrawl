# Changelog

## Unreleased

- Preserve explicit Granola deletions as note, transcript, panel, and source-object tombstones, and make portable snapshot imports merge by default with explicit `--replace` restores.

## v0.3.2 - 2026-07-09

- Add Developer ID signing and pre-publish verification for official macOS archives.
- Update CrawlKit to v0.13.4.
- Update Go to 1.26.5 to remove GO-2026-5856 from the `crypto/tls` dependency graph.

## v0.3.1 - 2026-06-19

- Update `golang.org/x/sys` to remove GO-2026-5024 from the Windows dependency graph.
- Refresh terminal rendering, Unicode text, and SQLite runtime dependencies.
- Treat `%`, `_`, and backslashes literally in archive search and update CrawlKit.

## v0.3.0 - 2026-06-11

- Add explicit in-memory encrypted Granola JSON unlock for cache import and private API authentication.
- Add a clear diagnostic when Granola desktop exposes encrypted-only state, thanks @elijahmuraoka.

## v0.2.1 - 2026-05-18

- Accept plaintext Granola cache version 8 for desktop-cache sync, including
  empty local caches with no recordings or documents.
- Expose an explicit desktop-cache import command in crawlkit metadata.

## v0.2.0 - 2026-05-18

- Preserve explicit `sync --no-transcripts` and `--no-panels` opt-outs when
  config defaults enable those archive sections.
- Bump routine Go module and GitHub Actions dependencies.
- Add `graincrawl check-update` and passive release notices backed by
  `crawlkit/releasecheck`.
- Move documented Homebrew installs to `openclaw/tap`.
- Add a repo-local `graincrawl` agent skill for local archive, freshness,
  source-unlock, and verification workflows.
- Add `graincrawl sql` for read-only local archive queries and document
  agent-friendly SQL examples in the repo-local skill.
- Add the project banner image and README placement.

## v0.1.0

- Scaffold `graincrawl` as a local-first Granola archive CLI.
- Add Granola private API and desktop cache sync adapters.
- Add SQLite archive commands for notes, transcripts, panels, people,
  workspaces, search, runs, Markdown export, snapshots, and TUI browsing.
- Add CI, release, Homebrew tap, and package distribution scaffolding.
