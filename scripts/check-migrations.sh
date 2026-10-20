#!/usr/bin/env bash
# Fails when a migration that exists on main was modified, renamed or deleted (goose keeps no checksums).
# New files are fine. Compares the working tree with the merge base of HEAD and origin/main (or main).
set -euo pipefail

base_ref=origin/main
git rev-parse --verify -q "$base_ref" >/dev/null || base_ref=main
if ! git rev-parse --verify -q "$base_ref" >/dev/null; then
  echo "check-migrations: neither origin/main nor main exists; fetch main first" >&2
  exit 2
fi
base=$(git merge-base HEAD "$base_ref")

changed=$(git diff --name-status --no-renames --diff-filter=MD "$base" -- migrations/ | grep '\.sql$' || true)
if [ -n "$changed" ]; then
  echo "check-migrations: migrations that exist on main must not change; add a new file instead:" >&2
  echo "$changed" >&2
  exit 1
fi
echo "check-migrations: ok (no migration from main changed)"
