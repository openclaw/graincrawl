# Releasing Graincrawl

`.github/workflows/release-unified.yml` owns the GitHub release, signed and
notarized macOS binaries, universal archive, Linux archives, deb/rpm packages,
checksums, and OpenClaw Homebrew handoff. It uses the shared Go CLI archetype.

Prepare a dated changelog section on protected main and require green push CI
on the exact source commit. With explicit release authorization, push an
annotated SSH-signed `vX.Y.Z` tag allowed by `.github/release-allowed-signers`.
The workflow validates the tag, builds, signs, independently rebuilds/verifies,
and publishes the frozen payload. It never replaces an existing version.

Dispatch from main with `version` set to a dated changelog version and
`dry_run=true` (default) for a full signing and verification rehearsal. Existing
historical tags do not redirect the rehearsal away from current main. No tag,
draft, release, tap update, or closeout PR is created during the rehearsal.
The shared workflow uses the fleet `@v1` compatibility tag, which includes
protected-main validation for no-publish rehearsals.

Required secrets are `MACOS_SIGNING_P12`, `MACOS_SIGNING_P12_PASSWORD`,
`ASC_KEY_ID`, `ASC_ISSUER_ID`, `ASC_PRIVATE_KEY_P8`, and `HOMEBREW_TAP_TOKEN`.
The legacy manual workflows are verification-only; they do not publish.
