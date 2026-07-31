#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

for compiler in clang gcc; do
  if ! command -v "$compiler" >/dev/null 2>&1; then
    echo "Missing C compiler: $compiler" >&2
    exit 1
  fi
done

mkdir -p build
source="code_examples/core/gui_toolkit.zum"
graphical="code_examples/core/gui_window.zum"
theme_source="code_examples/core/gui_theme.zum"
expected=$'headless\nZumbra UI\ntrue\nclicked\n640\ntrue\ntrue'

echo "Running the complete Z14 Go test suite..."
go test ./...

echo "Checking and running the deterministic GUI example in the VM..."
go run . check "$source"
vm_output="$(ZUMBRA_DESKTOP_HEADLESS=1 go run . run "$source")"
if [[ "$vm_output" != "$expected" ]]; then
  printf 'Unexpected VM GUI output:\n%s\n' "$vm_output" >&2
  exit 1
fi

echo "Checking the real SDL3 GUI example without opening a window..."
go run . check "$graphical"

echo "Checking dynamic dark theme and UTF-8 state in the VM..."
go run . check "$theme_source"
theme_vm_output="$(ZUMBRA_DESKTOP_HEADLESS=1 go run . run "$theme_source")"
if [[ "$theme_vm_output" != $'#141822\ntrue' ]]; then
  printf 'Unexpected VM theme output:\n%s\n' "$theme_vm_output" >&2
  exit 1
fi

for compiler in clang gcc; do
  echo "Building the headless GUI example with ${compiler}..."
  headless_output="build/z14-gui-headless-${compiler}"
  go run . build --release --compiler "$compiler" -o "$headless_output" "$source"
  native_output="$(ZUMBRA_DESKTOP_HEADLESS=1 "./$headless_output")"
  if [[ "$native_output" != "$expected" ]]; then
    printf 'VM/%s GUI output mismatch:\n%s\n' "$compiler" "$native_output" >&2
    exit 1
  fi

  echo "Building the real SDL3 GUI example with ${compiler}..."
  go run . build --release --compiler "$compiler" -o "build/z14-gui-window-${compiler}" "$graphical"

  echo "Building dynamic theme validation with ${compiler}..."
  theme_output="build/z14-gui-theme-${compiler}"
  go run . build --release --compiler "$compiler" -o "$theme_output" "$theme_source"
  theme_native_output="$(ZUMBRA_DESKTOP_HEADLESS=1 "./$theme_output")"
  if [[ "$theme_native_output" != $'#141822\ntrue' ]]; then
    printf 'VM/%s theme output mismatch:\n%s\n' "$compiler" "$theme_native_output" >&2
    exit 1
  fi
done

echo "Running GUI race checks..."
ZUMBRA_DESKTOP_HEADLESS=1 go test -race ./object ./object/builtins ./evaluator ./vm ./conformance

echo "Rechecking the Z13 desktop runtime and earlier milestones..."
scripts/test-desktop.sh

echo "Z14 GUI toolkit tests passed."

sdl_found=false
if ldconfig -p 2>/dev/null | grep -q 'libSDL3\.so'; then
  sdl_found=true
elif find /usr/lib /usr/local/lib /lib "${HOME}/.local/lib" -maxdepth 5 -name 'libSDL3.so*' -print -quit 2>/dev/null | grep -q .; then
  sdl_found=true
elif [[ -n "${LD_LIBRARY_PATH:-}" ]] && find ${LD_LIBRARY_PATH//:/ } -maxdepth 2 -name 'libSDL3.so*' -print -quit 2>/dev/null | grep -q .; then
  sdl_found=true
fi

ttf_found=false
if ldconfig -p 2>/dev/null | grep -q 'libSDL3_ttf\.so'; then
  ttf_found=true
elif find /usr/lib /usr/local/lib /lib "${HOME}/.local/lib" -maxdepth 5 -name 'libSDL3_ttf.so*' -print -quit 2>/dev/null | grep -q .; then
  ttf_found=true
elif [[ -n "${LD_LIBRARY_PATH:-}" ]] && find ${LD_LIBRARY_PATH//:/ } -maxdepth 2 -name 'libSDL3_ttf.so*' -print -quit 2>/dev/null | grep -q .; then
  ttf_found=true
fi

if [[ "$sdl_found" == false ]]; then
  echo "Note: SDL3 was not found by the filesystem probe. Headless validation passed; run the graphical example to verify dynamic loading."
else
  echo "SDL3 shared library detected."
fi

if [[ "$ttf_found" == false ]]; then
  echo "Note: SDL3_ttf was not found. GUI text will use the debug fallback until SDL3_ttf is installed or exposed through LD_LIBRARY_PATH."
else
  echo "SDL3_ttf shared library detected. Production font rendering is available."
fi

echo "Run: go run . run $graphical"
