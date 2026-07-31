#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

go test ./lexer ./ast ./parser ./object ./code ./compiler ./semantic ./types ./evaluator ./vm ./conformance ./runtime ./transpiler ./docs

go run . run code_examples/core/structured_types.zum
