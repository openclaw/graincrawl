#!/usr/bin/env bash
set -euo pipefail

TARGET=${1:-}
BINARY=${2:-}
IDENTIFIER=org.openclaw.graincrawl
EXPECTED_TEAM_ID=FWJYW4S8P8
EXPECTED_AUTHORITY='Developer ID Application: OpenClaw Foundation (FWJYW4S8P8)'
REQUIREMENT="identifier \"$IDENTIFIER\" and anchor apple generic and certificate 1[field.1.2.840.113635.100.6.2.6] exists and certificate leaf[field.1.2.840.113635.100.6.1.13] exists and certificate leaf[subject.OU] = \"$EXPECTED_TEAM_ID\""

case "$TARGET" in
  darwin_*) ;;
  *) exit 0 ;;
esac

[[ "${GRAINCRAWL_REQUIRE_CODESIGN:-0}" == 1 ]] || exit 0
[[ -n "$BINARY" && -f "$BINARY" ]] || {
  echo "missing graincrawl binary for target $TARGET: $BINARY" >&2
  exit 1
}
[[ "$(uname -s)" == Darwin ]] || {
  echo "official graincrawl macOS signing must run on macOS" >&2
  exit 1
}
[[ -n "${CODESIGN_IDENTITY:-}" ]] || {
  echo "CODESIGN_IDENTITY is required; run through mac-release codesign-run" >&2
  exit 1
}

codesign --force --options runtime --timestamp \
  --identifier "$IDENTIFIER" --sign "$CODESIGN_IDENTITY" "$BINARY"
codesign --verify --strict -R="$REQUIREMENT" --verbose=2 "$BINARY"

signature=$(codesign -dvvv "$BINARY" 2>&1)
grep -Fx "Identifier=$IDENTIFIER" <<<"$signature" >/dev/null
grep -Fx "TeamIdentifier=$EXPECTED_TEAM_ID" <<<"$signature" >/dev/null
grep -Fx "Authority=$EXPECTED_AUTHORITY" <<<"$signature" >/dev/null

case "$TARGET" in
  darwin_arm64*) expected_arch=arm64 ;;
  darwin_amd64*) expected_arch=x86_64 ;;
  *)
    echo "unsupported graincrawl macOS target: $TARGET" >&2
    exit 1
    ;;
esac
lipo -archs "$BINARY" | tr ' ' '\n' | grep -Fx "$expected_arch" >/dev/null
