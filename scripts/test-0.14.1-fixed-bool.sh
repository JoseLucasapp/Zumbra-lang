#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

printf 'Running Zumbra 0.14.1 fixed-boolean regression tests...\n'

go test ./evaluator -run 'TestBangOperatorUsesFixedComparisonBooleanValue|TestBangOperatorUsesBooleanValueStoredInStruct' -count=1
go test ./vm -run 'TestBangOperatorUsesFixedComparisonBooleanValueOnVM' -count=1
go test ./conformance -run 'TestFixedWidthComparisonBooleanConformance' -count=1

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

go build -o "$tmp_dir/zumbra" .
[[ "$($tmp_dir/zumbra version)" == "0.14.1" ]]

cat > "$tmp_dir/fixed-bool.zum" <<'ZUM'
struct Header { battery: bool; }
var flags << 0u8;
var parsed << Header((flags band 0x02u8) != 0u8);
if (!parsed.battery) {
    show("fixed boolean ok");
} else {
    panic("fixed boolean regression");
}
ZUM

[[ "$($tmp_dir/zumbra "$tmp_dir/fixed-bool.zum")" == "fixed boolean ok" ]]

printf 'Zumbra 0.14.1 fixed-boolean regression tests passed.\n'
