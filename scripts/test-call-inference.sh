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

echo "Testing Z9.1 contextual call inference..."
go test \
  ./types \
  ./hir \
  ./mir \
  ./pipeline \
  ./conformance \
  ./nativec \
  ./docs

echo "Checking ordinary calls, spawn, channels, atomics, and methods..."
go run . check code_examples/core/call_inference.zum

hir_dump="$(go run . ir code_examples/core/call_inference.zum hir | normalize_output)"
mir_dump="$(go run . ir code_examples/core/call_inference.zum optimized | normalize_output)"

grep -Fq 'fct(int) -> int' <<<"$hir_dump"
grep -Fq 'channel<int>' <<<"$hir_dump"
grep -Fq 'fct(atomic_int,int) -> null' <<<"$hir_dump"
grep -Fq 'function identity(value) -> int' <<<"$mir_dump"
grep -Fq ': channel<int>' <<<"$mir_dump"
grep -Fq 'function count(counter, amount) -> null' <<<"$mir_dump"

if grep -Fq 'function identity(value) -> unknown' <<<"$mir_dump"; then
  echo "identity retained an unknown return in MIR" >&2
  exit 1
fi
if grep -Fq 'function count(counter, amount) -> unknown' <<<"$mir_dump"; then
  echo "count retained an unknown return in MIR" >&2
  exit 1
fi

assert_output $'42\n7\n2000\n9' go run . run code_examples/core/call_inference.zum

echo "Building contextual call inference example natively..."
go run . build --release -o build/z91-call-inference code_examples/core/call_inference.zum
assert_output $'42\n7\n2000\n9' ./build/z91-call-inference

if command -v clang >/dev/null 2>&1 && command -v gcc >/dev/null 2>&1; then
  go run . build --release --compiler clang -o build/z91-call-inference-clang code_examples/core/call_inference.zum
  go run . build --release --compiler gcc -o build/z91-call-inference-gcc code_examples/core/call_inference.zum
  assert_output $'42\n7\n2000\n9' ./build/z91-call-inference-clang
  assert_output $'42\n7\n2000\n9' ./build/z91-call-inference-gcc
fi

echo "Z9.1 contextual call inference tests passed."
