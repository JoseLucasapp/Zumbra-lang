#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

go test \
  ./builtinspec \
  ./types \
  ./semantic \
  ./hir \
  ./mir \
  ./pipeline \
  ./compiler \
  ./evaluator \
  ./conformance \
  ./transpiler \
  ./docs

go run . check code_examples/core/typed_ir.zum

HIR_OUTPUT="$(mktemp)"
MIR_OUTPUT="$(mktemp)"
trap 'rm -f "$HIR_OUTPUT" "$MIR_OUTPUT"' EXIT

go run . ir code_examples/core/typed_ir.zum hir > "$HIR_OUTPUT"
go run . ir code_examples/core/typed_ir.zum optimized > "$MIR_OUTPUT"

grep -q 'struct name="Counter"' "$HIR_OUTPUT"
grep -q 'var name="label" : string' "$HIR_OUTPUT"
grep -q 'const value="14" : int' "$MIR_OUTPUT"
grep -q 'declare "label".*: string' "$MIR_OUTPUT"

EXPECTED=$'14\n19\nrunning'
ACTUAL="$(go run . run code_examples/core/typed_ir.zum)"
if [[ "$ACTUAL" != "$EXPECTED" ]]; then
  printf 'unexpected example output\nexpected:\n%s\nactual:\n%s\n' "$EXPECTED" "$ACTUAL" >&2
  exit 1
fi

printf 'Z6 typed IR pipeline tests passed.\n'
