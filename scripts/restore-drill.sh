#!/usr/bin/env bash
# The local restore drill (docs/deploy/restore-drills.md "Local drills"): docs/deploy/backup-restore.md rehearsed end to
# end on this checkout's own compose stack, with the real backup script (deployments/production/bootstrap/backup.sh)
# writing to a throwaway SFTP server that stands in for the Storage Box. Synthetic data only.
#
#   1. seed      the stack up, migrated; the drill's journey (test/e2e/restore_drill_test.go, tags e2e,drill) makes a
#                clinic, a patient, an attachment (scanned clean) and a fiscalized receipt
#   2. backup    the app services stopped (so the counts compare exactly), every table's row count noted, then
#                salus-backup and salus-backup objects, as the server runs them, to the SFTP server
#   3. destroy   the stack down and this project's postgres and objectstore volumes removed (nothing else)
#   4. restore   the runbook's steps: fetch the newest dump and archive, Postgres alone, the roles, decrypt and
#                pg_restore in one stream, migrate (passwords), the objects into Garage with rclone, the counts
#                compared table by table, rows against objects; then the rest of the stack
#   5. verify    the drill's journey signs in again (password and TOTP), downloads the attachment (decrypted by the
#                api with BINARY_KEK) and compares its SHA-1, and downloads the receipt's PDF
#
# It destroys data: it refuses the project `salus` (the main checkout's dev stack) and runs only on the project of
# deployments/.env (a worktree's, scripts/runner.sh start). Its working directory (keys, the encrypted files, the
# state) is a fresh 0700 directory under ${TMPDIR:-/tmp}, removed at the end. Prints one summary line per step and the
# times; exits non-zero on the first failure.
#   scripts/restore-drill.sh
set -euo pipefail
cd "$(dirname "$0")/.."

P=${COMPOSE_PROJECT_NAME:-$(sed -n 's/^COMPOSE_PROJECT_NAME=//p' deployments/.env 2>/dev/null | tail -n 1 || true)}
[[ -n $P && $P != salus ]] || {
	echo "restore-drill: refuses the project '${P:-salus}': it removes the project's database and object store." >&2
	echo "Run it in a worktree with its own COMPOSE_PROJECT_NAME (scripts/runner.sh start)." >&2
	exit 2
}
export COMPOSE_PROJECT_NAME=$P
# As the Makefile: the tools container runs as the host user and sees the git directory (go build stamps the commit).
HOST_UID=$(id -u) HOST_GID=$(id -g) GIT_COMMON_DIR=$(git rev-parse --path-format=absolute --git-common-dir)
export HOST_UID HOST_GID GIT_COMMON_DIR
tag=${SALUS_IMAGE_TAG:-$(sed -n 's/^SALUS_IMAGE_TAG=//p' deployments/.env 2>/dev/null | tail -n 1 || true)}
[[ -n $tag && $tag != dev ]] || {
	echo "restore-drill: set SALUS_IMAGE_TAG in deployments/.env, so the drill builds its own app image, not salus:dev" >&2
	exit 2
}
NET=${P}_default
C=(docker compose -p "$P" -f deployments/docker-compose.yml)
APP=(api worker scheduler notifier agent)
# The services the drill runs: the app and what it needs (not the landing page or the TLS Mailpits).
STACK=(postgres redis objectstore clamd mailpit livekit migrate "${APP[@]}")
IMAGE=salus-restore-drill:local # the backup test's image: age, sftp, the Docker CLI, sshd
RCLONE_IMAGE=$(sed -n 's/^readonly RCLONE_IMAGE=//p' deployments/production/bootstrap/backup.sh)
SFTP=$P-drill-sftp
# The dev Garage's synthetic key pair (deployments/docker-compose.yml).
S3_KEY=GK5a1a50000000000000000001
S3_SECRET=5a1a500000000000000000000000000000000000000000000000000000000001
W=$(mktemp -d "${TMPDIR:-/tmp}/salus-restore-drill.XXXXXX")
chmod 700 "$W"
mkdir -p "$W/keys" "$W/staging" "$W/fetched"

quiet() { grep -v 'level=warning' || true; }
step() { printf '\n== %s (%s)\n' "$*" "$(date -u +%H:%M:%S)"; }
cleanup() {
	docker rm -f "$SFTP" >/dev/null 2>&1 || true
	# The keys and the state are synthetic, but the files are root's when a container wrote them.
	docker run --rm -v "$W:/w" "$IMAGE" sh -c 'rm -rf /w/* /w/.[!.]*' >/dev/null 2>&1 || true
	rm -rf "$W"
}
trap cleanup EXIT
# in_runner ARGS...: a command in the drill's image on the stack's network with the working directory at /drill.
in_runner() { docker run --rm -i --network "$NET" -v /var/run/docker.sock:/var/run/docker.sock -v "$W:/drill" \
	-v "$PWD/deployments/production/bootstrap:/bootstrap:ro" "$IMAGE" "$@"; }
