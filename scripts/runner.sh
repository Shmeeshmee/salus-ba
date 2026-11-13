#!/usr/bin/env bash
# The package-runner's helpers (.claude/agents/package-runner.md, "Time, waits and cleanup"): a time budget per
# package, background commands with a pidfile and a hard timeout, checks in two layers, a stack it reuses and
# stops only when it started it, and the cleanup that ends every package. Runs on the host; every check still
# runs in Docker. State lives in .claude/runs/session/ (git-ignored), one session at a time per worktree.
#
#   runner.sh env                      COMPOSE_PROJECT_NAME and SALUS_*_PORT for this worktree (deployments/.env)
#   runner.sh start <plan> <package>   begin a package: clock, stack snapshot, env (idempotent for the same package)
#   runner.sh phase <name>             a phase begins; exits 3 once the budget (RUNNER_BUDGET_MIN, 60) is spent
#   runner.sh bg <name> [min] -- cmd   run cmd in the background: pidfile, log, timeout (default 10 min)
#   runner.sh wait <name> [min]        wait for a bg command (default 10 min); on timeout kill it, exit 124
#   runner.sh check <name> [min] -- cmd   run a check in the foreground with a timeout; one rerun at most per tree
#   runner.sh fast                     the fast layer: only what the package's diff touches
#   runner.sh full                     the full check set, once, at the end of the package
#   runner.sh cleanup                  kill what the session started, stop the stack services it started
#   runner.sh report                   the TIMINGS, FLAKY and CLEANUP lines for the report
#   runner.sh gate                     the Stop hook: make arch-gate, unless this exact tree already passed it
set -euo pipefail

root=$(git rev-parse --show-toplevel); cd "$root"
state="$root/.claude/runs/session"
compose=(docker compose -f deployments/docker-compose.yml)
budget=${RUNNER_BUDGET_MIN:-60}
export HOST_UID HOST_GID GIT_COMMON_DIR
HOST_UID=$(id -u); HOST_GID=$(id -g); GIT_COMMON_DIR=$(git rev-parse --path-format=absolute --git-common-dir)

