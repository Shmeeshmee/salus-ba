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
# Read the whole list first: piping git diff into grep -q lets grep exit on the first match, git dies of
# SIGPIPE, and pipefail turns a match into "changed=false". A failing git diff fails the job.
files=$(git diff --name-only "origin/${GITHUB_BASE_REF}...HEAD")
if grep -Eq "$pattern" <<<"$files"; then
  echo "changed=true"
else
  echo "changed=false"
fi
