#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

printf 'Running Z16 language foundation tests...\n'
go test ./mir ./nativec ./conformance ./object/builtins

go build -o "$TMP/zumbra" .
"$TMP/zumbra" check code_examples/core/ui_event_target.zum

VM_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/zumbra" run code_examples/core/ui_event_target.zum)"
EXPECTED=$'edit-42\nbutton'
if [[ "$VM_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected VM output:\n%s\n' "$VM_OUTPUT" >&2
  exit 1
fi

"$TMP/zumbra" build --release -o "$TMP/ui-event-target" code_examples/core/ui_event_target.zum
NATIVE_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/ui-event-target")"
if [[ "$NATIVE_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected native output:\n%s\n' "$NATIVE_OUTPUT" >&2
  exit 1
fi

printf 'Z16 language foundation tests passed.\n'
