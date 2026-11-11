#!/usr/bin/env bash
# Runs in the backup test's runner container (compose.yml, run.sh): salus-backup against a throwaway Postgres,
# Garage and SFTP server with a generated age key: the dump and the objects archive (upload, fetch, decrypt, restore,
# retention, failures), then the alerts (curl and systemctl stubbed: no mail leaves) and the freshness check.
set -euo pipefail

BACKUP=/bootstrap/backup.sh
WORK=/tmp/backup-test
BOX=salus/postgres
OBJBOX=salus/objects
PROJECT=${TEST_PROJECT:?}
BUCKET=salus-binaries
RCLONE=rclone/rclone:1.75.2@sha256:2687085f718d3c628f7fdfb77c52a1d344332aed543110bb88088f2c70d43eb5
S3_KEY=GK5a1a5e000000000000000001
S3_SECRET=5a1a5e0000000000000000000000000000000000000000000000000000000001

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
# rcl ARGS...: rclone on the test's network with the objects volume at /objects; remote S is the source Garage,
# R the one restored into.
rcl() {
	local r n args=()
	for r in S:objectstore R:restore-objectstore; do
		n=${r%%:*}
		args+=(-e "RCLONE_CONFIG_${n}_TYPE=s3" -e "RCLONE_CONFIG_${n}_PROVIDER=Other" -e "RCLONE_CONFIG_${n}_REGION=garage"
			-e "RCLONE_CONFIG_${n}_ENDPOINT=http://${r#*:}:3900" -e "RCLONE_CONFIG_${n}_ACCESS_KEY_ID=$S3_KEY"
			-e "RCLONE_CONFIG_${n}_SECRET_ACCESS_KEY=$S3_SECRET")
	done
	docker run --rm --network "${PROJECT}_default" -v "${PROJECT}_objects:/objects" "${args[@]}" "$RCLONE" --config= -q "$@"
}

mkdir -p "$WORK/staging"
rm -f /keys/backup /keys/backup.pub
ssh-keygen -q -t ed25519 -N '' -C synthetic-test-backup -f /keys/backup
age-keygen -o "$WORK/age.key" 2>/dev/null
recipient=$(age-keygen -y "$WORK/age.key")

for _ in $(seq 100); do
	ssh-keyscan -t ed25519 sftp >"$WORK/known_hosts" 2>/dev/null && [[ -s $WORK/known_hosts ]] && break
	sleep 0.2
done
[[ -s $WORK/known_hosts ]] || fail "the SFTP server did not come up"

EXTRA_CONF= # further lines for write_conf
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
BACKUP_WAIT_SECONDS=3
BACKUP_OBJECTS_PATH=$OBJBOX
BACKUP_ENV_FILE=$WORK/salus.env
ALERT_EMAIL=ops@salus.invalid
$EXTRA_CONF
EOF
}
export SALUS_BACKUP_CONFIG=$WORK/bootstrap.conf
run_backup() { # run_backup LOGNAME [ARGS...]
	local name=$1
	shift
	"$BACKUP" "$@" >"$WORK/$name.log" 2>&1
}
# The stack's env file as the deploy writes it (deployments/production/.env.production.example): quoted values, an
# inline comment, a single-quoted value with $, #, " and \ in it, a line that is no KEY=value (compose would refuse
# it; the backup and its alerts skip it), and keys the backup does not read. Synthetic.
write_env() { # write_env [EXTRA-LINE]
	cat >"$WORK/salus.env" <<EOF
POSTGRES_PASSWORD=synthetic-test-only
this line is not a setting
S3_ACCESS_KEY_ID=$S3_KEY
S3_SECRET_ACCESS_KEY="$S3_SECRET"
SMTP_ADDR=smtp-relay.invalid:587 # the relay
SMTP_USERNAME=synthetic-user
SMTP_PASSWORD='synth"pa\\ss\$x #y'
SMTP_FROM="Salus <noreply@salus.invalid>"
${1:-}
EOF
}
write_env

echo "== refuses an unpinned host key"
ssh-keygen -q -t ed25519 -N '' -f "$WORK/other" && printf 'sftp %s\n' "$(cut -d' ' -f1,2 "$WORK/other.pub")" >"$WORK/wrong_known_hosts"
write_conf "$WORK/wrong_known_hosts" no "$PROJECT"
if run_backup wronghost; then fail "uploaded to a server whose host key does not match"; fi
pass "exits non-zero"
expect "leaves no file in staging" bash -c "[[ -z \$(ls -A '$WORK/staging' | grep -vx .lock) ]]"

