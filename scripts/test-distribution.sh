#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
MANIFEST="code_examples/desktop_package/zumbra.toml"
OUTPUT="code_examples/desktop_package/dist/zumbra-packaged-app"

printf 'Running Z15.1 tests...\n'
go test ./appmanifest ./types ./object/builtins ./nativec ./conformance

printf 'Inspecting desktop manifest...\n'
go run . app inspect --manifest "$MANIFEST" >/tmp/zumbra-z15-inspect.json
grep -q 'assets/message.txt' /tmp/zumbra-z15-inspect.json

printf 'Running application in the VM...\n'
VM_OUTPUT="$(go run . app run --manifest "$MANIFEST")"
EXPECTED=$'Olá do asset incorporado!\n\ntrue\n2'
if [[ "$VM_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected VM output:\n%s\n' "$VM_OUTPUT" >&2
  exit 1
fi

printf 'Building application with embedded assets...\n'
rm -rf code_examples/desktop_package/build code_examples/desktop_package/dist
go run . app build --manifest "$MANIFEST" --compiler "${CC:-auto}"
NATIVE_OUTPUT="$("$OUTPUT")"
if [[ "$NATIVE_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected native output:\n%s\n' "$NATIVE_OUTPUT" >&2
  exit 1
fi

test -f "$OUTPUT.manifest.json"
grep -q 'assets/message.txt' "$OUTPUT.manifest.json"

printf 'Z15.1 distribution tests passed.\n'
