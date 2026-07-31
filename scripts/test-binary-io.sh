#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

rm -f tmp/z4-example.bin

go test \
  ./binarydata \
  ./object/builtins \
  ./types \
  ./semantic \
  ./compiler \
  ./evaluator \
  ./vm \
  ./conformance \
  ./runtime \
  ./transpiler \
  ./docs

go run . run code_examples/core/binary_io.zum
rm -f tmp/z4-example.bin
