#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

failed=0
while IFS= read -r path; do
  [[ -z "$path" ]] && continue
  printf 'Generated directory must not be committed or included in releases: %s\n' "$path" >&2
  failed=1
done < <(find . -mindepth 1 -type d \( -name build -o -name dist -o -name out -o -name release -o -name delivery -o -name tools -o -name tmp \) -not -path './.git/*' -print | sort)

while IFS= read -r entry; do
  [[ -z "$entry" ]] && continue
  printf 'Large repository file requires review: %s\n' "$entry" >&2
  failed=1
done < <(find . -type f -not -path './.git/*' -size +5M -printf '%s %p\n' | sort -nr)

if (( failed != 0 )); then
  exit 1
fi

printf 'Repository hygiene checks passed.\n'
