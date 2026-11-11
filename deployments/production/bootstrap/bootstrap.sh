#!/usr/bin/env bash
# Prepares an Ubuntu 24.04 server for the Salus compose project (docs/adr/0011-single-server-deployment.md):
# the deploy user, sshd hardening, ufw, fail2ban, unattended security upgrades, Docker, the /opt/salus layout and
# the nightly backups (Postgres and Garage's objects) with their alert e-mails, and the tools for the encrypted
# volume (salus-volume, volume.sh). It refuses automatic reboots and reports active swap (ADR 0011: the volume is
# unlocked by hand, the disk holds no personal data). Idempotent: a second run changes nothing. Run as root, from a
# copy of this directory (backup.sh is installed from beside this file):
#
#   bootstrap.sh [--check] [--config FILE]
#
#   --check        report what would change and change nothing
#   --config FILE  the settings (default /etc/salus/bootstrap.conf; see bootstrap.conf.example). A file given here
#                  is installed as /etc/salus/bootstrap.conf (root, 600), which the backup reads.
#
# It never touches other directories under /opt, other compose projects, or an existing Docker's configuration.
#
# Test-only environment flags (deployments/production/bootstrap/test/), default off, never set on a server:
#   SALUS_BOOTSTRAP_TEST_NO_SYSTEMD=1     skip every systemctl call (a container has no systemd)
#   SALUS_BOOTSTRAP_TEST_NO_UFW_ENABLE=1  skip enabling ufw (a container cannot load its netfilter rules); the
#                                         rules and defaults are still written
#   SALUS_BOOTSTRAP_TEST_PROC_SWAPS=FILE  read the active swap from FILE, not /proc/swaps (a container sees the
#                                         host's)
# All are refused on a host that runs systemd.
set -euo pipefail

readonly SALUS_ROOT=/opt/salus
readonly INSTALLED_CONF=/etc/salus/bootstrap.conf
readonly DEPLOY_USER=deploy
readonly SSHD_DROPIN=/etc/ssh/sshd_config.d/00-salus.conf
readonly BACKUP_BIN=/usr/local/sbin/salus-backup
readonly VOLUME_BIN=/usr/local/sbin/salus-volume
readonly SALUS_ETC=/etc/salus
# The public ports of ADR 0011, beside SSH: Caddy (80, 443, 443/udp for HTTP/3), TURN/UDP, LiveKit's media.
readonly SALUS_PORTS=(80/tcp 443/tcp 443/udp 3478/udp 7881/tcp 7882/udp)
# Docker's apt repository signing key (https://docs.docker.com/engine/install/ubuntu/).
readonly DOCKER_KEY_FINGERPRINT=9DC858229FC7DD38854AE2D88D81803C0EBFCD88

CHECK=0
CONFIG=$INSTALLED_CONF
CHANGES=0
FAILURES=0
NO_SYSTEMD=${SALUS_BOOTSTRAP_TEST_NO_SYSTEMD:-0}
NO_UFW_ENABLE=${SALUS_BOOTSTRAP_TEST_NO_UFW_ENABLE:-0}
PROC_SWAPS=${SALUS_BOOTSTRAP_TEST_PROC_SWAPS:-/proc/swaps}
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Settings (bootstrap.conf.example explains each).
DEPLOY_SSH_KEYS=()
DEPLOY_SUDO=no
SSH_PORT=22
UFW_EXTRA_PORTS=
UFW_ACCEPT_BLOCKING=no
UPGRADE_TIME=04:00
UPGRADE_AUTO_REBOOT=no
UPGRADE_REBOOT_TIME=04:30
BACKUP_HOST=
BACKUP_PORT=23
BACKUP_USER=
BACKUP_KNOWN_HOSTS=()
BACKUP_AGE_RECIPIENTS=()
BACKUP_TIME=02:30
BACKUP_RANDOM_DELAY=30min
BACKUP_RETENTION_DAYS=14
BACKUP_KEEP_LOCAL=no
BACKUP_SSH_KEY_FILE=/root/.ssh/salus-backup
BACKUP_KNOWN_HOSTS_FILE=$SALUS_ETC/backup_known_hosts
BACKUP_STAGING_DIR=$SALUS_ROOT/backups
BACKUP_CHECK_TIME=08:00
BACKUP_MAX_AGE_HOURS=26
ALERT_EMAIL=

# Flags set by the steps for the service actions at the end.
UNITS_CHANGED=0