echo "== fails without the database"
write_conf "$WORK/known_hosts" no no-such-project
if run_backup nodb; then fail "succeeded without a postgres container"; fi
expect "waits, then says why" grep -q 'no healthy postgres container in the compose project no-such-project after 3 s' "$WORK/nodb.log"

echo "== fails when pg_dump fails"
write_conf "$WORK/known_hosts" no "$PROJECT" sftp
if run_backup nodump; then fail "succeeded although pg_dump could not run"; fi
expect "says why" grep -q 'the dump failed' "$WORK/nodump.log"
if box 'ls -1 salus' >/dev/null 2>&1; then fail "uploaded something"; fi
pass "uploads nothing"
expect "leaves no file in staging" bash -c "[[ -z \$(ls -A '$WORK/staging' | grep -vx .lock) ]]"

echo "== retention: faked old dumps on the box"
printf 'not a dump\n' >"$WORK/filler"
old1=salus-20000101T000000Z.dump.age
old2=salus-$(stamp '-15 days').dump.age
recent=salus-$(stamp '-13 days').dump.age
box "-mkdir salus" "-mkdir $BOX" "put $WORK/filler $BOX/$old1" "put $WORK/filler $BOX/$old2" \
	"put $WORK/filler $BOX/$recent" "put $WORK/filler $BOX/notes.txt" \
	"put $WORK/filler $BOX/.partial-$old1" >/dev/null

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
expect "deleted the partial upload a failed night left" bash -c "! grep -qxF '.partial-$old1' <<<'$listing'"
expect "logs say how many were deleted" grep -q 'deleted 2 dumps older than 14 days and 1 partial uploads' "$WORK/run1.log"
expect "nothing left in staging but the success mark (and the lock)" bash -c "[[ \$(ls -A '$WORK/staging' | grep -vx .lock) == last-success ]]"
expect "logs hold no row data" bash -c "! grep -qi 'testperson' '$WORK/run1.log'"

