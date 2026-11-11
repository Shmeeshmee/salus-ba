#!/usr/bin/env bash
# The server bootstrap's checks, all in Docker (make deploy-check): shellcheck on every script, bootstrap.sh in a
# fresh ubuntu:24.04 container (--check, a real run, a second run that changes nothing), and the backup against a
# throwaway Postgres and SFTP server. Everything it starts is removed at the end.
#   run.sh [shellcheck|bootstrap|backup]...   (default: all three)
set -euo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
DIR=$(dirname "$HERE")
ID=${SALUS_WT_TAG:-local}
SHELLCHECK=koalaman/shellcheck:v0.11.0@sha256:61862eba1fcf09a484ebcc6feea46f1782532571a34ed51fedf90dd25f925a8d
UBUNTU=ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852
export TEST_PROJECT=salus-$ID-backup-test

shellcheck_() {
	echo "== shellcheck"
	docker run --rm -v "$DIR:/mnt:ro" -w /mnt "$SHELLCHECK" -x bootstrap.sh backup.sh test/run.sh test/bootstrap-test.sh test/backup-test.sh
	echo "shellcheck clean"
}

bootstrap() {
	echo "== bootstrap.sh in ubuntu:24.04"
	docker run --rm --name "salus-$ID-bootstrap-test" -v "$DIR:/bootstrap:ro" "$UBUNTU" bash /bootstrap/test/bootstrap-test.sh
}

backup() {
	echo "== backup.sh against a throwaway Postgres and SFTP server"
	COMPOSE_TEST=(docker compose -f "$HERE/compose.yml")
	trap '"${COMPOSE_TEST[@]}" down -v --remove-orphans >/dev/null 2>&1 || true' EXIT
	"${COMPOSE_TEST[@]}" build -q
	"${COMPOSE_TEST[@]}" run --rm runner
}

(($#)) || set -- shellcheck bootstrap backup
for step in "$@"; do
	case $step in
	shellcheck) shellcheck_ ;;
	bootstrap) bootstrap ;;
	backup) backup ;;
	*)
		echo "unknown step: $step" >&2
		exit 2
		;;
	esac
done
