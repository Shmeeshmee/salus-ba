#!/usr/bin/env bash
# The nightly backups and their alerts (docs/deploy/backup-restore.md), installed as /usr/local/sbin/salus-backup
# and run as root by the bootstrap's systemd units:
#
#   salus-backup [postgres]   pg_dump (custom format) in the salus project's postgres container
#                             (salus-backup.service)
#   salus-backup objects      every object of Garage's bucket as one tar, read through Garage's S3 API by a pinned
#                             rclone container on the compose network (salus-backup-objects.service, right after
#                             the dump)
#   salus-backup check        sends an alert and fails when the newest backup of either kind on the Storage Box is
#                             older than BACKUP_MAX_AGE_HOURS, or missing (salus-backup-check.timer)
#   salus-backup alert UNIT   sends the alert for a failed UNIT (salus-backup-failed@.service, from OnFailure=)
#
# A backup is encrypted with age on the way out, so no plaintext touches the disk, and the encrypted file is
# uploaded over SFTP to the Storage Box; files of that kind there older than BACKUP_RETENTION_DAYS are then deleted.
# Any failure exits non-zero (the unit fails and journald keeps the log). Logs and alerts carry the host, unit and
# file names, sizes, times and exit statuses only. An alert is an e-mail to ALERT_EMAIL through the SMTP relay of
# /opt/salus/.env (SMTP_ADDR, SMTP_USERNAME, SMTP_PASSWORD, SMTP_TLS, SMTP_FROM), sent with curl.
#
# Settings: /etc/salus/bootstrap.conf (SALUS_BACKUP_CONFIG overrides the path), the BACKUP_* keys and ALERT_EMAIL.
set -euo pipefail

CONFIG=${SALUS_BACKUP_CONFIG:-/etc/salus/bootstrap.conf}
TMP=

BACKUP_HOST=
BACKUP_PORT=23
BACKUP_USER=
BACKUP_PATH=salus/postgres
BACKUP_OBJECTS_PATH=salus/objects
BACKUP_AGE_RECIPIENTS=()
BACKUP_RETENTION_DAYS=14
BACKUP_MAX_AGE_HOURS=26
BACKUP_KEEP_LOCAL=no
BACKUP_COMPOSE_PROJECT=salus
BACKUP_PG_SERVICE=postgres
BACKUP_OBJECTS_SERVICE=objectstore
BACKUP_ENV_FILE=/opt/salus/.env
BACKUP_SSH_KEY_FILE=/root/.ssh/salus-backup
BACKUP_KNOWN_HOSTS_FILE=/etc/salus/backup_known_hosts
BACKUP_STAGING_DIR=/opt/salus/backups
BACKUP_WAIT_SECONDS=600
ALERT_EMAIL=

# The objects backup's S3 client, pinned by digest. Dependabot does not see it: move it by hand.
readonly RCLONE_IMAGE=rclone/rclone:1.75.2@sha256:2687085f718d3c628f7fdfb77c52a1d344332aed543110bb88088f2c70d43eb5

# File names carry the UTC time of the backup, so the retention and the check need no remote timestamps.
readonly PG_NAME_RE='^salus-([0-9]{8}T[0-9]{6}Z)\.dump\.age$'
readonly OBJECTS_NAME_RE='^salus-objects-([0-9]{8}T[0-9]{6}Z)\.tar\.age$'

log() { printf 'salus-backup: %s\n' "$*"; }
die() {
	printf 'salus-backup: error: %s\n' "$*" >&2
	exit 1
}

