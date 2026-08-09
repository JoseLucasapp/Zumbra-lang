#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

printf 'Running Z18 official tooling tests...\n'

go test ./diagnostics ./tooling/formatter ./tooling/docgen ./tooling/lint ./tooling/project ./tooling/profile ./tooling/lsp ./pipeline
go test . -run 'TestZ18' -count=1
go vet -unsafeptr=false ./...

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

go build -o "$tmp_dir/zumbra" .
[[ "$($tmp_dir/zumbra version)" == "0.14.5" ]]

"$tmp_dir/zumbra" check --help >/dev/null 2>&1
"$tmp_dir/zumbra" fmt --help >/dev/null 2>&1
"$tmp_dir/zumbra" lint --help >/dev/null 2>&1
"$tmp_dir/zumbra" doc --help >/dev/null 2>&1
"$tmp_dir/zumbra" project --help >/dev/null 2>&1
"$tmp_dir/zumbra" project init --help >/dev/null 2>&1
"$tmp_dir/zumbra" project build --help >/dev/null 2>&1
"$tmp_dir/zumbra" profile --help >/dev/null 2>&1
"$tmp_dir/zumbra" lsp --help >/dev/null 2>&1

"$tmp_dir/zumbra" project init --dir "$tmp_dir/sample" "Z18 Sample"
cd "$tmp_dir/sample"

"$tmp_dir/zumbra" fmt src tests
"$tmp_dir/zumbra" fmt --check src tests
"$tmp_dir/zumbra" lint --no-public-docs src tests
"$tmp_dir/zumbra" lint --no-public-docs --json src tests > "$tmp_dir/lint.json"
"$tmp_dir/zumbra" doc --private --output "$tmp_dir/api.md" src
"$tmp_dir/zumbra" doc --private --format json --output "$tmp_dir/api.json" src
"$tmp_dir/zumbra" project info --json > "$tmp_dir/project.json"
"$tmp_dir/zumbra" project check
"$tmp_dir/zumbra" project test
"$tmp_dir/zumbra" profile --runs 2 --warmup 1 --json src/main.zum > "$tmp_dir/profile.json"

[[ -s "$tmp_dir/lint.json" ]]
[[ -s "$tmp_dir/api.md" ]]
[[ -s "$tmp_dir/api.json" ]]
[[ -s "$tmp_dir/project.json" ]]
[[ -s "$tmp_dir/profile.json" ]]

cd "$ROOT"
if command -v node >/dev/null 2>&1; then
  node --check editors/vscode/extension.js
fi

scripts/check-repository-hygiene.sh
printf 'Z18 official tooling tests passed.\n'
