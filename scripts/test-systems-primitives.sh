#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

go test ./lexer ./parser ./types ./evaluator ./code ./compiler ./vm ./conformance \
  -run 'System|Bitwise|BitNot|Shift'