# read_kv FILE CALLBACK: calls CALLBACK KEY VALUE for each KEY=value line; '#' comment and blank lines are skipped,
# one pair of quotes around a value is removed. The file is parsed, never sourced.
read_kv() {
	local file=$1 callback=$2 line n=0 key value
	while IFS= read -r line || [[ -n $line ]]; do
		n=$((n + 1))
		line=${line%$'\r'}
		[[ $line =~ ^[[:space:]]*(#|$) ]] && continue
		[[ $line =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]] || die "$file line $n: expected KEY=value"
		key=${BASH_REMATCH[1]}
		value=${BASH_REMATCH[2]}
		if [[ $value =~ ^\"(.*)\"$ ]] || [[ $value =~ ^\'(.*)\'$ ]]; then
			value=${BASH_REMATCH[1]}
		fi
		"$callback" "$key" "$value"
	done <"$file"
}

# The same KEY=value format as bootstrap.sh reads; keys this script does not use are ignored.
config_key() {
	case $1 in
	BACKUP_AGE_RECIPIENT) [[ -n $2 ]] && BACKUP_AGE_RECIPIENTS+=("$2") ;;
	BACKUP_HOST | BACKUP_PORT | BACKUP_USER | BACKUP_PATH | BACKUP_OBJECTS_PATH | BACKUP_RETENTION_DAYS | \
		BACKUP_MAX_AGE_HOURS | BACKUP_KEEP_LOCAL | BACKUP_COMPOSE_PROJECT | BACKUP_PG_SERVICE | BACKUP_OBJECTS_SERVICE | \
		BACKUP_ENV_FILE | BACKUP_SSH_KEY_FILE | BACKUP_KNOWN_HOSTS_FILE | BACKUP_STAGING_DIR | BACKUP_WAIT_SECONDS | \
		ALERT_EMAIL)
		printf -v "$1" '%s' "$2"
		;;
	esac
	return 0
}

# load_config [ALERT-ONLY]: for an alert alone, the settings of the backup itself may be missing.
load_config() {
	[[ -r $CONFIG ]] || die "cannot read $CONFIG"
	read_kv "$CONFIG" config_key
	[[ -z ${1:-} ]] || return 0
	[[ -n $BACKUP_HOST && -n $BACKUP_USER ]] || die "BACKUP_HOST and BACKUP_USER are not set"
	((${#BACKUP_AGE_RECIPIENTS[@]})) || die "no BACKUP_AGE_RECIPIENT is set"
	if ! [[ $BACKUP_RETENTION_DAYS =~ ^[0-9]+$ ]] || ((BACKUP_RETENTION_DAYS < 1)); then
		die "BACKUP_RETENTION_DAYS must be at least 1"
	fi
	if ! [[ $BACKUP_MAX_AGE_HOURS =~ ^[0-9]+$ ]] || ((BACKUP_MAX_AGE_HOURS < 1)); then
		die "BACKUP_MAX_AGE_HOURS must be at least 1"
	fi
	[[ $BACKUP_WAIT_SECONDS =~ ^[0-9]+$ ]] || die "BACKUP_WAIT_SECONDS must be a number of seconds"
	[[ -r $BACKUP_SSH_KEY_FILE ]] || die "the SSH key $BACKUP_SSH_KEY_FILE is missing"
	[[ -s $BACKUP_KNOWN_HOSTS_FILE ]] || die "the pinned host key file $BACKUP_KNOWN_HOSTS_FILE is missing"
}

# What the objects backup and the alerts read from the stack's env file; nothing else of it is kept.
S3_ACCESS_KEY_ID=
S3_SECRET_ACCESS_KEY=
S3_BUCKET=salus-binaries
SMTP_ADDR=
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_TLS=starttls
SMTP_FROM=
env_key() {
	case $1 in
	S3_ACCESS_KEY_ID | S3_SECRET_ACCESS_KEY | S3_BUCKET | SMTP_ADDR | SMTP_USERNAME | SMTP_PASSWORD | SMTP_TLS | SMTP_FROM)
		[[ -z $2 ]] || printf -v "$1" '%s' "$2"
		;;
	esac
	return 0
}

load_env() {
	[[ -r $BACKUP_ENV_FILE ]] || die "cannot read $BACKUP_ENV_FILE"
	read_kv "$BACKUP_ENV_FILE" env_key
}

# sftp_batch: runs the sftp commands on stdin against the Storage Box; the host key must match the pinned one.
# Its messages (paths and errors, no data) are printed only when the batch fails: '-' commands that may fail
# (mkdir of an existing directory) are otherwise noisy.
sftp_batch() {
	local out
	if out=$(sftp_raw 2>&1); then
		printf '%s\n' "$out"
	else
		printf '%s\n' "$out" >&2
		return 1
	fi
}

sftp_raw() {
	sftp -q -b - -P "$BACKUP_PORT" -i "$BACKUP_SSH_KEY_FILE" \
		-o BatchMode=yes -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes \
		-o UserKnownHostsFile="$BACKUP_KNOWN_HOSTS_FILE" -o ConnectTimeout=30 -o ServerAliveInterval=30 \
		"$BACKUP_USER@$BACKUP_HOST"
}

# listing DIR: the names of the files in DIR on the Storage Box.
listing() {
	local out name
	out=$(printf 'ls -1a "%s"\n' "$1" | sftp_batch) || return 1
	while IFS= read -r name; do printf '%s\n' "${name##*/}"; done <<<"$out"
}

# container SERVICE: the id of the project's SERVICE container once it is healthy. Waits up to
# BACKUP_WAIT_SECONDS: after a boot, Persistent=true can start a missed backup before the stack is up.
container() {
	local service=$1 ids state waited=0
	while :; do
		ids=$(docker ps -q --filter "label=com.docker.compose.project=$BACKUP_COMPOSE_PROJECT" \
			--filter "label=com.docker.compose.service=$service" 2>/dev/null) || ids=""
		if [[ -n $ids ]]; then
			[[ $(wc -l <<<"$ids") -eq 1 ]] || die "more than one $service container in $BACKUP_COMPOSE_PROJECT"
			state=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$ids" 2>/dev/null) || state=""
			if [[ $state == healthy || $state == running ]]; then
				printf '%s\n' "$ids"
				return 0
			fi
		fi
		((waited < BACKUP_WAIT_SECONDS)) ||
			die "no healthy $service container in the compose project $BACKUP_COMPOSE_PROJECT after ${BACKUP_WAIT_SECONDS} s"
		sleep 5
		waited=$((waited + 5))
	done
}

# encrypt FILE: stdin encrypted to every recipient.
encrypt() {
	local r recipients=()
	for r in "${BACKUP_AGE_RECIPIENTS[@]}"; do recipients+=(-r "$r"); done
	age "${recipients[@]}" -o "$1"
}

# dump_postgres CONTAINER FILE: pg_dump as the image's superuser over the local socket (the image trusts local
# connections). POSTGRES_USER and POSTGRES_DB fall back to the image's defaults.
dump_postgres() {
	# pipefail makes a pg_dump failure fail the pipeline; its stderr names objects, not rows.
	# shellcheck disable=SC2016 # expanded inside the container
	docker exec -u postgres -i "$1" sh -c 'u=${POSTGRES_USER:-postgres}; exec pg_dump -U "$u" -d "${POSTGRES_DB:-$u}" -Fc' |
		encrypt "$2"
}

# dump_objects CONTAINER FILE: the bucket as one tar (rclone's archive, streamed to stdout), every object under its
# key. rclone runs on the objectstore container's network with the stack's own key pair, passed by name so it is in
# no command line. The objects are envelope-encrypted with BINARY_KEK already, which the backup never reads.
dump_objects() {
	local network
	network=$(docker inspect -f '{{range $k, $v := .NetworkSettings.Networks}}{{$k}}{{"\n"}}{{end}}' "$1" | head -1)
	[[ -n $network ]] || return 1
	RCLONE_CONFIG_SRC_ACCESS_KEY_ID=$S3_ACCESS_KEY_ID RCLONE_CONFIG_SRC_SECRET_ACCESS_KEY=$S3_SECRET_ACCESS_KEY \
		docker run --rm --network "$network" --read-only --tmpfs /tmp --cap-drop ALL \
		--security-opt no-new-privileges:true --user 65534:65534 -e HOME=/tmp \
		-e RCLONE_CONFIG_SRC_TYPE=s3 -e RCLONE_CONFIG_SRC_PROVIDER=Other -e RCLONE_CONFIG_SRC_REGION=garage \
		-e RCLONE_CONFIG_SRC_ENDPOINT="http://$BACKUP_OBJECTS_SERVICE:3900" \
		-e RCLONE_CONFIG_SRC_ACCESS_KEY_ID -e RCLONE_CONFIG_SRC_SECRET_ACCESS_KEY \
		"$RCLONE_IMAGE" --config= archive create --format tar "src:$S3_BUCKET" |
		encrypt "$2"
}

# upload FILE DIR NAME: put as .partial, then rename, so a broken upload never looks like a backup.
upload() {
	local file=$1 path=$2 name=$3 dir="" part parts=()
	{
		IFS=/ read -ra parts <<<"$path"
		for part in "${parts[@]}"; do
			[[ -n $part ]] || continue
			dir=${dir:+$dir/}$part
			printf -- '-mkdir "%s"\n' "$dir"
		done
		printf -- '-rm "%s/.partial-%s"\n' "$path" "$name"
		printf 'put "%s" "%s/.partial-%s"\n' "$file" "$path" "$name"
		printf 'rename "%s/.partial-%s" "%s/%s"\n' "$path" "$name" "$path" "$name"
	} | sftp_batch >/dev/null
}

# prune DIR RE NOUN [KEEP]: deletes on the Storage Box the files matching RE whose name is older than the retention,
# and the .partial files of that kind that earlier failed uploads left, but never the last one: when nothing newer
# than the retention is there (the backups have failed for that long), the old ones stay and the log says so. The
# same for the local copies.
prune() {
	local path=$1 re=$2 noun=$3 keep=${4:-} cutoff names name stale=() partials=() fresh=0 n=0
	cutoff=$(date -u -d "-$BACKUP_RETENTION_DAYS days" +%Y%m%dT%H%M%SZ)
	names=$(listing "$path") || die "listing the Storage Box failed"
	while IFS= read -r name; do
		if [[ $name == .partial-* && ${name#.partial-} =~ $re ]]; then
			partials+=("$name")
			continue
		fi
		[[ $name =~ $re ]] || continue
		if [[ $name == "$keep" || ! ${BASH_REMATCH[1]} < $cutoff ]]; then
			fresh=$((fresh + 1))
		else
			stale+=("$name")
		fi
	done <<<"$names"
	if ((${#stale[@]} && fresh == 0)); then
		log "error: no $noun newer than $BACKUP_RETENTION_DAYS days on the Storage Box; keeping the ${#stale[@]} older ones"
		stale=()
	fi
	if ((${#stale[@]} + ${#partials[@]})); then
		for name in "${stale[@]}" "${partials[@]}"; do printf 'rm "%s/%s"\n' "$path" "$name"; done |
			sftp_batch >/dev/null || die "deleting old ${noun}s failed"
		n=${#stale[@]}
	fi
	log "deleted $n ${noun}s older than $BACKUP_RETENTION_DAYS days and ${#partials[@]} partial uploads on the Storage Box"
	if [[ $BACKUP_KEEP_LOCAL == yes ]]; then
		for name in "$BACKUP_STAGING_DIR"/salus-*.age; do
			[[ -f $name ]] || continue
			name=${name##*/}
			[[ $name =~ $re && $name != "$keep" && ${BASH_REMATCH[1]} < $cutoff ]] && rm -f "$BACKUP_STAGING_DIR/$name"
		done
	fi
	return 0
}

# backup KIND: the postgres dump or the objects archive, encrypted, uploaded and pruned; then the time of the
# success in BACKUP_STAGING_DIR/last-success (postgres) or last-success-objects.
backup() {
	local kind=$1 start name size path re noun service lock mark cid
	start=$(date +%s)
	load_config
	umask 077
	if [[ $kind == postgres ]]; then
		name=salus-$(date -u +%Y%m%dT%H%M%SZ).dump.age path=$BACKUP_PATH re=$PG_NAME_RE noun=dump
		service=$BACKUP_PG_SERVICE lock=.lock mark=last-success
	else
		load_env
		[[ -n $S3_ACCESS_KEY_ID && -n $S3_SECRET_ACCESS_KEY ]] ||
			die "S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY are not set in $BACKUP_ENV_FILE"
		name=salus-objects-$(date -u +%Y%m%dT%H%M%SZ).tar.age path=$BACKUP_OBJECTS_PATH re=$OBJECTS_NAME_RE
		noun=archive service=$BACKUP_OBJECTS_SERVICE lock=.lock-objects mark=last-success-objects
	fi
	# One backup of a kind at a time (the timer and a manual start).
	exec 9>"$BACKUP_STAGING_DIR/$lock"
	flock -n 9 || die "another $kind backup is running"
	cid=$(container "$service")
	TMP=$BACKUP_STAGING_DIR/.partial-$name
	trap 'rm -f "$TMP"' EXIT
	log "backing up $kind to $name"
	"dump_$kind" "$cid" "$TMP" || die "the $noun failed"
	size=$(stat -c %s "$TMP")
	((size > 0)) || die "the $noun is empty"
	if ! upload "$TMP" "$path" "$name"; then
		# A full Storage Box shows here as a failed put (check with 'df -h' in sftp). Old files and partial uploads
		# are still pruned, which frees space without ever deleting the last one; tonight's is lost all the same.
		log "error: the upload to the Storage Box failed; pruning, then failing"
		prune "$path" "$re" "$noun" || true
		die "the upload to the Storage Box failed"
	fi
	log "uploaded $name ($size bytes)"
	if [[ $BACKUP_KEEP_LOCAL == yes ]]; then
		mv -f "$TMP" "$BACKUP_STAGING_DIR/$name"
		log "kept a local copy in $BACKUP_STAGING_DIR"
	fi
	prune "$path" "$re" "$noun" "$name"
	date -u +%Y-%m-%dT%H:%M:%SZ >"$BACKUP_STAGING_DIR/$mark"
	log "done in $(($(date +%s) - start)) s"
}

# --- alerts --------------------------------------------------------------------------------------------------

# no_newline VALUE...: false when a value would break out of its header line.
no_newline() {
	local v
	for v in "$@"; do [[ $v != *$'\r'* && $v != *$'\n'* ]] || return 1; done
}

# curl_quote VALUE: VALUE as a double-quoted string of curl's config file.
curl_quote() {
	local v=${1//\\/\\\\}
	printf '"%s"' "${v//\"/\\\"}"
}

# send_alert SUBJECT BODY: one plain-text e-mail to ALERT_EMAIL through the relay of BACKUP_ENV_FILE. The relay's
# login reaches curl through its config on stdin, never its command line.
send_alert() {
	local subject=$1 body=$2 host url ssl=() msg sender angle='<([^<>]+)>[[:space:]]*$'
	[[ -n $ALERT_EMAIL ]] || die "no ALERT_EMAIL is set in $CONFIG: the alert was not sent"
	load_env
	[[ -n $SMTP_ADDR && -n $SMTP_FROM ]] || die "SMTP_ADDR and SMTP_FROM are not set in $BACKUP_ENV_FILE: the alert was not sent"
	no_newline "$ALERT_EMAIL" "$SMTP_FROM" "$SMTP_ADDR" "$SMTP_USERNAME" "$SMTP_PASSWORD" ||
		die "ALERT_EMAIL or an SMTP_* setting holds a line break: the alert was not sent"
	case $SMTP_TLS in
	tls) url=smtps://$SMTP_ADDR ;;
	starttls) url=smtp://$SMTP_ADDR ssl=(--ssl-reqd) ;;
	none) url=smtp://$SMTP_ADDR ;;
	*) die "SMTP_TLS must be starttls, tls or none: the alert was not sent" ;;
	esac
	# SMTP_FROM may be a display name with the address in angle brackets; the envelope takes the address alone.
	sender=$SMTP_FROM
	[[ $sender =~ $angle ]] && sender=${BASH_REMATCH[1]}
	host=$(hostname)
	msg=$(mktemp)
	# shellcheck disable=SC2064 # the file's name now
	trap "rm -f '$msg'" EXIT
	printf '%s\r\n' "From: $SMTP_FROM" "To: $ALERT_EMAIL" "Subject: $subject" "Date: $(LC_ALL=C date -R)" \
		"Message-ID: <salus-backup.$(date +%s).$$@$host>" "MIME-Version: 1.0" "Content-Type: text/plain; charset=utf-8" \
		"Auto-Submitted: auto-generated" "" >"$msg"
	printf '%s\r\n' "${body//$'\n'/$'\r\n'}" >>"$msg"
	{
		printf 'url = %s\n' "$(curl_quote "$url")"
		[[ -z $SMTP_USERNAME ]] || printf 'user = %s\n' "$(curl_quote "$SMTP_USERNAME:$SMTP_PASSWORD")"
	} | curl -sS --max-time 60 --retry 3 "${ssl[@]}" -K - --mail-from "$sender" --mail-rcpt "$ALERT_EMAIL" \
		--upload-file "$msg" || die "sending the alert through $SMTP_ADDR failed"
	log "alert sent: $subject"
}

# alert UNIT: the failed unit's result and exit status, as systemd recorded them.
alert() {
	local unit=$1 host result status
	[[ $unit =~ ^[A-Za-z0-9@._-]+$ ]] || die "usage: salus-backup alert UNIT"
	load_config alert-only
	host=$(hostname)
	result=$(systemctl show -P Result "$unit" 2>/dev/null) || result=
	status=$(systemctl show -P ExecMainStatus "$unit" 2>/dev/null) || status=
	log "error: $unit failed (result ${result:-unknown}, exit status ${status:-unknown})"
	send_alert "Salus backup failed on $host: $unit" "The unit $unit failed on $host.

Time: $(date -u +%Y-%m-%dT%H:%M:%SZ)
Result: ${result:-unknown}
Exit status: ${status:-unknown}

On the server: journalctl -u $unit -n 50
See docs/deploy/backup-restore.md, \"Alerting\"."
}

# check: the newest file of each kind on the Storage Box, by the time in its name, must be younger than
# BACKUP_MAX_AGE_HOURS. Sends one alert naming every kind that is not, and fails.
check() {
	local now names newest stamp hours problems=() kind path re name host
	load_config
	now=$(date -u +%s)
	for kind in postgres objects; do
		if [[ $kind == postgres ]]; then path=$BACKUP_PATH re=$PG_NAME_RE; else path=$BACKUP_OBJECTS_PATH re=$OBJECTS_NAME_RE; fi
		if ! names=$(listing "$path" 2>/dev/null); then
			problems+=("$kind: the Storage Box directory $path could not be listed")
			continue
		fi
		newest=
		while IFS= read -r name; do
			[[ $name =~ $re && ${BASH_REMATCH[1]} > $newest ]] && newest=${BASH_REMATCH[1]}
		done <<<"$names"
		if [[ -z $newest ]]; then
			problems+=("$kind: no backup in $path")
			continue
		fi
		stamp=$(date -u -d "${newest:0:4}-${newest:4:2}-${newest:6:2}T${newest:9:2}:${newest:11:2}:${newest:13:2}Z" +%s)
		hours=$(((now - stamp) / 3600))
		if ((now - stamp > BACKUP_MAX_AGE_HOURS * 3600)); then
			problems+=("$kind: the newest backup is from $newest, $hours hours ago (limit $BACKUP_MAX_AGE_HOURS)")
		else
			log "$kind: the newest backup is from $newest, $hours hours ago"
		fi
	done
	((${#problems[@]})) || return 0
	printf 'salus-backup: error: %s\n' "${problems[@]}" >&2
	host=$(hostname)
	send_alert "Salus backup missing on $host" "A backup on $host is missing or too old.

Time: $(date -u +%Y-%m-%dT%H:%M:%SZ)
$(printf '%s\n' "${problems[@]}")

On the server: systemctl list-timers 'salus-backup*'
and journalctl -u salus-backup.service -u salus-backup-objects.service -n 50
See docs/deploy/backup-restore.md, \"Alerting\"."
	exit 1
}

case ${1:-postgres} in
postgres | objects) backup "${1:-postgres}" ;;
check) check ;;
alert) alert "${2:-}" ;;
*) die "usage: salus-backup [postgres|objects|check|alert UNIT]" ;;
esac
