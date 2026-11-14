#!/usr/bin/env bash
# Tests scripts/stop-gate.sh (the Claude Code Stop hook) in a throwaway git repository and worktree, with fake
# docker, make and ss on PATH: which exit code each outcome gives (0 passed, 2 a check failed, 1 the gate could
# not run), and which compose project and Mailpit port the stack is started with. Run by scripts/lint.sh.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
failures=0

mkdir "$work/bin"
cat > "$work/bin/docker" <<'EOF'
#!/usr/bin/env bash
echo "docker $*" >> "$FAKE_LOG"
case "$1" in
  info) exit "${FAKE_DOCKER_INFO:-0}" ;;
  compose)
    [[ " $* " == *" port mailpit 8025 "* ]] && [ -n "${FAKE_OWN_MAILPIT:-}" ] && echo "127.0.0.1:8025"
    [[ " $* " == *" ps -a --status created --services "* ]] && printf '%s\n' ${FAKE_CREATED:-} ;;
esac
exit 0
EOF
cat > "$work/bin/make" <<'EOF'
#!/usr/bin/env bash
echo "make $* mailpit=${SALUS_MAILPIT_PORT:-default}" >> "$FAKE_LOG"
case "$1" in
  arch-gate-stack) exit "${FAKE_STACK:-0}" ;;
  arch-gate-checks) exit "${FAKE_CHECKS:-0}" ;;
esac
exit 0
EOF
cat > "$work/bin/ss" <<'EOF'
#!/usr/bin/env bash
for p in ${FAKE_LISTEN:-}; do echo "LISTEN 0 4096 127.0.0.1:$p 0.0.0.0:*"; done
EOF
chmod +x "$work/bin/"*

# A main checkout with the script committed, and a worktree of it.
main="$work/main"
mkdir -p "$main/scripts" "$main/deployments"
cp "$here/stop-gate.sh" "$main/scripts/"
git -C "$main" init -q
git -C "$main" add .
git -C "$main" -c user.name=test -c user.email=test@example.invalid commit -qm init
git -C "$main" worktree add -q "$work/wt"

# gate <dir> <want exit> <name> [VAR=value...]: runs the script in dir with the fakes; its log and stderr are
# left in $work/log and $work/err.
gate() {
  local dir=$1 want=$2 name=$3 got=0
  shift 3
  : > "$work/log"
  env -u SALUS_MAILPIT_PORT PATH="$work/bin:$PATH" FAKE_LOG="$work/log" "$@" bash "$dir/scripts/stop-gate.sh" \
    > /dev/null 2> "$work/err" || got=$?
  if [ "$got" -ne "$want" ]; then
    echo "FAIL $name: exit $got, want $want; stderr:" >&2; sed 's/^/  /' "$work/err" >&2
    failures=$((failures + 1))
  fi
}
# expect <name> <grep -E pattern> <file>, and refute the same for a pattern that must not appear.
expect() { grep -qE "$2" "$3" || { echo "FAIL $1: no /$2/ in $(basename "$3"):" >&2; sed 's/^/  /' "$3" >&2; failures=$((failures + 1)); }; }
refute() { ! grep -qE "$2" "$3" || { echo "FAIL $1: /$2/ in $(basename "$3"):" >&2; sed 's/^/  /' "$3" >&2; failures=$((failures + 1)); }; }

gate "$main" 0 "a clean run"
expect "a clean run" '^make arch-gate-stack mailpit=default$' "$work/log"
expect "a clean run" '^make arch-gate-checks ' "$work/log"

gate "$main" 2 "a failing check" FAKE_CHECKS=2
expect "a failing check" 'archcheck, go vet or the tests failed' "$work/err"

gate "$main" 1 "a stack that does not come up" FAKE_STACK=2
expect "a stack that does not come up" 'not run: the test stack did not come up' "$work/err"
refute "a stack that does not come up" '^make arch-gate-checks' "$work/log"

