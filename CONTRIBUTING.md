# Contributing

`graincrawl` is a read-only local archive tool. Contributions must preserve that
boundary.

## Rules

- Do not add Granola mutation endpoints.
- Do not write to Granola app data.
- Do not print tokens, refresh tokens, note bodies, transcript text, emails, or
  decrypted key material from diagnostics.
- Use temp directories for tests.
- Keep provider-specific Granola logic in this repo, not in `crawlkit`.

## Checks

Run before handoff:

```bash
make check
```
