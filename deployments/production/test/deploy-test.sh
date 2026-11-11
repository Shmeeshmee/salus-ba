#!/usr/bin/env bash
# Tests of deployments/production/deploy.sh against a fake docker on PATH: no Docker daemon, no server. Each case
# lays out a throwaway /opt/salus in a temp dir, runs the script, and checks the files it leaves and the docker
# commands it ran. Run by make deploy-check (or on its own: bash deployments/production/test/deploy-test.sh).
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
# tags in FAKE_FAIL_TAG (space-separated); `image ls` lists FAKE_IMAGES.
tag=
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
	if [[ ${args[i]} == --env-file && ${args[i+1]} == *release.env ]]; then
		tag=$(sed -n 's/^SALUS_TAG=//p' "${args[i+1]}")
	fi
done
echo "tag=${tag:-none} $*" >>"$FAKE_LOG"
case " $* " in
*" config --services "*) printf '%s\n' caddy migrate api worker postgres ;;
*" image ls "*) for i in ${FAKE_IMAGES:-}; do echo "$i"; done ;;
*" up "*) for t in ${FAKE_FAIL_TAG:-}; do [[ $tag == "$t" ]] && exit 1; done ;;
esac
exit 0
EOF
	chmod +x "$WORK/bin/docker"
	for sha in "$@"; do
		mkdir -p "$ROOT/releases/$sha"
		echo "# compose of $sha" >"$ROOT/releases/$sha/docker-compose.prod.yml"
		echo "SALUS_DOMAIN=$sha.test" >"$ROOT/releases/$sha/.env"
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
	PATH="$WORK/bin:$PATH" FAKE_LOG=$LOG SALUS_ROOT=$ROOT bash "$DEPLOY" "$@" >"$WORK/out" 2>&1
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
expect_eq "$(cat "$ROOT/.env")" "SALUS_DOMAIN=$B.test" ".env"
expect_eq "$(stat -c %a "$ROOT/.env")" 600 ".env mode"
expect_log "^tag=$B .* pull"
expect_log "^tag=$B .*-p salus .* up -d --wait"
expect_no_log "--no-deps"

setup "a failed up rolls back to the previous release without migrate" "$A" "$B"
running "$A"
FAKE_FAIL_TAG=$B run deploy "$B"
expect_eq "$STATUS" 1 "exit status"
expect_eq "$(tag_of release.env)" "$A" "release.env after rollback"
expect_eq "$(cat "$ROOT/.env")" "SALUS_DOMAIN=$A.test" ".env after rollback"
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

echo "deploy-test: $PASSES passed, $FAILS failed"
((FAILS == 0))
