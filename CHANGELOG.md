# Changelog

## Unreleased

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
