#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

rm -rf build dist out release delivery tmp .cache
find . -mindepth 2 -type d \( -name build -o -name dist -o -name out -o -name release -o -name tmp -o -name .cache \) -not -path './.git/*' -prune -exec rm -rf {} + 2>/dev/null || true
find . -type f \( -name '*.test' -o -name '*.prof' -o -name 'coverage.out' -o -name 'coverage-*.out' \) -delete

if [[ "${1:-}" == "--local-tools" ]]; then
  find . -mindepth 1 -type d -name tools -not -path './.git/*' -prune -exec rm -rf {} + 2>/dev/null || true
fi

printf 'Generated files removed.\n'