echo "== restore drill"
box "get $BOX/$dump $WORK/fetched.age" >/dev/null
expect "the dump is age-encrypted" bash -c "[[ \$(head -c 21 '$WORK/fetched.age') == age-encryption.org/v1 ]]"
expect "no plaintext in it" bash -c "! grep -aq 'Synthetic Testperson' '$WORK/fetched.age'"
# The drill's steps (docs/deploy/backup-restore.md): the roles the migrations create, then the restore, which stops
# at the first error.
grep -ho 'CREATE ROLE [a-z_]* [A-Z ]*' /migrations/*.sql | sort -u | sed 's/ *$/;/' |
	docker exec -i "$(container restore-db)" psql -U salus -d salus -q -v ON_ERROR_STOP=1
age -d -i "$WORK/age.key" "$WORK/fetched.age" | docker exec -i "$(container restore-db)" pg_restore -U salus -d salus --exit-on-error
pass "restored with --exit-on-error"
for q in 'SELECT count(*) FROM synthetic_backup_rows' \
	'SELECT max(version_id) FROM goose_db_version WHERE is_applied' \
	"SELECT count(*) FROM pg_tables WHERE schemaname = 'public'" \
	'SELECT count(*) FROM pg_policies' \
	"SELECT string_agg(p.proname || ':' || r.rolname, ',' ORDER BY p.proname) FROM pg_proc p JOIN pg_roles r ON r.oid = p.proowner JOIN pg_namespace n ON n.oid = p.pronamespace WHERE n.nspname = 'public'" \
	"SELECT string_agg(grantee || ':' || privilege_type, ',' ORDER BY grantee, table_name, privilege_type) FROM information_schema.role_table_grants WHERE grantee LIKE 'salus_%'"; do
	src=$(psql_in postgres "$q")
	dst=$(psql_in restore-db "$q")
	expect "restored equal: ${q:0:60} (${dst:0:40})" test "$src" = "$dst"
done
expect "the schema is the real one" test "$(psql_in restore-db 'SELECT count(*) FROM pg_policies')" -gt 0
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

echo "== local pruning keeps the last copy"
find "$WORK/staging" -name 'salus-*.dump.age' -delete
stale1=salus-$(stamp '-20 days').dump.age
stale2=salus-$(stamp '-16 days').dump.age
printf 'old\n' >"$WORK/staging/$stale1"
printf 'old\n' >"$WORK/staging/$stale2"
box "-mkdir salus/readonly" "chmod 555 salus/readonly" >/dev/null 2>&1
EXTRA_CONF=BACKUP_PATH=salus/readonly
write_conf "$WORK/known_hosts" yes "$PROJECT"
EXTRA_CONF=
if run_backup fullbox; then fail "succeeded although the upload could not be written"; fi
expect "the upload failed" grep -q 'the upload to the Storage Box failed' "$WORK/fullbox.log"
expect "both stale local dumps are kept: there is no newer one" bash -c "[[ -f '$WORK/staging/$stale1' && -f '$WORK/staging/$stale2' ]]"
expect "and the log says so" grep -q 'no local dump newer than 14 days; keeping the 2 older ones' "$WORK/fullbox.log"
rm -f "$WORK/staging/$stale1" "$WORK/staging/$stale2"

echo "== objects: fails without the objectstore or its key pair"
write_conf "$WORK/known_hosts" no no-such-project
if run_backup noobj objects; then fail "succeeded without an objectstore container"; fi
expect "waits, then says why" grep -q 'no healthy objectstore container in the compose project no-such-project after 3 s' "$WORK/noobj.log"
write_conf "$WORK/known_hosts" no "$PROJECT"
write_env
sed -i '/^S3_SECRET_ACCESS_KEY=/d' "$WORK/salus.env"
if run_backup nokey objects; then fail "succeeded without the S3 key pair"; fi
expect "says the key pair is missing" grep -q 'S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY are not set' "$WORK/nokey.log"
write_env

echo "== objects: a bucket of synthetic objects"
uuid() { cat /proc/sys/kernel/random/uuid; }
rm -rf /objects/seed /objects/restore.tar
mkdir -p /objects/seed/composition
for i in $(seq 1 12); do head -c $((i * 997)) /dev/urandom >"/objects/seed/$(uuid)"; done
printf 'Synthetic Testperson, not envelope-encrypted in this test\n' >"/objects/seed/$(uuid)"
head -c 3000000 /dev/urandom >"/objects/seed/$(uuid)"
head -c 2048 /dev/urandom >"/objects/seed/composition/$(uuid).pdf"
rcl copy /objects/seed "S:$BUCKET"
seeded=$(cd /objects/seed && find . -type f | sed 's#^\./##' | sort)
expect "the source bucket holds the 15 objects" test "$(rcl lsf -R --files-only "S:$BUCKET" | grep -c .)" -eq 15
oldarch=salus-objects-$(stamp '-15 days').tar.age
recentarch=salus-objects-$(stamp '-13 days').tar.age
box "-mkdir salus" "-mkdir $OBJBOX" "put $WORK/filler $OBJBOX/$oldarch" "put $WORK/filler $OBJBOX/$recentarch" \
	"put $WORK/filler $OBJBOX/.partial-$oldarch" >/dev/null 2>&1

echo "== objects: fails when rclone cannot read the bucket"
write_env
sed -i "s/^S3_SECRET_ACCESS_KEY=.*/S3_SECRET_ACCESS_KEY=${S3_SECRET//1/2}/" "$WORK/salus.env"
before=$(box "ls -1a $OBJBOX" | sed -n "s#^$OBJBOX/##p" | sort)
if run_backup badsecret objects; then fail "succeeded with a wrong S3 secret"; fi
expect "says the archive failed" grep -q 'the objects archive or its upload failed' "$WORK/badsecret.log"
expect "with rclone's own error from Garage" grep -q 'AccessDenied' "$WORK/badsecret.log"
expect "nothing new reaches the Storage Box, no partial upload either" test "$(box "ls -1a $OBJBOX" | sed -n "s#^$OBJBOX/##p" | sort)" = "$before"
expect "nothing in staging" bash -c "[[ -z \$(ls -A '$WORK/staging' | grep -vx -e .lock -e .lock-objects -e last-success -e 'salus-.*\.dump\.age') ]]"
write_env

echo "== objects: fails when the upload stage fails (a wrong pinned host key)"
write_conf "$WORK/wrong_known_hosts" no "$PROJECT"
before=$(box "ls -1a $OBJBOX" | sed -n "s#^$OBJBOX/##p" | sort)
if run_backup badhost objects; then fail "uploaded to a server whose host key does not match"; fi
expect "nothing reaches the Storage Box" test "$(box "ls -1a $OBJBOX" | sed -n "s#^$OBJBOX/##p" | sort)" = "$before"
echo "== objects: fails when rclone's upload fails (a directory it cannot write)"
EXTRA_CONF=BACKUP_OBJECTS_PATH=salus/readonly
write_conf "$WORK/known_hosts" no "$PROJECT"
EXTRA_CONF=
if run_backup rofail objects; then fail "succeeded although the upload could not be written"; fi
expect "says the upload stage failed" grep -q 'the objects archive or its upload failed' "$WORK/rofail.log"
expect "no archive and no partial upload in that directory" test -z "$(box "ls -1a salus/readonly" | sed -n 's#^salus/readonly/##p' | grep -v '^\.\{1,2\}$')"
expect "the upload container, which held the key, is gone" test -z "$(docker ps -aq --filter "name=^salus-backup-rcat-$PROJECT\$")"
write_conf "$WORK/known_hosts" no "$PROJECT"

echo "== objects: a backup (streamed, BACKUP_KEEP_LOCAL is for the dumps only)"
EXTRA_CONF=BACKUP_KEEP_LOCAL=yes
write_conf "$WORK/known_hosts" yes "$PROJECT"
EXTRA_CONF=
run_backup objects1 objects || {
	cat "$WORK/objects1.log"
	fail "the objects backup failed"
}
cat "$WORK/objects1.log"
objlisting=$(box "ls -1a $OBJBOX" | sed -n "s#^$OBJBOX/##p" | sort)
arch=$(grep -E '^salus-objects-[0-9]{8}T[0-9]{6}Z\.tar\.age$' <<<"$objlisting" | grep -vxF -e "$recentarch" -e "$oldarch" || true)
expect "uploaded one new archive" test "$(grep -c . <<<"$arch")" -eq 1
expect "deleted the archive older than 14 days and the partial upload" bash -c "! grep -qxF -e '$oldarch' -e '.partial-$oldarch' <<<'$objlisting'"
expect "kept the 13-day-old archive" grep -qxF "$recentarch" <<<"$objlisting"
expect "logs say how many were deleted" grep -q 'deleted 1 archives older than 14 days and 1 partial uploads' "$WORK/objects1.log"
expect "left the dumps alone" test "$(box_ls | grep -cE '^salus-.*\.dump\.age$' | tr -d ' ')" -eq 3
expect "writes its own success mark" test -s "$WORK/staging/last-success-objects"
expect "no partial file left in staging" bash -c "! ls -A '$WORK/staging' | grep -q '^\.partial'"
expect "no local copy of the archive" bash -c "! ls -A '$WORK/staging' | grep -q '^salus-objects-'"
write_conf "$WORK/known_hosts" no "$PROJECT"
expect "logs hold no object data or key" bash -c "! grep -qi -e testperson -e '$S3_SECRET' '$WORK/objects1.log'"

echo "== objects: restore drill"
box "get $OBJBOX/$arch $WORK/objects.age" >/dev/null
expect "the archive is age-encrypted" bash -c "[[ \$(head -c 21 '$WORK/objects.age') == age-encryption.org/v1 ]]"
expect "no plaintext in it" bash -c "! grep -aq 'Synthetic Testperson' '$WORK/objects.age'"
age -d -i "$WORK/age.key" -o /objects/restore.tar "$WORK/objects.age"
expect "the tar holds every object under its key" test "$(tar -tvf /objects/restore.tar | awk '$1 !~ /^d/ {print $NF}' | sort)" = "$seeded"
rcl archive extract /objects/restore.tar "R:$BUCKET"
expect "restored equal: rclone check of the two buckets" rcl check "S:$BUCKET" "R:$BUCKET"
sample=$(grep -v / <<<"$seeded" | head -1)
rcl copyto "R:$BUCKET/$sample" /objects/sample.out
expect "a sample object downloads byte for byte" cmp -s "/objects/seed/$sample" /objects/sample.out
rm -rf /objects/seed /objects/restore.tar /objects/sample.out

echo "== alerts (curl and systemctl stubbed)"
mkdir -p "$WORK/stub"
cat >"$WORK/stub/curl" <<'EOF'
#!/bin/sh
# Records the call: the arguments, the config read from stdin (-K -) and the message.
: >"$STUB/curl.args"
for a in "$@"; do printf '%s\n' "$a" >>"$STUB/curl.args"; done
cat >"$STUB/curl.config"
while [ $# -gt 0 ]; do
	[ "$1" = --upload-file ] && cp "$2" "$STUB/curl.msg"
	shift
done
exit "${CURL_EXIT:-0}"
EOF
cat >"$WORK/stub/systemctl" <<'EOF'
#!/bin/sh
case "$*" in
"show -P Result "*) echo exit-code ;;
"show -P ExecMainStatus "*) echo 1 ;;
*) exit 1 ;;
esac
EOF
chmod +x "$WORK/stub/curl" "$WORK/stub/systemctl"
export STUB=$WORK/stub
stubbed() { PATH=$WORK/stub:$PATH run_backup "$@"; }
reset_stub() { rm -f "$WORK/stub/curl.args" "$WORK/stub/curl.config" "$WORK/stub/curl.msg"; }
after() { grep -A1 -x -- "$1" "$WORK/stub/curl.args" | tail -1; } # after FLAG: the argument after FLAG

