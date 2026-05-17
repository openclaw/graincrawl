# graincrawl spec

## intent

Build `graincrawl` as the local-first crawler/archive tool for Granola notes,
transcripts, summaries, panels, and meeting metadata.

The product should feel like the other crawl apps:

- local SQLite archive
- deterministic sync runs
- readable JSON command surfaces
- optional terminal browser
- portable snapshot/export support
- strict handling of live app stores and secrets

The important difference is that Granola has multiple overlapping access
surfaces: a private API, a public Enterprise API, plaintext desktop cache,
Electron safeStorage encrypted JSON, OPFS SQLCipher state, and a bundled
companion CLI. `graincrawl` should treat those as source adapters with clear
trust and support boundaries, not one mushy importer.

## current findings

Verified against local Granola `7.162.5` on macOS.

Private API:

- `supabase.json` contains `workos_tokens.access_token`, `refresh_token`,
  `obtained_at`, `expires_in`, and user/workspace metadata.
- `POST https://api.granola.ai/v2/get-documents` works with no body or `{}`.
- `POST https://api.granola.ai/v2/get-documents` with
  `{ "include_shared_with_me": true }` returns `docs`, `deleted`, and `shared`.
- `POST https://api.granola.ai/v1/get-documents-batch` works with
  `{ "document_ids": [...] }`.
- `POST https://api.granola.ai/v1/get-document-transcript` works with
  `{ "document_id": "..." }`.
- `documentId` and `id` are not accepted by `get-document-transcript`.
- `POST https://api.granola.ai/v1/get-document-panels` works with
  `{ "document_id": "..." }`.

Local cache:

- `~/Library/Application Support/Granola/cache-v6.json` is plaintext on the
  tested install.
- It includes `documents`, `transcripts`, `meetingsMetadata`, notes markdown,
  notes plain text, people, calendar data, and transcript chunks.
- Feature flags currently show `encrypted_cache_storage` and
  `encrypted_supabase_storage` as disabled on the tested install.

Encrypted JSON:

- `storage.dek` is wrapped by Electron `safeStorage`.
- JSON encrypted payloads use AES-256-GCM with:
  - 12 byte IV prefix
  - encrypted payload body
  - 16 byte auth tag suffix
- A generic Electron process could not unwrap `storage.dek`.
- An Electron helper with app name set to `Granola` could unwrap `storage.dek`
  and decrypt `user-preferences.json.enc`.
- On the tested install:
  - `cache-v6.json` has no `.enc` sibling
  - `supabase.json` has no `.enc` sibling
  - `user-preferences.json.enc` decrypts successfully

OPFS:

- Granola stores OPFS files under Chromium `File System` origins.
- Active UI origin is `app://ui`.
- The OPFS SQLite database is opened as:
  `file:granola.db?vfs=multipleciphers-opfs-sahpool`.
- It uses SQLCipher via `PRAGMA cipher = 'sqlcipher'` and `PRAGMA legacy = 4`.
- The raw SQLite key is stored indirectly in `IndexedDB` database
  `granola-encryption`, object store `keys`:
  - `kek`: non-extractable WebCrypto AES-GCM key
  - `dek`: wrapped key plus IV
- Running an Electron probe under `app://ui` against a copied profile recovered
  the raw 32 byte SQLCipher key and opened the OPFS DB.
- Verified tables:
  - `system`
  - `ydocs`
  - `document_panels`
  - `calendar_events`
  - `pre_meeting_briefs`
  - `document_panels_fts*`
- Verified row counts in the copied profile:
  - `ydocs`: 27
  - `document_panels`: 1
  - `calendar_events`: 14
  - `pre_meeting_briefs`: 0
- `ydocs` prefixes were `notes` and `summary`.

Companion CLI:

- Granola ships a bundled binary:
  `/Applications/Granola.app/Contents/Resources/bin/granola`.
- It exposes:
  - `granola notes list`
  - `granola notes get`
  - `granola notes transcript get`
- The CLI talks to a desktop app socket and metadata file:
  - metadata: app data `Granola/companion-cli/companion-cli.json`
  - socket: temp `granola-companion-cli.sock`
- The desktop server is gated by Granola feature flag `companion_cli`.
- The tested local app did not have the companion server active.

## product stance

Default to the private API because it is the richest verified source and does
not require local DB surgery.

Use local files for fallback, diagnostics, and offline recovery.

