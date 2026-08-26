#!/usr/bin/env bash
set -euo pipefail
# Resolve next llmhub version (v0.0.x) via GitHub Releases, never via stale local tags.
# Usage: ./next_version.sh [patch|minor|major]
BUMP="${1:-patch}"
REPO="therealtinhtute/llmhub"
LATEST="$(gh api repos/$REPO/releases/latest --jq .tag_name 2>/dev/null || gh release list --repo $REPO --limit 1 --json tagName --jq '.[0].tagName' 2>/dev/null || git ls-remote --tags origin | grep -E 'v0\.0\.[0-9]+' | sort -V | tail -1 | sed 's|.*refs/tags/||')"
if [[ ! "$LATEST" =~ ^v0\.0\.[0-9]+$ ]]; then
  echo "latest tag not v0.0.x: $LATEST" >&2
  exit 1
fi
VER="${LATEST#v}"
IFS='.' read -r MA MI PA <<< "$VER"
case "$BUMP" in
  patch) PA=$((PA+1));;
  minor) MI=$((MI+1)); PA=0;;
  major) MA=$((MA+1)); MI=0; PA=0;;
  *) echo "unknown bump $BUMP" >&2; exit 1;;
esac
echo "v$MA.$MI.$PA"
