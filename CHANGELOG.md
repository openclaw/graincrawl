# Changelog

## v0.3.7 - 2026-08-02

- Fall back to a readable desktop cache when implicit private API sync has no local credentials, while keeping explicit private API failures path-free.
- Honor trailing `--json` for structured subcommands and searches with an
  explicit query, while preserving literal `--json` arguments and free-form SQL.
- Restore Crabbox validation with a root volume that fits the current AWS
  developer image.
- Update CrawlKit to v0.14.5 and pick up the corrected `go-colorful` v1.4.1
  terminal color conversion.
- Avoid `SQLITE_BUSY` failures when concurrent commands open an archive whose
  schema is already current.

## v0.3.6 - 2026-08-01

- Standardize maintainer Make targets, keep local release commands fail-closed on unified CI, and require notarization when verifying legacy macOS archives.
- Refresh terminal width, syscall, text, and SQLite runtime dependencies.
- Update CrawlKit, SQLite, vulnerability/dead-code tooling, and GitHub Actions dependencies.

## v0.3.5 - 2026-07-26

- Re-release v0.3.4's content through the official signed and notarized release pipeline; v0.3.4's macOS archives were signed but not notarized.

## v0.3.4 - 2026-07-26

- Detect Granola 7.427+'s app-scoped Keychain migration, fail local sync and unlock with honest diagnostics, and return a nonzero status instead of silently importing nothing. Thanks @nwoonet.

## v0.3.3 - 2026-07-18

### Highlights

- Preserve explicit Granola deletions as durable tombstones across notes, transcripts, panels, and source objects.

### Snapshots

- Make portable snapshot imports merge by default, retaining destination-only records and local conflict winners; keep `--replace` for explicit exact restores.

### Dependencies

- Update CrawlKit to v0.14.3 and refresh its SQLite runtime dependencies.

### Release engineering

- Ship the first notarized macOS release through the unified OpenClaw pipeline, with `.deb` and `.rpm` packages bound into the verified asset inventory.

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