# A container whose start failed (its port taken) is left created without a network; started by `up --no-recreate`,
# it would fail the tests with exit 2. It is removed before the stack is started and after a failed start.
gate "$main" 1 "a stack left half created" FAKE_STACK=2 FAKE_CREATED="mailpit postgres"
expect "a stack left half created" '^docker compose -f deployments/docker-compose.yml rm -f mailpit postgres$' "$work/log"
grep -B1 '^make arch-gate-stack' "$work/log" | grep -q ' rm -f mailpit postgres$' ||
  { echo "FAIL a stack left half created: not removed before the stack is started" >&2; failures=$((failures + 1)); }
[ "$(grep -c ' rm -f ' "$work/log")" -eq 2 ] ||
  { echo "FAIL a stack left half created: not removed again after the failed start" >&2; failures=$((failures + 1)); }
gate "$main" 0 "nothing left created"
refute "nothing left created" ' rm ' "$work/log"

gate "$main" 1 "Docker down" FAKE_DOCKER_INFO=1
expect "Docker down" 'not run: Docker' "$work/err"
refute "Docker down" '^make' "$work/log"

gate "$main" 0 "8025 taken by another project" FAKE_LISTEN="8025 8026"
expect "8025 taken by another project" '^make arch-gate-stack mailpit=8027$' "$work/log"
expect "8025 taken by another project" '127.0.0.1:8027' "$work/err"

expect "8025 taken by another project" '^docker compose -f deployments/docker-compose.yml rm -f mailpit$' "$work/log"

# A stopped Mailpit of this project keeps its old 8025 mapping under `up --no-recreate`; it is removed when the port
# moves (above), and left alone when it does not.
gate "$main" 0 "8025 free"
refute "8025 free" ' rm -f mailpit$' "$work/log"

gate "$main" 1 "8025 to 8099 all taken" FAKE_LISTEN="$(seq -s ' ' 8025 8099)"
expect "8025 to 8099 all taken" 'not run: 127.0.0.1:8025 and 8026-8099 are all taken' "$work/err"
refute "8025 to 8099 all taken" '^make' "$work/log"

printf 'SALUS_MAILPIT_PORT=9125\n' > "$main/deployments/.env"
gate "$main" 0 "a Mailpit port in deployments/.env" FAKE_LISTEN=8025
expect "a Mailpit port in deployments/.env" '^make arch-gate-stack mailpit=default$' "$work/log"
rm "$main/deployments/.env"

gate "$main" 0 "8025 held by this project's own Mailpit" FAKE_LISTEN=8025 FAKE_OWN_MAILPIT=1
expect "8025 held by this project's own Mailpit" '^make arch-gate-stack mailpit=default$' "$work/log"

gate "$main" 0 "a Mailpit port set by the developer" FAKE_LISTEN=8025 SALUS_MAILPIT_PORT=9025
expect "a Mailpit port set by the developer" '^make arch-gate-stack mailpit=9025$' "$work/log"

gate "$work/wt" 1 "a worktree without scripts/runner.sh"
expect "a worktree without scripts/runner.sh" 'not run: this worktree has no compose project' "$work/err"
refute "a worktree without scripts/runner.sh" '^make' "$work/log"

printf '#!/usr/bin/env bash\necho "runner $*" >> "$FAKE_LOG"\n' > "$work/wt/scripts/runner.sh"
gate "$work/wt" 0 "a worktree with its slot from scripts/runner.sh" FAKE_LISTEN=8025
expect "a worktree with its slot from scripts/runner.sh" '^runner env$' "$work/log"
expect "a worktree with its slot from scripts/runner.sh" '^make arch-gate-stack mailpit=default$' "$work/log"

gate "$work/wt" 2 "a failing check in a worktree" FAKE_CHECKS=2

printf '#!/usr/bin/env bash\nexit 1\n' > "$work/wt/scripts/runner.sh"
gate "$work/wt" 1 "a worktree whose slot cannot be assigned"
refute "a worktree whose slot cannot be assigned" '^make' "$work/log"

if [ "$failures" -ne 0 ]; then
  echo "stop-gate.test.sh: $failures failed" >&2
  exit 1
fi
echo "stop-gate.test.sh: ok"
