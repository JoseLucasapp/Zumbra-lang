#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

normalize_output() {
  tr -d '\r'
}

assert_output() {
  local expected="$1"
  shift
  local actual
  actual="$("$@" | normalize_output)"
  if [[ "$actual" != "$expected" ]]; then
    printf 'Unexpected output from: %s\nExpected:\n%s\nActual:\n%s\n' "$*" "$expected" "$actual" >&2
    exit 1
  fi
}

echo "Testing Z9 concurrency packages..."
go test \
  ./lexer \
  ./parser \
  ./types \
  ./object \
  ./object/builtins \
  ./semantic \
  ./hir \
  ./mir \
  ./pipeline \
  ./compiler \
  ./evaluator \
  ./vm \
  ./conformance \
  ./nativec \
  ./transpiler \
  ./docs

echo "Checking task and channel example..."
go run . check code_examples/core/concurrency.zum

hir_dump="$(go run . ir code_examples/core/concurrency.zum hir | normalize_output)"
mir_dump="$(go run . ir code_examples/core/concurrency.zum optimized | normalize_output)"
grep -qi 'spawn' <<<"$hir_dump$mir_dump"
grep -qi 'await' <<<"$hir_dump$mir_dump"
grep -qi 'task<' <<<"$hir_dump$mir_dump"

assert_output $'36\n7\n8\n2000' go run . run code_examples/core/concurrency.zum

echo "Building native pthread runtime..."
go run . build --release -o build/z9-concurrency code_examples/core/concurrency.zum
assert_output $'36\n7\n8\n2000' ./build/z9-concurrency

if command -v clang >/dev/null 2>&1 && command -v gcc >/dev/null 2>&1; then
  go run . build --release --compiler clang -o build/z9-concurrency-clang code_examples/core/concurrency.zum
  go run . build --release --compiler gcc -o build/z9-concurrency-gcc code_examples/core/concurrency.zum
  assert_output $'36\n7\n8\n2000' ./build/z9-concurrency-clang
  assert_output $'36\n7\n8\n2000' ./build/z9-concurrency-gcc
fi

echo "Z9 concurrency tests passed."
