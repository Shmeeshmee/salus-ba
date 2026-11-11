#!/usr/bin/env bash
# Fails when the API paths the production Caddyfile routes on app. differ from the dev server's proxy in
# web/vite.config.ts. A string key '/x' there is a path prefix, `path /x*` here; a key starting with ^ is a regular
# expression, `path_regexp <key>` here. Only the Caddyfile's lines between `# vite-proxy:begin` and
# `# vite-proxy:end` count. Run by scripts/lint.sh.
set -euo pipefail

vite=${1:-web/vite.config.ts}
caddyfile=${2:-deployments/production/Caddyfile}

expected=$(sed -n '/proxy: {/,/^    },/p' "$vite" | sed -n "s/^ *'\([^']*\)':.*/\1/p" |
  awk '/^\^/ { print "path_regexp " $0; next } { print "path " $0 "*" }' | sort)
routed=$(sed -n '/# vite-proxy:begin/,/# vite-proxy:end/p' "$caddyfile" |
  grep -oE '(^|[[:space:]])(path|path_regexp) [^[:space:]]+' | sed 's/^[[:space:]]*//' | sort)

if [ -z "$expected" ] || [ -z "$routed" ]; then
  echo "check-caddy-routes: found no routes in $vite or $caddyfile" >&2
  exit 2
fi
if ! diff <(echo "$expected") <(echo "$routed") >/dev/null; then
  echo "check-caddy-routes: $caddyfile's app. routes differ from $vite's proxy (< vite, > Caddyfile):" >&2
  diff <(echo "$expected") <(echo "$routed") >&2 || true
  exit 1
fi
echo "check-caddy-routes: ok ($(echo "$expected" | wc -l | tr -d ' ') routes match $vite)"
