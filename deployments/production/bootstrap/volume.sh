#!/usr/bin/env bash
# The encrypted Hetzner Volume (docs/adr/0011-single-server-deployment.md, "Decided before real patient data"):
# Postgres's and Garage's data and the key files (BINARY_KEK, TOTP_KEY) on a LUKS2 volume that is unlocked by hand
# after every boot. Installed by bootstrap.sh as /usr/local/sbin/salus-volume. Run as root:
#
#   salus-volume setup DEVICE        once: LUKS2-format the new, empty volume DEVICE (/dev/disk/by-id/scsi-0HC_Volume_…),
#                                    create the filesystem, the layout and the key files, write /etc/salus/volume.conf
#   salus-volume unlock [--no-start] after a reboot: open the volume, mount it, check it, start the stack
#   salus-volume keys [--replace]    write the key files from Bitwarden (a restore onto a new volume); the volume open
#   salus-volume check               is the volume open, mounted, laid out, with both key files? (exit 1 if not;
#                                    as root: only root reads the key files)
#
# The passphrase is typed into cryptsetup, never into this script, and is kept only in the owner's Bitwarden. The
# keys are pasted from Bitwarden at a prompt that does not echo them. Nothing opens or mounts the volume at boot: no
# /etc/crypttab line, no /etc/fstab line, so a reboot never waits for a passphrase and the stack stays down until
# the owner unlocks.
#
# Layout (deployments/production/README.md "Encrypted volume"), on the volume's filesystem at MOUNTPOINT:
#   .                         root:deploy 0710   deploy may pass (deploy.sh checks the key files), nobody else
#   postgres/                 999:999 0700       Postgres's data (the postgres image's uid)
#   garage/, meta/, data/     root:root 0700     Garage's metadata and blocks (Garage runs as root)
#   clamd/                    100:101 0700       clamd's TemporaryDirectory: the uploads it scans (the clamav user)
#   keys/                     root:deploy 0710
#   keys/binary_kek, totp_key 65532:65532 0400   the app image's nonroot uid, which no host account has
#
# Settings: /etc/salus/volume.conf (SALUS_VOLUME_CONF), KEY=value lines written by setup: DEVICE (by LUKS UUID),
# NAME (the dm-crypt mapping) and MOUNTPOINT (SALUS_DATA_DIR at setup, default /mnt/salus-data; the compose file's
# SALUS_DATA_DIR must match). The stack is the compose project salus in /opt/salus (SALUS_ROOT); unlock starts it
# holding deploy.sh's lock, /opt/salus/.deploy.lock, so it never runs beside a deploy or a rollback.
set -euo pipefail

CONF=${SALUS_VOLUME_CONF:-/etc/salus/volume.conf}
ROOT=${SALUS_ROOT:-/opt/salus}
readonly DEPLOY_GROUP=deploy
readonly KEY_UID=65532 # gcr.io/distroless/static:nonroot, build/package/Dockerfile
readonly PG_UID=999    # the postgres image's postgres user
readonly CLAMAV_UID=100 CLAMAV_GID=101 # the clamav image's clamav user
readonly KEYS=(binary_kek totp_key)
readonly MOUNT_OPTIONS=nodev,nosuid,noexec

DEVICE=
NAME=salus-data
MOUNTPOINT=${SALUS_DATA_DIR:-/mnt/salus-data}

log() { printf 'salus-volume: %s\n' "$*"; }
warn() { printf 'salus-volume: warning: %s\n' "$*" >&2; }
die() {
	printf 'salus-volume: error: %s\n' "$*" >&2
	exit 1
}

usage() {
	sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed '$d; s/^# \{0,1\}//'
	exit "${1:-0}"
}

need_root() { [[ $EUID -eq 0 ]] || die "run as root"; }

