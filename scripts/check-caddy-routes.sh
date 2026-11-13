#!/usr/bin/env bash
# Fails when the API paths the production Caddyfile routes on app. differ from the dev server's proxy in
# web/vite.config.ts. Only the Caddyfile's lines between `# vite-proxy:begin` and `# vite-proxy:end` count. Checks:
# - every argument of every `path` matcher there is a vite key plus `*` (a string key is a path prefix), every
#   `path_regexp` pattern a vite key starting with ^, and every vite key appears there exactly once;
# - the keys whose GETs the dev server leaves to the app (`bypass`) are exactly the paths of the matchers with
#   `not method GET`;
# - every matcher there has a `handle @name` whose only line is `reverse_proxy api:8080`;
# - the global log filter deletes the request's URI and the response headers, so the token of a one-time link
#   (/a/<token> on api., and the Location of its 303, SAL-34) is never logged.
# Run by scripts/lint.sh.
set -euo pipefail

vite=${1:-web/vite.config.ts}
caddyfile=${2:-deployments/production/Caddyfile}
fail() { echo "check-caddy-routes: $*" >&2; exit 1; }

proxy=$(sed -n '/proxy: {/,/^    },/p' "$vite")
expected=$(sed -n "s/^ *'\([^']*\)':.*/\1/p" <<<"$proxy" |
  awk '/^\^/ { print "path_regexp " $0; next } { print "path " $0 "*" }' | sort)
bypassed=$(grep "bypass" <<<"$proxy" | sed -n "s/^ *'\([^']*\)':.*/path \1*/p" | sort || true)
region=$(sed -n '/# vite-proxy:begin/,/# vite-proxy:end/p' "$caddyfile" | sed 's/#.*//')
if [ -z "$expected" ] || [ -z "$region" ]; then
  echo "check-caddy-routes: no routes in $vite or $caddyfile" >&2
  exit 2
fi

# One line per matcher argument: "<matcher> path <arg>", "<matcher> path_regexp <pattern>", "<matcher> not-get".
args=$(awk '
  /^[[:space:]]*@[A-Za-z0-9_-]+/ { name = $1; sub(/^@/, "", name); inblock = ($NF == "{") }
  /^[[:space:]]*}/ { inblock = 0 }
  {
    start = 1
    if ($1 ~ /^@/) start = 2
    else if (!inblock) next
    if ($start == "path") for (i = start + 1; i <= NF; i++) print name " path " $i
    if ($start == "path_regexp") print name " path_regexp " $NF
    if ($start == "not" && $(start + 1) == "method" && $(start + 2) == "GET" && NF == start + 2) print name " not-get"
  }' <<<"$region")

routed=$(awk '$2 != "not-get" { print $2 " " $3 }' <<<"$args" | sort)
if [ "$routed" != "$expected" ]; then
  echo "check-caddy-routes: $caddyfile's app. routes differ from $vite's proxy (< vite, > Caddyfile):" >&2
  diff <(echo "$expected") <(echo "$routed") >&2 || true
  exit 1
fi

notget=$(awk '$2 == "not-get" { print $1 }' <<<"$args" | sort -u)
withget=$(for m in $notget; do awk -v m="$m" '$1 == m && $2 == "path" { print "path " $3 }' <<<"$args"; done | sort)
[ "$withget" = "$bypassed" ] ||
  fail "the paths whose GETs go to the app (${bypassed//$'\n'/, }) must be exactly those of the \`not method GET\` matchers"

while read -r m; do
  body=$(awk -v h="handle @$m {" '
    /^[[:space:]]*handle / { line = $0; sub(/^[[:space:]]+/, "", line); on = (line == h); next }
    on && /^[[:space:]]*}/ { on = 0; next }
    on && NF { sub(/^[[:space:]]+/, ""); print }' <<<"$region")
  [ "$body" = "reverse_proxy api:8080" ] || fail "handle @$m must hold exactly \`reverse_proxy api:8080\`, has: ${body:-nothing}"
done < <(awk '{ print $1 }' <<<"$args" | sort -u)

grep -Eq '^[[:space:]]*request>uri[[:space:]]+delete[[:space:]]*$' "$caddyfile" ||
  fail "the log filter must delete request>uri: a one-time link's token is in the path of /a/"
grep -Eq '^[[:space:]]*resp_headers[[:space:]]+delete[[:space:]]*$' "$caddyfile" ||
  fail "the log filter must delete resp_headers: a one-time link's 303 carries the token in Location"

echo "check-caddy-routes: ok ($(wc -l <<<"$expected" | tr -d ' ') routes match $vite)"