psql_() { "${C[@]}" exec -T postgres psql -v ON_ERROR_STOP=1 -U salus -d salus "$@"; }
# counts FILE: every table's exact row count, one "schema.table count" per line.
counts() {
	psql_ -Atc "SELECT n.nspname || '.' || c.relname || ' ' || (xpath('/row/c/text()', query_to_xml(format('SELECT count(*) AS c FROM %I.%I', n.nspname, c.relname), false, true, '')))[1]::text
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p') AND NOT c.relispartition AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY 1" >"$1"
}
# drill_test PHASE: the drill's journey in the tools container on the stack's network.
drill_test() {
	local status
	"${C[@]}" run --rm -T --no-deps -v "$W:/drill" -e E2E_BASE_URL=http://api:8080 -e E2E_MAILPIT_URL=http://mailpit:8025 \
		-e E2E_SALUSCTL=/tmp/salusctl -e E2E_REDIS_URL=redis://redis:6379/0 -e RESTORE_DRILL_STATE=/drill/state.json tools \
		sh -c "go build -o /tmp/salusctl ./cmd/salusctl && go test -tags e2e,drill -count=1 -v -run '^TestRestoreDrill_$1\$' ./test/e2e/" \
		>"$W/$1.log" 2>&1 && status=0 || status=$?
	grep -E '^(--- |ok|FAIL|PASS)|_test.go:' "$W/$1.log" || true
	[[ $status -eq 0 ]] || { echo "restore-drill: the $1 journey failed" >&2; exit 1; }
}

start=$(date +%s)
step "0. images"
docker build -q -t "$IMAGE" deployments/production/bootstrap/test >/dev/null
"${C[@]}" build -q migrate 2>&1 | quiet
"${C[@]}" --profile tools build -q tools 2>&1 | quiet

step "1. seed: the stack of $P up, the drill's journey"
"${C[@]}" up -d --wait "${STACK[@]}" 2>&1 | quiet | tail -1
drill_test Seed

step "2. backup with salus-backup to a throwaway SFTP server"
backup_start=$(date +%s)
"${C[@]}" stop "${APP[@]}" 2>&1 | quiet >/dev/null
counts "$W/before.counts"
echo "tables: $(wc -l <"$W/before.counts"), rows: $(awk '{s += $2} END {print s}' "$W/before.counts")"
in_runner sh -c 'ssh-keygen -q -t ed25519 -N "" -C synthetic-drill-backup -f /drill/keys/backup && age-keygen -o /drill/keys/age.key 2>/dev/null &&
	age-keygen -y /drill/keys/age.key >/drill/keys/age.pub'
docker run -d --name "$SFTP" --network "$NET" --network-alias drill-sftp -v "$W/keys:/keys:ro" "$IMAGE" sh -c \
	'install -d -m 700 -o storagebox -g storagebox /home/storagebox/.ssh && install -m 600 -o storagebox -g storagebox /keys/backup.pub /home/storagebox/.ssh/authorized_keys && exec /usr/sbin/sshd -D -e' >/dev/null
# shellcheck disable=SC2016 # expanded by the runner's shell
in_runner sh -c 'for i in $(seq 100); do ssh-keyscan -t ed25519 drill-sftp >/drill/known_hosts 2>/dev/null && [ -s /drill/known_hosts ] && exit 0; sleep 0.2; done; exit 1'
cat >"$W/salus.env" <<EOF
S3_ACCESS_KEY_ID=$S3_KEY
S3_SECRET_ACCESS_KEY=$S3_SECRET
S3_BUCKET=salus-binaries
EOF
cat >"$W/bootstrap.conf" <<EOF
BACKUP_HOST=drill-sftp
BACKUP_PORT=22
BACKUP_USER=storagebox
BACKUP_PATH=salus/postgres
BACKUP_OBJECTS_PATH=salus/objects
BACKUP_AGE_RECIPIENT=$(cat "$W/keys/age.pub")
BACKUP_COMPOSE_PROJECT=$P
BACKUP_ENV_FILE=/drill/salus.env
BACKUP_SSH_KEY_FILE=/drill/keys/backup
BACKUP_KNOWN_HOSTS_FILE=/drill/known_hosts
BACKUP_STAGING_DIR=/drill/staging
BACKUP_WAIT_SECONDS=60
EOF
in_runner env SALUS_BACKUP_CONFIG=/drill/bootstrap.conf bash /bootstrap/backup.sh postgres
in_runner env SALUS_BACKUP_CONFIG=/drill/bootstrap.conf bash /bootstrap/backup.sh objects
box() { printf '%s\n' "$@" | in_runner sftp -q -b - -i /drill/keys/backup -o UserKnownHostsFile=/drill/known_hosts storagebox@drill-sftp; }
backed_up=$(date +%s)

