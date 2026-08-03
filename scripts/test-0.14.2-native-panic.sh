#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

printf 'Running Zumbra 0.14.2 native panic regression tests...\n'

go test ./nativec -run 'TestNativePanic|TestNativeProgramWithDormantPanic|TestNativeImportedModuleWithPanicBuilds' -count=1

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

go build -o "$tmp_dir/zumbra" .
[[ "$("$tmp_dir/zumbra" version)" == "0.14.2" ]]

cat > "$tmp_dir/dormant-panic.zum" <<'ZUM'
var ok << true;
if (!ok) {
    panic("must not execute");
}
show("native panic path compiled");
ZUM

"$tmp_dir/zumbra" build -o "$tmp_dir/dormant-panic" "$tmp_dir/dormant-panic.zum"
[[ "$("$tmp_dir/dormant-panic")" == "native panic path compiled" ]]

cat > "$tmp_dir/triggered-panic.zum" <<'ZUM'
panic("native panic smoke");
ZUM

"$tmp_dir/zumbra" build -o "$tmp_dir/triggered-panic" "$tmp_dir/triggered-panic.zum"
set +e
"$tmp_dir/triggered-panic" >"$tmp_dir/stdout.txt" 2>"$tmp_dir/stderr.txt"
status=$?
set -e
[[ "$status" -ne 0 ]]
[[ ! -s "$tmp_dir/stdout.txt" ]]
grep -q 'panic: native panic smoke' "$tmp_dir/stderr.txt"

printf 'Zumbra 0.14.2 native panic regression tests passed.\n'
