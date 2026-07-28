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

if [[ "$sdl_found" == false ]]; then
  echo "Note: SDL3 was not found by the filesystem probe. Headless validation passed; run the graphical example to verify dynamic loading."
else
  echo "SDL3 shared library detected. Run: go run . run $graphical"
fi
