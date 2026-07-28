#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

for compiler in clang gcc; do
  if ! command -v "$compiler" >/dev/null 2>&1; then
    echo "Missing C compiler: $compiler" >&2
    exit 1
  fi
done

mkdir -p build

expected=$'headless\nZumbra Desktop 0.7.0\n960\n640\nhello desktop\ntrue\n7'
source="code_examples/core/desktop_runtime.zum"

echo "Running the complete Z13 Go test suite..."
go test ./...

echo "Checking and running the desktop example in the VM..."
go run . check "$source"
vm_output="$(ZUMBRA_DESKTOP_HEADLESS=1 go run . run "$source")"
if [[ "$vm_output" != "$expected" ]]; then
  printf 'Unexpected VM desktop output:\n%s\n' "$vm_output" >&2
  exit 1
fi

echo "Checking the real graphical SDL3 example without opening a window..."
go run . check code_examples/core/desktop_window.zum

for compiler in clang gcc; do
  go run . build --release --compiler "$compiler" -o "build/z13-window-${compiler}" code_examples/core/desktop_window.zum
done

for compiler in clang gcc; do
  output="build/z13-desktop-${compiler}"
  echo "Building desktop runtime with ${compiler}..."
  go run . build --release --compiler "$compiler" -o "$output" "$source"
  native_output="$(ZUMBRA_DESKTOP_HEADLESS=1 "./$output")"
  if [[ "$native_output" != "$expected" ]]; then
    printf 'VM/%s desktop output mismatch:\n%s\n' "$compiler" "$native_output" >&2
    exit 1
  fi
done

echo "Running desktop race checks..."
ZUMBRA_DESKTOP_HEADLESS=1 go test -race ./object ./object/builtins ./evaluator ./vm ./conformance

echo "Rechecking Z12 persistence and Z11 HTTP/WebSockets..."
scripts/test-data-persistence.sh

echo "Z13 desktop runtime tests passed."

if ! ldconfig -p 2>/dev/null | grep -q 'libSDL3.so'; then
  echo "Note: SDL3 was not found. Headless validation passed; install SDL3 before running graphical applications."
fi
