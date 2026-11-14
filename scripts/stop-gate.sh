#!/usr/bin/env bash
# The Claude Code Stop hook (.claude/settings.json): make arch-gate on this checkout's own compose stack.
#
# Exit 2, which keeps the session going with the failure in front of Claude, only when archcheck, go vet or the
# tests fail. When the gate cannot run (Docker unreachable, no compose project for this worktree, a port taken, a
# service that does not come up) it says why on stderr and exits 1, which Claude Code shows without blocking: a
# stack that cannot start must not loop the session.
#
# A worktree runs under the compose project and ports scripts/runner.sh env gives it (deployments/.env). The main
# checkout runs `salus` with the default ports, and Mailpit's web UI moves off 8025 when something else holds it.
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1

skip() {
  echo "stop-gate: not run: $*" >&2
  exit 1
}

# mailpit_port exports SALUS_MAILPIT_PORT, the first free port from 8026, when 8025 is held by anything but this
# project's own Mailpit. A port the developer set (environment or deployments/.env) is kept.
mailpit_port() {
  [ -n "${SALUS_MAILPIT_PORT:-}" ] && return
  grep -qs '^SALUS_MAILPIT_PORT=' deployments/.env && return
  local listening port
  listening=$(ss -Htln 2>/dev/null | awk '{ n = split($4, a, ":"); print a[n] }')
  grep -qx 8025 <<<"$listening" || return 0
  [ -n "$(docker compose -f deployments/docker-compose.yml port mailpit 8025 2>/dev/null)" ] && return
  for port in $(seq 8026 8099); do
    if ! grep -qx "$port" <<<"$listening"; then
      export SALUS_MAILPIT_PORT=$port
      echo "stop-gate: 127.0.0.1:8025 is taken; Mailpit's web UI on 127.0.0.1:$port" >&2
      return
    fi
  done
  skip "127.0.0.1:8025 and 8026-8099 are all taken (Mailpit's web UI)"
}

# drop_created removes the stack's containers that were created and never started. A service whose start failed (its
# port taken) is left so, without its network; `up --no-recreate` would start it as it is, report the stack up, and
# the tests would fail on it. Never started, it holds nothing, so it goes and `up` creates it afresh.
drop_created() {
  local -a compose=(docker compose -f deployments/docker-compose.yml) created
  mapfile -t created < <("${compose[@]}" ps -a --status created --services 2>/dev/null | grep .)
  [ "${#created[@]}" -eq 0 ] || "${compose[@]}" rm -f "${created[@]}" > /dev/null 2>&1
}

docker info > /dev/null 2>&1 || skip "Docker is not reachable"

if [ "$(git rev-parse --path-format=absolute --git-dir)" != "$(git rev-parse --path-format=absolute --git-common-dir)" ]; then
  [ -f scripts/runner.sh ] ||
    skip "this worktree has no compose project of its own: scripts/runner.sh env (#96) assigns it, and is not on this branch"
  bash scripts/runner.sh env >&2 || skip "scripts/runner.sh env could not assign this worktree its project and ports"
else
  mailpit_port
fi

drop_created
if ! make arch-gate-stack >&2; then
  drop_created
  skip "the test stack did not come up (above: Docker's reason, such as a port that is already allocated)"
fi
if ! make arch-gate-checks >&2; then
  echo "stop-gate: archcheck, go vet or the tests failed (above)" >&2
  exit 2
fi
