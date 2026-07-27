# Security Model

`graincrawl` treats Granola auth and encrypted local state as user-owned data.
The default path is private API sync using the desktop session file when it is
already present, plus plaintext cache fallback when Granola still writes one.

## Defaults

- `allow_private_api = true`
- `allow_desktop_cache = true`
- `allow_encrypted_json = false`
- `allow_opfs = false`
- `keychain_prompt_mode = "explicit"`
- `persist_helper_keys = false`

That means `graincrawl doctor`, `status`, `notes`, and `export` must not prompt
macOS Keychain. A Keychain prompt is allowed only after the user explicitly
enables encrypted sources and invokes an explicit unlock command.

## Keychain Boundary

Legacy encrypted JSON support runs only inside the authorized public command
path. When `storage.dek` is still present, graincrawl snapshots the requested
encrypted files into memory, reads the legacy `Granola Safe Storage` Keychain
item through `/usr/bin/security`, unwraps the DEK, and decrypts only the
requested JSON. Keychain access has a 30-second timeout. The DEK, Keychain
secret, and decrypted Granola JSON are never written to disk. There is no
hidden helper command that can bypass the public authorization checks. OPFS
remains unsupported.

`allow_encrypted_json = true` permits the feature, but never invokes it by
itself. The operator must also run `graincrawl unlock encrypted-json` or pass
`--unlock encrypted-json` to `sync`.

## Granola 7.427+ encrypted state

Granola 7.427+ moved its data-encryption key into the macOS data-protection
Keychain under the app-scoped access group `QZ7DHHLN25.granola`. Only
Granola-signed code can read that item. When `storage.dek` is absent and the
Granola profile contains at least one `*.enc` file, graincrawl diagnoses this
post-migration state without attempting to access the new Keychain item.
`unlock encrypted-json` is blocked outright; `private-api` and `desktop-cache`
are blocked per source, only when the file that source reads (`supabase.json`
and `cache-v6.json` respectively) is the encrypted one, so a source with usable
plaintext state is neither blocked nor advertised as unavailable.

This is an upstream code-signing and Keychain access-group boundary, not a
graincrawl bug. The legacy unlock path remains available only for profiles that
still contain `storage.dek`; existing graincrawl archives remain readable.

## Operational Rules

- Do not log tokens, refresh tokens, decrypted keys, cookies, or raw encrypted
  payloads.
- Do not read or mutate live Granola files in tests.
- Snapshot encrypted files into memory before Keychain access; never write
  decrypted Granola JSON to disk.
- Use temp config, cache, and database paths for tests.
- Prefer `graincrawl secrets --json` before debugging unlock issues.
- Prefer `graincrawl unlock --json` before enabling encrypted sources.