Treat Keychain, safeStorage, OPFS, and WebCrypto paths as explicit unlock
operations. No surprise Keychain prompts. No background helper. No raw key or
token persistence unless the operator opts in.

`graincrawl` should be a sober archive tool, not a stealth extractor. The
operator should always know which trust boundary is being crossed.

## non-goals

- no hosted service
- no browser UI in v1
- no mutation endpoints
- no writes to Granola app data
- no writes to `supabase.json`, `cache-v6.json`, OPFS files, IndexedDB files, or
  companion CLI metadata
- no automatic token refresh that mutates Granola state
- no hidden Keychain/safeStorage access during ordinary `status` or `notes`
  commands
- no vendoring Granola app bundle assets
- no crawlkit provider-specific code
- no transcript/audio recording or live meeting capture

## repository identity

- repo: `github.com/openclaw/graincrawl`
- module: `github.com/openclaw/graincrawl`
- binary: `graincrawl`
- language: Go for the CLI/archive
- helper language: minimal Electron/Node helper for macOS local encrypted
  surfaces
- shared library: `github.com/openclaw/crawlkit`
- default local app dir: `~/.config/graincrawl`

## source adapters

### `private-api`

Primary source.

Reads Granola's private API using the desktop WorkOS token from
`supabase.json`.

Capabilities:

- list notes
- hydrate notes by id
- hydrate transcript chunks
- hydrate panels/summaries
- include shared notes when requested
- eventually hydrate people, workspaces, document lists, attachments, and
  calendars if needed

Known read endpoints:

- `v2/get-documents`
- `v1/get-documents-batch`
- `v1/get-document-transcript`
- `v1/get-document-panels`
- `v1/get-document-lists`
- `v1/get-document-lists-metadata`
- `v1/get-people`
- `v1/get-workspaces`

Default request headers:

- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`
- `X-Client-Version: <detected Granola version>`
- `X-Granola-Platform: darwin`
- `X-Granola-Workspace-Id: <active workspace id>` when available

Token behavior:

- Read `supabase.json` plaintext when available.
- If `supabase.json.enc` exists and is newer, require explicit unlock through
  the helper.
- If token is still valid, use it.
- If token is expired, default behavior is to fail with a clear message:
  `open Granola or run graincrawl unlock --allow-refresh`.
- Optional later behavior: refresh into `graincrawl`'s own credential store,
  not Granola's `supabase.json`.
- Never print tokens, refresh tokens, auth headers, or decoded JWTs.

API stability:

- This is private and unsupported.
- Every request wrapper should centralize endpoint names and payload shapes.
- `doctor api --probe` should make one low-cost read call and report only shape,
  status, and counts.
- Endpoint drift should be a normal diagnostic, not a panic.

### `public-api`

Optional official adapter.

Use this only when the operator supplies an official Granola public API key.
Current public docs indicate Enterprise access and a public notes endpoint.

Capabilities are expected to be narrower than private API. The adapter should
not pretend it can provide transcripts or panels until proven.

Configuration:

- `GRANOLA_PUBLIC_API_KEY`
- `graincrawl config set public_api_key`
- optional keychain storage through `graincrawl secrets set public-api-key`

Commands should make source limitations obvious:

- `graincrawl sync --source public-api`
- `graincrawl doctor public-api`

### `desktop-cache`

Fallback and offline source.

Reads:

- `~/Library/Application Support/Granola/cache-v6.json`
- optionally configured profile path

Capabilities:

- notes and note metadata
- transcript chunks when present
- meeting metadata
- calendar event blobs
- feature flag/state diagnostics

Rules:

- Copy live files into a temp dir before parsing.
- Refuse files over a configured max size unless `--allow-large-cache`.
- Never write to the Granola profile.
- Validate top-level shape: `cache.version == 6`.
- Emit warnings for unknown major cache versions.

Encrypted cache behavior:

- If `cache-v6.json.enc` exists and is newer than plaintext, require
  `graincrawl unlock --surface encrypted-json`.
- Do not silently fall back to stale plaintext unless `--allow-stale-plaintext`
  is explicit.

### `encrypted-json`

Mac helper source for Electron safeStorage-backed JSON files.

Inputs:

- `storage.dek`
- `cache-v6.json.enc`
- `supabase.json.enc`
- `user-preferences.json.enc`

Process:

1. Operator runs an explicit command:
   - `graincrawl unlock encrypted-json`
   - or `graincrawl sync --source desktop-cache --unlock encrypted-json`
2. CLI starts a short-lived helper.
3. Helper sets Electron app name to `Granola`.
4. Helper calls `safeStorage.decryptString(storage.dek)`.
5. Helper AES-GCM decrypts requested `.enc` files.
6. Helper returns parsed JSON or a temp plaintext stream over stdout/pipe.
7. CLI imports data.
8. Helper exits.

Security rules:

- No long-running helper.
- No raw DEK logs.
- No raw token logs.
- No decrypted file written to disk by default.
- If a temp file is necessary for debugging, require `--debug-keep-temp` and
  write under a mode `0700` temp dir.
- Keychain prompts must only happen inside explicit unlock commands.
- CLI output must say which surface triggered Keychain/safeStorage access.

Potential Keychain friction:

- macOS may bind the safeStorage secret to app name/service/account identity.
- A helper may trigger a prompt that appears to reference Granola.
- If access is denied, `graincrawl` should degrade cleanly:
  `encrypted storage locked; rerun with --unlock or use private-api`.
- If helper identity breaks across Electron versions, fallback to asking the
  operator to open Granola and use API/companion CLI.

### `opfs`

Advanced local source.

This should not be v1 default. It is real, but it crosses too many fragile
boundaries to use silently.

Use cases:

- recover panel/calendar/Yjs state when private API is unavailable
- compare OPFS panel state to API/cache state
- future forensic/debug command

Process:

1. Copy Granola `IndexedDB` and `File System` directories into a temp profile.
2. Start a short-lived Electron helper with:
   - userData set to the temp profile
   - `app://ui` registered as secure/standard
   - no network requirement
