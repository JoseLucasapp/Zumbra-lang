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
go test ./appmanifest ./appdist ./nativec

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

printf 'Packaging Linux bundle, DEB and AppDir...\n'
SOURCE_DATE_EPOCH=1700000000 go run . app package \
  --manifest "$MANIFEST" --target linux --arch amd64 \
  --binary dist/zumbra-packaged-app --format bundle,deb,appdir

test -f "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.tar.gz"
test -f "$PACKAGES/zumbra-packaged-app_1.0.0_amd64.deb"
test -x "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppDir/AppRun"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppDir/zumbra-packaged-app.desktop"
if command -v dpkg-deb >/dev/null 2>&1; then
  dpkg-deb --info "$PACKAGES/zumbra-packaged-app_1.0.0_amd64.deb" >/dev/null
fi

printf 'Packaging AppImage through a deterministic test tool...\n'
TOOLS="$(mktemp -d)"
trap 'rm -rf "$TOOLS"' EXIT
cat > "$TOOLS/appimagetool" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
printf 'APPIMAGE\n' > "$2"
chmod +x "$2"
TOOL
chmod +x "$TOOLS/appimagetool"
SOURCE_DATE_EPOCH=1700000000 go run . app package \
  --manifest "$MANIFEST" --target linux --arch amd64 \
  --binary dist/zumbra-packaged-app --format appimage \
  --appimagetool "$TOOLS/appimagetool"
test -x "$PACKAGES/zumbra-packaged-app-1.0.0-linux-amd64.AppImage"

printf 'Packaging Windows portable ZIP and NSIS installer layout...\n'
cat > "$TOOLS/makensis" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
out=$(sed -n 's/^OutFile "\(.*\)"/\1/p' "$1")
printf 'MZ test installer\n' > "$out"
TOOL
chmod +x "$TOOLS/makensis"
SOURCE_DATE_EPOCH=1700000000 go run . app package \
  --manifest "$MANIFEST" --target windows --arch amd64 \
  --binary dist/zumbra-packaged-app --format all \
  --makensis "$TOOLS/makensis"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-windows-amd64-portable.zip"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-windows-amd64-setup.exe"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-windows-amd64-installer.nsi"

printf 'Packaging macOS application bundle and ZIP...\n'
SOURCE_DATE_EPOCH=1700000000 go run . app package \
  --manifest "$MANIFEST" --target macos --arch arm64 \
  --binary dist/zumbra-packaged-app --format all
test -f "$PACKAGES/Zumbra Packaged App.app/Contents/Info.plist"
test -x "$PACKAGES/Zumbra Packaged App.app/Contents/MacOS/zumbra-packaged-app"
test -f "$PACKAGES/zumbra-packaged-app-1.0.0-macos-arm64.zip"

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

printf 'Z15 desktop distribution tests passed.\n'
