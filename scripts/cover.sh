#!/usr/bin/env bash
# make cover, inside the tools container: test with coverage and fail below the floor (CLAUDE.md: 80%).
set -euo pipefail
floor=80

# -coverpkg: code counts as covered when any package's tests run it (the router's golden tests cover pkg/fhir).
go test -coverpkg=./... -coverprofile=coverage.out ./...
total=$(go tool cover -func=coverage.out | awk '/^total:/ { sub("%", "", $3); print $3 }')
echo "coverage: ${total}% (floor ${floor}%)"
awk -v t="$total" -v f="$floor" 'BEGIN { exit (t + 0 < f) }' || { echo "coverage is below ${floor}%" >&2; exit 1; }
