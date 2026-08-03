#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

failed=0
declare -A reported_generated=()

generated_directory_for() {
  local path="${1#./}"
  local prefix=""
  local component
  IFS='/' read -r -a components <<< "$path"
  for component in "${components[@]}"; do
    case "$component" in
      build|dist|out|release|delivery|tools|tmp)
        printf './%s%s\n' "$prefix" "$component"
        return 0
        ;;
    esac
    prefix+="$component/"
  done
  return 1
}

inspect_file() {
  local path="$1"
  local generated_dir
  local size

  if generated_dir="$(generated_directory_for "$path")"; then
    if [[ -z "${reported_generated[$generated_dir]+x}" ]]; then
      printf 'Generated directory contains tracked or unignored files: %s\n' "$generated_dir" >&2
      reported_generated["$generated_dir"]=1
      failed=1
    fi
  fi

  if [[ -f "$path" ]]; then
    size="$(stat -c '%s' -- "$path")"
    if (( size > 5 * 1024 * 1024 )); then
      printf 'Large repository file requires review: %s %s\n' "$size" "$path" >&2
      failed=1
    fi
  fi
}

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  # Inspect files that Git would commit: tracked files and untracked files that
  # are not excluded by .gitignore. Local compiler/test output may exist safely
  # when it is covered by the repository ignore rules.
  while IFS= read -r -d '' path; do
    inspect_file "$path"
  done < <(git ls-files --cached --others --exclude-standard -z)
else
  # Release archives do not include Git metadata. In that case, inspect every
  # file in the extracted tree; empty generated directories are harmless.
  while IFS= read -r -d '' path; do
    inspect_file "$path"
  done < <(find . -type f -not -path './.git/*' -print0)
fi

if (( failed != 0 )); then
  exit 1
fi

printf 'Repository hygiene checks passed.\n'
