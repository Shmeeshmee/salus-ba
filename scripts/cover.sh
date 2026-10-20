#!/usr/bin/env bash
# make cover, inside the tools container: test with coverage and fail when any package is below the floor
# (CLAUDE.md: 80%). Each package counts only the coverage its own tests give it. The one exclusion is
# cmd/*/main.go, listed file by file: composition only, exercised by the e2e suite.
set -euo pipefail
floor=80
module=$(go list -m)

go test -coverprofile=coverage.raw ./...

excluded=$(for f in cmd/*/main.go; do echo "${module}/${f}"; done)
awk -v excluded="$excluded" '
  BEGIN { n = split(excluded, list, "\n"); for (i = 1; i <= n; i++) skip[list[i]] = 1 }
  NR == 1 { print; next }
  { file = $0; sub(/:[^:]*$/, "", file); if (!(file in skip)) print }
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