die() { echo "runner: $*" >&2; exit 1; }
now() { date +%s; }
need_state() { [ -f "$state/start" ] || die "no session; run 'bash scripts/runner.sh start' first"; }
minutes() { echo $(( ($2 - $1 + 30) / 60 )); }
session_id() { echo "${RUNNER_SESSION:-${CLAUDE_CODE_SESSION_ID:-manual}}"; }
# One command string for bash -c: a single argument as written, several quoted one by one.
cmd_string() { if [ $# -eq 1 ]; then printf '%s' "$1"; else printf '%q ' "$@"; fi; }

# A worktree gets its own compose project and ports; the main checkout keeps `salus` and the defaults.
# Ports: 20000 + slot*20 + offset, the slot (10..99) from the worktree's name. Keys already set in
# deployments/.env outside the managed block win.
cmd_env() {
  local envf=deployments/.env begin="# >>> runner.sh env (generated; do not edit)" end="# <<< runner.sh env"
  if [ "$(git rev-parse --path-format=absolute --git-dir)" = "$GIT_COMMON_DIR" ]; then
    echo "main checkout: compose project salus, default ports"; return
  fi
  local name slot base project
  name=$(basename "$root")
  project=$(tr '[:upper:]' '[:lower:]' <<<"$name" | tr -c 'a-z0-9_\n-' '-')
  # Ports already given to this worktree stay; a new worktree takes the first slot from its name's hash whose
  # eight ports are free on the host.
  base=$( [ -f "$envf" ] && awk -v b="$begin" -v e="$end" '$0 == b { s = 1; next } $0 == e { s = 0 } s' "$envf" |
    sed -n 's/^SALUS_API_PORT=//p' || true)
  if [ -z "$base" ]; then
    local used i
    used=$(ss -Htln 2>/dev/null | awk '{ n = split($4, a, ":"); print a[n] }' | sort -u)
    slot=$(( $(cksum <<<"$name" | cut -d' ' -f1) % 90 ))
    for i in $(seq 0 89); do
      base=$(( 20000 + ((slot + i) % 90 + 10) * 20 ))
      grep -qxE "$base|$((base + 1))|$((base + 2))|$((base + 3))|$((base + 4))|$((base + 5))|$((base + 6))|$((base + 7))" <<<"$used" || break
    done
  fi
  local -a kv=("COMPOSE_PROJECT_NAME=$project" "SALUS_API_PORT=$base" "SALUS_LIVEKIT_PORT=$((base + 1))"
    "SALUS_LIVEKIT_TCP_PORT=$((base + 2))" "SALUS_LIVEKIT_UDP_PORT=$((base + 3))" "SALUS_MAILPIT_PORT=$((base + 4))"
    "SALUS_LANDING_PORT=$((base + 5))" "SALUS_WEB_PORT=$((base + 6))" "SALUS_LANDING_DEV_PORT=$((base + 7))")
  local kept="" line
  [ -f "$envf" ] && kept=$(awk -v b="$begin" -v e="$end" '$0 == b { s = 1; next } $0 == e { s = 0; next } !s' "$envf")
  {
    [ -n "$kept" ] && printf '%s\n' "$kept"
    echo "$begin"
    for line in "${kv[@]}"; do
      grep -q "^${line%%=*}=" <<<"$kept" || echo "$line"
    done
    echo "$end"
  } > "$envf.tmp"
  mv "$envf.tmp" "$envf"
  echo "worktree: $(awk -v b="$begin" -v e="$end" '$0 == b { s = 1; next } $0 == e { s = 0 } s' "$envf" | paste -sd' ')"
}

running_services() { { "${compose[@]}" ps --status running --services 2>/dev/null || true; } | grep -v '^$' | sort || true; }

cmd_start() {
  local id key; id=$(session_id); key="$id ${1:-} ${2:-}"
  if [ -f "$state/key" ] && [ "$(cat "$state/key")" = "$key" ]; then
    echo "session $id already started $(minutes "$(cat "$state/start")" "$(now)") min ago"; return
  fi
  rm -rf "$state"; mkdir -p "$state"
  echo "$id" > "$state/session"
  echo "$key" > "$state/key"
  now > "$state/start"
  git rev-parse HEAD > "$state/head"
  cmd_env
  running_services > "$state/services.before"
  : > "$state/checks"
  echo "$(now) start" > "$state/phases"
  echo "session $id started; budget $budget min, ends $(date -d "@$(( $(now) + budget * 60 ))" +%H:%M)"
  echo "stack: $(wc -l < "$state/services.before") services running (reused, never stopped by this session)"
}

cmd_phase() {
  need_state
  local name=${1:?phase name} start elapsed
  echo "$(now) $name" >> "$state/phases"
  start=$(cat "$state/start"); elapsed=$(minutes "$start" "$(now)")
  echo "phase $name at $elapsed min of $budget"
  if [ "$elapsed" -ge "$budget" ]; then
    echo "OVER BUDGET: $elapsed min. Stop, clean up and report STATUS: stopped (or done if committed)." >&2
    exit 3
  fi
}

# bg <name> [min] -- cmd...: setsid, so the whole process group can be killed; the exit code lands in
# <name>.exit, which is what wait looks for (no ps or log grep).
cmd_bg() {
  need_state
  local name=${1:?name}; shift
  local min=10; [ "${1:-}" != "--" ] && { min=$1; shift; }
  [ "${1:-}" = "--" ] && shift
  [ $# -gt 0 ] || die "bg $name: no command"
  rm -f "$state/$name.exit"
  echo "$min" > "$state/$name.min"
  # shellcheck disable=SC2016 # expanded by the inner bash
  RUNNER_SESSION=$(cat "$state/session") setsid bash -c '
    timeout --kill-after=30 "$1" bash -c "$2" > "$3.log" 2>&1 & echo $! > "$3.tpid"
    wait $!; echo $? > "$3.exit.tmp"; mv "$3.exit.tmp" "$3.exit"
  ' _ "$((min * 60))" "$(cmd_string "$@")" "$state/$name" < /dev/null > /dev/null 2>&1 &
  echo $! > "$state/$name.pid"
  echo "bg $name: pid $!, timeout $min min, log $state/$name.log"
}

cmd_wait() {
  need_state
  local name=${1:?name} min pid deadline
  min=${2:-$(cat "$state/$name.min" 2>/dev/null || echo 10)}
  pid=$(cat "$state/$name.pid" 2>/dev/null) || die "wait $name: no pidfile"
  deadline=$(( $(now) + min * 60 + 45 ))
  while [ ! -f "$state/$name.exit" ] && [ "$(now)" -lt "$deadline" ]; do
    kill -0 "$pid" 2>/dev/null || break
    sleep 2
  done
  if [ -f "$state/$name.exit" ]; then
    local code; code=$(cat "$state/$name.exit")
    echo "wait $name: exit $code"; tail -n 30 "$state/$name.log"
    return "$code"
  fi
  if kill -0 "$pid" 2>/dev/null; then
    kill -TERM -- "-$(cat "$state/$name.tpid" 2>/dev/null || echo "$pid")" 2>/dev/null || true
    kill -TERM -- "-$pid" 2>/dev/null || true
    echo "TIMEOUT $name after $min min; killed process group $pid. Report it and move on." >&2
    echo "$name timeout" >> "$state/killed"
  else
    echo "wait $name: the process ended without an exit file" >&2
  fi
  return 124
}

# The tree a check ran against: HEAD plus the working tree, so a rerun on an unchanged tree is recognised.
tree_hash() { { git rev-parse HEAD; git diff HEAD; git ls-files --others --exclude-standard | xargs -r cksum; } | cksum | cut -d' ' -f1; }

# check <name> [min] -- cmd: a check that already passed on this tree is not run again; a failed one may run once
# more on the same tree, and passing then marks it flaky; a third run is refused.
cmd_check() {
  need_state
  local name=${1:?name}; shift
  local min=10; [ "${1:-}" != "--" ] && { min=$1; shift; }
  [ "${1:-}" = "--" ] && shift
  local tree runs t0 code=0
  tree=$(tree_hash)
  if awk -v n="$name" -v t="$tree" '$1 == n && $2 == t && $3 == 0 { f = 1 } END { exit !f }' "$state/checks"; then
    echo "check $name: already passed on this tree; not run again"; return 0
  fi
  runs=$(awk -v n="$name" -v t="$tree" '$1 == n && $2 == t' "$state/checks" | wc -l)
  if [ "$runs" -ge 2 ]; then
    echo "check $name: already run twice on this tree; no third run. Record it in the report." >&2; return 2
  fi
  t0=$(now)
  timeout --kill-after=30 "$((min * 60))" bash -c "$(cmd_string "$@")" || code=$?
  echo "$name $tree $code $(( $(now) - t0 ))" >> "$state/checks"
  if [ "$code" = 124 ]; then echo "TIMEOUT check $name after $min min" >&2; fi
  if [ "$code" = 0 ] && [ "$runs" = 1 ]; then echo "$name" >> "$state/flaky"; echo "check $name: passed on the rerun; recorded as flaky"; fi
  return "$code"
}

changed_files() {
  local from; from=$(cat "$state/head" 2>/dev/null || git merge-base HEAD origin/main)
  { git diff --name-only "$from"; git ls-files --others --exclude-standard; } | sort -u | while read -r f; do [ -e "$f" ] && echo "$f"; done
}

cmd_fast() {
  need_state
  local files go_dirs pkgs web landing mobile i18n tools_mod worst=0
  # Every layer runs; the worst exit code is returned at the end.
  layer() { local c=0; cmd_check "$@" || c=$?; [ "$c" -le "$worst" ] || worst=$c; }
  files=$(changed_files)
  [ -n "$files" ] || { echo "fast: nothing changed"; return 0; }
  go_dirs=$(grep -E '^(cmd|internal|pkg)/.*\.go$|^(go\.mod|go\.sum)$|^migrations/|/query\.sql$' <<<"$files" |
    sed -E 's#/[^/]*$##; s#^go\.(mod|sum)$#go.mod#; s#^migrations.*#migrations#' | sort -u || true)
  tools_mod=$(grep -c '^tools/' <<<"$files" || true)
  web=$(grep -E '^web/src/.*\.(ts|tsx)$' <<<"$files" | sed 's#^web/##' || true)
  landing=$(grep -E '^landing/(src|scripts)/.*\.(ts|tsx)$' <<<"$files" | sed 's#^landing/##' || true)
  mobile=$(grep -E '^mobile/(lib|test)/.*\.dart$' <<<"$files" | sed 's#^mobile/##' || true)
  i18n=$(grep -cE '/i18n/|\.arb$|notification/templates/' <<<"$files" || true)
  if [ -n "$go_dirs" ]; then
    local gofiles fmt=""
    # shellcheck disable=SC2086
    pkgs=$("${compose[@]}" run --rm --no-deps -T tools bash scripts/go-affected.sh $go_dirs | paste -sd' ')
    gofiles=$(grep '\.go$' <<<"$files" | paste -sd' ' || true)
    [ -n "$gofiles" ] && fmt="test -z \"\$(gofmt -l $gofiles)\" || { gofmt -l $gofiles; exit 1; }; "
    if [ -n "$pkgs" ]; then
      # The test services start only when an affected package's tests use one (pgtest, redistest, TEST_*).
      local deps="--no-deps" d
      for d in $pkgs; do
        if [ "$d" = "./..." ] || grep -lqE 'pgtest|redistest|TEST_[A-Z_]+' "${d%/...}"/*_test.go 2>/dev/null; then deps=""; break; fi
      done
      echo "fast go: $pkgs${deps:+ (no test services)}"
      layer go-fast 10 -- "${compose[*]} run --rm $deps -T tools sh -c '${fmt}go vet $pkgs && go test -count=1 $pkgs'"
    fi
  fi
  if [ "$tools_mod" -gt 0 ]; then
    layer tools-fast 10 -- "${compose[*]} run --rm --no-deps -T tools sh -c 'go -C tools vet ./... && go -C tools test ./...'"
  fi
  if [ -n "$web" ]; then
    echo "fast web: $(wc -l <<<"$web") files"
    layer web-fast 10 -- "make -s web-install >/dev/null && ${compose[*]} run --rm -T node sh -c 'npx oxlint $(paste -sd' ' <<<"$web") && npx vitest related --run --passWithNoTests $(paste -sd' ' <<<"$web")'"
  fi
  if [ -n "$landing" ]; then
    layer landing-fast 10 -- "make -s landing-install >/dev/null && ${compose[*]} run --rm -T -w /src/landing node sh -c 'npx oxlint $(paste -sd' ' <<<"$landing") && npx vitest related --run --passWithNoTests $(paste -sd' ' <<<"$landing")'"
  fi
  if [ -n "$mobile" ]; then
    local tests
    tests=$(for f in $mobile; do case "$f" in test/*) echo "$f" ;; lib/*) t="test/${f#lib/}"; t="${t%.dart}_test.dart"; [ -f "mobile/$t" ] && echo "$t" ;; esac; done | sort -u | paste -sd' ')
    layer mobile-fast 10 -- "${compose[*]} run --rm -T flutter sh -c 'flutter analyze $(paste -sd' ' <<<"$mobile")${tests:+ && flutter test $tests}'"
  fi
  if [ "$i18n" -gt 0 ]; then layer i18n-fast 5 -- "make -s i18n-check"; fi
  echo "fast: done$([ "$worst" = 0 ] || echo ", exit $worst")"
  return "$worst"
}

# The full set, once: the Go checks always, the apps' checks when the package touched them. The plan's own check
# set, when it names more, is run with `check` by name.
cmd_full() {
  need_state
  local files targets="lint test archcheck"
  files=$(changed_files)
  grep -q '^web/' <<<"$files" && targets="$targets web-lint web-typecheck web-test web-build"
  grep -q '^landing/' <<<"$files" && targets="$targets landing-lint landing-typecheck landing-test landing-build"
  grep -q '^mobile/' <<<"$files" && targets="$targets mobile-analyze mobile-test"
  grep -q '^api/openapi.yaml' <<<"$files" && targets="gen-clients $targets"
  echo "full: make $targets"
  cmd_check full 20 -- "make $targets"
}

# Processes of this session: the session id in their environment (RUNNER_SESSION, set by run-feature.sh and bg,
# or the Claude Code session id, which subagents share with their parent), working in this worktree, and started
# after the session began. Never this shell, its parents, or claude.
session_pids() {
  local id start self p
  id=$(cat "$state/session"); start=$(cat "$state/start")
  local -A skip=()
  p=$$; while [ "$p" -gt 1 ]; do skip[$p]=1; p=$(awk '/^PPid:/ { print $2 }' "/proc/$p/status" 2>/dev/null || echo 1); done
  for d in /proc/[0-9]*; do
    p=${d#/proc/}
    [ -n "${skip[$p]:-}" ] && continue
    [ "$(cat "$d/comm" 2>/dev/null)" = claude ] && continue
    case "$(readlink "$d/cwd" 2>/dev/null)/" in "$root"/*) ;; *) continue ;; esac
    tr '\0' '\n' 2>/dev/null < "$d/environ" | grep -qxE "(RUNNER_SESSION|CLAUDE_CODE_SESSION_ID)=$id" || continue
    self=$(ps -o etimes= -p "$p" 2>/dev/null | tr -d ' ') || continue
    [ -n "$self" ] && [ $(( $(now) - self )) -ge "$start" ] && echo "$p"
  done
}

cmd_cleanup() {
  need_state
  local pids killed=() started down="" c
  local f p
  for f in "$state"/*.pid; do
    [ -e "$f" ] || continue
    p=$(cat "$f")
    if kill -0 "$p" 2>/dev/null; then
      kill -TERM -- "-$(cat "${f%.pid}.tpid" 2>/dev/null || echo "$p")" 2>/dev/null || true
      kill -TERM -- "-$p" 2>/dev/null || kill -TERM "$p" 2>/dev/null || true
      killed+=("$(basename "$f" .pid):$p")
    fi
  done
  pids=$(session_pids | paste -sd' ')
  for p in $pids; do
    c=$(tr -d '\n' 2>/dev/null < "/proc/$p/comm" || echo pid)
    kill -TERM "$p" 2>/dev/null && killed+=("$c:$p")
  done
  sleep 1
  for p in $pids; do kill -KILL "$p" 2>/dev/null || true; done
  # One-off containers (compose run) of this worktree's project created during the session. Not in the main
  # checkout's `salus` project: other sessions may be running checks there.
  local project start
  project=$("${compose[@]}" config --format json 2>/dev/null | jq -r .name || echo salus)
  start=$(cat "$state/start")
  [ "$project" = salus ] || while read -r cid created; do
    [ -n "$cid" ] || continue
    [ "$(date -d "$created" +%s)" -ge "$start" ] && docker rm -f "$cid" >/dev/null && killed+=("container:${cid:0:12}")
  done < <(docker ps -q --filter "label=com.docker.compose.project=$project" --filter label=com.docker.compose.oneoff=True |
    xargs -r docker inspect -f '{{.Id}} {{.Created}}')
  started=$(comm -13 "$state/services.before" <(running_services) | paste -sd' ')
  if [ ! -s "$state/services.before" ] && [ -n "$started" ]; then
    "${compose[@]}" --profile tools --profile web --profile landing --profile mobile --profile e2e down >/dev/null 2>&1 && down="down $project"
  elif [ -n "$started" ]; then
    # shellcheck disable=SC2086
    "${compose[@]}" stop $started >/dev/null 2>&1 && "${compose[@]}" rm -f $started >/dev/null 2>&1 && down="stopped $started"
  fi
  local line="CLEANUP: killed ${#killed[@]}${killed[*]:+ (${killed[*]})}; stack: ${down:-reused, left as found ($(wc -l < "$state/services.before") services)}"
  [ -f "$state/killed" ] && line="$line; timed out: $(cut -d' ' -f1 "$state/killed" | paste -sd' ')"
  echo "$line" | tee "$state/cleanup"
}

cmd_report() {
  need_state
  echo "$(now) end" >> "$state/phases"
  awk 'NR > 1 { printf "%s%s %d", sep, name, int(($1 - t + 30) / 60); sep = ", " } { t = $1; name = $2 } END { print "" }' \
    "$state/phases" | sed "s/^/TIMINGS: /; s/$/ (total $(minutes "$(cat "$state/start")" "$(now)") min)/"
  echo "FLAKY: $(sort -u "$state/flaky" 2>/dev/null | paste -sd' ' | grep . || echo none)"
  cat "$state/cleanup" 2>/dev/null || echo "CLEANUP: not run (run 'bash scripts/runner.sh cleanup' first)"
}

# The Stop hook runs at the end of every turn. A tree that already passed arch-gate, or the full set (which
# contains it), is not tested again.
cmd_gate() {
  local tree mark="$root/.claude/runs/gate.tree"
  tree=$(tree_hash)
  if [ "$(cat "$mark" 2>/dev/null)" = "$tree" ] ||
     awk -v t="$tree" '$1 == "full" && $2 == t && $3 == 0 { found = 1 } END { exit !found }' "$state/checks" 2>/dev/null; then
    echo "arch-gate: this tree already passed; skipped"; return 0
  fi
  make arch-gate
  mkdir -p "$(dirname "$mark")"; echo "$tree" > "$mark"
}

sub=${1:-}; shift || true
case "$sub" in
  env) cmd_env ;;
  start) cmd_start "$@" ;;
  phase) cmd_phase "$@" ;;
  bg) cmd_bg "$@" ;;
  wait) cmd_wait "$@" ;;
  check) cmd_check "$@" ;;
  fast) cmd_fast ;;
  full) cmd_full ;;
  cleanup) cmd_cleanup ;;
  report) cmd_report ;;
  gate) cmd_gate ;;
  *) sed -n '2,17p' "$0"; exit 2 ;;
esac
