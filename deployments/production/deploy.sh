#!/usr/bin/env bash
# Deploys one release of the production stack on the server, as the deploy user (docs/deploy/README.md,
# docs/adr/0011-single-server-deployment.md). The deploy workflow (.github/workflows/deploy.yml) uploads the
# release's deployments/production/ directory and its .env to /opt/salus/releases/<sha>/, then runs this script
# from there:
#
#   deploy.sh deploy <sha> [--no-migrate]   switch to the release; on any failure, back to the one before
#   deploy.sh rollback                      back to the release before the running one (no migrate)
#
# Layout (deployments/production/README.md): /opt/salus/deployments/ the running release's files, /opt/salus/.env
# its variables (600), /opt/salus/release.env its SALUS_TAG, /opt/salus/release.previous.env the one before.
# Every compose command names the project salus; no other project, image or volume on the server is touched.
#
# Migrations are forward-only (ADR 0004): the deploy runs migrate through `up` (every app service waits for it);
# a rollback never runs it and brings the previous images up against the schema as it is.
#
# Exit status: 0 deployed; 1 not deployed (refused, or failed and rolled back, or failed with nothing to roll back
# to); 2 failed and the rollback failed too: the stack needs a person.
set -euo pipefail

ROOT=${SALUS_ROOT:-/opt/salus}
WAIT_TIMEOUT=${SALUS_WAIT_TIMEOUT:-900}

log() { echo "salus-deploy: $*" >&2; }
die() {
	log "$*"
	exit 1
}

compose() {
	docker compose -p salus --env-file "$ROOT/.env" --env-file "$ROOT/release.env" \
		-f "$ROOT/deployments/docker-compose.prod.yml" "$@"
}

tag_of() { sed -n 's/^SALUS_TAG=//p' "$1" 2>/dev/null || true; }

# install puts the uploaded release SHA's files in place as the running release.
install() {
	local sha=$1 rel=$ROOT/releases/$1
	rm -rf "$ROOT/deployments.new"
	mkdir -p "$ROOT/deployments.new"
	cp -a "$rel/." "$ROOT/deployments.new/"
	rm -f "$ROOT/deployments.new/.env" "$ROOT/deployments.new/release.env"
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
	compose up -d --wait --wait-timeout "$WAIT_TIMEOUT" --remove-orphans --no-deps $services
}

# roll_back_to brings the release SHA back up without migrate. Exit 0 when it is up.
roll_back_to() {
	local sha=$1
	[[ -f $ROOT/releases/$sha/docker-compose.prod.yml && -f $ROOT/releases/$sha/.env ]] ||
		{ log "the files of release $sha are gone: cannot roll back"; return 1; }
	install "$sha"
	rm -f "$ROOT/release.previous.env"
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

	# Check and pull with the new files before anything running changes.
	echo "SALUS_TAG=$sha" >"$rel/release.env"
	docker compose -p salus --env-file "$rel/.env" --env-file "$rel/release.env" \
		-f "$rel/docker-compose.prod.yml" config -q || die "the compose file of $sha does not resolve"
	docker compose -p salus --env-file "$rel/.env" --env-file "$rel/release.env" \
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
	if ((migrate)); then
		compose up -d --wait --wait-timeout "$WAIT_TIMEOUT" --remove-orphans || up=$?
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
	if roll_back_to "$previous"; then
		log "rolled back from ${current:-none} to $previous"
		return 0
	fi
	log "the rollback to $previous failed"
	exit 2
}

case ${1:-} in
deploy) shift && cmd_deploy "$@" ;;
rollback) cmd_rollback ;;
*) die "usage: deploy.sh deploy <sha> [--no-migrate] | deploy.sh rollback" ;;
esac