reset_stub
stubbed alert1 alert salus-backup-objects.service || {
	cat "$WORK/alert1.log"
	fail "the alert failed"
}
expect "STARTTLS is required when SMTP_TLS is unset" grep -qx -- --ssl-reqd "$WORK/stub/curl.args"
expect "the envelope sender is the address in SMTP_FROM" test "$(after --mail-from)" = noreply@salus.invalid
expect "the recipient is ALERT_EMAIL" test "$(after --mail-rcpt)" = ops@salus.invalid
expect "the relay's URL comes through the config on stdin" grep -qxF 'url = "smtp://smtp-relay.invalid:587"' "$WORK/stub/curl.config"
# shellcheck disable=SC1003,SC2016 # the password's backslash, escaped for curl's config, and a literal $
expect "and its login, single quotes taken literally, quoted for curl" grep -qxF 'user = "synthetic-user:synth\"pa\\ss$x #y"' "$WORK/stub/curl.config"
expect "the login is in no argument" bash -c "! grep -q synth '$WORK/stub/curl.args'"
msg=$(tr -d '\r' <"$WORK/stub/curl.msg")
expect "the From header is SMTP_FROM, quotes removed" grep -qx 'From: Salus <noreply@salus.invalid>' <<<"$msg"
expect "the subject names the host and the unit" grep -qx "Subject: Salus backup failed on $(hostname): salus-backup-objects.service" <<<"$msg"
expect "the body has the result" grep -qx 'Result: exit-code' <<<"$msg"
expect "and the exit status" grep -qx 'Exit status: 1' <<<"$msg"
expect "lines end in CRLF" bash -c "head -1 '$WORK/stub/curl.msg' | grep -q \$'\\r\$'"
expect "nothing but host, unit, time and status" bash -c "! grep -qi -e testperson -e synth -e '$S3_KEY' '$WORK/stub/curl.msg'"

