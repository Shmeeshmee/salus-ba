#!/usr/bin/env bash
# Prints "changed=true" or "changed=false" for GITHUB_OUTPUT: whether a pull request touches a path matching
# the extended regex in $1. Pushes and manual runs always count as changed. Lets a workflow run on every
# pull request (so its required checks report) and skip its jobs when nothing relevant changed.
set -euo pipefail
pattern=$1

if [ "${GITHUB_EVENT_NAME:-}" != "pull_request" ]; then
  echo "changed=true"
  exit 0
fi
if git diff --name-only "origin/${GITHUB_BASE_REF}...HEAD" | grep -Eq "$pattern"; then
  echo "changed=true"
else
  echo "changed=false"
fi
