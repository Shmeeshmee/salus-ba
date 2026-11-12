#!/usr/bin/env bash
# shellcheck disable=SC2016 # the stubs expand when they run
# Runs inside a fresh ubuntu:24.04 container as root (run.sh): volume.sh's setup, unlock, keys and check against
# stubs of cryptsetup, mount, findmnt, lsblk, blkid, mkfs.ext4 and docker on PATH. Nothing is formatted, opened or
# mounted: the "volume" is a directory, the "device" a block device node made with mknod, and the stubs keep their
# state in files. Ownership, modes, the key files and the prompts are real.
set -euo pipefail

VOLUME=/bootstrap/volume.sh
export WORK=/tmp/volume-test # the stubs keep their state under it
FAILS=0
PASSES=0
# Synthetic keys: base64 of 32 bytes each.
KEK=c3ludGhldGljIDMyLWJ5dGUgS0VLIGZvciB0ZXN0cyE=
TOTP=c3ludGhldGljIDMyLWJ5dGUga2V5IGZvciB0ZXN0cyE=
HOOK=c3ludGhldGljIDMyLWJ5dGUgd2ViaG9vayBrZXkhISE=

ok() { PASSES=$((PASSES + 1)); }
fail() {
	FAILS=$((FAILS + 1))
	echo "FAIL [$CASE]: $*" >&2
}
expect_eq() { if [[ $1 == "$2" ]]; then ok; else fail "$3: got '$1', want '$2'"; fi; }
expect_out() { if grep -qE -- "$1" "$WORK/out"; then ok; else fail "output has no '$1': $(cat "$WORK/out")"; fi; }
expect_log() { if grep -qE -- "$1" "$WORK/calls"; then ok; else fail "no call matching '$1'"; fi; }
expect_no_log() { if grep -qE -- "$1" "$WORK/calls"; then fail "a call matches '$1'"; else ok; fi; }

id deploy >/dev/null 2>&1 || useradd --create-home --user-group deploy

# fresh CASE: a new work dir with the stubs, an empty "volume" at $WORK/mnt and a device node at $WORK/dev.
fresh() {
	CASE=$1
	rm -rf "$WORK"
	mkdir -p "$WORK/bin" "$WORK/state" "$WORK/etc" "$WORK/opt"
	: >"$WORK/calls"
	mknod "$WORK/dev" b 7 250
	cat >"$WORK/bin/cryptsetup" <<'STUB'
#!/usr/bin/env bash
echo "cryptsetup $*" >>"$WORK/calls"
case $1 in
luksFormat) touch "$WORK/state/formatted" ;;
luksUUID) echo 0b5e7a1c-0000-4000-8000-000000000001 ;;
open) [[ -z ${FAKE_WRONG_PASSPHRASE:-} ]] || exit 2; touch "$WORK/state/open" ;;
status) [[ -e $WORK/state/open ]] ;;
close) rm -f "$WORK/state/open" ;;
esac
STUB
	cat >"$WORK/bin/blkid" <<'STUB'
#!/usr/bin/env bash
[[ -e $WORK/state/formatted || -n ${FAKE_SIGNATURE:-} ]]
STUB
	cat >"$WORK/bin/mount" <<'STUB'
#!/usr/bin/env bash
echo "mount $*" >>"$WORK/calls"
touch "$WORK/state/mounted"
STUB
	cat >"$WORK/bin/findmnt" <<'STUB'
#!/usr/bin/env bash
[[ -e $WORK/state/mounted ]] || exit 1
echo "${FAKE_SOURCE:-/dev/mapper/salus-data}"
STUB
	cat >"$WORK/bin/lsblk" <<'STUB'
#!/usr/bin/env bash
case " $* " in
*" MOUNTPOINTS "*) echo "${FAKE_DEV_MOUNTS:-}" ;;
*) echo "${FAKE_DEV_TYPE:-crypt}" ;;
esac
STUB
	printf '#!/usr/bin/env bash\necho "umount $*" >>"$WORK/calls"\nrm -f "$WORK/state/mounted"\n' >"$WORK/bin/umount"
	printf '#!/usr/bin/env bash\necho "mkfs.ext4 $*" >>"$WORK/calls"\n[[ -z ${FAKE_MKFS_FAIL:-} ]]\n' >"$WORK/bin/mkfs.ext4"
	printf '#!/usr/bin/env bash\necho "docker $*" >>"$WORK/calls"\nexit ${FAKE_DOCKER_STATUS:-0}\n' >"$WORK/bin/docker"
	chmod +x "$WORK/bin/"*
}

