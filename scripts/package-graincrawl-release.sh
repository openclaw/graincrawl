#!/usr/bin/env bash
set -euo pipefail

echo "local release packaging is disabled because it cannot guarantee notarized macOS artifacts" >&2
echo "official releases must use: gh workflow run release-unified.yml --repo openclaw/graincrawl -f version=X.Y.Z" >&2
exit 1
