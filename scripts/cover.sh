#!/usr/bin/env bash
# make cover, inside the tools container: test with coverage and fail when any package is below the floor
# (CLAUDE.md: 80%). Each package counts only the coverage its own tests give it. Two exclusions:
# cmd/*/main.go, listed file by file (composition only, exercised by the e2e suite), and the generated sqlc
# packages internal/<domain>/<domain>db (never edited by hand, exercised by the domain's repository tests).
set -euo pipefail
floor=80
module=$(go list -m)

go test -p "${TEST_PACKAGES:-8}" -coverprofile=coverage.raw ./... # the cap of the Makefile's GO_TEST

excluded=$(for f in cmd/*/main.go; do echo "${module}/${f}"; done)
generated=$(for d in internal/*/; do n=$(basename "$d"); [ -d "${d}${n}db" ] && echo "${module}/${d}${n}db"; done || true)
awk -v excluded="$excluded" -v generated="$generated" '
  BEGIN {
    n = split(excluded, list, "\n"); for (i = 1; i <= n; i++) skip[list[i]] = 1
    n = split(generated, list, "\n"); for (i = 1; i <= n; i++) if (list[i] != "") skipdir[list[i]] = 1
  }
  NR == 1 { print; next }
  { file = $0; sub(/:[^:]*$/, "", file); dir = file; sub(/\/[^\/]*$/, "", dir); if (!(file in skip) && !(dir in skipdir)) print }
' coverage.raw > coverage.out
rm coverage.raw

# Profile lines: file:start,end statements count. Sum statements per package (the file's directory).
awk -v floor="$floor" '
  NR == 1 { next }
  {
    file = $1; sub(/:[^:]*$/, "", file); pkg = file; sub(/\/[^\/]*$/, "", pkg)
    total[pkg] += $2; all += $2
    if ($3 > 0) { covered[pkg] += $2; hit += $2 }
  }
  END {
    for (p in total) {
      pct = 100 * covered[p] / total[p]
      printf "%6.1f%%  %s\n", pct, p | "sort -k2"
      if (pct < floor) low = low "\n  " p
    }
    close("sort -k2")
    printf "total: %.1f%% (floor %d%% per package)\n", 100 * hit / all, floor
    if (low != "") { printf "below %d%%:%s\n", floor, low; exit 1 }
  }
' coverage.out
