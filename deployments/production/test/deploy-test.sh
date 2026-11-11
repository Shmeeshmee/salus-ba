#!/usr/bin/env bash
# Tests of deployments/production/deploy.sh against a fake docker on PATH: no Docker daemon, no server. Each case
# lays out a throwaway /opt/salus in a temp dir, runs the script, and checks the files it leaves and the docker
# commands it ran. findmnt and lsblk are faked too: the encrypted volume is a directory in the temp dir
# (SALUS_DATA_DIR in the release's env file), "mounted" from a dm-crypt device unless a case says otherwise. Run by
# make deploy-check, in a container as root (it chowns the key files to the app's uid).
set -euo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
DEPLOY=$HERE/../deploy.sh
FAILS=0
PASSES=0

A=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
B=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
C=cccccccccccccccccccccccccccccccccccccccc

ok() { PASSES=$((PASSES + 1)); }
fail() {
	FAILS=$((FAILS + 1))
	echo "FAIL [$CASE]: $*" >&2
}
expect_eq() { if [[ $1 == "$2" ]]; then ok; else fail "$3: got '$1', want '$2'"; fi; }
expect_log() { if grep -qE -- "$1" "$LOG"; then ok; else fail "docker log has no line matching '$1'"; fi; }
expect_no_log() { if grep -qE -- "$1" "$LOG"; then fail "docker log has a line matching '$1'"; else ok; fi; }

# setup lays out a fresh root with the fake docker, and an uploaded release for each SHA given.
setup() {
	CASE=$1
	shift
	WORK=$(mktemp -d)
	ROOT=$WORK/opt/salus
	LOG=$WORK/docker.log
	mkdir -p "$ROOT/deployments" "$ROOT/releases" "$WORK/bin"
	: >"$LOG"
	cat >"$WORK/bin/docker" <<'EOF'
#!/usr/bin/env bash
# Fake docker: logs every call with the SALUS_TAG of the release.env it was given, and fails `compose up` for the
# tags in FAKE_FAIL_TAG (space-separated); `image ls` lists FAKE_IMAGES; after `up`, `ps` shows the worker
# restarting for the tags in FAKE_UNSTABLE_TAG; `inspect` shows the api restarted (FAKE_RESTART_TAG) or unhealthy
# (FAKE_UNHEALTHY_TAG) for those tags. A fake `sleep` in the detached-start cases holds the deploy (FAKE_HOLD).
tag=
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
	if [[ ${args[i]} == --env-file && ${args[i+1]} == *release.env ]]; then
		tag=$(sed -n 's/^SALUS_TAG=//p' "${args[i+1]}")
	fi
done
echo "tag=${tag:-none} $*" >>"$FAKE_LOG"
# Plain docker calls (inspect) carry no env file: the running release is the root's.
[[ -n $tag ]] || tag=$(sed -n 's/^SALUS_TAG=//p' "$SALUS_ROOT/release.env" 2>/dev/null)
case " $* " in
*" config --services "*) printf '%s\n' caddy migrate api worker postgres ;;
*" image ls "*) for i in ${FAKE_IMAGES:-}; do echo "$i"; done ;;
*" ps --all --format "*)
	state=running
	for t in ${FAKE_UNSTABLE_TAG:-}; do [[ $tag == "$t" ]] && state=restarting; done
	printf '%s\n' "migrate exited" "caddy running" "api running" "worker $state" "postgres running"
	;;
*" ps -q "*) printf '%s\n' c1 c2 c3 ;;
*" inspect "*)
	# Name, created, restart count, health: a container created by this deploy (in the future of any run), and an
	# old one (postgres, untouched) that restarted long ago, which must not count.
	count=0 health=healthy
	for t in ${FAKE_RESTART_TAG:-}; do [[ $tag == "$t" ]] && count=1; done
	for t in ${FAKE_UNHEALTHY_TAG:-}; do [[ $tag == "$t" ]] && health=unhealthy; done
	printf '%s\n' "/salus-api-1 2099-01-01T00:00:00.000000000Z $count $health" "/salus-worker-1 2099-01-01T00:00:00.000000000Z 0 " \
		"/salus-postgres-1 2001-01-01T00:00:00.000000000Z 3 healthy"
	;;
