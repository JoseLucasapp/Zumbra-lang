#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

go test \
  ./collections \
  ./object \
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

go run . run code_examples/core/memory_collections.zum
