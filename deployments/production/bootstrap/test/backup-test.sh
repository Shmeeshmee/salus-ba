#!/usr/bin/env bash
# Runs in the backup test's runner container (compose.yml, run.sh): salus-backup against a throwaway Postgres and
# SFTP server with a generated age key: dump, upload, fetch, decrypt, restore, retention, and its failures.
set -euo pipefail

BACKUP=/bootstrap/backup.sh
WORK=/tmp/backup-test
BOX=salus/postgres
PROJECT=${TEST_PROJECT:?}

pass() { printf 'ok: %s\n' "$*"; }
fail() {
	printf 'FAIL: %s\n' "$*" >&2
	exit 1
}
expect() {
	local desc=$1
	shift
	if "$@"; then pass "$desc"; else fail "$desc"; fi
}
container() { docker ps -q --filter "label=com.docker.compose.project=$PROJECT" --filter "label=com.docker.compose.service=$1"; }
psql_in() { docker exec -i "$(container "$1")" psql -U salus -d salus -Atqc "$2"; }
box() { # box SFTP-COMMANDS...: runs them against the test server with the pinned host key
	printf '%s\n' "$@" | sftp -q -b - -i /keys/backup -o BatchMode=yes -o StrictHostKeyChecking=yes \
		-o UserKnownHostsFile="$WORK/known_hosts" storagebox@sftp
}
box_ls() { box "ls -1a $BOX" | sed -n "s#^$BOX/##p" | grep -v '^\.\{1,2\}$' | sort; }
stamp() { date -u -d "$1" +%Y%m%dT%H%M%SZ; } # a dump name's time, e.g. stamp '-13 days'

mkdir -p "$WORK/staging"
ssh-keygen -q -t ed25519 -N '' -C synthetic-test-backup -f /keys/backup
age-keygen -o "$WORK/age.key" 2>/dev/null
recipient=$(age-keygen -y "$WORK/age.key")

for _ in $(seq 100); do
	ssh-keyscan -t ed25519 sftp >"$WORK/known_hosts" 2>/dev/null && [[ -s $WORK/known_hosts ]] && break
	sleep 0.2
done
[[ -s $WORK/known_hosts ]] || fail "the SFTP server did not come up"

write_conf() { # write_conf KNOWN_HOSTS_FILE KEEP_LOCAL PROJECT [SERVICE]
	cat >"$WORK/bootstrap.conf" <<EOF
DEPLOY_SSH_KEY=ignored-by-the-backup
BACKUP_HOST=sftp
BACKUP_PORT=22
BACKUP_USER=storagebox
BACKUP_PATH=$BOX
BACKUP_AGE_RECIPIENT=$recipient
BACKUP_RETENTION_DAYS=14
BACKUP_KEEP_LOCAL=$2
BACKUP_COMPOSE_PROJECT=$3
BACKUP_PG_SERVICE=${4:-postgres}
BACKUP_SSH_KEY_FILE=/keys/backup
BACKUP_KNOWN_HOSTS_FILE=$1
BACKUP_STAGING_DIR=$WORK/staging
EOF
}
export SALUS_BACKUP_CONFIG=$WORK/bootstrap.conf
run_backup() { "$BACKUP" >"$WORK/$1.log" 2>&1; }

echo "== refuses an unpinned host key"
ssh-keygen -q -t ed25519 -N '' -f "$WORK/other" && printf 'sftp %s\n' "$(cut -d' ' -f1,2 "$WORK/other.pub")" >"$WORK/wrong_known_hosts"
write_conf "$WORK/wrong_known_hosts" no "$PROJECT"
if run_backup wronghost; then fail "uploaded to a server whose host key does not match"; fi
pass "exits non-zero"
expect "leaves no file in staging" bash -c "[[ -z \$(ls -A '$WORK/staging') ]]"

echo "== fails without the database"
write_conf "$WORK/known_hosts" no no-such-project
if run_backup nodb; then fail "succeeded without a postgres container"; fi
expect "says why" grep -q 'no running postgres container' "$WORK/nodb.log"