*" up "*) for t in ${FAKE_FAIL_TAG:-}; do [[ $tag == "$t" ]] && exit 1; done ;;
esac
exit 0
EOF
	chmod +x "$WORK/bin/docker"
	# The encrypted volume: mounted from /dev/mapper/salus-data, a crypt device, unless FAKE_UNMOUNTED is set or
	# FAKE_DEV_TYPE names another type.
	cat >"$WORK/bin/findmnt" <<'FAKE'
#!/usr/bin/env bash
echo "findmnt $*" >>"$FAKE_LOG"
[[ -z ${FAKE_UNMOUNTED:-} ]] || exit 1
echo /dev/mapper/salus-data
FAKE
	# shellcheck disable=SC2016 # expanded when the fake runs
	printf '#!/usr/bin/env bash\necho "${FAKE_DEV_TYPE:-crypt}"\n' >"$WORK/bin/lsblk"
	chmod +x "$WORK/bin/findmnt" "$WORK/bin/lsblk"
	DATA=$WORK/data
	mkdir -p "$DATA/keys"
	for key in binary_kek totp_key; do
		echo "c3ludGhldGljIDMyLWJ5dGUga2V5IGZvciB0ZXN0cyE=" >"$DATA/keys/$key"
		chown 65532:65532 "$DATA/keys/$key"
		chmod 400 "$DATA/keys/$key"
	done
	for sha in "$@"; do
		mkdir -p "$ROOT/releases/$sha"
		echo "# compose of $sha" >"$ROOT/releases/$sha/docker-compose.prod.yml"
		printf '%s\n' "SALUS_DOMAIN=$sha.test" "SALUS_DATA_DIR=$DATA" >"$ROOT/releases/$sha/.env"
	done
}

# running sets the root up as if SHA had been deployed (and PREV before it).
running() {
	cp -a "$ROOT/releases/$1/." "$ROOT/deployments/"
	mv "$ROOT/deployments/.env" "$ROOT/.env"
	echo "SALUS_TAG=$1" >"$ROOT/release.env"
	if [[ -n ${2:-} ]]; then echo "SALUS_TAG=$2" >"$ROOT/release.previous.env"; fi
}

run() {
	set +e
	PATH="$WORK/bin:$PATH" FAKE_LOG=$LOG SALUS_ROOT=$ROOT SALUS_SETTLE_SECONDS=0 bash "$DEPLOY" "$@" >"$WORK/out" 2>&1
	STATUS=$?
	set -e
}

tag_of() { sed -n 's/^SALUS_TAG=//p' "$ROOT/$1" 2>/dev/null; }

# --- cases ---------------------------------------------------------------------------------------------------

setup "a deploy switches files, env and tag, runs migrate through up, and keeps the one before" "$A" "$B"
running "$A"
run deploy "$B"
expect_eq "$STATUS" 0 "exit status"
expect_eq "$(tag_of release.env)" "$B" "release.env"
expect_eq "$(tag_of release.previous.env)" "$A" "release.previous.env"
expect_eq "$(cat "$ROOT/deployments/docker-compose.prod.yml")" "# compose of $B" "deployed compose file"
expect_eq "$(head -1 "$ROOT/.env")" "SALUS_DOMAIN=$B.test" ".env"
expect_eq "$(stat -c %a "$ROOT/.env")" 600 ".env mode"
expect_log "^tag=$B .* pull"
expect_log "^tag=$B .*-p salus .* up -d --wait"
expect_no_log "--no-deps"