reset_stub
write_env 'SMTP_TLS=tls'
stubbed alert2 alert salus-backup.service || fail "the alert with SMTP_TLS=tls failed"
expect "SMTP_TLS=tls is implicit TLS" grep -qxF 'url = "smtps://smtp-relay.invalid:587"' "$WORK/stub/curl.config"
expect "without STARTTLS" bash -c "! grep -qx -- --ssl-reqd '$WORK/stub/curl.args'"
write_env 'SMTP_TLS=ssl'
if stubbed alert3 alert salus-backup.service; then fail "accepted SMTP_TLS=ssl"; fi
expect "an unknown SMTP_TLS is refused" grep -q 'SMTP_TLS must be starttls, tls or none' "$WORK/alert3.log"
write_env 'SMTP_TLS=none'
if stubbed alert6 alert salus-backup.service; then fail "sent a login without TLS"; fi
expect "SMTP_TLS=none with a login is refused" grep -q 'SMTP_TLS=none sends the login in the clear' "$WORK/alert6.log"
reset_stub
write_env $'SMTP_TLS=none\nSMTP_USERNAME=\nSMTP_PASSWORD='
stubbed alert7 alert salus-backup.service || fail "the alert with SMTP_TLS=none and no login failed"
expect "SMTP_TLS=none without a login is plain SMTP" grep -qxF 'url = "smtp://smtp-relay.invalid:587"' "$WORK/stub/curl.config"
expect "with no STARTTLS and no login" bash -c "! grep -qx -- --ssl-reqd '$WORK/stub/curl.args' && ! grep -q '^user' '$WORK/stub/curl.config'"
write_env
if CURL_EXIT=67 stubbed alert4 alert salus-backup.service; then fail "succeeded although curl failed"; fi
expect "a failed send fails the alert" grep -q 'sending the alert through smtp-relay.invalid:587 failed' "$WORK/alert4.log"
EXTRA_CONF=ALERT_EMAIL=
write_conf "$WORK/known_hosts" no "$PROJECT"
reset_stub
if stubbed alert5 alert salus-backup.service; then fail "succeeded without ALERT_EMAIL"; fi
expect "no ALERT_EMAIL is an error" grep -q 'no ALERT_EMAIL is set' "$WORK/alert5.log"
expect "and nothing is sent" test ! -e "$WORK/stub/curl.args"
EXTRA_CONF=

