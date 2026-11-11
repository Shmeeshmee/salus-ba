#!/usr/bin/env bash
# Deploys one release of the production stack on the server, as the deploy user (docs/deploy/README.md,
# docs/adr/0011-single-server-deployment.md). The deploy workflow (.github/workflows/deploy.yml) uploads the
# release's deployments/production/ directory and its .env to /opt/salus/releases/<sha>/, then runs this script
# from there:
#
#   deploy.sh start <sha> [--no-migrate]    run `deploy` detached from the SSH session (the workflow's way), its
#                                           output in releases/<sha>/deploy.log, its exit status in deploy.status
#   deploy.sh status <sha>                  `none`, `running`, `died`, or the exit status of the detached deploy
#   deploy.sh deploy <sha> [--no-migrate]   switch to the release; on any failure, back to the one before
#   deploy.sh rollback                      back to the release before the running one (no migrate)
#
# A deploy and a rollback hold /opt/salus/.deploy.lock: a second one at the same time is refused. Detached, a dropped
# SSH connection or a cancelled workflow run cannot stop a deploy half-way: it finishes, or rolls back, on its own.
#
# Layout (deployments/production/README.md): /opt/salus/deployments/ the running release's files, /opt/salus/.env
# its variables (600), /opt/salus/release.env its SALUS_TAG, /opt/salus/release.previous.env the one before.
# Every compose command names the project salus; no other project, image or volume on the server is touched.
#
# Before anything is pulled, migrated or switched (a deploy, and a rollback too), the release's env file and the
# encrypted volume are checked (check_release): the env file must not hold BINARY_KEK or TOTP_KEY, which live only
# in the key files on the volume (ADR 0011), and SALUS_DATA_DIR (default /mnt/salus-data) must be the mount point
# of a dm-crypt (LUKS) device holding both key files. A locked volume after a reboot refuses the deploy; the owner
# unlocks it first (salus-volume unlock, docs/deploy/README.md).
#
# Migrations are forward-only (ADR 0004): the deploy runs migrate through `up` (every app service waits for it);
# a rollback never runs it and brings the previous images up against the schema as it is.
#
# Exit status: 0 deployed; 1 not deployed (refused, or failed and rolled back, or failed with nothing to roll back
# to); 2 failed and the rollback failed too: the stack needs a person.
set -euo pipefail

ROOT=${SALUS_ROOT:-/opt/salus}
# The compose project: always salus on the server; another name only for test/deploy-compose-test.sh.
PROJECT=${SALUS_PROJECT:-salus}
WAIT_TIMEOUT=${SALUS_WAIT_TIMEOUT:-900}
# After `up --wait`, how long the services must stay up: worker, scheduler, notifier and agent have no healthcheck,
# so `--wait` accepts them once they run, and one that exits at start would otherwise pass.
SETTLE_SECONDS=${SALUS_SETTLE_SECONDS:-30}
SELF=$(realpath "${BASH_SOURCE[0]}")

log() { echo "salus-deploy: $*" >&2; }
die() {
	log "$*"
	exit 1
}

compose() {
	docker compose -p "$PROJECT" --env-file "$ROOT/.env" --env-file "$ROOT/release.env" \
		-f "$ROOT/deployments/docker-compose.prod.yml" "$@"
}

tag_of() { sed -n 's/^SALUS_TAG=//p' "$1" 2>/dev/null || true; }

# env_value FILE KEY: the last value of KEY in an env file, without quotes.
env_value() { sed -n "s/^$2=//p" "$1" 2>/dev/null | tail -1 | sed -E "s/^\"(.*)\"\$/\1/; s/^'(.*)'\$/\1/"; }

# The app image's nonroot uid (build/package/Dockerfile): the owner of the key files.
readonly KEY_UID=65532

has_keys() { grep -Eq '^[[:space:]]*(export[[:space:]]+)?(BINARY_KEK|TOTP_KEY)[[:space:]]*=' "$1"; }

