#!/usr/bin/env bash
# The settings check of scripts/mobile-release.sh (--check, no build), run by make lint: real values pass; each
# placeholder, a missing value, http, localhost and .invalid are refused by name.
set -euo pipefail
cd "$(dirname "$0")"
fails=0

run() { # expected exit, expected names ("" for none), env assignments...
  local want=$1 names=$2 out got=0
  shift 2
  out=$(env -i PATH="$PATH" API_BASE_URL=https://api.clinic.example PRIVACY_NOTICE_URL=https://clinic.example/privacy \
    SUPPORT_EMAIL=support@clinic.example "$@" bash mobile-release.sh --check 2>&1) || got=$?
  if [[ $got != "$want" ]] || [[ -n $names && $out != *"placeholder: $names ("* ]]; then
    echo "FAIL: $* exited $got (want $want): $out" >&2
    fails=$((fails + 1))
  fi
}

run 0 ""
run 1 "API_BASE_URL PRIVACY_NOTICE_URL SUPPORT_EMAIL" API_BASE_URL=http://localhost:8080 \
  PRIVACY_NOTICE_URL=https://salus.invalid/privacy "SUPPORT_EMAIL=[SUPPORT EMAIL]"
for v in "" http://api.clinic.example https://localhost https://127.0.0.1:8080 http://10.0.2.2:8080 https://api.salus.invalid "https://[APP]"; do
  run 1 API_BASE_URL "API_BASE_URL=$v"
done
for v in "" https://salus.invalid/privacy http://clinic.example/privacy "[PRIVACY URL]"; do
  run 1 PRIVACY_NOTICE_URL "PRIVACY_NOTICE_URL=$v"
done
for v in "" "[SUPPORT EMAIL]" support "help@salus.invalid" "a b@clinic.example"; do
  run 1 SUPPORT_EMAIL "SUPPORT_EMAIL=$v"
done

if ((fails)); then
  echo "mobile-release.test: $fails failed" >&2
  exit 1
fi
echo "mobile-release.test: passed"