run() { # run STDIN ARGS...: volume.sh with the stubs, STDIN as its input
	local input=$1
	shift
	set +e
	printf '%b' "$input" | PATH="$WORK/bin:$PATH" SALUS_VOLUME_CONF=$WORK/etc/volume.conf \
		SALUS_DATA_DIR=$WORK/mnt SALUS_ROOT=$WORK/opt setsid -w bash "$VOLUME" "$@" >"$WORK/out" 2>&1
	STATUS=$?
	set -e
}

run_as_deploy() { # run_as_deploy ARGS...: volume.sh as the deploy user
	set +e
	runuser -u deploy -- env PATH="$WORK/bin:$PATH" WORK=$WORK SALUS_VOLUME_CONF=$WORK/etc/volume.conf \
		SALUS_DATA_DIR=$WORK/mnt SALUS_ROOT=$WORK/opt bash "$VOLUME" "$@" </dev/null >"$WORK/out" 2>&1
	STATUS=$?
	set -e
}

stat_of() { stat -c '%a %U:%G' "$1" 2>/dev/null || echo missing; }

# --- setup ---------------------------------------------------------------------------------------------------

fresh "setup formats, opens, lays out and writes the keys"
run "$KEK\n$TOTP\n$HOOK\n" setup "$WORK/dev"
expect_eq "$STATUS" 0 "exit status: $(cat "$WORK/out")"
expect_log "^cryptsetup luksFormat --type luks2 --verify-passphrase $WORK/dev$"
expect_log "^cryptsetup open --type luks2 $WORK/dev salus-data$"
expect_log "^mkfs.ext4 -q -L salus-data /dev/mapper/salus-data$"
expect_log "^mount -t ext4 -o nodev,nosuid,noexec /dev/mapper/salus-data $WORK/mnt$"
expect_eq "$(cat "$WORK/etc/volume.conf" | grep -v '^#' | tr '\n' ' ')" \
	"DEVICE=/dev/disk/by-uuid/0b5e7a1c-0000-4000-8000-000000000001 NAME=salus-data MOUNTPOINT=$WORK/mnt " "volume.conf"
expect_eq "$(stat_of "$WORK/mnt")" "710 root:deploy" "the volume's root"
expect_eq "$(stat -c '%a %u:%g' "$WORK/mnt/postgres")" "700 999:999" "postgres/"
for d in garage garage/meta garage/data; do expect_eq "$(stat_of "$WORK/mnt/$d")" "700 root:root" "$d/"; done
expect_eq "$(stat -c '%a %u:%g' "$WORK/mnt/clamd")" "700 100:101" "clamd/ (the clamav image's clamav user)"
expect_eq "$(stat_of "$WORK/mnt/keys")" "710 root:deploy" "keys/"
expect_eq "$(stat -c '%a %u:%g' "$WORK/mnt/keys/binary_kek")" "400 65532:65532" "binary_kek"
expect_eq "$(stat -c '%a %u:%g' "$WORK/mnt/keys/totp_key")" "400 65532:65532" "totp_key"
expect_eq "$(cat "$WORK/mnt/keys/binary_kek")" "$KEK" "binary_kek's value"
expect_eq "$(cat "$WORK/mnt/keys/totp_key")" "$TOTP" "totp_key's value"
expect_eq "$(stat -c '%a %u:%g' "$WORK/mnt/keys/webhook_key")" "400 65532:65532" "webhook_key"
expect_eq "$(cat "$WORK/mnt/keys/webhook_key")" "$HOOK" "webhook_key's value"
if grep -qF -e "$KEK" -e "$TOTP" -e "$HOOK" "$WORK/out"; then fail "a key was echoed"; else ok; fi
expect_eq "$(find "$WORK/mnt/keys" -name '.*' | wc -l)" 0 "no temporary key files left"
expect_eq "$(grep -c salus /etc/fstab /etc/crypttab 2>/dev/null | awk -F: '{s += $2} END {print s + 0}')" 0 "no fstab or crypttab line"

CASE="setup a second time is refused"
run "" setup "$WORK/dev"
expect_eq "$STATUS" 1 "exit status"
expect_out "is set up already"

fresh "setup refuses a device that holds a filesystem or LUKS header"
FAKE_SIGNATURE=1 run "" setup "$WORK/dev"
expect_eq "$STATUS" 1 "exit status"
expect_out "refusing to format it"
expect_no_log "luksFormat"

