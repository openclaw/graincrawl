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
under `dist/` without publishing.

## Release Notes

GitHub uses Release Drafter to auto-label PRs and generate release notes from
merged pull requests. The release workflow publishes the Release Drafter output
for the pushed tag, then uploads the GoReleaser artifacts to that release.

## Tagged Release

Create and push a semver tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow:

1. runs tests
2. builds GoReleaser artifacts
3. publishes Release Drafter notes for the tag
4. uploads GitHub release assets
5. optionally publishes APT/RPM packages to Cloudsmith
6. updates the Homebrew tap

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
