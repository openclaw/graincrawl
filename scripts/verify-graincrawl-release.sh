#!/usr/bin/env bash
set -euo pipefail

VERSION=${1:-}
shift || true
IDENTIFIER=org.openclaw.graincrawl
EXPECTED_TEAM_ID=FWJYW4S8P8
EXPECTED_AUTHORITY='Developer ID Application: OpenClaw Foundation (FWJYW4S8P8)'
REQUIREMENT="identifier \"$IDENTIFIER\" and anchor apple generic and certificate 1[field.1.2.840.113635.100.6.2.6] exists and certificate leaf[field.1.2.840.113635.100.6.1.13] exists and certificate leaf[subject.OU] = \"$EXPECTED_TEAM_ID\""

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ || "$#" -eq 0 ]]; then
  echo "usage: $0 vX.Y.Z graincrawl_VERSION_darwin_ARCH.tar.gz [...]" >&2
  exit 2
fi
[[ "$(uname -s)" == Darwin ]] || {
  echo "graincrawl macOS signature verification must run on macOS" >&2
  exit 1
}

WORK_DIR=$(mktemp -d "${TMPDIR:-/tmp}/graincrawl-verify.XXXXXX")
trap 'rm -rf "$WORK_DIR"' EXIT

release_version=${VERSION#v}
for archive in "$@"; do
  archive=$(cd "$(dirname "$archive")" && pwd)/$(basename "$archive")
  checksum="$archive.sha256"
  [[ -f "$archive" && -f "$checksum" ]] || {
    echo "missing artifact or checksum: $archive" >&2
    exit 1
  }

  line_count=$(awk 'END { print NR }' "$checksum")
  read -r expected_digest expected_name extra < "$checksum" || true
  if [[ "$line_count" != 1 || ! "$expected_digest" =~ ^[0-9a-f]{64}$ || "$expected_name" != "$(basename "$archive")" || -n "${extra:-}" ]]; then
    echo "invalid checksum entry for $(basename "$archive")" >&2
    exit 1
  fi
  actual_digest=$(shasum -a 256 "$archive" | awk '{ print $1 }')
  [[ "$actual_digest" == "$expected_digest" ]] || {
    echo "checksum mismatch for $(basename "$archive")" >&2
    exit 1
  }

  case "$(basename "$archive")" in
    "graincrawl_${release_version}_darwin_arm64.tar.gz") expected_arch=arm64 ;;
    "graincrawl_${release_version}_darwin_amd64.tar.gz") expected_arch=x86_64 ;;
    *)
      echo "unexpected graincrawl artifact name: $(basename "$archive")" >&2
      exit 1
      ;;
  esac

  stage="$WORK_DIR/$expected_arch"
  mkdir -p "$stage"
  tar -xzf "$archive" -C "$stage"
  binary="$stage/graincrawl"
  [[ -x "$binary" ]]

  codesign --verify --strict -R="$REQUIREMENT" --verbose=2 "$binary"
  codesign --verify --strict --check-notarization -R=notarized "$binary"
  signature=$(codesign -dvvv "$binary" 2>&1)
  grep -Fx "Identifier=$IDENTIFIER" <<<"$signature" >/dev/null
  grep -Fx "TeamIdentifier=$EXPECTED_TEAM_ID" <<<"$signature" >/dev/null
  grep -Fx "Authority=$EXPECTED_AUTHORITY" <<<"$signature" >/dev/null
  lipo -archs "$binary" | tr ' ' '\n' | grep -Fx "$expected_arch" >/dev/null
  env -i PATH="$PATH" HOME="${HOME:-}" TMPDIR="${TMPDIR:-/tmp}" \
    "$binary" --json version | grep -F "\"version\": \"$release_version\"" >/dev/null
done
