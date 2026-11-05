#!/usr/bin/env bash
# make e2e, after the journeys, in the playwright container: LiveKit's log (on stdin) holds none of the run's
# synthetic names, e-mails and ids, and does hold the run's (opaque) room names, so it is the right log
# (docs/consultation/SPEC.md §2 "LiveKit logs"). $1 is the check the journeys wrote, one line per value:
# "absent|present<TAB>label<TAB>value". An absent value is matched as a whole word, ignoring case (a short given
# name is also a part of ordinary words, as "Ena" of "enabled"). An empty absent value is skipped (it would match
# every blank line); an empty present value fails (it would prove nothing). Only labels are printed, never a value.
set -euo pipefail
check=$1
log=$(mktemp)
trap 'rm -f "$log"' EXIT
cat >"$log"

if [ ! -s "$check" ]; then
  echo "e2e-livekit-log: $check is empty; the consultation journeys write it" >&2
  exit 1
fi
if ! grep -q $'^present\t' "$check"; then
  echo "e2e-livekit-log: the check names no room to find in LiveKit's log" >&2
  exit 1
fi

failed=0 absent=0 present=0
while IFS=$'\t' read -r kind label value; do
  if [ -z "$value" ]; then # an empty pattern matches every line, or proves nothing
    if [ "$kind" = present ]; then
      echo "e2e-livekit-log: the check's $label is empty" >&2
      failed=1
    fi
    continue
  fi
  case $kind in
    absent)
      absent=$((absent + 1))
      if grep -qiwF -- "$value" "$log"; then
        echo "e2e-livekit-log: LiveKit's log holds the run's synthetic $label" >&2
        failed=1
      fi
      ;;
    present)
      present=$((present + 1))
      if ! grep -qF -- "$value" "$log"; then
        echo "e2e-livekit-log: LiveKit's log does not hold the run's $label; is it LiveKit's log?" >&2
        failed=1
      fi
      ;;
    *)
      echo "e2e-livekit-log: unknown line kind in $check" >&2
      failed=1
      ;;
  esac
done <"$check"

if [ "$failed" -ne 0 ]; then
  exit 1
fi
echo "e2e-livekit-log: LiveKit's log ($(wc -l <"$log") lines) holds none of the run's $absent synthetic values and each of its $present rooms"