3. Under `app://ui`, read IndexedDB `granola-encryption/keys`.
4. Use WebCrypto to unwrap the SQLCipher key.
5. Use Granola's installed app assets, not vendored assets, to load the same
   SQLite WASM/OPFS VFS path.
6. Open `granola.db` read-only when possible.
7. Query only whitelisted tables.
8. Return normalized rows to the Go CLI.
9. Delete temp profile.

Rules:

- Never operate directly on the live OPFS profile.
- Never persist the SQLCipher key.
- Never print the SQLCipher key.
- Refuse if Granola version cannot be detected.
- Version-gate the helper because minified export names may drift.
- Prefer read-only/select queries.
- Keep OPFS import behind `--source opfs` or `--unlock opfs`.

Whitelisted tables for initial support:

- `system`
- `document_panels`
- `calendar_events`
- `pre_meeting_briefs`
- `ydocs`

Initial OPFS import should store raw source rows plus enough normalized fields
for browsing. Do not attempt full Yjs reconstruction in v1 unless notes/panels
need it and the representation is stable.

### `companion-cli`

Potentially cleanest local source when Granola enables it.

The bundled CLI already exposes the product nouns we need:

- notes list
- notes get
- notes transcript get

`graincrawl` should detect it in `doctor` and prefer it only when:

- bundled binary exists
- companion metadata exists
- socket is reachable
- auth handshake succeeds

Do not depend on it for v1 because feature flag `companion_cli` may be off.

## command contract

Initial public commands:

- `version`
- `init`
- `doctor`
- `status`
- `sync`
- `refresh`
- `notes`
- `note`
- `transcripts`
- `panels`
- `people`
- `workspaces`
- `runs`
- `sources`
- `unlock`
- `secrets`
- `export`
- `snapshot`
- `import`
- `tui`
- `completion`

Suggested command details:

```text
graincrawl doctor
graincrawl doctor granola
graincrawl doctor api --probe
graincrawl doctor local --profile ~/Library/Application\ Support/Granola
graincrawl doctor opfs --unlock
graincrawl doctor encrypted-json --unlock

graincrawl sync --source private-api
graincrawl sync --source desktop-cache
graincrawl sync --source opfs --unlock
graincrawl sync --all-sources

graincrawl notes --json
graincrawl note get <id> --json
graincrawl transcripts get <id> --json
graincrawl panels get <id> --json

graincrawl unlock encrypted-json
graincrawl unlock opfs
graincrawl secrets status
graincrawl secrets set public-api-key
graincrawl secrets clear public-api-key

graincrawl export markdown --out ./notes
graincrawl snapshot create
graincrawl snapshot import <path>
graincrawl tui
```

`doctor` output should be brutally useful:

- Granola app installed/version
- profile path exists
- `cache-v6.json` present/version/counts
- `cache-v6.json.enc` present/newer
- `supabase.json` present/plaintext/encrypted
- WorkOS token present/expired without printing it
- active workspace id present without printing user email by default
- companion CLI binary present
- companion server metadata/socket active
- OPFS origins present
- OPFS helper supported for detected Granola version
- keychain/safeStorage access required/not required

## config contract

Default paths:

- config: `~/.config/graincrawl/config.toml`
- database: `~/.config/graincrawl/graincrawl.db`
- cache: `~/.config/graincrawl/cache`
- logs: `~/.config/graincrawl/logs`
- snapshots: `~/.config/graincrawl/snapshots`
- helper temp root: system temp dir, not config dir

Environment variables:

- `GRAINCRAWL_CONFIG`
- `GRAINCRAWL_DB_PATH`
- `GRAINCRAWL_GRANOLA_PROFILE`
- `GRAINCRAWL_SOURCE`
- `GRANOLA_PUBLIC_API_KEY`
- `GRAINCRAWL_ALLOW_PRIVATE_API`
- `GRAINCRAWL_HELPER_TIMEOUT`
- `GRAINCRAWL_NO_KEYCHAIN`

Config fields:

```toml
[granola]
profile_path = ""
app_path = "/Applications/Granola.app"
preferred_source = "private-api"
allow_private_api = true
allow_public_api = false
allow_companion_cli = true
allow_desktop_cache = true
allow_encrypted_json = false
allow_opfs = false

[api]
client_version = "auto"
platform = "darwin"
include_shared_with_me = true
refresh_mode = "never" # never, session, graincrawl-keychain

[sync]
default_limit = 100
include_transcripts = true
include_panels = true
include_calendar_events = true
include_people = true

[security]
redact_logs = true
keychain_prompt_mode = "explicit" # explicit only in v1
persist_helper_keys = false
debug_keep_temp = false
```

## database model

Use `crawlkit/store` for SQLite open/migration hygiene when available.

Core tables:

- `source_objects`
- `notes`
- `note_revisions`
- `transcript_chunks`
- `document_panels`
- `calendar_events`
- `people`
- `workspaces`
- `sync_runs`
- `sync_items`
- `source_state`
- `attachments`
- `export_manifests`

`source_objects`:

- `source`: `private-api`, `public-api`, `desktop-cache`, `encrypted-json`,
  `opfs`, `companion-cli`
- `kind`: provider object kind
- `source_id`: provider id
- `document_id`
- `payload_json`
- `payload_hash`
- `observed_at`

`notes`:

- `id`
- `title`
- `type`
- `status`
- `created_at`
- `updated_at`
- `deleted_at`
- `workspace_id`
- `owner_json`
- `people_json`
- `calendar_event_id`
- `notes_plain`
- `notes_markdown`
- `summary_text`
- `summary_markdown`
- `source`
- `payload_hash`
- `last_seen_at`

`transcript_chunks`:

- `id`
- `document_id`
- `start_timestamp`
- `end_timestamp`
- `source`
- `is_final`
- `speaker_user_id`
- `text`
- `payload_hash`

`document_panels`:

- `id`
- `document_id`
- `title`
- `template_slug`
- `content_plain`
- `content_markdown`
- `content_json`
- `created_at`
- `updated_at`
- `last_viewed_at`
- `ydoc_version`
- `ydoc_cached_at`
- `source`

Indexes:

- notes by `created_at`
- notes by `updated_at`
- notes by `workspace_id`
- transcript chunks by `document_id,start_timestamp`
- panels by `document_id`
- source objects by `source,kind,source_id`
- FTS over title, notes plain/markdown, summary text, panel text, transcript
  text

## sync semantics

Private API sync:

1. Load token source.
2. Check expiry.
3. Fetch documents with `include_shared_with_me`.
4. Upsert note summaries and source payloads.
5. For changed/new notes, call `get-documents-batch`.
6. If enabled, call transcript endpoint per changed/new note.
7. If enabled, call panels endpoint per changed/new note.
8. Record deleted ids from API response.
9. Record sync run counts and endpoint statuses.

Desktop cache sync:

1. Snapshot cache file into temp dir.
2. Parse shape and version.
3. Upsert documents, transcripts, meeting metadata, and source payloads.
4. Mark source as `desktop-cache`.
5. Do not interpret absent objects as deleted unless source has complete list
   semantics.

Encrypted JSON sync:

1. Require explicit unlock.
2. Decrypt target JSON files via helper.
3. Reuse desktop cache or supabase token parser.

OPFS sync:

1. Require explicit unlock.
2. Copy profile state to temp dir.
3. Open OPFS DB through helper.
4. Import whitelisted tables.
5. Normalize document panels/calendar events when practical.
6. Store opaque Yjs state as source payload until a stable reconstruction path
   exists.

Companion CLI sync:

1. Detect companion metadata and socket.
2. Authenticate using metadata token.
3. Use notes list/get/transcript commands.
4. Normalize output using the same note/transcript model.

Conflict behavior:

- Prefer private API for canonical note metadata.
- Prefer private API transcripts over cache transcripts when ids match.
- Prefer latest `updated_at` for notes/panels when source reliability is equal.
- Keep all raw source payloads for audit.
- Show source disagreements in `doctor data` or `graincrawl note diff <id>`.

## keychain and helper process

This is the riskiest part. Make it boring and explicit.

Helper binary:

- name: `graincrawl-granola-helper`
- implementation: small Electron app/script, launched by `graincrawl`
- lifetime: one command
- transport: stdout JSON lines or a local pipe
- timeout: default 30 seconds
- temp dirs: mode `0700`
- cleanup: always remove temp profile copies unless `--debug-keep-temp`

Unlock surfaces:

- `encrypted-json`: Electron safeStorage plus AES-GCM JSON decrypt
- `opfs`: IndexedDB/WebCrypto unwrap plus OPFS SQLCipher open
- future `api-refresh`: optional token refresh into graincrawl-owned keychain

Rules:

- `graincrawl status` and ordinary `graincrawl notes` must not trigger
  Keychain prompts.
- Commands that may prompt must include `unlock` or `--unlock`.
- Before launching helper, print the surface being accessed unless `--json`.
- In JSON mode, report `requires_unlock` instead of prompting unless
  `--unlock` is present.
- Helper returns only requested data, never the raw safeStorage DEK or OPFS
  SQLCipher key.
- Helper stderr is captured and redacted.
- Helper refuses unknown Granola app versions unless `--allow-unknown-version`.
- Helper refuses to read live OPFS directly; it must copy first.

Keychain UX:

- If macOS prompts, the prompt may name Granola or Electron depending on helper
  identity.
- Document this in `doctor encrypted-json --explain`.
- If access is denied, exit non-zero with an actionable message.
- Do not retry in a loop. One prompt attempt per command.

## security and privacy

Default redactions:

- tokens
- refresh tokens
- decoded JWT payloads
- email addresses in non-verbose doctor output
- transcript text in diagnostics
- note body text in diagnostics
- raw Keychain/safeStorage errors that include secret material

File permissions:

- config dir: `0700`
- database: `0600`
- temp dirs: `0700`
- exported snapshots: normal user-readable unless `--private-mode`

Network:

- private API source should only call allowlisted read endpoints.
- no mutation endpoints in v1.
- no telemetry.
- no upload.

Data deletion:

- `graincrawl reset` deletes only graincrawl state.
- Never delete Granola profile files.

## crawlkit boundary

Keep provider-specific code in `graincrawl`.

Use `crawlkit` for:

- config path/runtime dir helpers
- SQLite open/migration hygiene
- output envelopes
- progress logs
- snapshots
- mirror/export support
- TUI row presentation if it fits
- safe local cache snapshot helpers if generic enough

Do not add Granola-specific API clients, schemas, OPFS helpers, Electron
helpers, or token parsing to `crawlkit`.

If OPFS/keychain helper patterns later repeat across multiple crawl apps, move
only the generic process supervisor/temp-copy/redaction mechanics into
`crawlkit`, not Granola logic.

## implementation phases

### phase 0: repo bootstrap

- create `github.com/openclaw/graincrawl`
- add Go module
- add `cmd/graincrawl`
- add README, SPEC, LICENSE, CHANGELOG, CONTRIBUTING
- add `.gitignore`, `.editorconfig`, CI
- wire `version`, help, output modes
- commit scaffold separately

### phase 1: config, store, doctor

- add config defaults
- add SQLite migrations
- add `init`
- add `doctor granola`
- detect app install/version/profile
- detect cache/supabase/encrypted files
- detect OPFS/IndexedDB origins
- detect companion CLI binary/socket metadata
- add safe redacted JSON doctor output

### phase 2: private API sync