setup "a failed up rolls back to the previous release without migrate" "$A" "$B"
running "$A"
FAKE_FAIL_TAG=$B run deploy "$B"
expect_eq "$STATUS" 1 "exit status"
expect_eq "$(tag_of release.env)" "$A" "release.env after rollback"
expect_eq "$(head -1 "$ROOT/.env")" "SALUS_DOMAIN=$A.test" ".env after rollback"
expect_eq "$(cat "$ROOT/deployments/docker-compose.prod.yml")" "# compose of $A" "compose file after rollback"
expect_log "^tag=$A .* up -d --wait .*--no-deps caddy api worker postgres$"
expect_eq "$([[ -e $ROOT/release.previous.env ]] && echo yes || echo no)" no "no previous after a rollback"
if grep -q "rolled back to $A" "$WORK/out"; then ok; else fail "output does not say it rolled back"; fi

setup "a failed first deploy has nothing to roll back to" "$A"
FAKE_FAIL_TAG=$A run deploy "$A"
expect_eq "$STATUS" 1 "exit status"
expect_no_log "--no-deps"
if grep -q "no previous release" "$WORK/out"; then ok; else fail "output does not say there is no previous release"; fi

setup "a failed rollback exits 2" "$A" "$B"
running "$A"
FAKE_FAIL_TAG="$A $B" run deploy "$B"
expect_eq "$STATUS" 2 "exit status"

setup "--no-migrate brings up every service but migrate" "$A" "$B"
running "$A"
run deploy "$B" --no-migrate
expect_eq "$STATUS" 0 "exit status"
expect_log "^tag=$B .* up -d --wait .*--no-deps caddy api worker postgres$"

setup "rollback by hand goes back to the previous release" "$A" "$B"
running "$B" "$A"
run rollback
expect_eq "$STATUS" 0 "exit status"
expect_eq "$(tag_of release.env)" "$A" "release.env"
expect_log "^tag=$A .* up -d --wait .*--no-deps caddy api worker postgres$"

setup "rollback by hand without a previous release refuses" "$A"
running "$A"
run rollback
expect_eq "$STATUS" 1 "exit status"
expect_no_log " up "

setup "a SHA that is not 40 hex characters, or a release not uploaded, is refused" "$A"
run deploy main
expect_eq "$STATUS" 1 "a tag name"
run deploy "$C"
expect_eq "$STATUS" 1 "a release not uploaded"
expect_no_log " up "

setup "after a deploy, old releases and old Salus images go; other images stay" "$A" "$B" "$C"
running "$B" "$A"
export FAKE_IMAGES="ghcr.io/shmeeshmee/salus:$A ghcr.io/shmeeshmee/salus-web:$B ghcr.io/shmeeshmee/salus:$C ghcr.io/shmeeshmee/salus:main other/app:$A postgres:18.6"
mkdir -p "$ROOT/releases/dddddddddddddddddddddddddddddddddddddddd"
run deploy "$C"
unset FAKE_IMAGES
expect_eq "$STATUS" 0 "exit status"
expect_eq "$(find "$ROOT/releases" -mindepth 1 -maxdepth 1 -printf '%f\n' | sort | tr '\n' ' ')" "$B $C " "releases kept"
expect_log "image rm ghcr.io/shmeeshmee/salus:$A$"
expect_no_log "image rm .*(salus-web:$B|salus:$C|:main|other/app|postgres)"

setup "a service that does not stay up after the wait fails the deploy, which rolls back" "$A" "$B"
running "$A"
FAKE_UNSTABLE_TAG=$B run deploy "$B"
expect_eq "$STATUS" 1 "exit status"
expect_eq "$(tag_of release.env)" "$A" "release.env after rollback"
if grep -q "worker" "$WORK/out"; then ok; else fail "output does not name the unstable service"; fi

