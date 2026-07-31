#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

go test \
  ./object \
  ./numeric \
  ./lexer \
  ./parser \
  ./types \
  ./compiler \
  ./evaluator \
  ./vm \
  ./conformance \
  ./runtime \
  ./transpiler

go run . run code_examples/core/fixed_integers.zum
