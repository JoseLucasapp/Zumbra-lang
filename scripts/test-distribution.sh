#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
MANIFEST="code_examples/desktop_package/zumbra.toml"
TOOLS="$(mktemp -d)"
trap 'rm -rf "$TOOLS"' EXIT
CLI="$TOOLS/zumbra"
BINARY="$TOOLS/zumbra-packaged-app"
PACKAGES="$TOOLS/linux-packages"
EXPECTED=$'Olá do asset incorporado!\n\ntrue\n5'

printf 'Running complete Z15 tests...\n'
go test ./appmanifest ./appdist
go test ./nativec -run 'Test(WindowsResourceContainsApplicationMetadata|ExecutableNameUsesTarget|DetectTargetCompilerFromEnvironment|DesktopRuntimeContainsWindowsAndMacOSBackends|Z15AssetRuntimeIsConditionallyEnabled|Z15BuildWritesEmbeddedAssetSource|Z15EmbeddedAssetRunsNatively)$'

printf 'Building the Zumbra CLI once for integration tests...\n'
go build -o "$CLI" .

printf 'Inspecting application manifest...\n'
"$CLI" app inspect --manifest "$MANIFEST" > "$TOOLS/inspect.json"
grep -q 'icon_windows' "$TOOLS/inspect.json"
grep -q 'assets/message.txt' "$TOOLS/inspect.json"

