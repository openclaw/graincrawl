#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION=${1:-}

usage() {
  echo "usage: $0 vX.Y.Z" >&2
  exit 2
}

[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] || usage
[[ "$(uname -s)" == Darwin ]] || {
  echo "graincrawl macOS release packaging must run on macOS" >&2
  exit 1
}
[[ "$(uname -m)" == arm64 ]] || {
  echo "graincrawl release packaging requires Apple Silicon with Rosetta for both architecture smoke tests" >&2
  exit 1
}
[[ -n "${CODESIGN_IDENTITY:-}" ]] || {
  echo "CODESIGN_IDENTITY is required; run through mac-release codesign-run" >&2
  exit 1
}

for tool in codesign git go goreleaser lipo shasum tar; do
  command -v "$tool" >/dev/null || {
    echo "missing required tool: $tool" >&2
    exit 1
  }
done

required_go=$(awk '$1 == "go" { print $2; exit }' "$ROOT/go.mod")
actual_go=$(GOTOOLCHAIN=local go env GOVERSION)
[[ -n "$required_go" && "$actual_go" == "go$required_go" ]] || {
  echo "official graincrawl releases require Go $required_go exactly; found $actual_go" >&2
  exit 1
}

head_commit=$(git -C "$ROOT" rev-parse HEAD)
tag_commit=$(git -C "$ROOT" rev-parse "refs/tags/$VERSION^{commit}" 2>/dev/null) || {
  echo "release tag does not exist locally: $VERSION" >&2
  exit 1
}
tag_object=$(git -C "$ROOT" rev-parse "refs/tags/$VERSION")
[[ "$head_commit" == "$tag_commit" ]] || {
  echo "HEAD does not match release tag $VERSION" >&2
  exit 1
}
[[ -z "$(git -C "$ROOT" status --porcelain --untracked-files=normal)" ]] || {
  echo "release checkout is not clean" >&2
  exit 1
}
git -C "$ROOT" tag -v "$VERSION" >/dev/null 2>&1 || {
  echo "release tag is not signed by a trusted git signing key: $VERSION" >&2
  exit 1
}

remote_refs=$(git -C "$ROOT" ls-remote --tags origin \
  "refs/tags/$VERSION" "refs/tags/$VERSION^{}")
remote_tag_object=$(awk -v ref="refs/tags/$VERSION" '$2 == ref { print $1 }' <<<"$remote_refs")
remote_tag_commit=$(awk -v ref="refs/tags/$VERSION^{}" '$2 == ref { print $1 }' <<<"$remote_refs")
[[ -n "$remote_tag_object" && -n "$remote_tag_commit" ]] || {
  echo "release tag does not exist on origin as a signed annotated tag: $VERSION" >&2
  exit 1
}
[[ "$tag_object" == "$remote_tag_object" && "$tag_commit" == "$remote_tag_commit" ]] || {
  echo "local release tag does not match origin: $VERSION" >&2
  exit 1
}

(
  cd "$ROOT"
  GOTOOLCHAIN=local GOWORK=off GRAINCRAWL_REQUIRE_CODESIGN=1 \
    goreleaser release --clean --skip=publish
)

release_version=${VERSION#v}
archives=(
  "$ROOT/dist/graincrawl_${release_version}_darwin_arm64.tar.gz"
  "$ROOT/dist/graincrawl_${release_version}_darwin_amd64.tar.gz"
)
for archive in "${archives[@]}"; do
  (
    cd "$(dirname "$archive")"
    shasum -a 256 "$(basename "$archive")" > "$(basename "$archive").sha256"
  )
done

"$ROOT/scripts/verify-graincrawl-release.sh" "$VERSION" "${archives[@]}"
