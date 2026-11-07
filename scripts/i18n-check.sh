#!/usr/bin/env bash
# make i18n-check, inside the node service (web/ installed, for @formatjs/icu-messageformat-parser, which
# react-intl brings): the checker's own tests, then the check of every catalog (scripts/i18n-check.mjs;
# docs/i18n/SPEC.md §3 "Catalogs"). Runs in make web-lint, make mobile-analyze and make lint.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
node --test --test-reporter=dot "$root/scripts/i18n-check.test.mjs"
node "$root/scripts/i18n-check.mjs" "$root"
