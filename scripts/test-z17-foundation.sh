#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

printf 'Running Z17 systems programming tests...\n'

go test \
  ./compiler \
  ./mir \
  ./types \
  ./conformance \
  ./object/builtins \
  ./evaluator \
  ./vm

go test ./nativec -run 'TestZ17' -count=1

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir" /tmp/zumbra-z17-mapping.bin' EXIT

go build -o "$tmp_dir/zumbra" .
zumbra="$tmp_dir/zumbra"

examples=(
  code_examples/core/systems_programming.zum
  code_examples/core/systems_mapping.zum
  code_examples/core/systems_process.zum
  code_examples/core/systems_dynamic.zum
)

for example in "${examples[@]}"; do
  "$zumbra" check "$example"
done

check_output() {
  local actual="$1"
  local expected="$2"
  local label="$3"
  if [[ "$actual" != "$expected" ]]; then
    printf 'Unexpected output for %s.\nExpected:\n%s\nActual:\n%s\n' "$label" "$expected" "$actual" >&2
    exit 1
  fi
}

programming_expected=$'30\n4\n16\n20\ntrue\n80\ntrue\n7\ntrue\n9\ntrue\n7\n1\ntrue\ntrue\n16\n4\n8\n55\n56\ntrue\ntrue\ntrue'
mapping_expected=$'8\ntrue\ntrue\n42\n99'
process_expected=$'true\n7'
dynamic_expected=$'true\ntrue'

check_output "$("$zumbra" run code_examples/core/systems_programming.zum)" "$programming_expected" "systems_programming VM"
check_output "$("$zumbra" run code_examples/core/systems_mapping.zum)" "$mapping_expected" "systems_mapping VM"
check_output "$("$zumbra" run code_examples/core/systems_process.zum)" "$process_expected" "systems_process VM"
if [[ "$(uname -s)" == "Linux" ]]; then
  check_output "$("$zumbra" run code_examples/core/systems_dynamic.zum)" "$dynamic_expected" "systems_dynamic VM"
fi

run_native_and_check() {
  local executable="$1"
  local expected="$2"
  local label="$3"
  local output_file="$tmp_dir/${label//[^a-zA-Z0-9]/-}.out"
  printf 'Running %s...\n' "$label"
  if ! timeout -k 2s 30s env TERM=dumb "$executable" >"$output_file" 2>/dev/null; then
    printf 'Native executable failed or timed out: %s\n' "$label" >&2
    exit 1
  fi
  check_output "$(cat "$output_file")" "$expected" "$label"
}

# The nativec test package already builds all official Z17 examples with the
# default compiler. Explicit builds below validate the two supported C compilers
# and the sanitizer toolchain without repeating every expensive build.
for compiler in clang gcc; do
  if command -v "$compiler" >/dev/null 2>&1; then
    output="$tmp_dir/systems-programming-$compiler"
    "$zumbra" build --release --compiler "$compiler" -o "$output" code_examples/core/systems_programming.zum
    run_native_and_check "$output" "$programming_expected" "systems_programming native ($compiler)"
  fi
done

if command -v clang >/dev/null 2>&1; then
  sanitized="$tmp_dir/systems-programming-sanitized"
  "$zumbra" build --release --compiler clang --sanitize address,undefined -o "$sanitized" code_examples/core/systems_programming.zum
  output_file="$tmp_dir/systems-programming-sanitized.out"
  printf 'Running systems_programming ASan/UBSan...\n'
  if ! timeout -k 2s 30s env ASAN_OPTIONS=detect_leaks=1:abort_on_error=1 UBSAN_OPTIONS=halt_on_error=1 TERM=dumb "$sanitized" >"$output_file" 2>/dev/null; then
    printf 'Sanitized executable failed or timed out.\n' >&2
    exit 1
  fi
  check_output "$(cat "$output_file")" "$programming_expected" "systems_programming ASan/UBSan"
fi

printf 'Z17 systems programming tests passed.\n'
