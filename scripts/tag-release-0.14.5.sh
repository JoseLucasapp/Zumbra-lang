#!/usr/bin/env bash
set -euo pipefail

git status --short
git add .
git commit -m "release: zumbra 0.14.5 native static string literals"
git tag -a v0.14.5 -m "Zumbra 0.14.5"
git push origin main --tags
