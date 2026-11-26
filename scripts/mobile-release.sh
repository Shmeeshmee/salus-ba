#!/usr/bin/env bash
# The patient app's release build (the mobile workflow's release job, on a manual run or a mobile-v* tag): a release
# APK with the build-time settings a patient must never see as placeholders, passed as --dart-define:
#
#   API_BASE_URL        the api, https://api.<domain>
#   PRIVACY_NOTICE_URL  the published privacy notice the consent screen links to
#   SUPPORT_EMAIL       where a patient asks to reopen a closed account
#
# It refuses (exit 1, naming each variable, never a value) when one is missing, in brackets, not https (the two URLs),
# not an address (SUPPORT_EMAIL), or on localhost, the Android emulator's host (10.0.2.2) or an .invalid domain: the
# same rules as mobile/lib/release_settings.dart, which stops such a release build at start. Run on a machine with the
# Android SDK (the CI runner): the flutter image has none.
#
#   mobile-release.sh [--check]   --check: only the settings, no build (scripts/mobile-release.test.sh)
set -euo pipefail
cd "$(dirname "$0")/.."

placeholder_host() { # host
  case $1 in localhost | 127.0.0.1 | 10.0.2.2 | invalid | *.invalid) return 0 ;; esac
  return 1
}
bad_url() { # value
  local v=$1 rest host
  [[ $v == https://* && $v != *[\[\]]* ]] || return 0
  rest=${v#https://}
  host=${rest%%[/:?#]*}
  [[ -z $host ]] && return 0
  placeholder_host "$host"
}
bad_email() { # value
  local v=$1
  [[ $v =~ ^[^@[:space:]]+@[^@[:space:]]+$ && $v != *[\[\]]* ]] || return 0
  placeholder_host "${v##*@}"
}

problems=()
bad_url "${API_BASE_URL:-}" && problems+=(API_BASE_URL)
bad_url "${PRIVACY_NOTICE_URL:-}" && problems+=(PRIVACY_NOTICE_URL)
bad_email "${SUPPORT_EMAIL:-}" && problems+=(SUPPORT_EMAIL)
if ((${#problems[@]})); then
  echo "mobile-release: missing or still a placeholder: ${problems[*]} (docs/deploy/BEFORE-PRODUCTION.md)" >&2
  exit 1
fi
echo "mobile-release: API_BASE_URL, PRIVACY_NOTICE_URL and SUPPORT_EMAIL are set"
[[ ${1:-} == --check ]] && exit 0

cd mobile
flutter config --no-analytics
flutter build apk --release \
  --dart-define="API_BASE_URL=$API_BASE_URL" \
  --dart-define="PRIVACY_NOTICE_URL=$PRIVACY_NOTICE_URL" \
  --dart-define="SUPPORT_EMAIL=$SUPPORT_EMAIL"
