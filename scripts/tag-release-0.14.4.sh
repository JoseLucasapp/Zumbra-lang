#!/usr/bin/env bash
set -euo pipefail

version="0.14.4"
actual="$(./build/zumbra --version 2>/dev/null || true)"
if [[ "$actual" != "$version" ]]; then
  echo "Zumbra version mismatch: expected $version, got ${actual:-<missing>}" >&2
  echo "Run: go build -o build/zumbra ." >&2
  exit 1
fi

git diff --quiet -- . || {
  echo "Working tree has uncommitted changes. Commit before tagging." >&2
  exit 1
}

git tag -a "v$version" -m "Zumbra $version"
echo "Created tag v$version"
echo "Push with: git push origin main --tags"
