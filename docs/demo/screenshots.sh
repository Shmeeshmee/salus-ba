#!/usr/bin/env bash
# Retakes docs/demo/screenshots/*.png at 1280 px against this checkout's stack, after `make demo-seed` filled it
# (SCRIPT.md, "Retaking the screenshots"). The three TOTP secrets are the ones the seed printed:
#
#   DEMO_TOTP_ADMIN=... DEMO_TOTP_RECEPTION=... DEMO_TOTP_PRACTITIONER=... bash docs/demo/screenshots.sh
#
# DEMO_TOTP_PRACTITIONER is Dr. Emir Primjerović's. The ids of the visits it opens are read from the stack's database
# (synthetic, the seed's own). Needs `make web-install e2e-install` once.
set -euo pipefail
cd "$(dirname "$0")/../.."
: "${DEMO_TOTP_ADMIN:?the TOTP secret of the clinic admin, as make demo-seed printed it}"
: "${DEMO_TOTP_RECEPTION:?the TOTP secret of the receptionist}"
: "${DEMO_TOTP_PRACTITIONER:?the TOTP secret of Dr. Emir Primjerović}"
compose="docker compose -f deployments/docker-compose.yml"
sql() { $compose exec -T postgres psql -U salus -d salus -Atc "$1"; }

visit_questionnaire=$(sql "SELECT appointment_id FROM questionnaire_response WHERE status = 'completed' LIMIT 1")
visit_lab=$(sql "SELECT appointment_id FROM diagnostic_report LIMIT 1")
questionnaire=$(sql "SELECT id FROM questionnaire WHERE status = 'active' LIMIT 1")
chat=$(sql "SELECT id FROM conversation LIMIT 1")

$compose --profile e2e run --rm -T --no-deps -w /src/docs/demo \
	-e NODE_PATH=/src/test/e2e/web/node_modules \
	-e DEMO_TOTP_ADMIN -e DEMO_TOTP_RECEPTION -e DEMO_TOTP_PRACTITIONER \
	-e DEMO_VISIT_QUESTIONNAIRE="$visit_questionnaire" -e DEMO_VISIT_LAB="$visit_lab" \
	-e DEMO_QUESTIONNAIRE="$questionnaire" -e DEMO_CHAT="$chat" \
	playwright /src/test/e2e/web/node_modules/.bin/playwright test --config playwright.config.ts
