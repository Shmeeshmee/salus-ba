#!/usr/bin/env bash
# Tests scripts/check.sh (make check) in a throwaway git repository: which of the plan's check sets a change's files
# select, and the make targets each set runs (--dry-run). Run by scripts/lint.sh.
set -euo pipefail
unset GIT_DIR GIT_COMMON_DIR GIT_WORK_TREE GIT_INDEX_FILE

here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
failures=0
repo="$work/repo"
mkdir -p "$repo/scripts"
cp "$here/check.sh" "$repo/scripts/"
git -C "$repo" init -q
git -C "$repo" add .
git -C "$repo" -c user.name=test -c user.email=test@example.invalid commit -qm base
base=$(git -C "$repo" rev-parse HEAD)

go_set='make -k lint test archcheck cover i18n-check'
# case <name> <want output> <file...> [-- check.sh args]: the files are created (untracked), the dry run compared,
# and the files removed again.
case_() {
  local name=$1 want=$2; shift 2
  local files=() args=()
  while [ $# -gt 0 ] && [ "$1" != -- ]; do files+=("$1"); shift; done
  [ "${1:-}" = -- ] && { shift; args=("$@"); }
  for f in "${files[@]}"; do mkdir -p "$repo/$(dirname "$f")"; echo x > "$repo/$f"; done
  local got
  got=$(cd "$repo" && bash scripts/check.sh --dry-run --base "$base" "${args[@]}" 2>&1) || true
  if [ "$got" != "$want" ]; then
    echo "FAIL $name:" >&2; diff <(echo "$want") <(echo "$got") | sed 's/^/  /' >&2; failures=$((failures + 1))
  fi
  for f in "${files[@]}"; do rm -f "$repo/$f"; done
  git -C "$repo" clean -qfd
}

case_ "docs only" 'check: no check set for these files' docs/x/PROGRESS.md
case_ "a Go file" "check: sets go
make gen-clients (then no diff in the generated clients)
$go_set" internal/patient/service.go
case_ "a migration and a golden FHIR file" "check: sets go
make gen-clients (then no diff in the generated clients)
$go_set fhir-validate" migrations/00099_x.sql internal/patient/testdata/read.json
case_ "a script" "check: sets go
make gen-clients (then no diff in the generated clients)
$go_set" scripts/runner.sh
case_ "web and mobile" "check: sets web mobile
make -k web-lint web-typecheck web-test mobile-analyze mobile-test" web/src/App.tsx mobile/lib/main.dart
case_ "the landing page" "check: sets landing
make -k landing-lint landing-typecheck landing-test landing-build" landing/src/App.tsx
case_ "an e2e journey" "check: sets go e2e
make gen-clients (then no diff in the generated clients)
$go_set e2e" test/e2e/booking_test.go
case_ "a set the plan names" "check: sets go e2e
make gen-clients (then no diff in the generated clients)
$go_set e2e" internal/patient/service.go -- --sets e2e
case_ "an unknown set" "check: unknown set 'gui' (go, web, mobile, landing, e2e)" internal/x.go -- --sets gui

# A committed change counts from the base on, as an uncommitted one does.
mkdir -p "$repo/web/src" && echo x > "$repo/web/src/a.ts"
git -C "$repo" add . && git -C "$repo" -c user.name=test -c user.email=test@example.invalid commit -qm web
case_ "a committed web file" "check: sets web
make -k web-lint web-typecheck web-test"

if [ "$failures" -ne 0 ]; then
  echo "check.test.sh: $failures failed" >&2
  exit 1
fi
echo "check.test.sh: ok"
