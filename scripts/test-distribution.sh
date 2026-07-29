#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
MANIFEST="code_examples/desktop_package/zumbra.toml"
PROJECT="code_examples/desktop_package"
BINARY="$PROJECT/dist/zumbra-packaged-app"
PACKAGES="$PROJECT/dist/packages"
EXPECTED=$'Olá do asset incorporado!\n\ntrue\n5'

printf 'Running complete Z15 tests...\n'
go test ./appmanifest ./appdist
go test ./nativec -run 'Test(WindowsResourceContainsApplicationMetadata|ExecutableNameUsesTarget|DetectTargetCompilerFromEnvironment|Z15AssetRuntimeIsConditionallyEnabled|Z15BuildWritesEmbeddedAssetSource|Z15EmbeddedAssetRunsNatively)$'

printf 'Inspecting application manifest...\n'
go run . app inspect --manifest "$MANIFEST" > /tmp/zumbra-z15-inspect.json
grep -q 'icon_windows' /tmp/zumbra-z15-inspect.json
grep -q 'assets/message.txt' /tmp/zumbra-z15-inspect.json

printf 'Running application with embedded assets in the VM...\n'
VM_OUTPUT="$(go run . app run --manifest "$MANIFEST")"
if [[ "$VM_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected VM output:\n%s\n' "$VM_OUTPUT" >&2
  exit 1
fi

printf 'Building reproducible native application...\n'
rm -rf "$PROJECT/build" "$PROJECT/dist"
SOURCE_DATE_EPOCH=1700000000 go run . app build --manifest "$MANIFEST" --compiler "${CC:-auto}"
NATIVE_OUTPUT="$("$BINARY")"
if [[ "$NATIVE_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected native output:\n%s\n' "$NATIVE_OUTPUT" >&2
  exit 1
fi

TOOLS="$(mktemp -d)"
trap 'rm -rf "$TOOLS"' EXIT

cat > "$TOOLS/appimagetool" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
printf '\177ELF\002\001\001' > "$2"
printf ' deterministic AppImage test artifact\n' >> "$2"
chmod +x "$2"
TOOL
chmod +x "$TOOLS/appimagetool"

cat > "$TOOLS/makensis" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
out=$(sed -n 's/^OutFile "\(.*\)"/\1/p' "$1")
python3 - "$out" <<'PY'
import pathlib, struct, sys
p=pathlib.Path(sys.argv[1]); data=bytearray(256); data[0:2]=b'MZ'; struct.pack_into('<I',data,0x3c,0x80); data[0x80:0x84]=b'PE\0\0'; struct.pack_into('<H',data,0x84,0x8664); p.write_bytes(data)
PY
TOOL
chmod +x "$TOOLS/makensis"

printf 'Checking Linux package prerequisites...\n'
go run . app doctor --manifest "$MANIFEST" --target linux --arch amd64 --format bundle,deb,appdir
go run . app doctor --manifest "$MANIFEST" --target linux --arch amd64 --format appimage --appimagetool "$TOOLS/appimagetool"

printf 'Packaging Linux bundle, DEB, AppDir and AppImage...\n'
SOURCE_DATE_EPOCH=1700000000 go run . app package \
  --manifest "$MANIFEST" --target linux --arch amd64 \
  --binary dist/zumbra-packaged-app --format all \
  --appimagetool "$TOOLS/appimagetool"

test -f "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.tar.gz"
test -f "$PACKAGES/zumbra-packaged-app_1.0.0_amd64.deb"
test -x "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppDir/AppRun"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppDir/zumbra-packaged-app.desktop"
test -x "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppImage"
if command -v dpkg-deb >/dev/null 2>&1; then
  dpkg-deb --info "$PACKAGES/zumbra-packaged-app_1.0.0_amd64.deb" >/dev/null
fi

printf 'Creating target-correct Windows and macOS package fixtures...\n'
PE_BINARY="$TOOLS/zumbra-packaged-app.exe"
MACHO_BINARY="$TOOLS/zumbra-packaged-app-macos"
python3 - "$PE_BINARY" "$MACHO_BINARY" <<'PY'
import pathlib, struct, sys
pe=bytearray(256); pe[0:2]=b'MZ'; struct.pack_into('<I',pe,0x3c,0x80); pe[0x80:0x84]=b'PE\0\0'; struct.pack_into('<H',pe,0x84,0x8664); pathlib.Path(sys.argv[1]).write_bytes(pe)
mo=bytearray(64); mo[0:4]=bytes([0xcf,0xfa,0xed,0xfe]); struct.pack_into('<I',mo,4,0x0100000c); pathlib.Path(sys.argv[2]).write_bytes(mo)
PY
chmod +x "$PE_BINARY" "$MACHO_BINARY"

printf 'Checking Windows package prerequisites with a real PE header...\n'
go run . app doctor --manifest "$MANIFEST" --target windows --arch amd64 --format all --binary "$PE_BINARY" --makensis "$TOOLS/makensis"

printf 'Packaging Windows portable ZIP and NSIS installer layout...\n'
SOURCE_DATE_EPOCH=1700000000 go run . app package \
  --manifest "$MANIFEST" --target windows --arch amd64 \
  --binary "$PE_BINARY" --format all \
  --makensis "$TOOLS/makensis"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-windows-amd64-portable.zip"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-windows-amd64-setup.exe"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-windows-amd64-installer.nsi"

printf 'Checking macOS package prerequisites with a Mach-O header...\n'
go run . app doctor --manifest "$MANIFEST" --target macos --arch arm64 --format all --binary "$MACHO_BINARY"

printf 'Packaging macOS application bundle and ZIP...\n'
SOURCE_DATE_EPOCH=1700000000 go run . app package \
  --manifest "$MANIFEST" --target macos --arch arm64 \
  --binary "$MACHO_BINARY" --format all
test -f "$PACKAGES/Zumbra Packaged App.app/Contents/Info.plist"
test -x "$PACKAGES/Zumbra Packaged App.app/Contents/MacOS/zumbra-packaged-app"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-macos-arm64.zip"

printf 'Rejecting foreign target binaries...\n'
if go run . app package --manifest "$MANIFEST" --target windows --arch amd64 --binary dist/zumbra-packaged-app --format portable >/tmp/zumbra-z15-foreign.log 2>&1; then
  printf 'Foreign binary package unexpectedly succeeded.\n' >&2
  exit 1
fi
grep -q 'binary target mismatch' /tmp/zumbra-z15-foreign.log

printf 'Checking reproducible Linux archives...\n'
REPRO_A="$(mktemp -d)"
REPRO_B="$(mktemp -d)"
SOURCE_DATE_EPOCH=1700000000 go run . app package --manifest "$MANIFEST" --target linux --arch amd64 --binary dist/zumbra-packaged-app --format bundle,deb --output-dir "$REPRO_A" >/dev/null
SOURCE_DATE_EPOCH=1700000000 go run . app package --manifest "$MANIFEST" --target linux --arch amd64 --binary dist/zumbra-packaged-app --format bundle,deb --output-dir "$REPRO_B" >/dev/null
cmp "$REPRO_A/zumbra-packaged-app-1.0.0-linux-amd64.tar.gz" "$REPRO_B/zumbra-packaged-app-1.0.0-linux-amd64.tar.gz"
cmp "$REPRO_A/zumbra-packaged-app_1.0.0_amd64.deb" "$REPRO_B/zumbra-packaged-app_1.0.0_amd64.deb"
rm -rf "$REPRO_A" "$REPRO_B"

grep -q 'zumbra-packaged-app-1.0.0' "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64-SHA256SUMS.txt"
grep -q 'stable' "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64-update.json"
grep -q '"binary"' "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64-package-report.json"

printf 'Z15 desktop distribution tests passed.\n'
