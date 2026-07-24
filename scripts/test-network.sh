#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "Testing Z10 network packages..."
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

echo "Checking and running local TCP, UDP and DNS example..."
go run . check code_examples/core/network.zum
vm_output="$(go run . run code_examples/core/network.zum 2>/dev/null)"
expected_network=$'true\ntrue\n2\ntrue'
if [[ "$vm_output" != "$expected_network" ]]; then
  printf 'Unexpected VM network output:\n%s\n' "$vm_output" >&2
  exit 1
fi

echo "Building POSIX socket runtime with Clang and GCC..."
go run . build --release --compiler clang -o build/z10-network-clang code_examples/core/network.zum
go run . build --release --compiler gcc -o build/z10-network-gcc code_examples/core/network.zum
clang_output="$(./build/z10-network-clang)"
gcc_output="$(./build/z10-network-gcc)"
if [[ "$clang_output" != "$expected_network" || "$gcc_output" != "$expected_network" ]]; then
  echo "Clang/GCC network output mismatch" >&2
  exit 1
fi

if ! pkg-config --exists openssl; then
  echo "OpenSSL development files are required for Z10 TLS tests." >&2
  echo "Debian/Ubuntu: sudo apt install libssl-dev pkg-config" >&2
  exit 1
fi

echo "Checking and building TLS 1.2+ example..."
go run . check code_examples/core/network_tls.zum
tls_vm_output="$(go run . run code_examples/core/network_tls.zum 2>/dev/null)"
if [[ "$tls_vm_output" != "4" ]]; then
  printf 'Unexpected TLS VM output: %s\n' "$tls_vm_output" >&2
  exit 1
fi
go run . build --release --compiler clang -o build/z10-tls-clang code_examples/core/network_tls.zum
go run . build --release --compiler gcc -o build/z10-tls-gcc code_examples/core/network_tls.zum
if [[ "$(./build/z10-tls-clang)" != "4" || "$(./build/z10-tls-gcc)" != "4" ]]; then
  echo "Clang/GCC TLS output mismatch" >&2
  exit 1
fi

echo "Running network race checks..."
go test -race ./object ./object/builtins ./evaluator ./vm ./conformance

echo "Z10 network tests passed."