log() { printf '%s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}
fail() {
	printf 'error: %s\n' "$*" >&2
	FAILURES=$((FAILURES + 1))
}

# change DESCRIPTION COMMAND...: counts the change; runs it unless --check.
change() {
	local desc=$1
	shift
	CHANGES=$((CHANGES + 1))
	if ((CHECK)); then
		log "would: $desc"
		return 0
	fi
	log "change: $desc"
	"$@"
}

# put_file PATH MODE OWNER:GROUP CONTENT: writes the file only when its content, mode or owner differ.
# Returns 0 when it changed (or would change), 1 when it was already in place.
put_file() {
	local path=$1 mode=$2 owner=$3 content=$4
	if [[ -f $path ]] && [[ $(cat "$path") == "$content" ]] &&
		[[ $(stat -c '%a %U:%G' "$path") == "$mode $owner" ]]; then
		return 1
	fi
	change "write $path" write_file "$path" "$mode" "$owner" "$content"
}

write_file() {
	local path=$1 mode=$2 owner=$3 content=$4 tmp
	mkdir -p "$(dirname "$path")"
	tmp=$(mktemp "$path.XXXXXX")
	printf '%s\n' "$content" >"$tmp"
	chown "$owner" "$tmp"
	chmod "$mode" "$tmp"
	mv -f "$tmp" "$path"
}

# put_dir PATH MODE OWNER:GROUP
put_dir() {
	local path=$1 mode=$2 owner=$3
	if [[ -d $path ]] && [[ $(stat -c '%a %U:%G' "$path") == "$mode $owner" ]]; then
		return 0
	fi
	change "directory $path ($mode $owner)" install -d -m "$mode" -o "${owner%%:*}" -g "${owner##*:}" "$path"
}

systemctl_() {
	if ((NO_SYSTEMD)); then
		log "skipped (test flag): systemctl $*"
		return 0
	fi
	systemctl "$@"
}

usage() {
	sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed '$d; s/^# \{0,1\}//'
	exit "${1:-0}"
}

parse_args() {
	while (($#)); do
		case $1 in
		--check) CHECK=1 ;;
		--config)
			[[ $# -ge 2 ]] || die "--config needs a file"
			CONFIG=$2
			shift
			;;
		-h | --help) usage 0 ;;
		*)
			warn "unknown argument: $1"
			usage 1
			;;
		esac
		shift
	done
}

# load_config FILE: KEY=value lines, '#' comments, no inline comments; the file is parsed, never sourced.
load_config() {
	local file=$1 line key value n=0
	[[ -r $file ]] || die "cannot read the settings file $file (see bootstrap.conf.example)"
	while IFS= read -r line || [[ -n $line ]]; do
		n=$((n + 1))
		line=${line%$'\r'}
		[[ $line =~ ^[[:space:]]*(#|$) ]] && continue
		[[ $line =~ ^([A-Z][A-Z0-9_]*)=(.*)$ ]] || die "$file line $n: expected KEY=value"
		key=${BASH_REMATCH[1]}
		value=${BASH_REMATCH[2]}
		if [[ $value =~ ^\"(.*)\"$ ]] || [[ $value =~ ^\'(.*)\'$ ]]; then
			value=${BASH_REMATCH[1]}
		fi
		case $key in
		DEPLOY_SSH_KEY) [[ -n $value ]] && DEPLOY_SSH_KEYS+=("$value") ;;
		BACKUP_KNOWN_HOST) [[ -n $value ]] && BACKUP_KNOWN_HOSTS+=("$value") ;;
		BACKUP_AGE_RECIPIENT) [[ -n $value ]] && BACKUP_AGE_RECIPIENTS+=("$value") ;;
		DEPLOY_SUDO | SSH_PORT | UFW_EXTRA_PORTS | UFW_ACCEPT_BLOCKING | UPGRADE_TIME | UPGRADE_AUTO_REBOOT | UPGRADE_REBOOT_TIME | \
			BACKUP_HOST | BACKUP_PORT | BACKUP_USER | BACKUP_TIME | BACKUP_RANDOM_DELAY | \
			BACKUP_RETENTION_DAYS | BACKUP_KEEP_LOCAL | BACKUP_SSH_KEY_FILE | BACKUP_KNOWN_HOSTS_FILE | \
			BACKUP_STAGING_DIR | BACKUP_CHECK_TIME | BACKUP_MAX_AGE_HOURS | ALERT_EMAIL)
			printf -v "$key" '%s' "$value"
			;;
		BACKUP_PATH | BACKUP_OBJECTS_PATH | BACKUP_COMPOSE_PROJECT | BACKUP_PG_SERVICE | BACKUP_OBJECTS_SERVICE | \
			BACKUP_ENV_FILE | BACKUP_WAIT_SECONDS) ;; # read by backup.sh only
		*) die "$file line $n: unknown setting $key" ;;
		esac
	done <"$file"
}

