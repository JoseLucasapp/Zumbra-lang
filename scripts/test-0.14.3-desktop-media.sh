#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

echo "Running Zumbra 0.14.5 desktop media regression tests..."

go test ./object/builtins -run 'TestDesktopMediaHeadlessContract|TestProcessArgsAndUnixTimeBuiltins' -count=1
go test ./types -run 'TestZumbra0143DesktopMediaAndProcessBuiltinsAreTyped' -count=1
go test ./nativec -run 'TestZumbra0143DesktopMediaBuildsNatively|TestZ13DesktopRuntimeIsConditionallyEnabled' -count=1

clang -std=c11 -Wall -Wextra -Werror \
  -fsyntax-only nativec/runtime/zumbra_runtime.c \
  -I nativec/runtime \
  $(pkg-config --cflags sqlite3 2>/dev/null || true)

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

go build -o "$tmp_dir/zumbra" .
[[ "$("$tmp_dir/zumbra" --version)" == "0.14.5" ]]

cat > "$tmp_dir/media.zum" <<'ZUM'
var app << desktopApp({"backend":"headless","name":"Media","version":"0.14.5"});
var window << desktopWindow(app,{"title":"Media","width":2,"height":2});
var pixels << bytes(16);
show(desktopWindowPresentRGBA(window,pixels,2,2));
desktopWindowSetVSync(window,true);
show(desktopKeyDown(app,29));
show(desktopGamepadButton(app,1,0));
var samples << bytes(4);
fill(samples,32u8);
show(desktopAudioQueue(app,samples,80,false));
show(desktopAudioQueued(app));
show(sizeOf(processArgs()) > 0);
show(unixTimeSeconds() > 0u64);
show(createFile("data/zumbra-runtime.txt","0.14.5"));
desktopClose(app);
ZUM

"$tmp_dir/zumbra" build "$tmp_dir/media.zum" -o "$tmp_dir/media"
output="$(cd "$tmp_dir" && ZUMBRA_DESKTOP_HEADLESS=1 ./media game.nes)"
expected=$'true\nfalse\nfalse\n4\n0\ntrue\ntrue\ndata/zumbra-runtime.txt'
[[ -f "$tmp_dir/data/zumbra-runtime.txt" ]]
[[ "$(cat "$tmp_dir/data/zumbra-runtime.txt")" == "0.14.5" ]]
[[ "$output" == "$expected" ]] || {
  printf 'unexpected media output:\n%s\n' "$output" >&2
  exit 1
}

echo "Zumbra 0.14.5 desktop media regression tests passed."
