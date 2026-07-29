#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

rm -rf build dist out release .cache
find code_examples -type d \( -name build -o -name dist -o -name out \) -prune -exec rm -rf {} + 2>/dev/null || true
find . -type f \( -name '*.test' -o -name '*.prof' -o -name 'coverage.out' -o -name 'coverage-*.out' \) -delete

if [[ "${1:-}" == "--local-tools" ]]; then
  rm -rf tools
fi

printf 'Generated files removed.\n'
