#!/usr/bin/env bash
# make lint, inside the tools container: gofmt, go vet and golangci-lint over both modules, then the
# migration immutability check, the check that the production Caddyfile routes the web app's API paths, and the
# Stop hook's tests.
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