setup "a second deploy while one holds the lock is refused" "$A" "$B"
running "$A"
exec 8>"$ROOT/.deploy.lock"
flock 8
run deploy "$B"
exec 8>&-
expect_eq "$STATUS" 1 "exit status"
expect_no_log " up "
if grep -q "another deploy" "$WORK/out"; then ok; else fail "output does not say another deploy runs"; fi

setup "start runs the deploy detached and status reports its exit status" "$A" "$B"
running "$A"
run status "$B"
expect_eq "$(cat "$WORK/out")" "none" "status before start"
run start "$B"
expect_eq "$STATUS" 0 "start returns at once"
for _ in $(seq 50); do
	run status "$B"
	[[ $(cat "$WORK/out") == running ]] || break
	sleep 0.1
done
expect_eq "$(cat "$WORK/out")" "0" "status after the detached deploy"
expect_eq "$(tag_of release.env)" "$B" "release.env"
if grep -q "deployed $B" "$ROOT/releases/$B/deploy.log"; then ok; else fail "the deploy log has no result"; fi

setup "a container this deploy created that restarted, or one unhealthy, fails it; an old restart does not" "$A" "$B" "$C"
running "$A"
FAKE_RESTART_TAG=$B run deploy "$B"
expect_eq "$STATUS" 1 "a restarted new container"
if grep -q "salus-api-1" "$WORK/out"; then ok; else fail "output does not name the restarted container"; fi
FAKE_UNHEALTHY_TAG=$C run deploy "$C"
expect_eq "$STATUS" 1 "an unhealthy container"
run deploy "$C"
expect_eq "$STATUS" 0 "the old postgres restart does not count"

setup "rollback is refused while a deploy holds the lock" "$A" "$B"
running "$B" "$A"
exec 8>"$ROOT/.deploy.lock"
flock 8
run rollback
exec 8>&-
expect_eq "$STATUS" 1 "exit status"
expect_no_log " up "

setup "start while a deploy runs is refused and leaves its status and log alone" "$A" "$B"
running "$A"
printf '#!/usr/bin/env bash\nwhile [[ -e "%s" ]]; do command sleep 0.05; done\n' "$WORK/hold" >"$WORK/bin/sleep"
chmod +x "$WORK/bin/sleep"
: >"$WORK/hold"
run start "$B"
expect_eq "$STATUS" 0 "the first start"
run status "$B"
expect_eq "$(cat "$WORK/out")" running "status while held"
echo "first deploy's line" >>"$ROOT/releases/$B/deploy.log"
run start "$B"
expect_eq "$STATUS" 1 "a second start"
run status "$B"
expect_eq "$(cat "$WORK/out")" running "status after the refused start"
if grep -q "first deploy's line" "$ROOT/releases/$B/deploy.log"; then ok; else fail "the refused start emptied the log"; fi
rm -f "$WORK/hold"
for _ in $(seq 100); do
	run status "$B"
	[[ $(cat "$WORK/out") == running ]] || break
	command sleep 0.1
done
expect_eq "$(cat "$WORK/out")" 0 "status once the first deploy finished"

setup "start passes --no-migrate to the detached deploy" "$A" "$B"
running "$A"
run start "$B" --no-migrate
for _ in $(seq 50); do
	run status "$B"
	[[ $(cat "$WORK/out") == running ]] || break
	command sleep 0.1
done
expect_eq "$(cat "$WORK/out")" 0 "status"
expect_log "^tag=$B .* up -d --wait .*--no-deps caddy api worker postgres$"

setup "status says died when the detached deploy is gone without a result" "$A" "$B"
mkdir -p "$ROOT/releases/$B"
echo "partial" >"$ROOT/releases/$B/deploy.log"
echo 999999999 >"$ROOT/releases/$B/deploy.pid"
run status "$B"
expect_eq "$(cat "$WORK/out")" died "status"

# --- the encrypted volume and the keys (ADR 0011) ---------------------------------------------------------------

