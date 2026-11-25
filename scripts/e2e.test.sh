#!/usr/bin/env bash
# Tests scripts/e2e.sh (make e2e) with a fake docker on PATH: every compose call runs on the e2e project of this
# checkout (<project>-e2e), never on the dev stack's; the project's data volumes are removed before the run (fresh)
# and after it, also when a journey fails; no other volume is removed; a failing journey fails the script; and with
# E2E_LOGS a failing run writes the e2e project's full logs to that file before the cleanup (CI's failure step).
# Run by scripts/lint.sh.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
failures=0
fail() { echo "FAIL $*" >&2; failures=$((failures + 1)); }

mkdir -p "$work/bin" "$work/repo/scripts" "$work/repo/deployments"
cp "$here/e2e.sh" "$work/repo/scripts/"
cat > "$work/bin/docker" <<'FAKE'
#!/usr/bin/env bash
echo "docker $*" >> "$FAKE_LOG"
case " $* " in
  *" volume inspect "*) for v in $FAKE_VOLUMES; do [ "${*: -1}" = "$v" ] && exit 0; done; exit 1 ;;
  *" go test "*) exit "${FAKE_JOURNEYS:-0}" ;;
esac
exit 0
FAKE
chmod +x "$work/bin/docker"

run() { # run <want exit> <name> [VAR=value...]
  local want=$1 name=$2 got=0; shift 2
  : > "$work/log"
  (cd "$work/repo" && env -u COMPOSE_PROJECT_NAME PATH="$work/bin:$PATH" FAKE_LOG="$work/log" \
    FAKE_VOLUMES="wt-x-e2e_postgres wt-x-e2e_objectstore wt-x-e2e_clamav wt-x-e2e_e2e salus_gomod wt-x_postgres" \
    "$@" bash scripts/e2e.sh) > "$work/out" 2>&1 || got=$?
  [ "$got" -eq "$want" ] || { fail "$name: exit $got, want $want"; sed 's/^/  /' "$work/out" >&2; }
}

printf 'COMPOSE_PROJECT_NAME=wt-x\nSALUS_API_PORT=20400\n' > "$work/repo/deployments/.env"
run 0 "a passing run"
grep '^docker compose' "$work/log" | grep -v -- ' -p wt-x-e2e ' && fail "a compose call off the e2e project"
grep -q -- '-p wt-x-e2e .* up ' "$work/log" || fail "the e2e stack was not started"
removed=$(sed -n 's/^docker volume rm -f //p' "$work/log" | tr ' ' '\n' | sort -u | paste -sd' ')
[ "$removed" = "wt-x-e2e_objectstore wt-x-e2e_postgres" ] || fail "volumes removed: '$removed'"
[ "$(grep -c '^docker volume rm' "$work/log")" -eq 2 ] || fail "the volumes were not removed both before and after"
first_rm=$(grep -n '^docker volume rm' "$work/log" | head -n1 | cut -d: -f1)
first_up=$(grep -n -- ' up ' "$work/log" | head -n1 | cut -d: -f1)
[ "$first_rm" -lt "$first_up" ] || fail "the old volumes were not removed before the start"

run 1 "a failing journey" FAKE_JOURNEYS=1
[ "$(grep -c '^docker volume rm' "$work/log")" -eq 2 ] || fail "a failing journey: not cleaned up after"
grep -q -- ' logs --no-color --tail ' "$work/log" || fail "a failing journey: the stack's logs were not printed"
grep -q -- ' logs --no-color --timestamps' "$work/log" && fail "a failing journey: full logs written without E2E_LOGS"

run 1 "a failing journey with E2E_LOGS" FAKE_JOURNEYS=1 E2E_LOGS="$work/e2e-logs.txt"
grep -q -- '-p wt-x-e2e .* logs --no-color --timestamps$' "$work/log" || fail "E2E_LOGS: the e2e project's full logs were not read"
[ -f "$work/e2e-logs.txt" ] || fail "E2E_LOGS: no log file"
log_line=$(grep -n -- ' logs --no-color --timestamps' "$work/log" | head -n1 | cut -d: -f1) || true
last_rm=$(grep -n '^docker volume rm' "$work/log" | tail -n1 | cut -d: -f1)
[ -n "$log_line" ] && [ "$log_line" -lt "$last_rm" ] || fail "E2E_LOGS: the logs were read after the cleanup"
run 0 "a passing run with E2E_LOGS" E2E_LOGS="$work/e2e-logs-pass.txt"
[ -f "$work/e2e-logs-pass.txt" ] && fail "E2E_LOGS: a passing run wrote the logs"

: > "$work/log"
(cd "$work/repo" && env PATH="$work/bin:$PATH" FAKE_LOG="$work/log" FAKE_VOLUMES= bash scripts/e2e.sh report) > /dev/null 2>&1 ||
  fail "the report: exit $?"
[ "$(wc -l < "$work/log")" -eq 1 ] && grep -q -- '-p wt-x-e2e .* run --rm --no-deps -T playwright cat /e2e/consult-latency.md$' "$work/log" ||
  { fail "the report did more than read the report:"; sed 's/^/  /' "$work/log" >&2; }

rm "$work/repo/deployments/.env"
run 0 "the main checkout"
grep '^docker compose' "$work/log" | grep -v -- ' -p salus-e2e ' && fail "the main checkout: a compose call off salus-e2e"

if [ "$failures" -ne 0 ]; then
  echo "e2e.test.sh: $failures failed" >&2
  exit 1
fi
echo "e2e.test.sh: ok"
