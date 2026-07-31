#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "Testing Z11 HTTP, API and WebSocket packages..."
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

echo "Checking and running HTTP API example in the VM..."
go run . check code_examples/core/http_api.zum
vm_output="$(go run . run code_examples/core/http_api.zum 2>/dev/null)"
for expected in "200" "hello zumbra" "201" "local-user" "event: status"; do
  if [[ "$vm_output" != *"$expected"* ]]; then
    printf 'VM HTTP output is missing %q:\n%s\n' "$expected" "$vm_output" >&2
    exit 1
  fi
done

echo "Checking and running WebSocket example in the VM..."
go run . check code_examples/core/websocket.zum
ws_vm_output="$(go run . run code_examples/core/websocket.zum 2>/dev/null)"
expected_ws=$'text\necho:ping\nclose\ntrue'
if [[ "$ws_vm_output" != "$expected_ws" ]]; then
  printf 'Unexpected VM WebSocket output:\n%s\n' "$ws_vm_output" >&2
  exit 1
fi

echo "Building HTTP API with Clang and GCC..."
go run . build --release --compiler clang -o build/z11-http-clang code_examples/core/http_api.zum
go run . build --release --compiler gcc -o build/z11-http-gcc code_examples/core/http_api.zum
clang_output="$(./build/z11-http-clang)"
gcc_output="$(./build/z11-http-gcc)"
if [[ "$clang_output" != "$vm_output" || "$gcc_output" != "$vm_output" ]]; then
  echo "VM/Clang/GCC HTTP output mismatch" >&2
  exit 1
fi

echo "Building WebSocket example with Clang and GCC..."
go run . build --release --compiler clang -o build/z11-ws-clang code_examples/core/websocket.zum
go run . build --release --compiler gcc -o build/z11-ws-gcc code_examples/core/websocket.zum
if [[ "$(./build/z11-ws-clang)" != "$expected_ws" || "$(./build/z11-ws-gcc)" != "$expected_ws" ]]; then
  echo "Clang/GCC WebSocket output mismatch" >&2
  exit 1
fi

echo "Running HTTP race checks..."
go test -race ./object ./object/builtins ./evaluator ./vm ./conformance

echo "Z11 HTTP, API and WebSocket tests passed."