- parse `supabase.json`
- parse active workspace id
- build private API client
- implement token expiry checks
- implement `get-documents-v2`
- implement `get-documents-batch`
- implement transcript and panel hydration
- persist notes/transcripts/panels/source payloads
- add sync run accounting
- add tests with fixture JSON only

### phase 3: desktop cache source

- snapshot live cache to temp
- parse `cache-v6.json`
- import documents/transcripts/meeting metadata
- compare cache vs API for a known fixture
- add `sync --source desktop-cache`
- add stale plaintext/encrypted detection

### phase 4: encrypted JSON helper

- add helper scaffold
- implement safeStorage unwrap
- implement AES-256-GCM decrypt
- support decrypting `cache-v6.json.enc` and `supabase.json.enc`
- add `unlock encrypted-json`
- add tests using generated safeStorage-compatible fixtures where possible
- add manual macOS verification notes for real Keychain behavior

### phase 5: OPFS helper

- copy profile dirs to temp
- open helper under `app://ui`
- unwrap OPFS SQLCipher key from IndexedDB/WebCrypto
- open installed Granola OPFS DB path
- query whitelisted tables
- return counts and normalized panels/calendar rows
- add `doctor opfs --unlock`
- add `sync --source opfs --unlock`
- version-gate helper against Granola app version

### phase 6: operator surfaces

- implement note listing/detail/transcript/panel JSON commands
- implement FTS search
- implement Markdown export
- implement snapshot create/import via crawlkit
- implement TUI after JSON contracts settle

### phase 7: hardening

- fixture coverage for API shapes
- fixture coverage for cache-v6 shape
- source disagreement reports
- helper timeout/crash tests
- token redaction tests
- temp cleanup tests
- `go test ./...`
- `go vet ./...`
- `GOWORK=off go test ./...`
- local smoke with temp `HOME`, `XDG_CONFIG_HOME`, and `XDG_CACHE_HOME`

## validation plan

Unit tests:

- config defaults
- migrations
- API response parsers
- cache parser
- transcript ordering
- panel markdown/plain extraction
- source conflict handling
- redaction
- output envelopes

Integration tests:

- private API client with fixture transport
- desktop cache import from temp fixture
- encrypted JSON helper with synthetic encrypted payloads
- OPFS helper behind a build tag or manual macOS test

Manual proof commands:

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
graincrawl doctor --json
graincrawl doctor api --probe --json
graincrawl sync --source private-api --limit 1 --json
graincrawl sync --source desktop-cache --json
graincrawl unlock encrypted-json --json
graincrawl doctor opfs --unlock --json
graincrawl notes --json
graincrawl transcripts get <id> --json
```

All tests and smokes must use temp graincrawl config/db paths. Live Granola
profile reads are allowed only for explicit manual local verification commands,
and those commands must be read-only.

## commit strategy

Use many small semantic commits:

- `chore: scaffold graincrawl module`
- `feat(config): add local runtime paths`
- `feat(store): add initial archive schema`
- `feat(doctor): detect Granola install and profile`
- `feat(api): add private Granola read client`
- `feat(sync): import notes from private API`
- `feat(sync): import transcripts and panels`
- `feat(cache): import plaintext desktop cache`
- `feat(helper): decrypt Granola encrypted JSON`
- `feat(helper): inspect Granola OPFS database`
- `feat(export): add markdown export`
- `feat(tui): browse archived notes`

Avoid one huge "initial implementation" commit. This app has too many sharp
edges for that.

## open questions

- Should v1 default to private API only, or private API plus desktop-cache
  reconciliation?
- Should token refresh be supported at all, or should `graincrawl` require the
  user to reopen Granola when the WorkOS token expires?
- Should the helper be packaged in the Go repo or generated/downloaded as a
  local dev artifact?
- Should OPFS import attempt Yjs reconstruction, or keep Yjs as raw audit data
  until there is a product need?
- Should companion CLI become the preferred source if the feature flag is on,
  even though it depends on the running desktop app?

## recommended v1 cut

Do this first:

- repo scaffold
- config/store/doctor
- private API sync for notes/transcripts/panels
- plaintext cache fallback
- JSON commands for notes/transcripts/panels
- Markdown export
- redaction and temp-path discipline

Defer:

- OPFS import
- encrypted JSON import
- token refresh
- companion CLI integration
- TUI

But design the command/config surface now so adding those later does not require
renaming everything.
