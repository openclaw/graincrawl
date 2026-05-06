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
enables encrypted sources and changes the prompt mode.

## Keychain Boundary

Encrypted JSON and OPFS support should be implemented through an explicit
companion/helper process. The helper owns the macOS prompt, returns only the
minimum unlock material needed for the current operation, and exits. The main
archive path should avoid keeping helper keys on disk unless
`persist_helper_keys` is intentionally enabled.

## Operational Rules

- Do not log tokens, refresh tokens, decrypted keys, cookies, or raw encrypted
  payloads.
- Do not read or mutate live Granola files in tests.
- Use temp config, cache, and database paths for tests.
- Prefer `graincrawl secrets --json` before debugging unlock issues.
- Prefer `graincrawl unlock --json` before enabling encrypted sources.