step "3. destroy: the stack down, the volumes ${P}_postgres and ${P}_objectstore removed"
"${C[@]}" down --remove-orphans 2>&1 | quiet >/dev/null
docker volume rm "${P}_postgres" "${P}_objectstore" >/dev/null
docker volume inspect "${P}_postgres" >/dev/null 2>&1 && { echo "restore-drill: ${P}_postgres is still there" >&2; exit 1; }
echo "removed"

step "4. restore by docs/deploy/backup-restore.md"
restore_start=$(date +%s)
# Step 1 and 7 of the runbook's drill: the newest dump and archive, fetched (still encrypted).
dump=$(box 'ls -1 salus/postgres' | sed 's#.*/##' | grep '\.dump\.age$' | sort | tail -1)
archive=$(box 'ls -1 salus/objects' | sed 's#.*/##' | grep '\.tar\.age$' | sort | tail -1)
box "get salus/postgres/$dump /drill/fetched/drill.dump.age" "get salus/objects/$archive /drill/fetched/drill.objects.tar.age" >/dev/null
echo "dump $dump ($(stat -c %s "$W/fetched/drill.dump.age") bytes), archive $archive ($(stat -c %s "$W/fetched/drill.objects.tar.age") bytes)"
# Step 2: Postgres alone, on its new empty volume.
"${C[@]}" up -d --wait postgres 2>&1 | quiet >/dev/null
# Step 3: the roles the migrations create.
grep -ho 'CREATE ROLE [a-z_]* [A-Z ]*' migrations/*.sql | sort -u | sed 's/ *$/;/' | psql_ -q
# Step 5: decrypt and restore in one stream.
in_runner age -d -i /drill/keys/age.key /drill/fetched/drill.dump.age |
	"${C[@]}" exec -T postgres pg_restore -U salus -d salus --exit-on-error
# "Restoring onto a new server" step 7: migrate sets the roles' passwords and applies nothing new.
"${C[@]}" run --rm -T migrate 2>&1 | quiet | tail -1
psql_ -Atc "SELECT 'migration version ' || max(version_id) FROM goose_db_version WHERE is_applied"
# Steps 7 and 8: the archive decrypted into the working directory, extracted into the new Garage with rclone.
"${C[@]}" up -d --wait objectstore 2>&1 | quiet >/dev/null
in_runner sh -c 'umask 077 && age -d -i /drill/keys/age.key /drill/fetched/drill.objects.tar.age >/drill/fetched/drill.objects.tar'
rcl() { docker run --rm --network "$NET" -v "$W/fetched:/fetched:ro" -e RCLONE_CONFIG_D_TYPE=s3 -e RCLONE_CONFIG_D_PROVIDER=Other \
	-e RCLONE_CONFIG_D_REGION=garage -e RCLONE_CONFIG_D_ENDPOINT=http://objectstore:3900 -e RCLONE_CONFIG_D_ACCESS_KEY_ID=$S3_KEY \
	-e RCLONE_CONFIG_D_SECRET_ACCESS_KEY=$S3_SECRET "$RCLONE_IMAGE" --config= -q "$@"; }
rcl archive extract /fetched/drill.objects.tar D:salus-binaries
echo "objects: $(rcl size D:salus-binaries | tr '\n' ' ')"
restored=$(date +%s)
# Step 6, exactly: every table's count against the one noted before the backup.
counts "$W/after.counts"
if ! diff "$W/before.counts" "$W/after.counts"; then
	echo "restore-drill: row counts differ between the backup and the restore" >&2
	exit 1
fi
echo "row counts: equal in all $(wc -l <"$W/after.counts") tables"
# Step 9: rows against objects (questionnaire attachments, lab result PDFs and documents' PDFs are rows of binary).
psql_ -Atc "SELECT id FROM \"binary\" WHERE status <> 'rejected' ORDER BY 1" >"$W/rows"
rcl lsf --files-only D:salus-binaries | grep -v / | sort >"$W/objects" || true
echo "binary rows: $(wc -l <"$W/rows"); top-level objects: $(wc -l <"$W/objects");" \
	"rows without an object: $(comm -23 "$W/rows" "$W/objects" | wc -l); objects without a row: $(comm -13 "$W/rows" "$W/objects" | wc -l)"
# "Restoring onto a new server", the sessions step: every session of the dump is closed, so a sign-out or a password
# change made after the dump's time cannot come back.
psql_ -Atc "UPDATE session SET revoked_at = now() WHERE revoked_at IS NULL" | sed 's/^/sessions closed: /'
"${C[@]}" up -d --wait "${STACK[@]}" 2>&1 | quiet | tail -1

step "5. verify: sign in again, the attachment decrypted, the receipt's PDF"
drill_test Verify

now=$(date +%s)
printf '\nrestore-drill: passed. backup %d s; fetch to restored %d s; whole drill %d s\n' \
	"$((backed_up - backup_start))" "$((restored - restore_start))" "$((now - start))"
