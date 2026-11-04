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
