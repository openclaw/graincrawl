# Distribution

`graincrawl` ships through GitHub Releases, Homebrew tap updates, and optional
Cloudsmith APT/RPM repositories.

## Local Checks

```bash
GOWORK=off go mod tidy
git diff --exit-code -- go.mod go.sum
GOWORK=off go vet ./...
GOWORK=off go test -count=1 ./...
make smoke
graincrawl check-update --json
```

The smoke target uses temp `HOME`, temp XDG dirs, temp config, temp cache, and a
temp SQLite database. Do not run distribution checks against a live personal
archive.

If GoReleaser is installed:

```bash
make release-snapshot
```

That creates local snapshot archives, checksums, `.deb`, and `.rpm` packages
under `dist/` without publishing or requiring signing credentials.

## Signed macOS Assets

Official macOS archives use the identifier `org.openclaw.graincrawl` and the
identity `Developer ID Application: OpenClaw Foundation (FWJYW4S8P8)`. Build
them only from a clean checkout whose `HEAD` exactly matches a trusted signed
release tag:

```bash
make release-artifacts VERSION=v0.3.2
scripts/verify-graincrawl-release.sh v0.3.2 \
  dist/graincrawl_0.3.2_darwin_arm64.tar.gz \
  dist/graincrawl_0.3.2_darwin_amd64.tar.gz
```

`release-artifacts` uses the shared managed-keychain helper. Credential routing
belongs in an ignored `.mac-release.env` or approved private environment,
never in Git. The same GoReleaser configuration remains credential-free for
ordinary local, Linux, Windows, and snapshot builds.

## Release Notes

GitHub uses Release Drafter to auto-label PRs and maintain release notes from
merged pull requests. Copy those notes, including the matching changelog
entries, into the tagged draft release before staging the signed archives.

## Tagged Release

Create and push a signed semver tag:

```bash
git tag -s v0.3.2
git push origin v0.3.2
```

The official release runs locally on an authorized maintainer Mac:

1. run `make release-artifacts VERSION=<tag>`
2. verify both signed Darwin archives with
   `scripts/verify-graincrawl-release.sh`
3. create a tagged draft GitHub release with the Release Drafter notes and
   matching changelog entries
4. upload every archive, Linux package, Darwin `.sha256` sidecar, and
   `sha256sums.txt` from `dist/`
5. verify the uploaded draft assets, then publish it

Publishing the release triggers independent native macOS signature checks and
the Homebrew tap update. Dispatch the APT/RPM workflows after publication when
Cloudsmith publishing is enabled.

GitHub Actions never receives the Developer ID private key and never publishes
GitHub Release artifacts. The `Release Validation` workflow is an optional
credential-free test and snapshot check for an existing signed tag. The
`Release Assets` workflow only verifies already-published files.

## Secrets

- `HOMEBREW_TAP_GITHUB_TOKEN`: optional; when set, updates the tap repository
  automatically
- `CLOUDSMITH_API_KEY`: optional; enables package publishing

## Optional Variables

- `HOMEBREW_TAP_REPO`: defaults to `openclaw/homebrew-tap`, which installs as
  `brew install openclaw/tap/graincrawl`
- `CODEQL_ENABLED`: set to `true` after code scanning is enabled for the
  repository
- `CLOUDSMITH_APT_TARGETS`: comma-separated targets like `ubuntu/jammy,debian/trixie`
- `CLOUDSMITH_DISTRIBUTION` and `CLOUDSMITH_RELEASE`: legacy single APT target
- `CLOUDSMITH_RPM_DISTRIBUTION`: defaults to `el`
- `CLOUDSMITH_RPM_RELEASE`: defaults to `9`

## Manual Reruns

If Cloudsmith publish fails after GitHub release assets exist:

```bash
gh workflow run publish-apt.yml -f tag_name=v0.1.0
gh workflow run publish-rpm.yml -f tag_name=v0.1.0
```

If the Homebrew tap update fails:

```bash
gh workflow run homebrew-tap.yml -f tag_name=v0.1.0
```
