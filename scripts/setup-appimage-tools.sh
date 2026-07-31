#!/usr/bin/env bash
set -euo pipefail

VERSION="${APPIMAGETOOL_VERSION:-1.9.1}"
CACHE_ROOT="${XDG_CACHE_HOME:-$HOME/.cache}/zumbra/tools"
MACHINE="${APPIMAGETOOL_ARCH:-$(uname -m)}"

case "$MACHINE" in
  x86_64|amd64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="aarch64" ;;
  *)
    printf 'Unsupported AppImage architecture: %s\n' "$MACHINE" >&2
    exit 1
    ;;
esac

NAME="appimagetool-${ARCH}.AppImage"
URL="https://github.com/AppImage/appimagetool/releases/download/${VERSION}/${NAME}"
DEST="$CACHE_ROOT/$NAME"
TMP="$DEST.tmp"

mkdir -p "$CACHE_ROOT"
rm -f "$TMP"

if command -v curl >/dev/null 2>&1; then
  curl -fL "$URL" -o "$TMP"
elif command -v wget >/dev/null 2>&1; then
  wget -O "$TMP" "$URL"
else
  printf 'curl or wget is required to download appimagetool.\n' >&2
  exit 1
fi

chmod 0755 "$TMP"
"$TMP" --version >/dev/null
mv -f "$TMP" "$DEST"

printf 'Installed appimagetool: %s\n' "$DEST"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$DEST"
fi
printf 'Zumbra discovers this cache path automatically.\n'