# check_release ENV_FILE: returns 1, saying why, when the env file holds a key or the encrypted volume is not
# mounted with both key files. Reads only names, owners and modes, never a key.
check_release() {
	local env=$1 dir src type key file
	if has_keys "$env"; then
		log "the env file sets BINARY_KEK or TOTP_KEY: they live only in the key files on the encrypted volume (ADR 0011); remove them from SALUS_ENV_FILE and from Bitwarden's env note"
		return 1
	fi
	dir=$(env_value "$env" SALUS_DATA_DIR)
	dir=${dir:-/mnt/salus-data}
	src=$(findmnt -n -o SOURCE --mountpoint "$dir" 2>/dev/null) || src=
	if [[ -z $src ]]; then
		log "the encrypted volume is not mounted at $dir: unlock it first (as root: salus-volume unlock, docs/deploy/README.md)"
		return 1
	fi
	type=$(lsblk -ndo TYPE "$src" 2>/dev/null) || type=
	if [[ $type != crypt ]]; then
		log "$dir is mounted from $src, which is not a dm-crypt (LUKS) device: refusing to put data on it"
		return 1
	fi
	for key in binary_kek totp_key; do
		file=$dir/keys/$key
		if [[ ! -f $file ]]; then
			log "the key file $file is missing (salus-volume keys, docs/deploy/README.md)"
			return 1
		fi
		if [[ $(stat -c '%u %a' "$file") != "$KEY_UID 400" ]]; then
			log "the key file $file is not owned by uid $KEY_UID with mode 400 (salus-volume check)"
			return 1
		fi
	done
}

# lock takes /opt/salus/.deploy.lock on fd 9, or trusts the one `start` took and handed down (SALUS_LOCK_HELD).
lock() {
	[[ ${SALUS_LOCK_HELD:-} == 1 ]] && return 0
	exec 9>"$ROOT/.deploy.lock"
	flock -n 9 || die "another deploy or rollback is running"
}

# settled is true when, SETTLE_SECONDS after the wait, every service but the one-shot migrate is running, none is
# unhealthy, and no container this deploy created (since STARTED) has restarted. Containers it left alone (a
# database whose image did not change) keep their old restart counts, which say nothing about this release.
STARTED=0
settled() {
	sleep "$SETTLE_SECONDS"
	local bad name created count health
	bad=$(compose ps --all --format '{{.Service}} {{.State}}' | awk '$1 != "migrate" && $2 != "running"')
	if [[ -n $bad ]]; then
		log "not running after ${SETTLE_SECONDS}s: $(echo "$bad" | tr '\n' ' ')"
		return 1
	fi
	bad=
	# shellcheck disable=SC2046 # one word per container id
	while read -r name created count health; do
		[[ $health == unhealthy ]] && bad+="$name unhealthy; "
		if ((count > 0)) && (($(date -u -d "$created" +%s) >= STARTED)); then
			bad+="$name restarted $count times; "
		fi
	done < <(docker inspect --format '{{.Name}} {{.Created}} {{.RestartCount}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' \
		$(compose ps -q))
	if [[ -n $bad ]]; then
		log "not stable after ${SETTLE_SECONDS}s: $bad"
		return 1
	fi
}

# install puts the uploaded release SHA's files in place as the running release.
install() {
	local sha=$1 rel=$ROOT/releases/$1
	rm -rf "$ROOT/deployments.new"
	mkdir -p "$ROOT/deployments.new"
	cp -a "$rel/." "$ROOT/deployments.new/"
	rm -f "$ROOT/deployments.new/.env" "$ROOT/deployments.new/release.env" "$ROOT/deployments.new/deploy.log" \
		"$ROOT/deployments.new/deploy.status" "$ROOT/deployments.new/deploy.pid"
	# The env file is written mode 600 from the start, never readable by another user in between.
	(umask 077 && cp "$rel/.env" "$ROOT/.env.new")
	mv "$ROOT/.env.new" "$ROOT/.env"
	rm -rf "$ROOT/deployments.old"
	mv "$ROOT/deployments" "$ROOT/deployments.old"
	mv "$ROOT/deployments.new" "$ROOT/deployments"
	rm -rf "$ROOT/deployments.old"
	echo "SALUS_TAG=$sha" >"$ROOT/release.env.new"
	mv "$ROOT/release.env.new" "$ROOT/release.env"
}

