#!/usr/bin/env bash
# make e2e, on the host (the Makefile installs web/ and test/e2e/web first): the end-to-end suite on a compose project
# of its own, <this checkout's project>-e2e (salus-e2e in the main checkout), never on the dev stack, whose old
# data would fail the journeys. Its Postgres and object store volumes are removed before the run, so every run starts
# fresh, and again after it, pass or fail; nothing else is touched. Two volumes stay: clamav (the virus signatures,
# a cache with no personal data, downloaded again only when missing) and e2e (the run's output: make e2e-report
# reads the latency report from it). The e2e project publishes no host ports (docker-compose.e2e-run.yml): the
# journeys run on its compose network, so it runs beside the dev stack.
#
# The stack runs with docker-compose.e2e.yml on top: the fake engines play the scripted conversations of
# test/testdata/consult (SAL-10). Each run first clears the login rate-limit counters (ratelimit:*) in the
# project's Redis, and each Go journey clears them again before it signs in (clearLoginLimits): every sign-in comes
# from one IP, and a run signs in more often than the production limit of 30 per 15 minutes allows. After the
# journeys, LiveKit's log must hold none of the run's synthetic names, e-mails and ids (scripts/e2e-livekit-log.sh).
#
#   e2e.sh           the run; with E2E_LOGS=<file>, a failed run also writes the project's full logs there before
#                    its containers are removed (CI's failure step prints them)
#   e2e.sh report    print the last run's latency report (make e2e-report), touching nothing else
set -euo pipefail
cd "$(dirname "$0")/.."

base=${COMPOSE_PROJECT_NAME:-$(sed -n 's/^COMPOSE_PROJECT_NAME=//p' deployments/.env 2>/dev/null | tail -n 1 || true)}
project="${base:-salus}-e2e"
compose=(docker compose -p "$project" -f deployments/docker-compose.yml -f deployments/docker-compose.e2e.yml
  -f deployments/docker-compose.e2e-run.yml)
run=("${compose[@]}" run --rm -T playwright)
data_volumes=(postgres objectstore)

if [ "${1:-}" = report ]; then
  exec "${compose[@]}" run --rm --no-deps -T playwright cat /e2e/consult-latency.md
fi

clean() {
  "${compose[@]}" --profile e2e down --remove-orphans --timeout 10 >/dev/null 2>&1 || true
  local v names=()
  for v in "${data_volumes[@]}"; do
    docker volume inspect "${project}_$v" >/dev/null 2>&1 && names+=("${project}_$v")
  done
  [ "${#names[@]}" -eq 0 ] || docker volume rm -f "${names[@]}" >/dev/null
}
finish() {
  local code=$?
  if [ "$code" -ne 0 ]; then
    echo "e2e: failed (exit $code); the last lines of the $project stack's logs:" >&2
    "${compose[@]}" logs --no-color --tail 200 >&2 || true
    if [ -n "${E2E_LOGS:-}" ]; then
      "${compose[@]}" logs --no-color --timestamps > "$E2E_LOGS" 2>&1 || true
      echo "e2e: the $project stack's full logs are in $E2E_LOGS" >&2
    fi
  fi
  clean
  echo "e2e: $project removed (containers, network, ${data_volumes[*]} volumes)" >&2
  exit "$code"
}

echo "e2e: compose project $project" >&2
clean
trap finish EXIT
"${compose[@]}" up -d --build --wait --remove-orphans
"${compose[@]}" exec -T redis sh -c "redis-cli --scan --pattern 'ratelimit:*' | xargs -r redis-cli del >/dev/null"
"${run[@]}" sh -c 'rm -f /e2e/livekit-check.tsv /e2e/consult-latency.md && go build -o /tmp/salusctl ./cmd/salusctl && go test -tags e2e -count=1 ./test/e2e/...'
"${compose[@]}" logs --no-color --no-log-prefix livekit | "${run[@]}" bash scripts/e2e-livekit-log.sh /e2e/livekit-check.tsv
"${run[@]}" cat /e2e/consult-latency.md
