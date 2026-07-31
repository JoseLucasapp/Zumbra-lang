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

echo "Testing Z8 packages..."
go test \
  ./cbinding \
  ./lexer \
  ./parser \
  ./modules \
  ./semantic \
  ./types \
  ./hir \
  ./mir \
  ./pipeline \
  ./nativec \
  ./compiler \
  ./evaluator \
  ./transpiler

echo "Checking and running aliased modules..."
go run . check code_examples/core/modules.zum
module_graph="$(go run . modules code_examples/core/modules.zum | normalize_output)"
grep -q 'exports BASE, Point, add' <<<"$module_graph"
assert_output $'42\n15' go run . run code_examples/core/modules.zum

go run . build --release -o build/z8-modules code_examples/core/modules.zum
assert_output $'42\n15' ./build/z8-modules

echo "Building C FFI and synchronous callback example..."
go run . check code_examples/core/ffi.zum
go run . build --release -o build/z8-ffi code_examples/core/ffi.zum
assert_output $'42\n42\nzumbra\ntrue' ./build/z8-ffi

echo "Generating a conservative binding from a C header..."
rm -f build/z8_math_binding.zum build/z8_bind_test.zum build/z8-bind-test
go run . bind-c \
  --pub \
  --link ../code_examples/native/z8_math.c \
  -o build/z8_math_binding.zum \
  code_examples/native/z8_math.h

grep -q 'fct z8_add(left: i32, right: i32) -> i32;' build/z8_math_binding.zum
cat > build/z8_bind_test.zum <<'ZUM'
import "z8_math_binding.zum" as native;
unsafe { show(native.z8_add(20i32, 22i32)); }
ZUM

go run . build --release -o build/z8-bind-test build/z8_bind_test.zum
assert_output '42' ./build/z8-bind-test

echo "Z8 modules and C FFI tests passed."
