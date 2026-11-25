#!/usr/bin/env bash
# make check, on the host: exactly the check sets of the plans (docs/clinic-cms/IMPLEMENTATION_PLAN.md, "Check
# sets") that the change's files select, so every package and every person runs the same thing. The change is every
# file that differs from the base (committed or not) plus the untracked ones.
#
#   go       Go, SQL, the API contract, tools/, scripts/, deployments/, build/package/ and the Makefile (the paths of
#            .github/workflows/go.yml): make gen-clients leaves no diff in the generated clients, then make lint
#            test archcheck cover i18n-check, and fhir-validate when a golden FHIR file (testdata/*.json, api/fhir/)
#            changed
#   web      web/: make web-lint web-typecheck web-test
#   mobile   mobile/: make mobile-analyze mobile-test
#   landing  landing/: make landing-lint landing-typecheck landing-test landing-build
#   e2e      test/e2e/: make e2e
#
#   check.sh [--base <rev>] [--sets <set,...>] [--dry-run]
#     --base     what the change is measured from (default: the merge base with origin/main)
#     --sets     sets to run besides the ones the files select, as a package's check sets name them (e.g. e2e)
#     --dry-run  print the sets and the commands only
# The targets of all sets run in one `make -k`, so one failure does not hide the next; the exit code is non-zero
# when any check failed.
set -euo pipefail
cd "$(dirname "$0")/.."

base="" extra="" dry=""
while [ $# -gt 0 ]; do
  case "$1" in
    --base) base=$2; shift 2 ;;
    --sets) extra=$2; shift 2 ;;
    --dry-run) dry=1; shift ;;
    *) echo "check: unknown argument $1" >&2; exit 2 ;;
  esac
done
[ -n "$base" ] || base=$(git merge-base HEAD origin/main 2>/dev/null || git merge-base HEAD main)

files=$({ git diff --name-only "$base"; git ls-files --others --exclude-standard; } | sort -u)

go_paths='^(cmd|internal|pkg|migrations|api|tools|test|deployments|scripts|docs/architecture/archcheck)/|^build/package/|^(go\.mod|go\.sum|Makefile|\.golangci\.yml|\.dockerignore|\.github/workflows/go\.yml)$'
golden='^(cmd|internal|pkg)/.*/testdata/.*\.json$|^api/fhir/'
declare -A on=()
grep -qE "$go_paths" <<<"$files" && on[go]=1
grep -q '^web/' <<<"$files" && on[web]=1
grep -q '^mobile/' <<<"$files" && on[mobile]=1
grep -q '^landing/' <<<"$files" && on[landing]=1
grep -q '^test/e2e/' <<<"$files" && on[e2e]=1
for s in ${extra//,/ }; do
  case "$s" in
    go | web | mobile | landing | e2e) on[$s]=1 ;;
    *) echo "check: unknown set '$s' (go, web, mobile, landing, e2e)"; exit 2 ;;
  esac
done

sets=() targets=()
for s in go web mobile landing e2e; do [ -n "${on[$s]:-}" ] && sets+=("$s"); done
if [ "${#sets[@]}" -eq 0 ]; then echo "check: no check set for these files"; exit 0; fi
for s in "${sets[@]}"; do
  case "$s" in
    go) targets+=(lint test archcheck cover i18n-check)
        grep -qE "$golden" <<<"$files" && targets+=(fhir-validate) ;;
    web) targets+=(web-lint web-typecheck web-test) ;;
    mobile) targets+=(mobile-analyze mobile-test) ;;
    landing) targets+=(landing-lint landing-typecheck landing-test landing-build) ;;
    e2e) targets+=(e2e) ;;
  esac
done

echo "check: sets ${sets[*]}"
[ -z "${on[go]:-}" ] || echo "make gen-clients (then no diff in the generated clients)"
echo "make -k ${targets[*]}"
[ -z "$dry" ] || exit 0

code=0
if [ -n "${on[go]:-}" ]; then
  clients=(web/src/api mobile/packages/salus_api)
  # The clients as they stand, committed or not: gen-clients must leave them exactly so.
  state() { { git status --porcelain -- "${clients[@]}"; git diff HEAD -- "${clients[@]}"
    git ls-files --others --exclude-standard -- "${clients[@]}" | xargs -r cksum; } | cksum; }
  before=$(state)
  if ! make gen-clients; then
    echo "check: make gen-clients failed" >&2; code=1
  elif [ "$(state)" != "$before" ]; then
    echo "check: the generated clients are stale; make gen-clients changed:" >&2
    git status --porcelain -- "${clients[@]}" >&2
    code=1
  fi
fi
make -k "${targets[@]}" || code=$?
if [ "$code" -eq 0 ]; then echo "check: passed (${sets[*]})"; else echo "check: FAILED (${sets[*]})" >&2; fi
exit "$code"