# up_without_migrate brings every service but migrate up and waits for their healthchecks.
up_without_migrate() {
	local services
	services=$(compose config --services | grep -vx migrate | tr '\n' ' ')
	# shellcheck disable=SC2086 # one word per service
	compose up -d --wait --wait-timeout "$WAIT_TIMEOUT" --remove-orphans --no-deps $services && settled
}

# roll_back_to brings the release SHA back up without migrate. Exit 0 when it is up.
roll_back_to() {
	local sha=$1
	[[ -f $ROOT/releases/$sha/docker-compose.prod.yml && -f $ROOT/releases/$sha/.env ]] ||
		{ log "the files of release $sha are gone: cannot roll back"; return 1; }
	check_release "$ROOT/releases/$sha/.env" || { log "not rolling back to $sha"; return 1; }
	install "$sha"
	rm -f "$ROOT/release.previous.env"
	STARTED=$(date -u +%s)
	up_without_migrate
}

# prune keeps the running and previous releases' directories and Salus images and removes the rest. Only images of
# this project's repositories (ghcr.io/<owner>/salus*) tagged with a SHA are candidates; `docker image rm` refuses
# an image a container still uses.
prune() {
	local current previous keep dir image
	current=$(tag_of "$ROOT/release.env")
	previous=$(tag_of "$ROOT/release.previous.env")
	for dir in "$ROOT"/releases/*/; do
		dir=${dir%/}
		keep=${dir##*/}
		[[ $keep == "$current" || $keep == "$previous" ]] || rm -rf "$dir"
	done
	while read -r image; do
		[[ $image =~ ^ghcr\.io/[a-z0-9._-]+/salus(-[a-z]+)?:([0-9a-f]{40})$ ]] || continue
		keep=${BASH_REMATCH[2]}
		[[ $keep == "$current" || $keep == "$previous" ]] && continue
		docker image rm "$image" >/dev/null 2>&1 || log "kept $image (in use)"
	done < <(docker image ls --format '{{.Repository}}:{{.Tag}}')
}

cmd_deploy() {
	local sha=${1:-} migrate=1 previous
	[[ ${2:-} == --no-migrate ]] && migrate=0
	[[ $sha =~ ^[0-9a-f]{40}$ ]] || die "not a full commit SHA: '$sha'"
	local rel=$ROOT/releases/$sha
	[[ -f $rel/docker-compose.prod.yml && -f $rel/.env ]] || die "release $sha is not uploaded to $rel"
	chmod 600 "$rel/.env"
	if ! check_release "$rel/.env"; then
		# An env file with the keys does not stay in a release directory either.
		has_keys "$rel/.env" && rm -f "$rel/.env" && log "removed $rel/.env"
		die "not deployed: nothing was pulled, migrated or changed"
	fi
	lock

	# Check and pull with the new files before anything running changes.
	echo "SALUS_TAG=$sha" >"$rel/release.env"
	docker compose -p "$PROJECT" --env-file "$rel/.env" --env-file "$rel/release.env" \
		-f "$rel/docker-compose.prod.yml" config -q || die "the compose file of $sha does not resolve"
	docker compose -p "$PROJECT" --env-file "$rel/.env" --env-file "$rel/release.env" \
		-f "$rel/docker-compose.prod.yml" pull --quiet || die "could not pull the images of $sha"

	previous=$(tag_of "$ROOT/release.env")
	if [[ $previous == "$sha" ]]; then
		previous=$(tag_of "$ROOT/release.previous.env")
	fi
	install "$sha"
	if [[ -n $previous ]]; then
		echo "SALUS_TAG=$previous" >"$ROOT/release.previous.env"
	else
		rm -f "$ROOT/release.previous.env"
	fi

	local up=0
	STARTED=$(date -u +%s)
	if ((migrate)); then
		{ compose up -d --wait --wait-timeout "$WAIT_TIMEOUT" --remove-orphans && settled; } || up=$?
	else
		up_without_migrate || up=$?
	fi
	if ((up == 0)); then
		log "deployed $sha"
		prune
		return 0
	fi

	log "release $sha did not come up healthy"
	compose ps --format '{{.Service}} {{.State}} {{.Health}}' >&2 || true
	if [[ -z $previous ]]; then
		log "no previous release to roll back to: the stack stays as it is"
		exit 1
	fi
	if roll_back_to "$previous"; then
		log "rolled back to $previous"
		exit 1
	fi
	log "the rollback to $previous failed too"
	exit 2
}

