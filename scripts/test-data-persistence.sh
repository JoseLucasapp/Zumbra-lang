#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

for dependency in sqlite3 libpq hiredis; do
  if ! pkg-config --exists "$dependency"; then
    echo "Missing native dependency: $dependency" >&2
    echo "Debian/Ubuntu: sudo apt install libsqlite3-dev libpq-dev libhiredis-dev pkg-config" >&2
    exit 1
  fi
done

for compiler in clang gcc; do
  if ! command -v "$compiler" >/dev/null 2>&1; then
    echo "Missing C compiler: $compiler" >&2
    exit 1
  fi
done

mkdir -p build
rm -f build/z12-data.json build/z12-data.zb

echo "Running the complete Z12 Go test suite..."
go test ./...

run_portable_example() {
  local name="$1"
  local expected="$2"
  local source="code_examples/core/${name}.zum"

  echo "Checking ${source}..."
  go run . check "$source"
  local vm_output
  vm_output="$(go run . run "$source" 2>/dev/null)"
  if [[ "$vm_output" != "$expected" ]]; then
    printf 'Unexpected VM output for %s:\n%s\n' "$name" "$vm_output" >&2
    exit 1
  fi

  for compiler in clang gcc; do
    local output="build/z12-${name}-${compiler}"
    go run . build --release --compiler "$compiler" -o "$output" "$source"
    local native_output
    native_output="$("./$output")"
    if [[ "$native_output" != "$expected" ]]; then
      printf 'VM/%s output mismatch for %s:\n%s\n' "$compiler" "$name" "$native_output" >&2
      exit 1
    fi
  done
}

run_portable_example "data_persistence" $'2\n2\nLucas\n42\nLucas\nZumbra\ntrue\n43\ntrue'
run_portable_example "config_observability" $'8080\ntrue\n[REDACTED]\n2\ntrue\nok\ntrue\ntrue\nfalse\nLucas\ntrue\nLucas\ntrue'
run_portable_example "data_serialization" $'true\nLucas\n43981\ntrue\n42'

echo "Checking PostgreSQL and Redis APIs without requiring running servers..."
go run . check code_examples/core/postgres_redis.zum
go run . build --release --compiler clang -o build/z12-postgres-redis-clang code_examples/core/postgres_redis.zum
go run . build --release --compiler gcc -o build/z12-postgres-redis-gcc code_examples/core/postgres_redis.zum

echo "Running race checks for portable Z12 runtimes..."
go test -race ./object ./object/builtins ./evaluator ./vm ./conformance

echo "Rechecking prior SQLite and HTTP mini-milestones..."
scripts/test-sqlite.sh
scripts/test-http.sh

echo "Z12 data and persistence tests passed."
