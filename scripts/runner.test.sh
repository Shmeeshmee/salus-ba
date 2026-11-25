#!/usr/bin/env bash
# Tests scripts/runner.sh's port allocation (`env`) and the refusal of `start` on a port in use, in a throwaway git
# repository with worktrees and fake docker and ss on PATH: two worktrees never get the same ports, whether or not
# the other's stack is running, ports already given stay, and a port held by anything but the worktree's own
# compose project stops `start`. Run by scripts/lint.sh.
set -euo pipefail
unset GIT_DIR GIT_COMMON_DIR GIT_WORK_TREE GIT_INDEX_FILE COMPOSE_PROJECT_NAME

here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
failures=0
fail() { echo "FAIL $*" >&2; failures=$((failures + 1)); }

mkdir "$work/bin"
# docker: `ps` lists FAKE_OWN_PORTS as the published ports of the project's containers; compose ps lists nothing.
cat > "$work/bin/docker" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = ps ]; then for p in ${FAKE_OWN_PORTS:-}; do echo "127.0.0.1:$p->8080/tcp"; done; fi
exit 0
EOF
cat > "$work/bin/ss" <<'EOF'
#!/usr/bin/env bash
for p in ${FAKE_LISTEN:-}; do echo "LISTEN 0 4096 127.0.0.1:$p 0.0.0.0:*"; done
EOF
chmod +x "$work/bin/"*

main="$work/main"
mkdir -p "$main/scripts" "$main/deployments"
cp "$here/runner.sh" "$main/scripts/"
printf '.claude/runs/\n.env\n' > "$main/.gitignore"
touch "$main/deployments/docker-compose.yml"
git -C "$main" init -q
git -C "$main" add .
git -C "$main" -c user.name=test -c user.email=test@example.invalid commit -qm init

run() { # run <dir> <subcommand...>: runner.sh with the fakes; output in $work/out, the exit code returned
  local dir=$1; shift
  (cd "$dir" && env PATH="$work/bin:$PATH" RUNNER_SESSION=test "$@") > "$work/out" 2>&1
}
api_port() { sed -n 's/^SALUS_API_PORT=//p' "$1/deployments/.env"; }
slot_of() { echo $(( $(cksum <<<"$1" | cut -d' ' -f1) % 90 )); }

# Two names whose hash picks the same slot: the case that gave R3 the ports of R1.
a=wt-a; b=
for i in $(seq 1 2000); do
  [ "$(slot_of "wt-b$i")" = "$(slot_of "$a")" ] && { b="wt-b$i"; break; }
done
[ -n "$b" ] || { echo "runner.test.sh: no colliding name found" >&2; exit 1; }
git -C "$main" worktree add -q "$work/$a"
git -C "$main" worktree add -q "$work/$b"

run "$work/$a" bash scripts/runner.sh env || fail "env in $a: $(cat "$work/out")"
pa=$(api_port "$work/$a")
# Nothing listens (a's stack is down): b must still not get a's ports.
run "$work/$b" bash scripts/runner.sh env || fail "env in $b: $(cat "$work/out")"
pb=$(api_port "$work/$b")
[ -n "$pa" ] && [ -n "$pb" ] || fail "no ports assigned: a=$pa b=$pb"
[ $(( pa > pb ? pa - pb : pb - pa )) -ge 8 ] 2>/dev/null || fail "two worktrees of one slot share ports: $a=$pa $b=$pb"

# Ports already given stay on a rerun.
run "$work/$a" bash scripts/runner.sh env || fail "env rerun in $a"
[ "$(api_port "$work/$a")" = "$pa" ] || fail "rerun moved $a's ports: $pa -> $(api_port "$work/$a")"

# A worktree whose deployments/.env already holds another worktree's ports (written before this fix) moves.
cp "$work/$a/deployments/.env" "$work/$b/deployments/.env"
sed -i "s/^COMPOSE_PROJECT_NAME=.*/COMPOSE_PROJECT_NAME=$b/" "$work/$b/deployments/.env"
run "$work/$b" bash scripts/runner.sh env || fail "env in $b with $a's ports"
[ "$(api_port "$work/$b")" != "$pa" ] || fail "$b kept $a's ports"
[ "$(api_port "$work/$a")" = "$pa" ] || fail "$a's ports changed when $b moved"

# A port someone else listens on is skipped when ports are first given.
git -C "$main" worktree add -q "$work/wt-c"
c_slot=$(slot_of wt-c)
c_first=$(( 20000 + ((c_slot) % 90 + 10) * 20 ))
run "$work/wt-c" env FAKE_LISTEN="$((c_first + 3))" bash scripts/runner.sh env || fail "env in wt-c"
pc=$(api_port "$work/wt-c")
[ "$pc" != "$c_first" ] || fail "wt-c was given $c_first although $((c_first + 3)) is in use"

# start refuses a port held by something else, and accepts one held by the worktree's own stack.
if run "$work/$a" env FAKE_LISTEN="$((pa + 4))" bash scripts/runner.sh start docs/x 1; then
  fail "start in $a ran with $((pa + 4)) held by another process"
fi
grep -q "$((pa + 4))" "$work/out" || fail "start's refusal does not name the port: $(cat "$work/out")"
run "$work/$a" env FAKE_LISTEN="$pa $((pa + 4))" FAKE_OWN_PORTS="$pa $((pa + 4))" bash scripts/runner.sh start docs/x 2 ||
  fail "start in $a refused its own stack's ports: $(cat "$work/out")"
run "$work/$a" bash scripts/runner.sh start docs/x 3 || fail "start in $a with nothing listening: $(cat "$work/out")"

if [ "$failures" -ne 0 ]; then
  echo "runner.test.sh: $failures failed" >&2
  exit 1
fi
echo "runner.test.sh: ok"
