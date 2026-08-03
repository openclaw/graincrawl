# Command reference

`graincrawl` reads Granola data into its own local SQLite archive. Commands that inspect or sync Granola never write to the Granola profile.

## Global options

```text
graincrawl [--json] [--config <path>] [--version] <command> [args]
```

| Option | Purpose |
|---|---|
| `--json` | Write structured JSON. Put it before the command so free-form search and SQL arguments remain unambiguous. |
| `--config <path>` | Use a specific TOML config instead of the default or `GRAINCRAWL_CONFIG`. |
| `--version`, `-v` | Print build metadata. |
| `--help`, `-h` | Print the built-in command overview. |

## Setup and sync

| Command | Purpose |
|---|---|
| `graincrawl init` | Create the config and archive directories. |
| `graincrawl doctor` | Inspect Granola, config, archive, and unlock state without prompting for Keychain access. |
| `graincrawl sources` | Show which source adapters the current config allows. |
| `graincrawl sync` | Sync from the configured source; `refresh` is an alias. |
| `graincrawl sync --source private-api` | Require the private desktop API session. |
| `graincrawl sync --source public-api` | Use Granola's official API with `GRANOLA_PUBLIC_API_KEY`. |
| `graincrawl sync --source desktop-cache` | Import the local plaintext desktop cache. |
| `graincrawl sync --limit <n>` | Limit the number of notes processed. |
| `graincrawl sync --no-transcripts --no-panels` | Skip transcript and panel hydration for this run. |
| `graincrawl runs --limit <n>` | List recent sync runs. |
| `graincrawl status` | Show archive counts and the SQLite path. |

`private-api`, `public-api`, and `desktop-cache` are supported sync sources. `public-api` must be explicitly enabled with `allow_public_api = true` or `GRAINCRAWL_ALLOW_PUBLIC_API=true`; its key is read only from `GRANOLA_PUBLIC_API_KEY`. It archives notes, summaries, and transcripts available to the key, but the official API does not expose panels or deletion events.

## Read the archive

| Command | Purpose |
|---|---|
| `graincrawl notes --limit <n>` | List archived notes. |
| `graincrawl search <query> --limit <n>` | Search archived notes. |
| `graincrawl note get <id>` | Show one archived note. |
| `graincrawl transcripts get <id>` | Show transcript chunks for a note. |
| `graincrawl panels get <id>` | Show panels for a note. |
| `graincrawl people --limit <n>` | List retained people source objects. |
| `graincrawl workspaces --limit <n>` | List retained workspace source objects. |
| `graincrawl sql <query>` | Run a read-only SQL query against the archive. |
| `graincrawl tui --limit <n>` | Browse archived notes in the terminal. |

The TUI detail pane is assembled from the SQLite archive, including note text, transcript chunks, panels, and retained source metadata.

## Export and portability

| Command | Purpose |
|---|---|
| `graincrawl export markdown --out <dir> --limit <n>` | Export notes as Markdown. |
| `graincrawl snapshot create --out <dir>` | Create a portable crawlkit snapshot. |
| `graincrawl import <snapshot-dir>` | Merge a snapshot into the archive. |
| `graincrawl import --replace <snapshot-dir>` | Restore exactly, removing rows absent from the snapshot. |

Merge imports keep existing local payloads on identity conflicts and preserve tombstones found on either side.

## Security and diagnostics

| Command | Purpose |
|---|---|
| `graincrawl unlock` | Report whether an explicit unlock is possible. |
| `graincrawl unlock encrypted-json` | Explicitly access legacy encrypted JSON when enabled and still supported. |
| `graincrawl sync --source <source> --unlock encrypted-json` | Unlock legacy encrypted source state in memory for that sync. |
| `graincrawl secrets` | Report graincrawl-managed secret policy without printing secret values. |
| `graincrawl metadata` | Print crawlkit control metadata for scripts and dashboards. |
| `graincrawl check-update` | Check GitHub Releases for a newer version. |
| `graincrawl version` | Show version, commit, and build date. |

Legacy encrypted JSON requires `allow_encrypted_json = true` and an explicit unlock command or sync flag. Decrypted payloads stay in process memory. The [security model](security.md) documents the Keychain boundary and Granola 7.427+ behavior.

The public API key is never written to graincrawl config or SQLite. Inject it at runtime, for example with `op run --env-file <template> -- graincrawl sync --source public-api`.

## JSON and automation

Every command that returns data accepts the global `--json` option. For example:

```bash
graincrawl --json doctor
graincrawl --json status
graincrawl --json search "decision"
graincrawl --json sql "select source, count(*) as notes from notes group by source"
```

`graincrawl metadata` describes the stable crawlkit control surfaces available to scripts and status dashboards.

## Shell completion

```bash
# bash
source <(graincrawl completion bash)

# zsh
source <(graincrawl completion zsh)
```