# load_conf: KEY=value lines, parsed, never sourced.
load_conf() {
	local line key value
	[[ -r $CONF ]] || die "$CONF is missing: the volume is not set up on this server (salus-volume setup, docs/deploy/README.md)"
	while IFS= read -r line || [[ -n $line ]]; do
		[[ $line =~ ^[[:space:]]*(#|$) ]] && continue
		[[ $line =~ ^(DEVICE|NAME|MOUNTPOINT)=(.+)$ ]] || die "$CONF: unexpected line '$line'"
		key=${BASH_REMATCH[1]}
		value=${BASH_REMATCH[2]}
		printf -v "$key" '%s' "$value"
	done <"$CONF"
	[[ -n $DEVICE ]] || die "$CONF has no DEVICE"
}

deploy_gid() { getent group "$DEPLOY_GROUP" | cut -d: -f3; }

is_open() { cryptsetup status "$NAME" >/dev/null 2>&1; }

# mounted_from: the device mounted at MOUNTPOINT, empty when nothing is.
mounted_from() { findmnt -n -o SOURCE --mountpoint "$MOUNTPOINT" 2>/dev/null || true; }

# expect_path PATH TYPE MODE UID:GID: prints the problem and returns 1 when PATH is not as expected.
expect_path() {
	local path=$1 type=$2 want="$3 $4" got
	if [[ $type == d && ! -d $path ]] || [[ $type == f && ! -f $path ]]; then
		printf 'salus-volume: %s is missing\n' "$path" >&2
		return 1
	fi
	got=$(stat -c '%a %u:%g' "$path")
	[[ $got == "$want" ]] && return 0
	printf 'salus-volume: %s is %s (mode uid:gid), want %s\n' "$path" "$got" "$want" >&2
	return 1
}

# key_ok FILE: base64 of exactly 32 bytes.
key_ok() {
	local n
	n=$(base64 -d <"$1" 2>/dev/null | wc -c) || return 1
	[[ $n == 32 ]]
}

# check_layout prints every problem; returns 1 when there is one.
check_layout() {
	local bad=0 src gid key
	src=$(mounted_from)
	if [[ -z $src ]]; then
		printf 'salus-volume: nothing is mounted at %s\n' "$MOUNTPOINT" >&2
		return 1
	fi
	if [[ $src != "/dev/mapper/$NAME" ]] || [[ $(lsblk -ndo TYPE "$src" 2>/dev/null) != crypt ]]; then
		printf 'salus-volume: %s is mounted from %s, not from the dm-crypt device /dev/mapper/%s\n' "$MOUNTPOINT" "$src" "$NAME" >&2
		return 1
	fi
	gid=$(deploy_gid)
	[[ -n $gid ]] || { printf 'salus-volume: there is no %s group (run bootstrap.sh first)\n' "$DEPLOY_GROUP" >&2; return 1; }
	expect_path "$MOUNTPOINT" d 710 "0:$gid" || bad=1
	expect_path "$MOUNTPOINT/postgres" d 700 "$PG_UID:$PG_UID" || bad=1
	expect_path "$MOUNTPOINT/garage" d 700 0:0 || bad=1
	expect_path "$MOUNTPOINT/garage/meta" d 700 0:0 || bad=1
	expect_path "$MOUNTPOINT/garage/data" d 700 0:0 || bad=1
	expect_path "$MOUNTPOINT/clamd" d 700 "$CLAMAV_UID:$CLAMAV_GID" || bad=1
	expect_path "$MOUNTPOINT/keys" d 710 "0:$gid" || bad=1
	for key in "${KEYS[@]}"; do
		if [[ ! -f $MOUNTPOINT/keys/$key ]]; then
			printf 'salus-volume: %s is missing: write it with salus-volume keys (from Bitwarden)\n' "$MOUNTPOINT/keys/$key" >&2
			bad=1
		elif expect_path "$MOUNTPOINT/keys/$key" f 400 "$KEY_UID:$KEY_UID"; then
			key_ok "$MOUNTPOINT/keys/$key" || { printf 'salus-volume: %s is not base64 of 32 bytes\n' "$MOUNTPOINT/keys/$key" >&2; bad=1; }
		else
			bad=1
		fi
	done
	if getent passwd "$KEY_UID" >/dev/null; then
		printf 'salus-volume: the host account %s has uid %s, the key files owner: it could read the keys\n' \
			"$(getent passwd "$KEY_UID" | cut -d: -f1)" "$KEY_UID" >&2
		bad=1
	fi
	return "$bad"
}

mount_volume() {
	local src
	src=$(mounted_from)
	if [[ -n $src ]]; then
		[[ $src == "/dev/mapper/$NAME" ]] || die "$MOUNTPOINT is mounted from $src, not from the volume: unmount it first"
		log "already mounted at $MOUNTPOINT"
		return 0
	fi
	# Underneath, on the unencrypted disk, the mount point is an empty directory only root may enter.
	[[ -d $MOUNTPOINT ]] || install -d -m 700 -o root -g root "$MOUNTPOINT"
	mount -t ext4 -o "$MOUNT_OPTIONS" "/dev/mapper/$NAME" "$MOUNTPOINT"
	log "mounted at $MOUNTPOINT"
}

lay_out() {
	local gid
	gid=$(deploy_gid)
	install -d -m 710 -o root -g "$gid" "$MOUNTPOINT" "$MOUNTPOINT/keys"
	install -d -m 700 -o "$PG_UID" -g "$PG_UID" "$MOUNTPOINT/postgres"
	install -d -m 700 -o root -g root "$MOUNTPOINT/garage" "$MOUNTPOINT/garage/meta" "$MOUNTPOINT/garage/data"
	install -d -m 700 -o "$CLAMAV_UID" -g "$CLAMAV_GID" "$MOUNTPOINT/clamd"
}

# read_key LABEL: the key from the owner, not echoed; checked to be base64 of 32 bytes.
read_key() {
	local value n
	if { exec 3</dev/tty; } 2>/dev/null; then
		read -rsp "Paste $1 from Bitwarden (base64 of 32 bytes; not shown), then Enter: " value <&3 2>/dev/tty || true
		exec 3<&-
		echo >/dev/tty
	else
		IFS= read -r value || true # no terminal: from stdin (the tests)
	fi
	value=${value//[[:space:]]/}
	n=$(printf '%s' "$value" | base64 -d 2>/dev/null | wc -c) || n=0
	[[ $n == 32 ]] || die "$1 is not base64 of 32 bytes: no key file was written"
	printf '%s' "$value"
}

# write_keys [--replace]: both key files, 65532:65532 0400; an existing file with another value is kept unless
# --replace (a different key makes the data encrypted with the old one unreadable).
write_keys() {
	local replace=${1:-} key file tmp value values=()
	is_open && [[ $(mounted_from) == "/dev/mapper/$NAME" ]] || die "the volume is not open and mounted: salus-volume unlock --no-start first"
	lay_out
	for key in "${KEYS[@]}"; do
		values+=("$(read_key "${key^^}")")
	done
	[[ ${values[0]} != "${values[1]}" ]] || die "BINARY_KEK and TOTP_KEY are the same value: paste each from its own Bitwarden field"
	for i in "${!KEYS[@]}"; do
		key=${KEYS[i]}
		value=${values[i]}
		file=$MOUNTPOINT/keys/$key
		if [[ -f $file ]]; then
			if [[ $(tr -d '[:space:]' <"$file") == "$value" ]]; then
				log "$key unchanged"
				chown "$KEY_UID:$KEY_UID" "$file" && chmod 400 "$file"
				continue
			fi
			[[ $replace == --replace ]] ||
				die "$file holds another key: a different key cannot open what the old one encrypted. Pass --replace only if that is meant"
		fi
		tmp=$(mktemp "$MOUNTPOINT/keys/.$key.XXXXXX")
		printf '%s\n' "$value" >"$tmp"
		chown "$KEY_UID:$KEY_UID" "$tmp"
		chmod 400 "$tmp"
		mv -f "$tmp" "$file"
		log "wrote $file"
	done
}

compose_cmd() {
	printf 'docker compose -p salus --env-file %s/.env --env-file %s/release.env -f %s/deployments/docker-compose.prod.yml' \
		"$ROOT" "$ROOT" "$ROOT"
}

start_stack() {
	local no_start=$1 cmd
	cmd=$(compose_cmd)
	if [[ ! -f $ROOT/release.env ]]; then
		log "no release is deployed yet: the first Deploy run starts the stack (docs/deploy/README.md)"
		return 0
	fi
	if ((no_start)); then
		log "not started (--no-start). To start the stack: $cmd up -d --wait"
		return 0
	fi
	# deploy.sh's lock, so the stack never starts beside a deploy or a rollback. Created as deploy's when missing:
	# deploy.sh, run as deploy, opens it for writing.
	[[ -e $ROOT/.deploy.lock ]] || install -m 644 -o "$DEPLOY_GROUP" -g "$DEPLOY_GROUP" /dev/null "$ROOT/.deploy.lock"
	exec 9<"$ROOT/.deploy.lock"
	flock -n 9 ||
		die "a deploy or rollback is running (it holds $ROOT/.deploy.lock) and brings the stack up itself: as deploy, deploy.sh status <sha>"
	log "starting the stack: $cmd up -d --wait"
	# shellcheck disable=SC2086 # the command's words
	$cmd up -d --wait --wait-timeout 900 ||
		die "the stack did not come up healthy: look at '$cmd ps' and '$cmd logs --tail 100 <service>'"
	exec 9<&-
	log "the stack is up"
}

# undo_setup, on any exit of a setup before volume.conf is written: unmounts and closes the volume, so nothing is
# left open, and says how to start again (the device now holds a LUKS header, which setup refuses).
SETUP_DEV=
undo_setup() {
	if [[ -n $(mounted_from) ]]; then umount "$MOUNTPOINT"; fi
	if is_open; then cryptsetup close "$NAME"; fi
	printf 'salus-volume: error: setup did not finish, and the volume is closed again. Fix the cause above, then wipefs -a %s and run setup again\n' \
		"$SETUP_DEV" >&2
}

cmd_setup() {
	local dev=${1:-} uuid
	need_root
	[[ -n $dev ]] || die "usage: salus-volume setup DEVICE (the Hetzner Volume, /dev/disk/by-id/scsi-0HC_Volume_…)"
	[[ -e $CONF ]] && die "$CONF exists: the volume is set up already (salus-volume unlock)"
	[[ -b $dev ]] || die "$dev is not a block device: attach the volume in Hetzner's console first"
	[[ -n $(deploy_gid) ]] || die "there is no $DEPLOY_GROUP group: run bootstrap.sh first"
	[[ -n $(lsblk -nro MOUNTPOINTS "$dev" 2>/dev/null | tr -d '[:space:]') ]] &&
		die "$dev (or a partition on it) is mounted: this is not the new, empty volume"
	if blkid -p "$dev" >/dev/null 2>&1; then
		die "$dev already holds a filesystem, a partition table or a LUKS header: refusing to format it. If it is the new volume (Hetzner formats one when asked), check its id in the console, then 'wipefs -a $dev' and run setup again"
	fi
	if grep -Eq "^[^#]*($(basename "$dev")|HC_Volume)" /etc/fstab 2>/dev/null; then
		die "/etc/fstab mounts a Hetzner Volume (the console's automatic mount): remove that line first"
	fi
	SETUP_DEV=$dev
	trap undo_setup EXIT
	log "formatting $dev with LUKS2: cryptsetup asks to confirm, then for the new passphrase twice. Use the passphrase you stored in Bitwarden; it is never stored on this server"
	cryptsetup luksFormat --type luks2 --verify-passphrase "$dev"
	uuid=$(cryptsetup luksUUID "$dev")
	log "opening the volume: type the passphrase again"
	cryptsetup open --type luks2 "$dev" "$NAME"
	mkfs.ext4 -q -L salus-data "/dev/mapper/$NAME"
	DEVICE=/dev/disk/by-uuid/$uuid
	mount_volume
	lay_out
	write_keys
	check_layout || die "the layout check failed (see above)"
	# Last: until volume.conf exists, a failed setup is undone (undo_setup) and can be run again.
	install -d -m 755 -o root -g root "$(dirname "$CONF")"
	printf '# Written by salus-volume setup. The encrypted volume: no secret here.\nDEVICE=%s\nNAME=%s\nMOUNTPOINT=%s\n' \
		"$DEVICE" "$NAME" "$MOUNTPOINT" >"$CONF.new"
	chmod 644 "$CONF.new"
	mv -f "$CONF.new" "$CONF"
	trap - EXIT
	log "done. After every reboot: salus-volume unlock. If MOUNTPOINT is not /mnt/salus-data, set SALUS_DATA_DIR=$MOUNTPOINT in the env file"
}

cmd_unlock() {
	local no_start=0
	[[ ${1:-} == --no-start ]] && no_start=1
	need_root
	load_conf
	if is_open; then
		log "the volume is open already"
	else
		[[ -b $DEVICE ]] || die "$DEVICE is not there: is the volume attached to this server (Hetzner console)?"
		log "opening the volume: type the passphrase (Bitwarden)"
		cryptsetup open --type luks2 "$DEVICE" "$NAME" || die "cryptsetup could not open the volume (wrong passphrase?)"
	fi
	mount_volume
	# The directories' owners and modes are this script's to set (a surviving volume may carry another server's
	# deploy gid); the key files are only checked.
	lay_out
	check_layout || die "the volume is mounted but not as expected (see above): the stack is not started"
	log "the volume is open, mounted and laid out; both key files are in place"
	start_stack "$no_start"
}

cmd_check() {
	need_root
	load_conf
	check_layout || exit 1
	log "ok: $MOUNTPOINT is the open, encrypted volume, laid out, with both key files"
}

case ${1:-} in
setup) shift && cmd_setup "$@" ;;
unlock) shift && cmd_unlock "$@" ;;
keys)
	shift
	need_root
	load_conf
	write_keys "$@"
	check_layout || die "the layout check failed (see above)"
	;;
check) cmd_check ;;
-h | --help) usage 0 ;;
*) usage 1 ;;
esac
