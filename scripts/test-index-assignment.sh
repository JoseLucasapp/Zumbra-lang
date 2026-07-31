#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

go test \
  ./ast \
  ./parser \
  ./semantic \
  ./types \
  ./code \
  ./compiler \
  ./evaluator \
  ./vm \
  ./conformance \
  ./transpiler

go run . run code_examples/core/index_assignment.zum