echo "== the freshness check"
write_conf "$WORK/known_hosts" no "$PROJECT"
reset_stub
stubbed check1 check || {
	cat "$WORK/check1.log"
	fail "the check failed although both backups are fresh"
}
expect "fresh backups: no alert" test ! -e "$WORK/stub/curl.args"
expect "it logs the dump" grep -q 'postgres: the newest backup is from' "$WORK/check1.log"
expect "and the archive" grep -q 'objects: the newest backup is from' "$WORK/check1.log"
box "-mkdir salus/stale" "put $WORK/filler salus/stale/salus-objects-$(stamp '-27 hours').tar.age" >/dev/null
EXTRA_CONF=$'BACKUP_OBJECTS_PATH=salus/stale\nBACKUP_PATH=salus/nothing-here'
write_conf "$WORK/known_hosts" no "$PROJECT"
stubbed check2 check || {
	cat "$WORK/check2.log"
	fail "the check failed although it sent its alert (OnFailure would mail a second time)"
}
expect "it logs the problems" grep -q 'salus-backup: error: objects: the newest backup is from' "$WORK/check2.log"
expect "and sftp's own error for the listing" grep -q 'not found' "$WORK/check2.log"
JOURNAL_STREAM=8:1 stubbed check5 check || fail "the check under journald failed"
expect "under journald the problems are at err priority (<3>)" grep -q '^<3>salus-backup: error: objects: the newest backup is from' "$WORK/check5.log"
expect "a plain run has no priority prefix" bash -c "! grep -q '^<' '$WORK/check2.log'"
msg=$(tr -d '\r' <"$WORK/stub/curl.msg")
expect "one alert names the stale kind" grep -q '^objects: the newest backup is from .*, 27 hours ago (limit 26)$' <<<"$msg"
expect "and the missing one" grep -qx 'postgres: the Storage Box directory salus/nothing-here could not be listed' <<<"$msg"
expect "the subject says a backup is missing" grep -qx "Subject: Salus backup missing on $(hostname)" <<<"$msg"
if CURL_EXIT=67 stubbed check4 check; then fail "the check passed although its alert could not be sent"; fi
expect "an alert it cannot send fails the check (OnFailure tries again)" grep -q 'sending the alert through' "$WORK/check4.log"
EXTRA_CONF=$'BACKUP_MAX_AGE_HOURS=30\nBACKUP_OBJECTS_PATH=salus/stale'
write_conf "$WORK/known_hosts" no "$PROJECT"
reset_stub
stubbed check3 check || fail "the check failed within BACKUP_MAX_AGE_HOURS=30"
expect "BACKUP_MAX_AGE_HOURS sets the limit" test ! -e "$WORK/stub/curl.args"
EXTRA_CONF=

echo "backup tests passed"
