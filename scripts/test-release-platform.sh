#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

host_name="${RUNNER_OS:-}"
if [[ -z "$host_name" ]]; then
  case "$(uname -s)" in
    Linux*) host_name="Linux" ;;
    Darwin*) host_name="macOS" ;;
    MINGW*|MSYS*|CYGWIN*) host_name="Windows" ;;
    *) host_name="$(uname -s)" ;;
  esac
fi

portable_packages=(
  .
  ./appmanifest
  ./ast
  ./benchmark
  ./binarydata
  ./builtinspec
  ./cbinding
  ./code
  ./collections
  ./compiler
  ./diagnostics
  ./docs
  ./evaluator
  ./hir
  ./lexer
  ./mir
  ./modules
  ./numeric
  ./object
  ./parser
  ./pipeline
  ./repl
  ./runtime
  ./semantic
  ./token
  ./tooling/docgen
  ./tooling/formatter
  ./tooling/lint
  ./tooling/lsp
  ./tooling/profile
  ./tooling/project
  ./transpiler
  ./types
  ./vm
)

echo "Running release validation on ${host_name}..."

case "$host_name" in
  Linux)
    # Linux is the canonical full-suite host because the current native runtime,
    # SDL backend and several distribution tests intentionally target Linux.
    go test ./...
    go vet -unsafeptr=false ./...
    scripts/test-z18-tooling.sh
    ;;
  Windows|macOS)
    # Validate the compiler/CLI and all host-portable packages. Packages omitted
    # here contain Linux-only native-runtime tests or external packaging-tool
    # assertions and are exercised by the Linux canonical gate instead.
    go test "${portable_packages[@]}"
    go vet -unsafeptr=false "${portable_packages[@]}"
    ;;
  *)
    echo "Unsupported release host: ${host_name}" >&2
    exit 1
    ;;
esac

exe_suffix=""
if [[ "$host_name" == "Windows" ]]; then
  exe_suffix=".exe"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

go build -trimpath -o "$tmp_dir/zumbra${exe_suffix}" .
actual_version="$("$tmp_dir/zumbra${exe_suffix}" --version | tr -d '\r\n')"

expected_version="${EXPECTED_VERSION:-}"
expected_version="${expected_version#v}"
if [[ -n "$expected_version" && "$actual_version" != "$expected_version" ]]; then
  echo "Version mismatch: expected ${expected_version}, got ${actual_version}" >&2
  exit 1
fi

echo "Release host validation passed: ${host_name} (${actual_version})"