fresh "setup refuses a mounted device, and a path that is not a block device"
FAKE_DEV_MOUNTS=/srv run "" setup "$WORK/dev"
expect_eq "$STATUS" 1 "mounted"
expect_out "is mounted"
run "" setup "$WORK/calls"
expect_eq "$STATUS" 1 "not a block device"
expect_out "not a block device"
expect_no_log "luksFormat"

fresh "setup refuses a key that is not base64 of 32 bytes, and two equal keys"
run "c2hvcnQ=\n$TOTP\n$HOOK\n" setup "$WORK/dev"
expect_eq "$STATUS" 1 "a short key"
expect_out "BINARY_KEK is not base64 of 32 bytes: no key file was written"
expect_eq "$(stat_of "$WORK/mnt/keys/binary_kek")" missing "nothing written"
expect_out "setup did not finish"
expect_log "^umount $WORK/mnt$"
expect_log "^cryptsetup close salus-data$"
expect_eq "$(stat_of "$WORK/etc/volume.conf")" missing "no volume.conf after a failed setup"

fresh "a setup that fails half-way closes the volume again and writes no volume.conf"
FAKE_MKFS_FAIL=1 run "$KEK\n$TOTP\n$HOOK\n" setup "$WORK/dev"
expect_eq "$STATUS" 1 "exit status"
expect_out "setup did not finish.*wipefs -a $WORK/dev"
expect_log "^cryptsetup close salus-data$"
expect_no_log "^mount"
expect_eq "$([[ -e $WORK/state/open ]] && echo open || echo closed)" closed "the mapping"
expect_eq "$(stat_of "$WORK/etc/volume.conf")" missing "volume.conf"

# --- unlock ----------------------------------------------------------------------------------------------------

# locked: the state after a reboot of a set-up server: laid out, closed, not mounted.
locked() {
	fresh "$1"
	run "$KEK\n$TOTP\n$HOOK\n" setup "$WORK/dev"
	[[ $STATUS == 0 ]] || fail "setup for the case failed: $(cat "$WORK/out")"
	sed -i "s|^DEVICE=.*|DEVICE=$WORK/dev|" "$WORK/etc/volume.conf"
	rm -f "$WORK/state/open" "$WORK/state/mounted"
	: >"$WORK/calls"
}

locked "unlock opens, mounts, checks and says how to start when no release is deployed yet"
run "" unlock
expect_eq "$STATUS" 0 "exit status: $(cat "$WORK/out")"
expect_log "^cryptsetup open --type luks2 $WORK/dev salus-data$"
expect_log "^mount -t ext4 -o nodev,nosuid,noexec /dev/mapper/salus-data $WORK/mnt$"
expect_out "no release is deployed yet"
expect_no_log "^docker"

CASE="unlock again is idempotent"
: >"$WORK/calls"
run "" unlock --no-start
expect_eq "$STATUS" 0 "exit status"
expect_out "open already"
expect_out "already mounted"
expect_no_log "cryptsetup open|^mount"

locked "unlock starts the deployed stack"
echo "SALUS_TAG=aaaa" >"$WORK/opt/release.env"
run "" unlock
expect_eq "$STATUS" 0 "exit status: $(cat "$WORK/out")"
expect_log "^docker compose -p salus --env-file $WORK/opt/.env --env-file $WORK/opt/release.env -f $WORK/opt/deployments/docker-compose.prod.yml up -d --wait"
CASE="unlock --no-start prints the command instead"
: >"$WORK/calls"
run "" unlock --no-start
expect_out "To start the stack: docker compose -p salus"
expect_no_log "^docker"
CASE="a stack that does not come up fails the unlock"
FAKE_DOCKER_STATUS=1 run "" unlock
expect_eq "$STATUS" 1 "exit status"
expect_out "did not come up healthy"

locked "a wrong passphrase stops it before mounting"
FAKE_WRONG_PASSPHRASE=1 run "" unlock
expect_eq "$STATUS" 1 "exit status"
expect_out "wrong passphrase"
expect_no_log "^mount"

locked "a detached volume is named"
rm "$WORK/dev"
run "" unlock
expect_eq "$STATUS" 1 "exit status"
expect_out "is the volume attached"

locked "something else mounted at the mount point is refused"
touch "$WORK/state/mounted"
FAKE_SOURCE=/dev/sda1 run "" unlock
expect_eq "$STATUS" 1 "exit status"
expect_out "mounted from /dev/sda1, not from the volume"