valid_port_spec() { [[ $1 =~ ^[0-9]{1,5}/(tcp|udp)$ ]] && ((${1%/*} >= 1 && ${1%/*} <= 65535)); }
valid_time() { [[ $1 =~ ^([01][0-9]|2[0-3]):[0-5][0-9]$ ]]; }
valid_yes_no() { [[ $1 == yes || $1 == no ]]; }

validate_config() {
	local key port
	((${#DEPLOY_SSH_KEYS[@]} > 0)) || die "no DEPLOY_SSH_KEY is configured: refusing to continue, it would lock you out"
	for key in "${DEPLOY_SSH_KEYS[@]}"; do
		[[ $key =~ ^(ssh-[a-z0-9-]+|ecdsa-sha2-[a-z0-9-]+|sk-[a-z0-9@.-]+)\ [A-Za-z0-9+/]+=*(\ .*)?$ ]] ||
			die "a DEPLOY_SSH_KEY is not a public key line (type, key, optional comment)"
		if command -v ssh-keygen >/dev/null 2>&1; then
			ssh-keygen -l -f /dev/stdin <<<"$key" >/dev/null 2>&1 || die "a DEPLOY_SSH_KEY is not a valid OpenSSH public key"
		fi
	done
	valid_port_spec "$SSH_PORT/tcp" || die "SSH_PORT must be a port number"
	for port in $UFW_EXTRA_PORTS; do
		valid_port_spec "$port" || die "UFW_EXTRA_PORTS: '$port' is not PORT/tcp or PORT/udp"
	done
	valid_yes_no "$DEPLOY_SUDO" || die "DEPLOY_SUDO must be yes or no"
	valid_yes_no "$UFW_ACCEPT_BLOCKING" || die "UFW_ACCEPT_BLOCKING must be yes or no"
	valid_yes_no "$UPGRADE_AUTO_REBOOT" || die "UPGRADE_AUTO_REBOOT must be yes or no"
	[[ $UPGRADE_AUTO_REBOOT == no ]] ||
		die "UPGRADE_AUTO_REBOOT=yes is refused: after a reboot the encrypted volume waits for the owner's passphrase and the stack stays down (ADR 0011); reboot by hand and unlock in one sitting"
	valid_yes_no "$BACKUP_KEEP_LOCAL" || die "BACKUP_KEEP_LOCAL must be yes or no"
	valid_time "$UPGRADE_TIME" || die "UPGRADE_TIME must be HH:MM"
	valid_time "$UPGRADE_REBOOT_TIME" || die "UPGRADE_REBOOT_TIME must be HH:MM"
	valid_time "$BACKUP_TIME" || die "BACKUP_TIME must be HH:MM"
	valid_time "$BACKUP_CHECK_TIME" || die "BACKUP_CHECK_TIME must be HH:MM"
	if ! [[ $BACKUP_MAX_AGE_HOURS =~ ^[0-9]+$ ]] || ((BACKUP_MAX_AGE_HOURS < 1)); then
		die "BACKUP_MAX_AGE_HOURS must be a whole number of hours, at least 1"
	fi
	# A bare address: it goes into a mail header and the SMTP envelope.
	local address='^[^[:space:]<>@"]+@[^[:space:]<>@"]+$'
	[[ -z $ALERT_EMAIL || $ALERT_EMAIL =~ $address ]] ||
		die "ALERT_EMAIL must be one bare e-mail address"
	[[ $BACKUP_PORT =~ ^[0-9]+$ ]] || die "BACKUP_PORT must be a port number"
	if ! [[ $BACKUP_RETENTION_DAYS =~ ^[0-9]+$ ]] || ((BACKUP_RETENTION_DAYS < 1)); then
		die "BACKUP_RETENTION_DAYS must be a whole number of days, at least 1"
	fi
	[[ $BACKUP_RANDOM_DELAY =~ ^[0-9]+(s|min|h)?$ ]] || die "BACKUP_RANDOM_DELAY must be like 30min"
	[[ -f $SCRIPT_DIR/backup.sh && -f $SCRIPT_DIR/volume.sh ]] ||
		die "backup.sh or volume.sh is not beside bootstrap.sh: copy the whole bootstrap directory to the server"
}

check_os() {
	local id version
	[[ $EUID -eq 0 ]] || die "run as root"
	# shellcheck source=/dev/null
	id=$(. /etc/os-release && echo "$ID")
	# shellcheck source=/dev/null
	version=$(. /etc/os-release && echo "$VERSION_ID")
	[[ $id == ubuntu && $version == 24.04 ]] || die "this script supports Ubuntu 24.04 only (found $id $version)"
	if { ((NO_SYSTEMD || NO_UFW_ENABLE)) || [[ $PROC_SWAPS != /proc/swaps ]]; } && [[ -d /run/systemd/system ]]; then
		die "the SALUS_BOOTSTRAP_TEST_* flags are for the container test only; this host runs systemd"
	fi
}

# --- packages ----------------------------------------------------------------------------------------------

APT_UPDATED=0
apt_install() {
	local missing=() pkg
	for pkg in "$@"; do
		dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q 'install ok installed' || missing+=("$pkg")
	done
	((${#missing[@]})) || return 0
	change "install packages: ${missing[*]}" apt_get_install "${missing[@]}"
}

apt_get_install() {
	if ((!APT_UPDATED)); then
		apt-get update -qq
		APT_UPDATED=1
	fi
	DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends "$@" >/dev/null
}

step_packages() {
	# iproute2 for ss, openssh-client for ssh-keygen and sftp (backup), age for the backup's encryption,
	# python3-systemd for fail2ban's journal backend, cryptsetup for the encrypted volume (salus-volume).
	apt_install ca-certificates curl gnupg iproute2 openssh-server openssh-client ufw fail2ban python3-systemd \
		unattended-upgrades age cryptsetup
}

# --- deploy user ---------------------------------------------------------------------------------------------

step_deploy_user() {
	local home keys
	if ! id -u "$DEPLOY_USER" >/dev/null 2>&1; then
		change "create user $DEPLOY_USER (no password, SSH key only)" \
			useradd --create-home --user-group --shell /bin/bash "$DEPLOY_USER"
	fi
	if id -u "$DEPLOY_USER" >/dev/null 2>&1; then
		home=$(getent passwd "$DEPLOY_USER" | cut -d: -f6)
		# '*' is no usable password; unlike a lock ('!') it does not block key logins under PAM.
		if [[ $(getent shadow "$DEPLOY_USER" | cut -d: -f2) != '*' ]]; then
			change "remove the password of $DEPLOY_USER" usermod -p '*' "$DEPLOY_USER"
		fi
	else
		home=/home/$DEPLOY_USER
	fi
	put_dir "$home/.ssh" 700 "$DEPLOY_USER:$DEPLOY_USER"
	keys=$(printf '%s\n' "${DEPLOY_SSH_KEYS[@]}")
	put_file "$home/.ssh/authorized_keys" 600 "$DEPLOY_USER:$DEPLOY_USER" "$keys" || true
	if [[ $DEPLOY_SUDO == yes ]]; then
		put_file /etc/sudoers.d/salus-deploy 440 root:root "$DEPLOY_USER ALL=(ALL) NOPASSWD:ALL" || true
	elif [[ -e /etc/sudoers.d/salus-deploy ]]; then
		change "remove /etc/sudoers.d/salus-deploy (DEPLOY_SUDO=no)" rm -f /etc/sudoers.d/salus-deploy
	fi
	if [[ -f /etc/sudoers.d/salus-deploy ]] && ! ((CHECK)); then
		visudo -cqf /etc/sudoers.d/salus-deploy || die "/etc/sudoers.d/salus-deploy does not validate"
	fi
}

deploy_key_in_place() {
	local home file key
	((CHECK)) && return 0
	home=$(getent passwd "$DEPLOY_USER" | cut -d: -f6)
	file=$home/.ssh/authorized_keys
	[[ -s $file ]] || return 1
	[[ $(stat -c '%U %a' "$file") == "$DEPLOY_USER 600" ]] || return 1
	for key in "${DEPLOY_SSH_KEYS[@]}"; do
		grep -qxF -- "$key" "$file" || return 1
	done
}

# --- sshd ---------------------------------------------------------------------------------------------------

current_ssh_port() { sshd -T 2>/dev/null | awk '$1 == "port" {print $2; exit}'; }

step_sshd() {
	local content previous=""
	deploy_key_in_place || die "the deploy user's authorized_keys is not in place: not touching sshd"
	content="# Managed by Salus bootstrap.sh; changes are overwritten. Sorts first, so these values win.
Port $SSH_PORT
PermitRootLogin no
PasswordAuthentication no
KbdInteractiveAuthentication no
PubkeyAuthentication yes"
	if [[ -f $SSHD_DROPIN ]] && [[ $(cat "$SSHD_DROPIN") == "$content" ]] &&
		[[ $(stat -c '%a %U:%G' "$SSHD_DROPIN") == "644 root:root" ]]; then
		((CHECK)) || sshd_effective_ok || die "sshd's effective configuration would lock the deploy user out (see above)"
		return 0
	fi
	if ((CHECK)); then
		put_file "$SSHD_DROPIN" 644 root:root "$content" || true
		return 0
	fi
	mkdir -p /run/sshd
	[[ -f $SSHD_DROPIN ]] && previous=$(cat "$SSHD_DROPIN")
	put_file "$SSHD_DROPIN" 644 root:root "$content" || true
	# The new file is kept only when sshd accepts it and the deploy user can still log in with it.
	if ! sshd -t || ! sshd_effective_ok; then
		if [[ -n $previous ]]; then write_file "$SSHD_DROPIN" 644 root:root "$previous"; else rm -f "$SSHD_DROPIN"; fi
		die "the new sshd configuration was rejected (see above); the previous one is back"
	fi
}

# in_list WORD PATTERN...: WORD (or '*') among the patterns, ignoring an @host part.
in_list() {
	local want=$1 tok
	shift
	for tok in "$@"; do
		tok=${tok%%@*}
		[[ $tok == "*" || $tok == "$want" ]] && return 0
	done
	return 1
}

# The effective configuration for the deploy user, after every drop-in (cloud-init's 50-cloud-init.conf may set
# PasswordAuthentication yes; ours sorts first and wins). Prints each problem; returns 1 when there is one.
sshd_effective_ok() {
	local eff bad=0 groups=() list=() g found key values
	eff=$(sshd -T -C "user=$DEPLOY_USER,host=localhost,addr=127.0.0.1") || return 1
	read -ra groups <<<"$(id -nG "$DEPLOY_USER") docker"
	grep -qx 'permitrootlogin no' <<<"$eff" || { warn "sshd: PermitRootLogin is not no"; bad=1; }
	grep -qx 'passwordauthentication no' <<<"$eff" || { warn "sshd: PasswordAuthentication is not no"; bad=1; }
	grep -qx 'kbdinteractiveauthentication no' <<<"$eff" || { warn "sshd: KbdInteractiveAuthentication is not no"; bad=1; }
	grep -qx 'pubkeyauthentication yes' <<<"$eff" || { warn "sshd: PubkeyAuthentication is not yes"; bad=1; }
	while read -r key values; do
		read -ra list <<<"$values"
		case $key in
		authorizedkeysfile)
			in_list .ssh/authorized_keys "${list[@]}" || in_list %h/.ssh/authorized_keys "${list[@]}" ||
				in_list "/home/%u/.ssh/authorized_keys" "${list[@]}" ||
				{ warn "sshd: AuthorizedKeysFile ($values) does not read ~/.ssh/authorized_keys"; bad=1; }
			;;
		authenticationmethods)
			in_list any "${list[@]}" || in_list publickey "${list[@]}" ||
				{ warn "sshd: AuthenticationMethods ($values) asks for more than a key"; bad=1; }
			;;
		allowusers) in_list "$DEPLOY_USER" "${list[@]}" || { warn "sshd: AllowUsers does not include $DEPLOY_USER"; bad=1; } ;;
		denyusers) in_list "$DEPLOY_USER" "${list[@]}" && { warn "sshd: DenyUsers includes $DEPLOY_USER"; bad=1; } ;;
		allowgroups)
			found=0
			for g in "${groups[@]}"; do in_list "$g" "${list[@]}" && found=1; done
			((found)) || { warn "sshd: AllowGroups has none of $DEPLOY_USER's groups"; bad=1; }
			;;
		denygroups)
			for g in "${groups[@]}"; do
				in_list "$g" "${list[@]}" && { warn "sshd: DenyGroups includes $g, a group of $DEPLOY_USER"; bad=1; }
			done
			;;
		esac
	done < <(grep -E '^(authorizedkeysfile|authenticationmethods|allowusers|denyusers|allowgroups|denygroups) ' <<<"$eff")
	return "$bad"
}

# Reloads sshd on every run, not only when the file changed: an earlier run may have stopped before reloading.
# When nothing listens on SSH_PORT yet the port changed: Ubuntu 24.04 starts sshd from ssh.socket, whose ports a
# generator takes from sshd_config at daemon-reload, so the socket and then the service are restarted.
# Established sessions are kept either way.
apply_sshd() {
	if ss -Hltn "sport = :$SSH_PORT" | grep -q .; then
		systemctl try-reload-or-restart ssh.service
		return 0
	fi
	systemctl daemon-reload
	if systemctl is-enabled --quiet ssh.socket 2>/dev/null; then
		systemctl restart ssh.socket
	fi
	systemctl restart ssh.service
	if ss -Hltn "sport = :$SSH_PORT" | grep -q .; then
		log "sshd listens on port $SSH_PORT. Keep this session open and test 'ssh -p $SSH_PORT $DEPLOY_USER@<server>' first."
	else
		warn "sshd does not listen on port $SSH_PORT yet: keep this session open and check 'systemctl status ssh.socket ssh.service'"
	fi
}

# --- ufw ----------------------------------------------------------------------------------------------------

# shellcheck disable=SC2086 # UFW_EXTRA_PORTS is a space-separated list
allowed_ports() { printf '%s\n' "$SSH_PORT/tcp" "${SALUS_PORTS[@]}" $UFW_EXTRA_PORTS; }

is_allowed() { allowed_ports | grep -qxF "$1"; }

# Sockets listening on a non-loopback address: "proto port process".
public_listeners() {
	ss -Hlntup 2>/dev/null | awk '{
		proto = $1; local = $5; proc = $7
		port = local; sub(/.*:/, "", port)
		addr = local; sub(/:[^:]*$/, "", addr)
		if (addr ~ /^127\./ || addr == "[::1]" || addr ~ /%lo$/) next
		name = "?"; if (match(proc, /"[^"]+"/)) name = substr(proc, RSTART + 1, RLENGTH - 2)
		print proto, port, name
	}' | sort -u
}

# blocked_listeners RULES: the public listeners ("port/proto (process)") no allowed port or existing ufw rule covers.
blocked_listeners() {
	local rules=$1 proto port name
	while read -r proto port name; do
		[[ -n $port ]] || continue
		# DHCP client replies are let in by ufw's own rules.
		[[ $proto == udp && ($port == 68 || $port == 546) ]] && continue
		is_allowed "$port/$proto" && continue
		grep -qE "^ufw allow $port(/$proto)?( |$)" <<<"$rules" && continue
		# Published by Docker, which bypasses ufw; reported by check_docker_published.
		[[ $name == docker-proxy ]] && continue
		printf '%s/%s (%s)\n' "$port" "$proto" "$name"
	done < <(public_listeners)
}

# Allow rules first, then the default policy, then enabling, so SSH is never blocked in between. Refuses when
# enabling ufw (or switching its default to deny) would cut off a service listening on a public address, unless
# UFW_ACCEPT_BLOCKING=yes.
step_ufw() {
	local active=0 deny=0 rules spec blocked item
	ufw status 2>/dev/null | grep -q '^Status: active' && active=1
	grep -qx 'DEFAULT_INPUT_POLICY="DROP"' /etc/default/ufw 2>/dev/null && deny=1
	rules=$(ufw show added 2>/dev/null || true)
	if ((!active || !deny)); then
		blocked=$(blocked_listeners "$rules")
		if [[ -n $blocked ]]; then
			while IFS= read -r item; do
				if [[ $UFW_ACCEPT_BLOCKING == yes ]]; then
					warn "$item listens on a public address; ufw will block it (UFW_ACCEPT_BLOCKING=yes)"
				else
					printf 'error: %s listens on a public address and ufw would block it\n' "$item" >&2
				fi
			done <<<"$blocked"
			if [[ $UFW_ACCEPT_BLOCKING != yes ]]; then
				((CHECK)) || die "refusing to enable ufw: add those ports to UFW_EXTRA_PORTS, or set UFW_ACCEPT_BLOCKING=yes to block them"
				fail "ufw would block the ports above: add them to UFW_EXTRA_PORTS, or set UFW_ACCEPT_BLOCKING=yes"
			fi
		fi
	fi
	while read -r spec; do
		grep -qE "^ufw allow $spec( |$)" <<<"$rules" ||
			change "ufw: allow $spec" ufw allow "$spec" comment salus
	done < <(allowed_ports)
	grep -qx 'DEFAULT_OUTPUT_POLICY="ACCEPT"' /etc/default/ufw 2>/dev/null ||
		change "ufw: default allow outgoing" ufw default allow outgoing
	((deny)) || change "ufw: default deny incoming" ufw default deny incoming
	if ((!active)); then
		if ((NO_UFW_ENABLE)); then
			log "skipped (test flag): ufw --force enable"
		else
			change "ufw: enable" ufw --force enable
		fi
	fi
	if grep -qE '^ufw allow 22/tcp( |$)' <<<"$rules" && [[ $SSH_PORT != 22 ]]; then
		log "note: ufw still allows 22/tcp from before; remove it with 'ufw delete allow 22/tcp' once port $SSH_PORT works"
	fi
}

# --- fail2ban and unattended upgrades -----------------------------------------------------------------------

step_fail2ban() {
	put_file /etc/fail2ban/jail.d/salus-sshd.conf 644 root:root "# Managed by Salus bootstrap.sh.
[sshd]
enabled = true
port = $SSH_PORT
backend = systemd
maxretry = 5
findtime = 10m
bantime = 1h" || true
}

# The pockets unattended-upgrades installs from: Ubuntu's own default list, whose -updates line is commented out:
# the release pocket (unchanged since the release; a security fix's new dependencies come from it) and the
# security pockets.
# shellcheck disable=SC2016 # apt expands these, not the shell
readonly UPGRADE_ORIGINS=('${distro_id}:${distro_codename}' '${distro_id}:${distro_codename}-security'
	'${distro_id}ESMApps:${distro_codename}-apps-security' '${distro_id}ESM:${distro_codename}-infra-security')
readonly UPGRADES_CONF=/etc/apt/apt.conf.d/52salus-unattended-upgrades

# The origins the rest of the server's apt configuration allows ("kind origin"): apt-config over a copy of the
# parts directory without our file (APT_CONFIG is read before Dir::Etc::Parts).
other_upgrade_origins() {
	local tmp
	command -v apt-config >/dev/null 2>&1 || return 0
	tmp=$(mktemp -d)
	find /etc/apt/apt.conf.d -maxdepth 1 -type f ! -name "${UPGRADES_CONF##*/}" -exec cp -t "$tmp" {} +
	printf 'Dir::Etc::Parts "%s";\n' "$tmp" >"$tmp/.apt.conf"
	APT_CONFIG=$tmp/.apt.conf apt-config dump 2>/dev/null |
		sed -nE 's/^Unattended-Upgrade::(Allowed-Origins|Origins-Pattern):: "(.*)";$/\1 \2/p' || true
	rm -r "$tmp"
}

step_upgrades() {
	local reboot=false content kind origin o origins=""
	[[ $UPGRADE_AUTO_REBOOT == yes ]] && reboot=true
	for o in "${UPGRADE_ORIGINS[@]}"; do origins+=$'\t'"\"$o\";"$'\n'; done
	# '#clear' empties the lists the package's 50unattended-upgrades (and any other file) fills.
	content="// Managed by Salus bootstrap.sh.
APT::Periodic::Update-Package-Lists \"1\";
APT::Periodic::Unattended-Upgrade \"1\";
#clear Unattended-Upgrade::Allowed-Origins;
#clear Unattended-Upgrade::Origins-Pattern;
Unattended-Upgrade::Allowed-Origins {
${origins}};
Unattended-Upgrade::Automatic-Reboot \"$reboot\";
Unattended-Upgrade::Automatic-Reboot-Time \"$UPGRADE_REBOOT_TIME\";"
	if ! [[ -f $UPGRADES_CONF && $(cat "$UPGRADES_CONF") == "$content" ]]; then
		while read -r kind origin; do
			[[ -n $origin ]] || continue
			if [[ $kind == Allowed-Origins ]] && printf '%s\n' "${UPGRADE_ORIGINS[@]}" | grep -qxF -- "$origin"; then
				continue
			fi
			warn "unattended-upgrades: the server's own $kind entry '$origin' is replaced (security pockets only)"
		done < <(other_upgrade_origins)
	fi
	put_file "$UPGRADES_CONF" 644 root:root "$content" || true
	put_file /etc/systemd/system/apt-daily-upgrade.timer.d/salus.conf 644 root:root "# Managed by Salus bootstrap.sh.
[Timer]
OnCalendar=
OnCalendar=*-*-* $UPGRADE_TIME
RandomizedDelaySec=15min" && UNITS_CHANGED=1
	return 0
}

# --- Docker -------------------------------------------------------------------------------------------------

install_docker() {
	local key=/etc/apt/keyrings/docker.asc tmp
	install -m 0755 -d /etc/apt/keyrings
	tmp=$(mktemp)
	curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o "$tmp"
	gpg --show-keys --with-colons "$tmp" | grep -q "^fpr:::::::::$DOCKER_KEY_FINGERPRINT:" || {
		rm -f "$tmp"
		die "Docker's apt key does not have the expected fingerprint"
	}
	install -m 0644 "$tmp" "$key"
	rm -f "$tmp"
	# shellcheck source=/dev/null
	printf 'Types: deb\nURIs: https://download.docker.com/linux/ubuntu\nSuites: %s\nComponents: stable\nArchitectures: %s\nSigned-By: %s\n' \
		"$(. /etc/os-release && echo "$VERSION_CODENAME")" "$(dpkg --print-architecture)" "$key" \
		>/etc/apt/sources.list.d/docker.sources
	apt-get update -qq
	DEBIAN_FRONTEND=noninteractive apt-get install -y -qq docker-ce docker-ce-cli containerd.io \
		docker-buildx-plugin docker-compose-plugin >/dev/null
	systemctl_ enable --now docker.service
}

step_docker() {
	if ! command -v docker >/dev/null 2>&1; then
		change "install Docker Engine and the compose plugin from Docker's apt repository" install_docker
	elif ! docker compose version >/dev/null 2>&1; then
		# An existing Docker is never reconfigured: its owner installs the plugin.
		fail "Docker is installed but the compose plugin is not; install it (docker-compose-plugin) and run this again"
	else
		log "ok: Docker is installed; its configuration is left alone"
	fi
	if getent group docker >/dev/null && id -u "$DEPLOY_USER" >/dev/null 2>&1; then
		id -nG "$DEPLOY_USER" | tr ' ' '\n' | grep -qx docker ||
			change "add $DEPLOY_USER to the docker group" usermod -aG docker "$DEPLOY_USER"
	elif ((CHECK)); then
		change "add $DEPLOY_USER to the docker group" true
	fi
}

# Docker-published ports bypass ufw; the compose files are the real boundary. Reports, never changes.
check_docker_published() {
	local ids name ip port key
	command -v docker >/dev/null 2>&1 || return 0
	if ! ids=$(docker ps -q 2>/dev/null); then
		warn "the Docker daemon is not reachable: cannot check which ports containers publish"
		return 0
	fi
	[[ -n $ids ]] || return 0
	while IFS='|' read -r name ip port key; do
		[[ -n $port ]] || continue
		[[ $ip == 127.* || $ip == ::1 ]] && continue
		is_allowed "$port/${key##*/}" && continue
		warn "container ${name#/} publishes $port/${key##*/} on ${ip:-all addresses}, outside the allowed list; Docker bypasses ufw (not changed)"
	done < <(
		# shellcheck disable=SC2086 # one argument per container id
		docker inspect --format \
			'{{$n := .Name}}{{range $p, $b := .NetworkSettings.Ports}}{{range $b}}{{$n}}|{{.HostIp}}|{{.HostPort}}|{{$p}}{{"\n"}}{{end}}{{end}}' \
			$ids
	)
}

# --- /opt/salus and the backup ------------------------------------------------------------------------------

step_layout() {
	local owner=root:root
	id -u "$DEPLOY_USER" >/dev/null 2>&1 && owner=$DEPLOY_USER:$DEPLOY_USER
	put_dir "$SALUS_ROOT" 750 "$owner"
	put_dir "$SALUS_ROOT/deployments" 750 "$owner"
	# Only the root backup job writes here.
	put_dir "$BACKUP_STAGING_DIR" 700 root:root
	put_dir "$SALUS_ETC" 755 root:root
	# Root reads the settings (the backup runs as root), so they live outside the deploy-owned /opt/salus.
	if [[ $(realpath -m "$CONFIG") != "$INSTALLED_CONF" ]]; then
		put_file "$INSTALLED_CONF" 600 root:root "$(cat "$CONFIG")" || true
	elif [[ $(stat -c '%a %U:%G' "$INSTALLED_CONF") != "600 root:root" ]]; then
		change "permissions of $INSTALLED_CONF (600 root:root)" chmod 600 "$INSTALLED_CONF"
		((CHECK)) || chown root:root "$INSTALLED_CONF"
	fi
}

backup_configured() {
	[[ -n $BACKUP_HOST && -n $BACKUP_USER ]] && ((${#BACKUP_AGE_RECIPIENTS[@]} > 0 && ${#BACKUP_KNOWN_HOSTS[@]} > 0))
}

step_backup() {
	put_file "$BACKUP_BIN" 755 root:root "$(cat "$SCRIPT_DIR/backup.sh")" || true
	if [[ ! -f $BACKUP_SSH_KEY_FILE ]]; then
		change "generate the backup's SSH key $BACKUP_SSH_KEY_FILE" generate_backup_key
	fi
	if ((${#BACKUP_KNOWN_HOSTS[@]})); then
		put_file "$BACKUP_KNOWN_HOSTS_FILE" 644 root:root "$(printf '%s\n' "${BACKUP_KNOWN_HOSTS[@]}")" || true
	fi
	# The dump, then the objects (Wants= starts them together, After= orders the objects behind the dump, whatever
	# its result), so every row of the dump finds its object in the archive (docs/deploy/backup-restore.md).
	put_file /etc/systemd/system/salus-backup.service 644 root:root "$(backup_unit "Salus nightly Postgres backup to the Storage Box" \
		postgres "Wants=salus-backup-objects.service")" && UNITS_CHANGED=1
	put_file /etc/systemd/system/salus-backup-objects.service 644 root:root "$(backup_unit \
		"Salus nightly backup of Garage's objects to the Storage Box" objects "After=salus-backup.service")" && UNITS_CHANGED=1
	# A failure is logged at err priority and mailed to ALERT_EMAIL (docs/deploy/backup-restore.md "Alerting").
	put_file /etc/systemd/system/salus-backup-failed@.service 644 root:root "# Managed by Salus bootstrap.sh.
[Unit]
Description=Report the failed Salus backup %i

[Service]
Type=oneshot
SyslogIdentifier=salus-backup
ExecStart=/usr/bin/systemd-cat -t salus-backup -p err /bin/echo \"salus-backup FAILED: see journalctl -u %i\"
ExecStart=$BACKUP_BIN alert %i
TimeoutStartSec=5min
PrivateTmp=yes
NoNewPrivileges=yes
ProtectSystem=full
ProtectHome=read-only" && UNITS_CHANGED=1
	if [[ -e /etc/systemd/system/salus-backup-failed.service ]]; then
		change "remove salus-backup-failed.service (replaced by salus-backup-failed@.service)" \
			rm -f /etc/systemd/system/salus-backup-failed.service
		UNITS_CHANGED=1
	fi
	# The freshness check: mails ALERT_EMAIL when a backup is missing on the Storage Box, also when no unit failed
	# (a timer that never ran, a server that was off). It succeeds once it has sent that mail; it fails, and the
	# failure hook tries to mail, only when it could not run or not send.
	put_file /etc/systemd/system/salus-backup-check.service 644 root:root "# Managed by Salus bootstrap.sh.
[Unit]
Description=Check that the Salus backups on the Storage Box are recent
After=network-online.target
Wants=network-online.target
OnFailure=salus-backup-failed@%n.service

[Service]
Type=oneshot
SyslogIdentifier=salus-backup
ExecStart=$BACKUP_BIN check
TimeoutStartSec=10min
PrivateTmp=yes
NoNewPrivileges=yes
ProtectSystem=full
ProtectHome=read-only" && UNITS_CHANGED=1
	put_file /etc/systemd/system/salus-backup-check.timer 644 root:root "# Managed by Salus bootstrap.sh.
[Unit]
Description=Daily check of the Salus backups

[Timer]
OnCalendar=*-*-* $BACKUP_CHECK_TIME
RandomizedDelaySec=10min
Persistent=true

[Install]
WantedBy=timers.target" && UNITS_CHANGED=1
	put_file /etc/systemd/system/salus-backup.timer 644 root:root "# Managed by Salus bootstrap.sh.
[Unit]
Description=Salus nightly backups (Postgres, then Garage's objects)

[Timer]
OnCalendar=*-*-* $BACKUP_TIME
RandomizedDelaySec=$BACKUP_RANDOM_DELAY
Persistent=true

[Install]
WantedBy=timers.target" && UNITS_CHANGED=1
	return 0
}

# backup_unit DESCRIPTION KIND DEPENDENCY: a backup's service unit.
backup_unit() {
	printf '%s\n' "# Managed by Salus bootstrap.sh.
[Unit]
Description=$1
After=docker.service network-online.target
Wants=network-online.target
$3
OnFailure=salus-backup-failed@%n.service

[Service]
Type=oneshot
SyslogIdentifier=salus-backup
ExecStart=$BACKUP_BIN $2
TimeoutStartSec=2h
Nice=10
IOSchedulingClass=idle
PrivateTmp=yes
NoNewPrivileges=yes
ProtectSystem=full
ProtectHome=read-only
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectKernelLogs=yes
ProtectControlGroups=yes
ProtectClock=yes"
}

# --- the encrypted volume and swap (ADR 0011) ----------------------------------------------------------------

# Swap would page the stack's memory, personal data included, to the server's unencrypted disk. Reported, never
# changed: turning swap off is the owner's call on a server that may run other software.
check_swap() {
	local active
	active=$(awk 'NR > 1 {printf "%s ", $1}' "$PROC_SWAPS" 2>/dev/null || true)
	[[ -z $active ]] ||
		fail "swap is active (${active% }): memory, personal data included, could be written to the unencrypted disk. Turn it off (swapoff -a), remove it from /etc/fstab, and run this again"
	if awk '$1 !~ /^#/ && $3 == "swap" {found = 1} END {exit !found}' /etc/fstab 2>/dev/null; then
		fail "/etc/fstab has a swap entry, which comes back at the next boot: remove it"
	fi
}

step_volume() {
	put_file "$VOLUME_BIN" 755 root:root "$(cat "$SCRIPT_DIR/volume.sh")" || true
	if [[ ! -f /etc/salus/volume.conf ]]; then
		warn "the encrypted volume is not set up: attach the Hetzner Volume and run 'salus-volume setup DEVICE' (docs/deploy/README.md); until then deploys are refused"
	fi
}

generate_backup_key() {
	install -d -m 700 "$(dirname "$BACKUP_SSH_KEY_FILE")"
	ssh-keygen -q -t ed25519 -N '' -C "salus-backup@$(hostname)" -f "$BACKUP_SSH_KEY_FILE"
}

apply_services() {
	((CHECK)) && return 0
	((UNITS_CHANGED)) && systemctl_ daemon-reload
	if ((NO_SYSTEMD)); then
		log "skipped (test flag): reloading sshd"
	else
		apply_sshd
	fi
	# Every run, as for sshd: an earlier run may have written the jail and stopped before this.
	systemctl_ enable fail2ban.service
	systemctl_ reload-or-restart fail2ban.service
	if backup_configured; then
		if [[ $NO_SYSTEMD == 1 ]] || ! systemctl is-enabled --quiet salus-backup.timer 2>/dev/null; then
			systemctl_ enable --now salus-backup.timer
		fi
		if [[ $NO_SYSTEMD == 1 ]] || ! systemctl is-enabled --quiet salus-backup-check.timer 2>/dev/null; then
			systemctl_ enable --now salus-backup-check.timer
		fi
		[[ -n $ALERT_EMAIL ]] ||
			warn "ALERT_EMAIL is not set: a failed or missing backup is logged on this server only, nobody is mailed"
	else
		warn "the backup is not configured (BACKUP_HOST, BACKUP_USER, BACKUP_KNOWN_HOST, BACKUP_AGE_RECIPIENT): the timer is installed but not enabled"
	fi
}

main() {
	parse_args "$@"
	check_os
	load_config "$CONFIG"
	validate_config
	((CHECK)) && log "check mode: nothing is changed"
	step_packages
	step_deploy_user
	step_ufw
	step_sshd
	step_fail2ban
	step_upgrades
	step_docker
	step_layout
	step_backup
	step_volume
	check_swap
	apply_services
	check_docker_published
	if [[ -f ${BACKUP_SSH_KEY_FILE}.pub ]]; then
		log "the backup's public key (install it on the Storage Box, docs/deploy/backup-restore.md):"
		log "  $(cat "${BACKUP_SSH_KEY_FILE}.pub")"
	fi
	if ((CHECK)); then log "$CHANGES changes would be made"; else log "$CHANGES changes made"; fi
	((FAILURES == 0)) || die "$FAILURES problems need attention (see above)"
}

main "$@"
