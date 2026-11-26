#!/usr/bin/env bash
# The migration drill (docs/deploy/restore-drills.md "Migration drills"), in Docker, on this checkout's compose
# Postgres, in three databases of its own that it drops at the end (migdrill_*); no other database is touched.
#   1. From empty to head: cmd/migrate up on an empty database; the version must be the newest file in migrations/.
#   2. From a dump at 00059 (the last migration before the clinic system) to head, with data: goose up-to 59, the
#      synthetic rows of migration-drill/seed-00059.sql, a pg_dump -Fc of it restored into a third database the way
#      the restore runbook restores (pg_restore --exit-on-error), cmd/migrate up, then the checks of
#      migration-drill/check-head.sql: the old rows are all there and the later migrations' backfills hold.
# Prints the times and the counts; exits non-zero on the first failure.
#   scripts/migration-drill.sh
set -euo pipefail
cd "$(dirname "$0")/.."

COMPOSE=(docker compose -f deployments/docker-compose.yml)
PSQL=("${COMPOSE[@]}" exec -T postgres psql -v ON_ERROR_STOP=1 -q -U salus)
TOOLS=("${COMPOSE[@]}" run --rm -T --no-deps tools)
URL=postgres://salus:salus@postgres:5432
head=$(find migrations -name '[0-9]*.sql' | sed 's#.*/0*\([0-9]*\)_.*#\1#' | sort -n | tail -1)
dbs=(migdrill_empty migdrill_59 migdrill_restored)

drop() {
	local d
	for d in "${dbs[@]}"; do "${PSQL[@]}" -d salus -c "DROP DATABASE IF EXISTS $d WITH (FORCE)" >/dev/null 2>&1 || true; done
}
version() { "${PSQL[@]}" -d "$1" -Atc 'SELECT max(version_id) FROM goose_db_version WHERE is_applied'; }
quiet() { grep -v 'level=warning' || true; }

"${COMPOSE[@]}" up -d --wait postgres 2>&1 | quiet >/dev/null
drop
trap drop EXIT
for d in "${dbs[@]}"; do "${PSQL[@]}" -d salus -c "CREATE DATABASE $d"; done

echo "== 1. empty to head ($(printf %05d "$head"))"
start=$(date +%s)
"${TOOLS[@]}" env DATABASE_URL="$URL/migdrill_empty?sslmode=disable" go run ./cmd/migrate up 2>&1 | quiet | tail -1
v=$(version migdrill_empty)
[[ $v == "$head" ]] || { echo "migration-drill: empty to head stopped at $v, want $head" >&2; exit 1; }
echo "ok: version $v in $(($(date +%s) - start)) s"

echo "== 2. a dump at 00059 with data, restored, to head"
"${TOOLS[@]}" go tool -modfile=tools/go.mod goose -dir migrations postgres "$URL/migdrill_59?sslmode=disable" up-to 59 2>&1 |
	quiet | tail -1
"${PSQL[@]}" -d migdrill_59 <scripts/migration-drill/seed-00059.sql
before=$("${PSQL[@]}" -d migdrill_59 -Atc "SELECT string_agg(t || '=' || n, ' ' ORDER BY t) FROM (
	SELECT 'account' t, count(*) n FROM account UNION ALL SELECT 'patient', count(*) FROM patient
	UNION ALL SELECT 'appointment', count(*) FROM appointment UNION ALL SELECT 'encounter', count(*) FROM encounter
	UNION ALL SELECT 'questionnaire_response', count(*) FROM questionnaire_response
	UNION ALL SELECT 'communication', count(*) FROM communication UNION ALL SELECT 'audit_event', count(*) FROM audit_event
	UNION ALL SELECT 'consent', count(*) FROM consent) c")
echo "at 00059: $before"
start=$(date +%s)
"${COMPOSE[@]}" exec -T postgres pg_dump -U salus -d migdrill_59 -Fc |
	"${COMPOSE[@]}" exec -T postgres pg_restore -U salus -d migdrill_restored --exit-on-error
"${TOOLS[@]}" env DATABASE_URL="$URL/migdrill_restored?sslmode=disable" go run ./cmd/migrate up 2>&1 | quiet | tail -1
v=$(version migdrill_restored)
[[ $v == "$head" ]] || { echo "migration-drill: 00059 to head stopped at $v, want $head" >&2; exit 1; }
echo "ok: restored and migrated to version $v in $(($(date +%s) - start)) s"
"${PSQL[@]}" -d migdrill_restored -v before="$before" <scripts/migration-drill/check-head.sql
echo "migration-drill: passed"
