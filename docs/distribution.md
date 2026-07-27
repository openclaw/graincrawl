# Distribution

`graincrawl` ships through GitHub Releases, Homebrew tap updates, and optional
Cloudsmith APT/RPM repositories.

## Local Checks

```bash
make check
graincrawl check-update --json
```

The smoke target uses temp `HOME`, temp XDG dirs, temp config, temp cache, and a
temp SQLite database. Do not run distribution checks against a live personal
archive.

For credential-free development artifacts:

```bash
make snapshot
```

That creates local snapshot archives, checksums, `.deb`, and `.rpm` packages
under `dist/` without publishing or requiring signing credentials. The target
uses a pinned GoReleaser module version, so Go can fetch the tool when needed.

## Unified Release

Official releases use `.github/workflows/release-unified.yml`, which calls the
fleet release pipeline pinned to its compatible `@v1` contract. Dispatch it
from the protected `main` head with the changelog version already prepared:

```bash
make release
# release refuses locally and prints:
gh workflow run release-unified.yml --repo openclaw/graincrawl -f version=X.Y.Z
```

The pipeline freezes an annotated `v0.3.3` tag, runs the GoReleaser matrix,
signs every Darwin binary as
`org.openclaw.graincrawl.graincrawl` with
`Developer ID Application: OpenClaw Foundation (FWJYW4S8P8)`, notarizes the
signed bytes, and independently verifies both macOS architectures before
publishing. Release notes are the byte-exact dated `CHANGELOG.md` section.

GoReleaser's nFPM configuration is auto-detected. The generated `.deb` and
`.rpm` files are included as top-level release assets and bound into
`ASSET-INVENTORY.json` and `SHA256SUMS` alongside the signed archives. The
pipeline then hands exact archive names and digests to
`openclaw/homebrew-tap` and opens a closeout pull request restoring the next
`## Unreleased` section.

## Cloudsmith Packages

Cloudsmith publishing remains an explicit post-release operation. The APT and
RPM workflows download their package type from the published tag and verify
each package against the pipeline's `SHA256SUMS` before upload:

```bash
gh workflow run publish-apt.yml -f tag_name=v0.3.3
gh workflow run publish-rpm.yml -f tag_name=v0.3.3
```

## Legacy Manual Tools

`release-legacy.yml`, `release-assets.yml`, and `homebrew-tap.yml` are
manual-only fallbacks. They exist to validate or recover releases made by the
old local signing path and never trigger from tags or release publication.
`make release-artifacts` remains an alias for the fail-closed `release` target,
and `scripts/package-graincrawl-release.sh` also points maintainers to
`release-unified.yml`; the former local path could sign macOS binaries without
notarizing them. `make release-snapshot` remains an alias for the credential-free
`snapshot` target. `make verify-release TAG=vX.Y.Z ARCHIVES='path ...'` remains
available for inspecting legacy archives and their per-archive `.sha256`
sidecars, including Apple notarization assessment, but it is not a publishing
path.

## Secrets

- `MACOS_SIGNING_P12` and `MACOS_SIGNING_P12_PASSWORD`: Foundation Developer ID
  identity for the ephemeral CI keychain
- `ASC_KEY_ID`, `ASC_ISSUER_ID`, and `ASC_PRIVATE_KEY_P8`: notarization API
  credentials
- `HOMEBREW_TAP_TOKEN`: token with Actions access to
  `openclaw/homebrew-tap`
- `CLOUDSMITH_API_KEY`: optional; enables manual APT/RPM publishing

## Optional Variables

- `CODEQL_ENABLED`: set to `true` after code scanning is enabled for the
  repository
- `CLOUDSMITH_APT_TARGETS`: comma-separated targets like `ubuntu/jammy,debian/trixie`
- `CLOUDSMITH_DISTRIBUTION` and `CLOUDSMITH_RELEASE`: legacy single APT target
- `CLOUDSMITH_RPM_DISTRIBUTION`: defaults to `el`
- `CLOUDSMITH_RPM_RELEASE`: defaults to `9`

## Manual Reruns

If Cloudsmith publishing needs to be retried after the GitHub release exists:

```bash
gh workflow run publish-apt.yml -f tag_name=v0.3.3
gh workflow run publish-rpm.yml -f tag_name=v0.3.3
```

If the unified Homebrew handoff needs a manual fallback:

```bash
gh workflow run homebrew-tap.yml -f tag_name=v0.3.3
```
