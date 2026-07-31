#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

normalize_output() {
  tr -d '\r'
}

echo "Testing Z8.1 contextual callback inference..."
go test ./types ./hir ./mir ./pipeline ./nativec ./docs

go run . check code_examples/core/contextual_callbacks.zum

hir_dump="$(go run . ir code_examples/core/contextual_callbacks.zum hir | normalize_output)"
mir_dump="$(go run . ir code_examples/core/contextual_callbacks.zum optimized | normalize_output)"

grep -q 'function name="triple" : fct(i32) -> i32' <<<"$hir_dump"
grep -q 'identifier name="value" : i32' <<<"$hir_dump"
grep -q 'function triple(value) -> i32' <<<"$mir_dump"
grep -q 'function_ref "triple" : fct(i32) -> i32' <<<"$mir_dump"

if grep -q 'fct(unknown) -> unknown' <<<"$hir_dump$mir_dump"; then
  echo "Contextual callback remained unknown in IR" >&2
  exit 1
fi

go run . build --release -o build/z81-contextual-callbacks code_examples/core/contextual_callbacks.zum
actual="$(./build/z81-contextual-callbacks | normalize_output)"
if [[ "$actual" != "42" ]]; then
  printf 'Expected 42, got:\n%s\n' "$actual" >&2
  exit 1
fi

echo "Z8.1 contextual callback inference tests passed."
