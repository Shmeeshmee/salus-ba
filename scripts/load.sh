#!/usr/bin/env bash
# make load, on the host (docs/clinic-cms/load/REPORT.md, WAVE7.md RD2): the busy year's load run on a compose project
# of its own, <this checkout's project>-load, with test/load/docker-compose.load.yml on top (Postgres with
# pg_stat_statements and cluster_name=salus-load, no host ports). It starts that project's Postgres and Redis,
# migrates its database, then runs TestLoad in its tools container: the seed on the first run (kept in the project's
# postgres volume for the next runs) and the readings. The dev stack and its data are never touched.
#
# The seeder and the driver refuse to run unless LOAD_CONFIRM=synthetic-busy-year is given, SALUS_ENV is dev and the
# server is the load stack's (test/load/guard_test.go).
#
#   load.sh        the run (LOAD_REQUESTS, LOAD_CONCURRENCY, LOAD_SEED_ONLY are passed on)
#   load.sh down   stop the load project and remove its volumes (the seed is several GB)
set -euo pipefail
cd "$(dirname "$0")/.."

base=${COMPOSE_PROJECT_NAME:-$(sed -n 's/^COMPOSE_PROJECT_NAME=//p' deployments/.env 2>/dev/null | tail -n 1 || true)}
project="${base:-salus}-load"
compose=(docker compose -p "$project" -f deployments/docker-compose.yml -f test/load/docker-compose.load.yml)

if [ "${1:-}" = down ]; then
  "${compose[@]}" down --volumes --remove-orphans --timeout 10
  exit
fi

echo "load: compose project $project" >&2
"${compose[@]}" up -d --wait postgres redis
"${compose[@]}" run --rm -T tools go run ./cmd/migrate up
"${compose[@]}" run --rm -T -e LOAD_CONFIRM -e LOAD_REQUESTS -e LOAD_CONCURRENCY -e LOAD_SEED_ONLY tools \
  go test -tags loadtest -count=1 -timeout 180m -v -run '^TestLoad$' ./test/load/
