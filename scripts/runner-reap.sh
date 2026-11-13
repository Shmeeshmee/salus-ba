#!/usr/bin/env bash
# Lists, and with --kill ends, what runner and Claude Code sessions left behind, older than N hours:
# - processes in a salus checkout that carry a session id in their environment (RUNNER_SESSION from
#   run-feature.sh and runner.sh, or CLAUDE_CODE_SESSION_ID from a Claude Code shell);
# - the one-off `compose run` containers of salus projects;
# - the compose stacks of salus worktrees (projects salus-*) whose containers are all older than N hours.
# The main checkout's `salus` stack, the claude processes themselves and the caller's own session are never
# touched. Dry run by default.
#
# Usage: bash scripts/runner-reap.sh [--hours N] [--kill] [--session ID]
#   --hours N     age threshold, default 2 (0 with --session)
#   --session ID  only that session's processes (run-feature.sh uses it after each package)
set -euo pipefail

hours=""; kill=0; only=""
while [ $# -gt 0 ]; do
  case "$1" in
    --hours) hours=$2; shift 2 ;;
    --kill) kill=1; shift ;;
    --session) only=$2; shift 2 ;;
    *) sed -n '2,12p' "$0"; exit 2 ;;
  esac
done
[ -n "$hours" ] || hours=$([ -n "$only" ] && echo 0 || echo 2)
limit=$(( hours * 3600 ))
now=$(date +%s)
mine="${RUNNER_SESSION:-}|${CLAUDE_CODE_SESSION_ID:-}"
act() { if [ "$kill" = 1 ]; then "$@"; fi; }
verb=$([ "$kill" = 1 ] && echo killed || echo would-kill)
count=0

# The caller and its parents are never candidates.
declare -A skip=()
p=$$; while [ "$p" -gt 1 ]; do skip[$p]=1; p=$(awk '/^PPid:/ { print $2 }' "/proc/$p/status" 2>/dev/null || echo 1); done

echo "== processes (older than ${hours} h)"
for d in /proc/[0-9]*; do
  pid=${d#/proc/}
  [ -n "${skip[$pid]:-}" ] && continue
  [ "$(cat "$d/comm" 2>/dev/null)" = claude ] && continue
  env=$(tr '\0' '\n' 2>/dev/null < "$d/environ") || continue
  sid=$(sed -n 's/^RUNNER_SESSION=//p' <<<"$env" | head -1)
  [ -n "$sid" ] || sid=$(sed -n 's/^CLAUDE_CODE_SESSION_ID=//p' <<<"$env" | head -1)
  [ -n "$sid" ] || continue
  # Only salus checkouts (…/salus, …/salus-<worktree>) and their scratch directories.
  cwd=$(readlink "$d/cwd" 2>/dev/null) || continue
  [[ "$cwd" =~ /salus(-[^/]*)?(/|$) ]] || [[ "$cwd" =~ -home-[^/]*-salus ]] || continue
  if [ -n "$only" ]; then [ "$sid" = "$only" ] || continue
  else case "|$mine|" in *"|$sid|"*) continue ;; esac
  fi
  age=$(ps -o etimes= -p "$pid" 2>/dev/null | tr -d ' ') || continue
  if [ -z "$age" ] || [ "$age" -lt "$limit" ]; then continue; fi
  owner=$(sed -n 's/^CLAUDE_PID=//p' <<<"$env" | head -1)
  state="session alive"
  { [ -z "$owner" ] || ! kill -0 "$owner" 2>/dev/null; } && state="orphan"
  printf '%s pid %s, %s h, %s, session %s: %s\n' "$verb" "$pid" "$((age / 3600))" "$state" "${sid:0:12}" \
    "$(tr '\0\n' '  ' 2>/dev/null < "$d/cmdline" | cut -c1-120)"
  act kill -TERM "$pid" 2>/dev/null || true
  count=$((count + 1))
done

[ -z "$only" ] || { echo "== $count processes"; exit 0; }

age_of() { echo $(( $(date +%s) - $(date -d "$1" +%s) )); }

echo "== one-off compose run containers of salus projects (older than ${hours} h)"
while read -r id name created project; do
  [ -n "$id" ] || continue
  case "$project" in salus|salus-*) ;; *) continue ;; esac
  a=$(age_of "$created"); [ "$a" -ge "$limit" ] || continue
  echo "$verb container $name ($project), $((a / 3600)) h"
  act docker rm -f "$id" >/dev/null
  count=$((count + 1))
done < <(docker ps -q --filter label=com.docker.compose.oneoff=True |
  xargs -r docker inspect -f '{{.Id}} {{.Name}} {{.Created}} {{index .Config.Labels "com.docker.compose.project"}}')

echo "== worktree compose stacks (salus-*, every container older than ${hours} h)"
while read -r project; do
  [ -n "$project" ] || continue
  newest=$(docker ps -aq --filter "label=com.docker.compose.project=$project" | xargs -r docker inspect -f '{{.Created}}' |
    while read -r c; do age_of "$c"; done | sort -n | head -1)
  if [ -z "$newest" ] || [ "$newest" -lt "$limit" ]; then echo "keep stack $project (a container is younger than ${hours} h)"; continue; fi
  echo "$verb stack $project (newest container $((newest / 3600)) h): docker compose -p $project down"
  act docker compose -p "$project" down --remove-orphans >/dev/null 2>&1
  count=$((count + 1))
done < <(docker compose ls -a --format json | jq -r '.[].Name' | grep '^salus-' || true)

echo "== $count found$([ "$kill" = 1 ] || echo '; dry run, pass --kill to act')"
