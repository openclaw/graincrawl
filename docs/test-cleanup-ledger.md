# Test cleanup ledger

## CLI flag-like arguments

- Removed an unrelated seeded note and its database setup from
  `TestAppPreservesFlagLikeCommandArguments`. The SQL selects a literal and the
  JSON search asserts its query; neither uses that note.
- Strengthened `TestAppPreservesTrailingJSONSearchArgument`: seed a note that
  actually matches the literal `--json`, then assert the matching text row.
  Previously only a nil error was checked, allowing an empty result or wrong
  output format to pass.
- Retained parser tables and command-level tests: they cover different layers.
  Kept distinct archive, encryption, pagination, and publication regressions;
  no assertion-free tests or standalone unused fixture files were found in
  the initial inventory of 25 Go test files.

Validation: focused CLI tests, full repository checks, isolated CLI smoke,
credential-free snapshot packaging, and structured autoreview before landing.
These command-level tests use temporary archives, not live Granola data.
The optional live API contract test remains opt-in; no credentials or releases
are required for this cleanup.

Remaining: complete the semantic audit of the other packages and crawl apps.

## CLI control contracts

- Consolidated the exact duplicate metadata invocation and trailing-JSON status
  fixture into the existing temporary-config command test.
- Decode success envelopes, launcher metadata, both status flag positions, and
  empty TUI rows instead of accepting JSON-looking substrings. Require the
  desktop import command's arguments and flags, app identity, selected config,
  and the empty archive's note count.
- Keep the seeded TUI detail test and source-specific security tests. Isolate
  the touched command test's home and XDG paths as well as its archive/profile.

No CLI behavior, Granola data, or release artifacts changed.

## Plaintext desktop-cache source selection

Consolidated two nearly identical profile fixtures into one test with named
explicit desktop-cache and implicit private-API fallback cases. Both routes
still run separately against their own temporary profile and SQLite archive;
only the repeated setup and assertions are shared.

Both cases now require one imported note, the selected desktop-cache source,
and the archived note's exact synthetic title/body/source. The old checks could
pass without proving that the plaintext document reached the archive. Kept the
no-cache migration block, encrypted-only cache, and expired-token cases.

Validation: syncer race tests, complete `make check` including isolated built
CLI smoke and credential-free snapshots, independent review, and PR CI.
No production behavior, live Granola data, credentials, or releases changed.

## HTTP fixture failure paths

- Replace three fatal request assertions in public-API HTTP and ServeMux
  callbacks with nonfatal errors, HTTP 400, and immediate returns. Keep every
  authorization/cursor/transcript check and successful payload unchanged.
- Report shared JSON response-write failures nonfatally and return without
  another response attempt. Callers finish their handlers after the writer.
- Forced authorization, cursor, and transcript-include failures reproduce EOF
  before the change and keep their diagnostics with HTTP 400 afterward. A failed
  writer probe verifies normal return while preserving its error diagnostic.
- Root-test assertions and persisted note/summary/transcript/source-object
  checks remain intact. Validate with publicapi/syncer race tests, full local
  gates, isolated CLI smoke, credential-free snapshots, independent review, and
  PR CI. No live Granola profiles, archives, credentials, APIs, or releases.

## Decoded unlock redaction

- Decode the full unlock JSON envelope and check its success, surface, Keychain,
  cache count and token-summary fields. Keep the existing decrypt-call checks.
- Scan the normalized decoded payload, including unknown fields, for all five
  private fixture values. The old raw-text scan missed Unicode-escaped leaks.
- Five escaped-leak overlays pass the old test and fail the repaired test under
  the race detector. Validate malformed output, normal CLI race tests, full
  local gates, independent review and CI with synthetic temporary profiles only.

## Child tombstone inheritance

- Remove `TestChildUpsertsInheritExistingDocumentTombstone`: the retained
  `TestNoteDeletionStubHydration` already creates a parent tombstone, inserts all
  three child types, and checks their inherited deletion timestamp, source and
  reason before any hydration. Its three source cases repeat those checks after
  hydration and precedence updates.
- Keep migration reconciliation, snapshot import, Unicode search and export
  publication tests; these exercise different boundaries. Disable inheritance
  separately for transcript, panel and source-object rows in temporary overlays
  and require the retained test to reject each fault before and after removal.
- Run store race tests, full `make check`, independent review and platform CI.
  Only synthetic temporary archives and credential-free build artifacts are used;
  no production behavior, Granola data, credentials or releases change.

## Snapshot replacement of shared notes

- Require the existing merge/replace test to find the shared note after explicit
  replacement, restore its snapshot title, and clear the destination's deletion
  timestamp, source and reason. Keep the default merge assertions and legacy
  tombstone reconciliation checks intact.
- The original test passed a generic replacement fault that retained overlapping
  destination notes' payload and tombstones. The repaired test rejects that fault,
  each independently retained field, and omission of previously tombstoned notes
  under the race detector; the healthy implementation passes.
- Validate with portable race tests, full `make check`, independent review and
  platform CI. Only synthetic temporary archives and credential-free build
  artifacts are used; no production behavior, Granola data, credentials or
  releases change.

## Migrated source support presence

- Require the migrated-profile test to observe all three local source identities,
  plus both private-API and desktop-cache entries in its mixed-profile case.
  Keep the existing allowed-state, migration-message and unlock checks.
- The old test passed when private-API, desktop-cache or encrypted-JSON support
  disappeared. It also passed when either mixed-profile source disappeared on
  its own. Temporary omission overlays reproduce those false greens; the repaired
  test rejects every omission under the race detector, while healthy code passes.
- Validate with security race tests, full `make check`, independent review and
  platform CI. Only synthetic temporary profiles and credential-free build
  artifacts are used; no production behavior, live Granola data, credentials or
  releases change.