locked "a missing key file or a wrong mode keeps the stack down"
echo "SALUS_TAG=aaaa" >"$WORK/opt/release.env"
rm "$WORK/mnt/keys/totp_key"
chmod 444 "$WORK/mnt/keys/binary_kek"
run "" unlock
expect_eq "$STATUS" 1 "exit status"
expect_out "totp_key is missing: write it with salus-volume keys"
expect_out "binary_kek is 444 65532:65532 \(mode uid:gid\), want 400 65532:65532"
expect_out "the stack is not started"
expect_no_log "^docker"

locked "unlock lays the directories out again (another server's deploy gid, a surviving volume)"
chgrp root "$WORK/mnt" "$WORK/mnt/keys"
chmod 755 "$WORK/mnt/garage"
run "" unlock --no-start
expect_eq "$STATUS" 0 "exit status: $(cat "$WORK/out")"
expect_eq "$(stat_of "$WORK/mnt")" "710 root:deploy" "the volume's root"
expect_eq "$(stat_of "$WORK/mnt/keys")" "710 root:deploy" "keys/"
expect_eq "$(stat_of "$WORK/mnt/garage")" "700 root:root" "garage/"

locked "unlock does not start the stack while a deploy holds the lock"
echo "SALUS_TAG=aaaa" >"$WORK/opt/release.env"
install -m 644 -o deploy -g deploy /dev/null "$WORK/opt/.deploy.lock"
exec 8>"$WORK/opt/.deploy.lock"
flock 8
run "" unlock
exec 8>&-
expect_eq "$STATUS" 1 "exit status"
expect_out "a deploy or rollback is running"
expect_no_log "^docker"
CASE="unlock takes the deploy lock, creating it as deploy's when missing"
rm "$WORK/opt/.deploy.lock"
run "" unlock
expect_eq "$STATUS" 0 "exit status: $(cat "$WORK/out")"
expect_log "^docker compose -p salus"
expect_eq "$(stat_of "$WORK/opt/.deploy.lock")" "644 deploy:deploy" "the lock file"

# --- keys and check --------------------------------------------------------------------------------------------

locked "keys rewrites a missing key file and keeps the same value"
run "" unlock --no-start
rm "$WORK/mnt/keys/totp_key"
run "$KEK\n$TOTP\n$HOOK\n" keys
expect_eq "$STATUS" 0 "exit status: $(cat "$WORK/out")"
expect_out "binary_kek unchanged"
expect_out "wrote $WORK/mnt/keys/totp_key"
expect_eq "$(stat -c '%a %u:%g' "$WORK/mnt/keys/totp_key")" "400 65532:65532" "totp_key"
CASE="keys refuses to replace a key with another value unless --replace"
OTHER=b3RoZXIgc3ludGhldGljIDMyLWJ5dGUga2V5LCBvayE=
run "$OTHER\n$TOTP\n$HOOK\n" keys
expect_eq "$STATUS" 1 "without --replace"
expect_out "holds another key"
expect_eq "$(cat "$WORK/mnt/keys/binary_kek")" "$KEK" "binary_kek kept"
run "$OTHER\n$TOTP\n$HOOK\n" keys --replace
expect_eq "$STATUS" 0 "with --replace"
expect_eq "$(cat "$WORK/mnt/keys/binary_kek")" "$OTHER" "binary_kek replaced"

CASE="keys refuses two equal keys"
run "$KEK\n$TOTP\n$KEK\n" keys --replace
expect_eq "$STATUS" 1 "two equal keys"
expect_out "BINARY_KEK and WEBHOOK_KEY are the same value"
expect_eq "$(cat "$WORK/mnt/keys/webhook_key")" "$HOOK" "webhook_key kept"

locked "keys needs the volume open and mounted"
run "$KEK\n$TOTP\n$HOOK\n" keys
expect_eq "$STATUS" 1 "exit status"
expect_out "not open and mounted"

locked "check: locked, then not a crypt device, then ok"
run "" check
expect_eq "$STATUS" 1 "locked"
expect_out "nothing is mounted"
run "" unlock --no-start
FAKE_DEV_TYPE=disk run "" check
expect_eq "$STATUS" 1 "not crypt"
expect_out "not from the dm-crypt device"
run "" check
expect_eq "$STATUS" 0 "ok"
CASE="check needs root, which alone can read the key files"
run_as_deploy check
expect_eq "$STATUS" 1 "as deploy"
expect_out "run as root"
CASE="check: a host account with the key files' uid could read them"
useradd --uid 65532 --no-create-home synthetic-nonroot 2>/dev/null
run "" check
userdel synthetic-nonroot
expect_eq "$STATUS" 1 "exit status"
expect_out "host account synthetic-nonroot has uid 65532"

echo "volume-test: $PASSES passed, $FAILS failed"
((FAILS == 0))