cmd_rollback() {
	local current previous
	current=$(tag_of "$ROOT/release.env")
	previous=$(tag_of "$ROOT/release.previous.env")
	[[ -n $previous ]] || die "no previous release to roll back to"
	[[ -f $ROOT/releases/$previous/.env ]] || die "the files of release $previous are gone: cannot roll back"
	check_release "$ROOT/releases/$previous/.env" || die "not rolled back: nothing was changed"
	lock
	if roll_back_to "$previous"; then
		log "rolled back from ${current:-none} to $previous"
		return 0
	fi
	log "the rollback to $previous failed"
	exit 2
}

# cmd_start starts `deploy` in its own session, so it outlives the SSH connection that started it. It takes the
# lock first, so a start refused while another deploy runs leaves that deploy's status and log alone, and hands
# the lock down to the detached deploy, which drops it before it writes its status: a rollback the workflow starts
# on reading the status is never refused by the deploy it follows.
cmd_start() {
	local sha=${1:-} rel
	[[ $sha =~ ^[0-9a-f]{40}$ ]] || die "not a full commit SHA: '$sha'"
	rel=$ROOT/releases/$sha
	[[ -d $rel ]] || die "release $sha is not uploaded to $rel"
	lock
	rm -f "$rel/deploy.status" "$rel/deploy.pid"
	: >"$rel/deploy.log"
	# shellcheck disable=SC2016 # expanded by the inner shell
	SALUS_LOCK_HELD=1 setsid nohup bash -c \
		'echo $$ >"$2/deploy.pid"; bash "$1" deploy "${@:3}" >"$2/deploy.log" 2>&1; s=$?; exec 9>&-; echo $s >"$2/deploy.status"' \
		_ "$SELF" "$rel" "$@" </dev/null >/dev/null 2>&1 &
	log "started the deploy of $sha (log: $rel/deploy.log)"
}

cmd_status() {
	local sha=${1:-} rel
	[[ $sha =~ ^[0-9a-f]{40}$ ]] || die "not a full commit SHA: '$sha'"
	rel=$ROOT/releases/$sha
	if [[ -s $rel/deploy.status ]]; then
		cat "$rel/deploy.status"
	elif [[ -s $rel/deploy.pid ]] && ! kill -0 "$(cat "$rel/deploy.pid")" 2>/dev/null; then
		# Gone without a result (killed, or the server rebooted): the files may be switched with nothing verified.
		echo died
	elif [[ -f $rel/deploy.log ]]; then
		echo running
	else
		echo none
	fi
}

case ${1:-} in
start) shift && cmd_start "$@" ;;
status) shift && cmd_status "$@" ;;
deploy) shift && cmd_deploy "$@" ;;
rollback) cmd_rollback ;;
*) die "usage: deploy.sh start|deploy <sha> [--no-migrate] | deploy.sh status <sha> | deploy.sh rollback" ;;
esac
