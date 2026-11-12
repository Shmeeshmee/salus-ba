#!/usr/bin/env bash
# deploy.sh against the real Docker Compose (what the fake docker of deploy-test.sh cannot show): a one-shot migrate
# that `up --wait` accepts once it exited 0, the rollback with `--no-deps` after a failed migrate, and a service that
# crashes after the wait. A tiny stack of busybox images retagged per release, in its own compose project
# (salus-deploy-test-<id>, never salus) and a temp root; everything is removed at the end. Run by make deploy-check.
set -euo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
DEPLOY=$HERE/../deploy.sh
BUSYBOX=busybox:1.37.0@sha256:a7dbc319b1438115a207e11930dc4bb3e643dcb3ec0bdd694bb36cd8fad289ec
OWNER=salus-deploy-test
export SALUS_PROJECT=salus-deploy-test-${SALUS_WT_TAG:-local}
export SALUS_SETTLE_SECONDS=4 SALUS_WAIT_TIMEOUT=60
WORK=$(mktemp -d)
export SALUS_ROOT=$WORK/opt/salus
FAILS=0

A=a000000000000000000000000000000000000000
B=b000000000000000000000000000000000000000
C=c000000000000000000000000000000000000000

cleanup() {
	SALUS_TAG=$A docker compose -p "$SALUS_PROJECT" -f "$WORK/compose.yml" down -v --remove-orphans >/dev/null 2>&1 || true
	for sha in $A $B $C; do docker image rm "ghcr.io/$OWNER/salus:$sha" >/dev/null 2>&1 || true; done
	[[ $WORK == /tmp/* ]] && find "$WORK" -delete 2>/dev/null || true
}
trap cleanup EXIT

fail() {
	FAILS=$((FAILS + 1))
	echo "FAIL: $*" >&2
}

cat >"$WORK/compose.yml" <<'EOF'
# migrate fails when FAIL_MIGRATE=yes; app (healthcheck) and worker (none) exit at once when CRASH=yes.
x-app: &app
  image: ghcr.io/salus-deploy-test/salus:${SALUS_TAG:?}
  pull_policy: never
services:
  migrate:
    <<: *app
    command: ["sh", "-c", "test \"$$FAIL_MIGRATE\" != yes"]
    environment: {FAIL_MIGRATE: "${FAIL_MIGRATE:-no}"}
    restart: "no"
  app:
    <<: *app
    command: ["sh", "-c", "test \"$$CRASH\" = yes && exit 1; exec sleep 3600"]
    environment: {CRASH: "${CRASH:-no}"}
    healthcheck: {test: ["CMD", "true"], interval: 1s, timeout: 2s, retries: 3}
    restart: unless-stopped
    depends_on: {migrate: {condition: service_completed_successfully}}
  worker:
    <<: *app
    command: ["sh", "-c", "sleep 2; test \"$$CRASH\" = yes && exit 1; exec sleep 3600"]
    environment: {CRASH: "${CRASH:-no}"}
    restart: unless-stopped
    depends_on: {migrate: {condition: service_completed_successfully}}
EOF

mkdir -p "$SALUS_ROOT/deployments"
docker pull -q "$BUSYBOX" >/dev/null

# The encrypted volume, faked: findmnt and lsblk say SALUS_DATA_DIR is mounted from a dm-crypt device, and the key
# files are owned by the app's uid (written from a container, which may chown), as deploy.sh checks.
mkdir -p "$WORK/bin" "$WORK/data/keys"
printf '#!/bin/sh\necho /dev/mapper/salus-data\n' >"$WORK/bin/findmnt"
printf '#!/bin/sh\necho crypt\n' >"$WORK/bin/lsblk"
chmod +x "$WORK/bin/findmnt" "$WORK/bin/lsblk"
export PATH=$WORK/bin:$PATH
docker run --rm -v "$WORK/data/keys:/keys" "$BUSYBOX" \
	sh -c 'for k in binary_kek totp_key webhook_key; do echo synthetic >/keys/$k; chown 65532:65532 /keys/$k; chmod 400 /keys/$k; done'
release() { # sha, env lines...
	local sha=$1
	shift
	docker tag "$BUSYBOX" "ghcr.io/$OWNER/salus:$sha"
	mkdir -p "$SALUS_ROOT/releases/$sha"
	cp "$WORK/compose.yml" "$SALUS_ROOT/releases/$sha/docker-compose.prod.yml"
	printf '%s\n' "SALUS_DOMAIN=test" "SALUS_DATA_DIR=$WORK/data" "$@" >"$SALUS_ROOT/releases/$sha/.env"
}
running_tag() { # the release tag of a service's container
	docker inspect --format '{{.Config.Image}}' "$(docker compose -p "$SALUS_PROJECT" -f "$SALUS_ROOT/deployments/docker-compose.prod.yml" --env-file "$SALUS_ROOT/.env" --env-file "$SALUS_ROOT/release.env" ps -q "$1")" | sed 's/.*://'
}
deploy() { # expected status, args...
	local want=$1 got=0
	shift
	bash "$DEPLOY" "$@" >"$WORK/out" 2>&1 || got=$?
	if [[ $got != "$want" ]]; then
		fail "deploy.sh $* exited $got, want $want"
		sed 's/^/    /' "$WORK/out" >&2
	fi
}

echo "== a first deploy: migrate runs once and exits 0, up --wait accepts it"
release $A
deploy 0 deploy $A
[[ $(running_tag app) == "$A" && $(running_tag worker) == "$A" ]] || fail "app and worker are not on $A"

echo "== a release whose migrate fails: back to the previous release without migrate"
release $B FAIL_MIGRATE=yes
deploy 1 deploy $B
grep -q "rolled back to $A" "$WORK/out" || fail "deploy.sh did not report the rollback to $A"
[[ $(sed -n 's/^SALUS_TAG=//p' "$SALUS_ROOT/release.env") == "$A" ]] || fail "release.env is not back on $A"
[[ $(running_tag app) == "$A" && $(running_tag worker) == "$A" ]] || fail "app and worker are not back on $A"

echo "== a release whose services crash after the wait: back to the previous release"
release $C CRASH=yes
deploy 1 deploy $C
grep -q "rolled back to $A" "$WORK/out" || fail "deploy.sh did not report the rollback to $A"
[[ $(running_tag app) == "$A" && $(running_tag worker) == "$A" ]] || fail "app and worker are not back on $A"
state=$(docker compose -p "$SALUS_PROJECT" -f "$SALUS_ROOT/deployments/docker-compose.prod.yml" --env-file "$SALUS_ROOT/.env" \
	--env-file "$SALUS_ROOT/release.env" ps --format '{{.Service}} {{.State}}' app worker | sort | tr '\n' ' ')
[[ $state == "app running worker running " ]] || fail "after the rollback: $state"

if ((FAILS)); then
	echo "deploy-compose-test: $FAILS failed"
	exit 1
fi
echo "deploy-compose-test: passed"
