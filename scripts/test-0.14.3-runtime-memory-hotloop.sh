#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

zumbra_bin="${ZUMBRA_BIN:-./build/zumbra}"
if [[ ! -x "$zumbra_bin" ]]; then
    zumbra_bin="${ZUMBRA_BIN:-zumbra}"
fi
command -v "$zumbra_bin" >/dev/null 2>&1 || { echo "Zumbra CLI not found: $zumbra_bin" >&2; exit 1; }

work="build/runtime-memory-hotloop"
rm -rf "$work"
mkdir -p "$work"
cat > "$work/probe.zum" <<'ZUM'
struct Pair {
    left: int;
    right: int;
}

fct makePair(value) {
    Pair(value, value + 1);
}

var baselineStats << runtimeMemoryStats();
var baseline << toInt(baselineStats["activeBytes"]);
var mark << runtimeMemoryMark();
var index << 0;
var checksum << 0;
while (index < 200000) {
    var pair << makePair(index);
    checksum << (checksum + pair.left + pair.right) band 1048575;
    runtimeMemoryReset(mark);
    index << index + 1;
}
runtimeMemoryReset(mark);
var afterStats << runtimeMemoryStats();
var after << toInt(afterStats["activeBytes"]);
var delta << after - baseline;
if (delta < 0) {
    delta << 0;
}
if (delta > 262144) {
    panic("runtime memory hot loop grew by " + toString(delta) + " bytes");
}
show("runtime-memory-hotloop: ok");
show(delta);
show(checksum);
ZUM

"$zumbra_bin" build --release "$work/probe.zum" -o "$work/probe"
"$work/probe" | tee "$work/output.txt"
grep -q '^runtime-memory-hotloop: ok$' "$work/output.txt"
