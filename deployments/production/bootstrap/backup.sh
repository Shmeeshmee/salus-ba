#!/usr/bin/env bash
# The nightly Postgres backup (docs/deploy/backup-restore.md), installed as /usr/local/sbin/salus-backup and run
# as root by salus-backup.timer. pg_dump (custom format) runs in the salus project's postgres container, its output
# is encrypted with age on the way out, so no plaintext dump touches the disk, and the encrypted file is uploaded
# over SFTP to the Storage Box; dumps there older than BACKUP_RETENTION_DAYS are then deleted. Any failure exits
# non-zero (the unit fails and journald keeps the log). Logs carry file names, sizes and timings only.
#
# Settings: /etc/salus/bootstrap.conf (SALUS_BACKUP_CONFIG overrides the path), the BACKUP_* keys.
set -euo pipefail

CONFIG=${SALUS_BACKUP_CONFIG:-/etc/salus/bootstrap.conf}
TMP=

BACKUP_HOST=
BACKUP_PORT=23
BACKUP_USER=
BACKUP_PATH=salus/postgres
BACKUP_AGE_RECIPIENTS=()
BACKUP_RETENTION_DAYS=14
BACKUP_KEEP_LOCAL=no
BACKUP_COMPOSE_PROJECT=salus
BACKUP_PG_SERVICE=postgres
BACKUP_SSH_KEY_FILE=/root/.ssh/salus-backup
BACKUP_KNOWN_HOSTS_FILE=/etc/salus/backup_known_hosts
BACKUP_STAGING_DIR=/opt/salus/backups
BACKUP_WAIT_SECONDS=600

# Dump file names: the UTC time of the dump, so the retention needs no remote timestamps.
readonly NAME_RE='^salus-[0-9]{8}T[0-9]{6}Z\.dump\.age$'

log() { printf 'salus-backup: %s\n' "$*"; }
die() {
	printf 'salus-backup: error: %s\n' "$*" >&2
	exit 1
}