printf 'Running application with embedded assets in the VM...\n'
VM_OUTPUT="$("$CLI" app run --manifest "$MANIFEST")"
if [[ "$VM_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected VM output:\n%s\n' "$VM_OUTPUT" >&2
  exit 1
fi

printf 'Building reproducible native application...\n'
SOURCE_DATE_EPOCH=1700000000 "$CLI" app build --manifest "$MANIFEST" --compiler "${CC:-auto}" -o "$BINARY"
NATIVE_OUTPUT="$("$BINARY")"
if [[ "$NATIVE_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected native output:\n%s\n' "$NATIVE_OUTPUT" >&2
  exit 1
fi

printf 'Building the cross-platform desktop/GUI smoke test in headless mode...\n'
ZUMBRA_DESKTOP_HEADLESS=1 "$CLI" build --release -o "$TOOLS/z15-desktop-gui-smoke" code_examples/core/desktop_gui_smoke.zum
GUI_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TOOLS/z15-desktop-gui-smoke")"
if [[ "$GUI_OUTPUT" != $'headless\ntrue' ]]; then
  printf 'Unexpected desktop/GUI smoke output:\n%s\n' "$GUI_OUTPUT" >&2
  exit 1
fi

printf 'Checking that operational doctor errors do not print global usage...\n'
if "$CLI" app doctor --manifest "$MANIFEST" --target linux --arch amd64 --format appimage --appimagetool /definitely/missing > "$TOOLS/doctor-error.log" 2>&1; then
  printf 'Doctor unexpectedly accepted a missing appimagetool.\n' >&2
  exit 1
fi
if grep -q '^Usage:' "$TOOLS/doctor-error.log"; then
  printf 'Operational doctor error unexpectedly printed global usage.\n' >&2
  exit 1
fi

cat > "$TOOLS/appimagetool" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
for output do :; done
printf '\177ELF\002\001\001' > "$output"
printf ' deterministic AppImage test artifact\n' >> "$output"
chmod +x "$output"
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
"$CLI" app doctor --manifest "$MANIFEST" --target linux --arch amd64 --format bundle,deb,appdir
"$CLI" app doctor --manifest "$MANIFEST" --target linux --arch amd64 --format appimage --appimagetool "$TOOLS/appimagetool"

printf 'Packaging Linux bundle, DEB and AppDir...\n'
SOURCE_DATE_EPOCH=1700000000 "$CLI" app package \
  --manifest "$MANIFEST" --target linux --arch amd64 \
  --binary "$BINARY" --format bundle,deb,appdir --output-dir "$PACKAGES"

test -f "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.tar.gz"
test -f "$PACKAGES/zumbra-packaged-app_1.0.0_amd64.deb"
test -x "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppDir/AppRun"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppDir/zumbra-packaged-app.desktop"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppDir/usr/share/metainfo/zumbra-packaged-app.appdata.xml"

printf 'Testing AppImage integration in an isolated output directory...\n'
APPIMAGE_PACKAGES="$TOOLS/appimage-packages"
SOURCE_DATE_EPOCH=1700000000 "$CLI" app package \
  --manifest "$MANIFEST" --target linux --arch amd64 \
  --binary "$BINARY" --format appimage \
  --output-dir "$APPIMAGE_PACKAGES" --appimagetool "$TOOLS/appimagetool"
test -x "$APPIMAGE_PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppImage"
test ! -e "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppImage"
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
"$CLI" app doctor --manifest "$MANIFEST" --target windows --arch amd64 --format all --binary "$PE_BINARY" --makensis "$TOOLS/makensis"

printf 'Packaging Windows portable ZIP and NSIS installer layout...\n'
WINDOWS_PACKAGES="$TOOLS/windows-packages"
SOURCE_DATE_EPOCH=1700000000 "$CLI" app package \
  --manifest "$MANIFEST" --target windows --arch amd64 \
  --binary "$PE_BINARY" --format all --output-dir "$WINDOWS_PACKAGES" \
  --makensis "$TOOLS/makensis"
test -f "$WINDOWS_PACKAGES/zumbra-packaged-app-1.0.0-windows-amd64-portable.zip"
test -f "$WINDOWS_PACKAGES/zumbra-packaged-app-1.0.0-windows-amd64-setup.exe"
test -f "$WINDOWS_PACKAGES/zumbra-packaged-app-1.0.0-windows-amd64-installer.nsi"

printf 'Checking macOS package prerequisites with a Mach-O header...\n'
"$CLI" app doctor --manifest "$MANIFEST" --target macos --arch arm64 --format all --binary "$MACHO_BINARY"

printf 'Packaging macOS application bundle and ZIP...\n'
MACOS_PACKAGES="$TOOLS/macos-packages"
SOURCE_DATE_EPOCH=1700000000 "$CLI" app package \
  --manifest "$MANIFEST" --target macos --arch arm64 \
  --binary "$MACHO_BINARY" --format all --output-dir "$MACOS_PACKAGES"
test -f "$MACOS_PACKAGES/Zumbra Packaged App.app/Contents/Info.plist"
test -x "$MACOS_PACKAGES/Zumbra Packaged App.app/Contents/MacOS/zumbra-packaged-app"
test -f "$MACOS_PACKAGES/zumbra-packaged-app-1.0.0-macos-arm64.zip"

printf 'Rejecting foreign target binaries...\n'
if "$CLI" app package --manifest "$MANIFEST" --target windows --arch amd64 --binary "$BINARY" --format portable >"$TOOLS/foreign.log" 2>&1; then
  printf 'Foreign binary package unexpectedly succeeded.\n' >&2
  exit 1
fi
grep -q 'binary target mismatch' "$TOOLS/foreign.log"

printf 'Checking reproducible Linux archives...\n'
REPRO_A="$(mktemp -d)"
REPRO_B="$(mktemp -d)"
SOURCE_DATE_EPOCH=1700000000 "$CLI" app package --manifest "$MANIFEST" --target linux --arch amd64 --binary "$BINARY" --format bundle,deb --output-dir "$REPRO_A" >/dev/null
SOURCE_DATE_EPOCH=1700000000 "$CLI" app package --manifest "$MANIFEST" --target linux --arch amd64 --binary "$BINARY" --format bundle,deb --output-dir "$REPRO_B" >/dev/null
cmp "$REPRO_A/zumbra-packaged-app-1.0.0-linux-amd64.tar.gz" "$REPRO_B/zumbra-packaged-app-1.0.0-linux-amd64.tar.gz"
cmp "$REPRO_A/zumbra-packaged-app_1.0.0_amd64.deb" "$REPRO_B/zumbra-packaged-app_1.0.0_amd64.deb"
rm -rf "$REPRO_A" "$REPRO_B"

grep -q 'zumbra-packaged-app-1.0.0' "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64-SHA256SUMS.txt"
grep -q 'stable' "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64-update.json"
grep -q '"binary"' "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64-package-report.json"

printf 'Z15 desktop distribution tests passed.\n'
