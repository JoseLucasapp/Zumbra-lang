#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if ! pkg-config --exists sqlite3; then
  echo "SQLite development files are required." >&2
  echo "Debian/Ubuntu: sudo apt install libsqlite3-dev pkg-config" >&2
  exit 1
fi

echo "Testing Z12.1 SQLite packages..."
go test \
  ./object \
  ./object/builtins \
  ./types \
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

echo "Checking and running SQLite example in the VM..."
go run . check code_examples/core/sqlite.zum
vm_output="$(go run . run code_examples/core/sqlite.zum 2>/dev/null)"
expected=$'1\nLucas\n42\n1\ntrue\ntrue'
if [[ "$vm_output" != "$expected" ]]; then
  printf 'Unexpected VM SQLite output:\n%s\n' "$vm_output" >&2
  exit 1
fi

echo "Building SQLite example with Clang and GCC..."
go run . build --release --compiler clang -o build/z12-sqlite-clang code_examples/core/sqlite.zum
go run . build --release --compiler gcc -o build/z12-sqlite-gcc code_examples/core/sqlite.zum
clang_output="$(./build/z12-sqlite-clang)"
gcc_output="$(./build/z12-sqlite-gcc)"
if [[ "$clang_output" != "$expected" || "$gcc_output" != "$expected" ]]; then
  echo "VM/Clang/GCC SQLite output mismatch" >&2
  exit 1
fi

echo "Running SQLite race checks..."
go test -race ./object ./object/builtins ./evaluator ./vm ./conformance

echo "Z12.1 SQLite tests passed."
