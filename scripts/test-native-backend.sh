#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "Testing native backend packages..."
go test ./nativec ./hir ./mir ./semantic ./pipeline

compiler=""
for candidate in clang gcc cc; do
    if command -v "$candidate" >/dev/null 2>&1; then
        compiler="$candidate"
        break
    fi
done
if [[ -z "$compiler" ]]; then
    echo "Clang or GCC is required for the Z7 native backend." >&2
    exit 1
fi

echo "Building with $compiler..."
go run . build --release --compiler "$compiler" code_examples/core/native_build.zum

expected=$'14\n19\n9\nrunning'
actual="$(./build/native_build)"
if [[ "$actual" != "$expected" ]]; then
    echo "Unexpected native output:" >&2
    printf '%s\n' "$actual" >&2
    exit 1
fi

go run . build --emit-c code_examples/core/native_build.zum

test -f build/native/native_build/main.c
test -f build/native/native_build/zumbra_runtime.c
test -f build/native/native_build/zumbra_runtime.h

echo "Z7 native backend tests passed."