echo "== fails when pg_dump fails"
write_conf "$WORK/known_hosts" no "$PROJECT" sftp
if run_backup nodump; then fail "succeeded although pg_dump could not run"; fi
expect "says why" grep -q 'the dump failed' "$WORK/nodump.log"
if box 'ls -1 salus' >/dev/null 2>&1; then fail "uploaded something"; fi
pass "uploads nothing"
expect "leaves no file in staging" bash -c "[[ -z \$(ls -A '$WORK/staging') ]]"

echo "== retention: faked old dumps on the box"
printf 'not a dump\n' >"$WORK/filler"
old1=salus-20000101T000000Z.dump.age
old2=salus-$(stamp '-15 days').dump.age
recent=salus-$(stamp '-13 days').dump.age
box "-mkdir salus" "-mkdir $BOX" "put $WORK/filler $BOX/$old1" "put $WORK/filler $BOX/$old2" \
	"put $WORK/filler $BOX/$recent" "put $WORK/filler $BOX/notes.txt" >/dev/null

echo "== a backup"
write_conf "$WORK/known_hosts" no "$PROJECT"
start=$(date +%s)
run_backup run1 || {
	cat "$WORK/run1.log"
	fail "the backup failed"
}
cat "$WORK/run1.log"
listing=$(box_ls)
dump=$(grep -E '^salus-[0-9]{8}T[0-9]{6}Z\.dump\.age$' <<<"$listing" | grep -vxF -e "$recent" -e "$old1" -e "$old2" || true)
expect "uploaded one new dump" test "$(grep -c . <<<"$dump")" -eq 1
expect "deleted the dumps older than 14 days" bash -c "! grep -qxF -e '$old1' -e '$old2' <<<'$listing'"
expect "kept the 13-day-old dump and the unrelated file" bash -c "grep -qxF '$recent' <<<'$listing' && grep -qxF notes.txt <<<'$listing'"
expect "left no partial upload" bash -c "! grep -q '^\.partial' <<<'$listing'"
expect "logs say how many were deleted" grep -q 'deleted 2 dumps older than 14 days' "$WORK/run1.log"
expect "nothing left in staging but the success mark" bash -c "[[ \$(ls -A '$WORK/staging') == last-success ]]"
expect "logs hold no row data" bash -c "! grep -qi 'testperson' '$WORK/run1.log'"

echo "== restore drill"
box "get $BOX/$dump $WORK/fetched.age" >/dev/null
expect "the dump is age-encrypted" bash -c "[[ \$(head -c 21 '$WORK/fetched.age') == age-encryption.org/v1 ]]"
expect "no plaintext in it" bash -c "! grep -aq 'Synthetic Testperson' '$WORK/fetched.age'"
age -d -i "$WORK/age.key" "$WORK/fetched.age" | docker exec -i "$(container restore-db)" pg_restore -U salus -d salus --no-owner --exit-on-error
for q in 'SELECT count(*) FROM patient' 'SELECT max(version_id) FROM goose_db_version WHERE is_applied'; do
	src=$(psql_in postgres "$q")
	dst=$(psql_in restore-db "$q")
	expect "restored: $q = $dst" test "$src" = "$dst"
done
pass "dump to restored in $(($(date +%s) - start)) s"

echo "== a local copy when asked"
sleep 1
write_conf "$WORK/known_hosts" yes "$PROJECT"
run_backup run2 || {
	cat "$WORK/run2.log"
	fail "the backup with a local copy failed"
}
local_copy=$(find "$WORK/staging" -name 'salus-*.dump.age')
expect "keeps one encrypted local copy" bash -c "[[ \$(grep -c . <<<'$local_copy') -eq 1 && \$(head -c 21 '$local_copy') == age-encryption.org/v1 ]]"
expect "both new dumps and the 13-day-old one are on the box" test "$(box_ls | grep -cE '^salus-.*\.dump\.age$' | tr -d ' ')" -eq 3

echo "backup tests passed"
