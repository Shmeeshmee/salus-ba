#!/usr/bin/env bash
# make lint, inside the tools container: gofmt, go vet and golangci-lint over both modules, then the
# migration immutability check, the check that the production Caddyfile routes the web app's API paths, and the
# tests of the Stop hook, the runner's port allocation, make e2e's project and make check's sets.
set -euo pipefail

unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "gofmt: these files need formatting:" >&2
  echo "$unformatted" >&2
  exit 1
fi
go vet ./...
go vet -tags e2e,loadtest ./test/... # the e2e suite and the load test beside it
go tool -modfile=tools/go.mod golangci-lint run ./...
(cd tools && go vet ./... && go tool golangci-lint run ./...)
bash scripts/check-migrations.sh
bash scripts/check-caddy-routes.sh
bash scripts/stop-gate.test.sh
bash scripts/runner.test.sh
bash scripts/e2e.test.sh
bash scripts/check.test.sh
