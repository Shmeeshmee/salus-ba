#!/usr/bin/env bash
# Inside the tools container: the Go packages of the main module that the given directories change, and the
# packages that import one of them directly (their tests included), as ./relative paths: the fast layer of
# scripts/runner.sh. Importers further up are left to the full run at the end of the package. A change to
# go.mod, go.sum or migrations/ (which every repository test migrates) affects every package: prints ./...
# Usage: bash scripts/go-affected.sh internal/patient pkg/clock ...
set -euo pipefail
[ $# -gt 0 ] || exit 0
mod=$(go list -m)
changed=()
for d in "$@"; do
  case "$d" in go.mod|go.sum|migrations|migrations/*) echo "./..."; exit 0 ;; esac
  [ -d "$d" ] && changed+=("$mod/${d#./}")
done
[ ${#changed[@]} -gt 0 ] || exit 0
pattern=$(printf '%s\n' "${changed[@]}" | sed 's/[.]/\\./g' | paste -sd'|')
# One line per package: its path, then what it and its tests import.
go list -f '{{.ImportPath}} {{join .Imports " "}} {{join .TestImports " "}} {{join .XTestImports " "}}' ./... |
  awk -v re="^($pattern)$" -v mod="$mod/" '
    { p = $1
      for (i = 1; i <= NF; i++) if ($i ~ re) { hit[p] = 1; break } }
    END { for (p in hit) if (index(p, mod) == 1) print "./" substr(p, length(mod) + 1) }' | sort -u