refused() { # refused WHAT PATTERN: exit 1, PATTERN in the output, and nothing pulled, migrated or switched
	expect_eq "$STATUS" 1 "$1: exit status"
	if grep -qE -- "$2" "$WORK/out"; then ok; else fail "$1: output has no '$2': $(cat "$WORK/out")"; fi
	expect_no_log " (pull|up|config) "
}

setup "a deploy is refused when the encrypted volume is not mounted" "$A" "$B"
running "$A"
FAKE_UNMOUNTED=1 run deploy "$B"
refused "unmounted" "not mounted at $WORK/data: unlock it first"
expect_eq "$(tag_of release.env)" "$A" "release.env unchanged"
expect_log "^findmnt .*--mountpoint $WORK/data$"

setup "a deploy is refused when the mount point is not a dm-crypt device" "$A"
FAKE_DEV_TYPE=disk run deploy "$A"
refused "not crypt" "not a dm-crypt \(LUKS\) device"

setup "the mount point defaults to /mnt/salus-data" "$A"
echo "SALUS_DOMAIN=$A.test" >"$ROOT/releases/$A/.env"
FAKE_UNMOUNTED=1 run deploy "$A"
refused "default" "not mounted at /mnt/salus-data"

setup "a deploy is refused when a key file is missing, or not 65532's with mode 400" "$A"
rm "$WORK/data/keys/totp_key"
run deploy "$A"
refused "missing key" "key file $WORK/data/keys/totp_key is missing"
cp -p "$WORK/data/keys/binary_kek" "$WORK/data/keys/totp_key"
chmod 440 "$WORK/data/keys/binary_kek"
run deploy "$A"
refused "group-readable key" "binary_kek is not owned by uid 65532 with mode 400"
chmod 400 "$WORK/data/keys/binary_kek"
chown 0:0 "$WORK/data/keys/binary_kek"
run deploy "$A"
refused "root-owned key" "binary_kek is not owned by uid 65532"

for line in "BINARY_KEK=c2VjcmV0" "TOTP_KEY=" "export TOTP_KEY=c2VjcmV0"; do
	setup "an env file with '$line' is refused and removed from the release" "$A" "$B"
	running "$A"
	echo "$line" >>"$ROOT/releases/$B/.env"
	run deploy "$B"
	refused "keys in env" "sets BINARY_KEK or TOTP_KEY"
	expect_eq "$([[ -e $ROOT/releases/$B/.env ]] && echo yes || echo no)" no "the env file with the keys is gone"
	expect_eq "$(tag_of release.env)" "$A" "release.env unchanged"
done

setup "BINARY_KEK_ID in the env file is not a key" "$A"
echo "BINARY_KEK_ID=prod-1" >>"$ROOT/releases/$A/.env"
run deploy "$A"
expect_eq "$STATUS" 0 "exit status"

setup "a rollback by hand is refused when the volume is not mounted" "$A" "$B"
running "$B" "$A"
FAKE_UNMOUNTED=1 run rollback
refused "rollback unmounted" "not mounted"
expect_eq "$(tag_of release.env)" "$B" "release.env unchanged"

setup "a rollback by hand is refused when the previous release's env file holds a key" "$A" "$B"
running "$B" "$A"
echo "TOTP_KEY=c2VjcmV0" >>"$ROOT/releases/$A/.env"
run rollback
refused "rollback keys" "sets BINARY_KEK or TOTP_KEY"

setup "a failed deploy does not roll back onto an unmounted volume either" "$A" "$B"
running "$A"
echo "SALUS_DATA_DIR=$WORK/elsewhere" >>"$ROOT/releases/$A/.env"
FAKE_FAIL_TAG=$B run deploy "$B"
expect_eq "$STATUS" 2 "exit status"
if grep -q "key file $WORK/elsewhere/keys/binary_kek is missing" "$WORK/out"; then ok; else fail "the rollback did not check the volume"; fi

echo "deploy-test: $PASSES passed, $FAILS failed"
((FAILS == 0))