# The same KEY=value format as bootstrap.sh reads; keys this script does not use are ignored. Never sourced.
load_config() {
	local line key value n=0
	[[ -r $CONFIG ]] || die "cannot read $CONFIG"
	while IFS= read -r line || [[ -n $line ]]; do
		n=$((n + 1))
		line=${line%$'\r'}
		[[ $line =~ ^[[:space:]]*(#|$) ]] && continue
		[[ $line =~ ^([A-Z][A-Z0-9_]*)=(.*)$ ]] || die "$CONFIG line $n: expected KEY=value"
		key=${BASH_REMATCH[1]}
		value=${BASH_REMATCH[2]}
		if [[ $value =~ ^\"(.*)\"$ ]] || [[ $value =~ ^\'(.*)\'$ ]]; then
			value=${BASH_REMATCH[1]}
		fi
		case $key in
		BACKUP_AGE_RECIPIENT) [[ -n $value ]] && BACKUP_AGE_RECIPIENTS+=("$value") ;;
		BACKUP_HOST | BACKUP_PORT | BACKUP_USER | BACKUP_PATH | BACKUP_RETENTION_DAYS | BACKUP_KEEP_LOCAL | \
			BACKUP_COMPOSE_PROJECT | BACKUP_PG_SERVICE | BACKUP_SSH_KEY_FILE | BACKUP_KNOWN_HOSTS_FILE | \
			BACKUP_STAGING_DIR | BACKUP_WAIT_SECONDS)
			printf -v "$key" '%s' "$value"
			;;
		esac
	done <"$CONFIG"
	[[ -n $BACKUP_HOST && -n $BACKUP_USER ]] || die "BACKUP_HOST and BACKUP_USER are not set"
	((${#BACKUP_AGE_RECIPIENTS[@]})) || die "no BACKUP_AGE_RECIPIENT is set"
	if ! [[ $BACKUP_RETENTION_DAYS =~ ^[0-9]+$ ]] || ((BACKUP_RETENTION_DAYS < 1)); then
		die "BACKUP_RETENTION_DAYS must be at least 1"
	fi
	[[ $BACKUP_WAIT_SECONDS =~ ^[0-9]+$ ]] || die "BACKUP_WAIT_SECONDS must be a number of seconds"
	[[ -r $BACKUP_SSH_KEY_FILE ]] || die "the SSH key $BACKUP_SSH_KEY_FILE is missing"
	[[ -s $BACKUP_KNOWN_HOSTS_FILE ]] || die "the pinned host key file $BACKUP_KNOWN_HOSTS_FILE is missing"
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

# postgres_container: the id of the project's postgres container once it is healthy. Waits up to
# BACKUP_WAIT_SECONDS: after a boot, Persistent=true can start a missed backup before the stack is up.
postgres_container() {
	local ids state waited=0
	while :; do
		ids=$(docker ps -q --filter "label=com.docker.compose.project=$BACKUP_COMPOSE_PROJECT" \
			--filter "label=com.docker.compose.service=$BACKUP_PG_SERVICE" 2>/dev/null) || ids=""
		if [[ -n $ids ]]; then
			[[ $(wc -l <<<"$ids") -eq 1 ]] || die "more than one $BACKUP_PG_SERVICE container in $BACKUP_COMPOSE_PROJECT"
			state=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$ids" 2>/dev/null) || state=""
			if [[ $state == healthy || $state == running ]]; then
				printf '%s\n' "$ids"
				return 0
			fi
		fi
		((waited < BACKUP_WAIT_SECONDS)) ||
			die "no healthy $BACKUP_PG_SERVICE container in the compose project $BACKUP_COMPOSE_PROJECT after ${BACKUP_WAIT_SECONDS} s"
		sleep 5
		waited=$((waited + 5))
	done
}

# dump CONTAINER FILE: pg_dump as the image's superuser over the local socket (the image trusts local
# connections), encrypted to every recipient. POSTGRES_USER and POSTGRES_DB fall back to the image's defaults.
dump() {
	local cid=$1 out=$2 recipients=() r
	for r in "${BACKUP_AGE_RECIPIENTS[@]}"; do recipients+=(-r "$r"); done
	# pipefail makes a pg_dump failure fail the pipeline; its stderr names objects, not rows.
	# shellcheck disable=SC2016 # expanded inside the container
	docker exec -u postgres -i "$cid" sh -c 'u=${POSTGRES_USER:-postgres}; exec pg_dump -U "$u" -d "${POSTGRES_DB:-$u}" -Fc' |
		age "${recipients[@]}" -o "$out"
}

# upload FILE NAME: put as .partial, then rename, so a broken upload never looks like a dump.
upload() {
	local file=$1 name=$2 dir="" part parts=()
	{
		IFS=/ read -ra parts <<<"$BACKUP_PATH"
		for part in "${parts[@]}"; do
			[[ -n $part ]] || continue
			dir=${dir:+$dir/}$part
			printf -- '-mkdir "%s"\n' "$dir"
		done
		printf -- '-rm "%s/.partial-%s"\n' "$BACKUP_PATH" "$name"
		printf 'put "%s" "%s/.partial-%s"\n' "$file" "$BACKUP_PATH" "$name"
		printf 'rename "%s/.partial-%s" "%s/%s"\n' "$BACKUP_PATH" "$name" "$BACKUP_PATH" "$name"
	} | sftp_batch >/dev/null
}

# prune [KEEP]: deletes on the Storage Box the dumps whose name is older than the retention, and the .partial files
# that earlier failed uploads left, but never the last dump: when nothing newer than the retention is there (the
# backups have failed for that long), the old dumps stay and the log says so. The same for the local copies.
prune() {
	local keep=${1:-} cutoff listing name stale=() partials=() fresh=0 n=0
	cutoff=$(date -u -d "-$BACKUP_RETENTION_DAYS days" +%Y%m%dT%H%M%SZ)
	listing=$(printf 'ls -1a "%s"\n' "$BACKUP_PATH" | sftp_batch) || die "listing the Storage Box failed"
	while IFS= read -r name; do
		name=${name##*/}
		if [[ $name == .partial-salus-* ]]; then
			partials+=("$name")
			continue
		fi
		[[ $name =~ $NAME_RE ]] || continue
		if [[ $name == "$keep" || ! ${name:6:16} < $cutoff ]]; then
			fresh=$((fresh + 1))
		else
			stale+=("$name")
		fi
	done <<<"$listing"
	if ((${#stale[@]} && fresh == 0)); then
		log "error: no dump newer than $BACKUP_RETENTION_DAYS days on the Storage Box; keeping the ${#stale[@]} older ones"
		stale=()
	fi
	if ((${#stale[@]} + ${#partials[@]})); then
		for name in "${stale[@]}" "${partials[@]}"; do printf 'rm "%s/%s"\n' "$BACKUP_PATH" "$name"; done |
			sftp_batch >/dev/null || die "deleting old dumps failed"
		n=${#stale[@]}
	fi
	log "deleted $n dumps older than $BACKUP_RETENTION_DAYS days and ${#partials[@]} partial uploads on the Storage Box"
	if [[ $BACKUP_KEEP_LOCAL == yes ]]; then
		for name in "$BACKUP_STAGING_DIR"/salus-*.dump.age; do
			[[ -f $name ]] || continue
			name=${name##*/}
			[[ $name =~ $NAME_RE && $name != "$keep" && ${name:6:16} < $cutoff ]] && rm -f "$BACKUP_STAGING_DIR/$name"
		done
	fi
	return 0
}

main() {
	local start cid name size
	start=$(date +%s)
	load_config
	umask 077
	# One backup at a time (the timer and a manual start).
	exec 9>"$BACKUP_STAGING_DIR/.lock"
	flock -n 9 || die "another backup is running"
	cid=$(postgres_container)
	name=salus-$(date -u +%Y%m%dT%H%M%SZ).dump.age
	TMP=$BACKUP_STAGING_DIR/.partial-$name
	trap 'rm -f "$TMP"' EXIT
	log "dumping to $name"
	dump "$cid" "$TMP" || die "the dump failed"
	size=$(stat -c %s "$TMP")
	((size > 0)) || die "the dump is empty"
	if ! upload "$TMP" "$name"; then
		# A full Storage Box shows here as a failed put (check with 'df -h' in sftp). Old dumps and partial uploads are
		# still pruned, which frees space without ever deleting the last dump; tonight's dump is lost all the same.
		log "error: the upload to the Storage Box failed; pruning, then failing"
		prune || true
		die "the upload to the Storage Box failed"
	fi
	log "uploaded $name ($size bytes)"
	if [[ $BACKUP_KEEP_LOCAL == yes ]]; then
		mv -f "$TMP" "$BACKUP_STAGING_DIR/$name"
		log "kept a local copy in $BACKUP_STAGING_DIR"
	fi
	prune "$name"
	date -u +%Y-%m-%dT%H:%M:%SZ >"$BACKUP_STAGING_DIR/last-success"
	log "done in $(($(date +%s) - start)) s"
}

main "$@"
