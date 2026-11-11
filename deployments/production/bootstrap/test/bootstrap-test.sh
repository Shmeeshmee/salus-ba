#!/usr/bin/env bash
# shellcheck disable=SC2016 # the bash -c checks expand inside the child shell
# Runs inside a fresh ubuntu:24.04 container (run.sh): bootstrap.sh in --check mode, for real, and again to show the
# second run changes nothing. Without systemd and netfilter in a container, the test flags skip systemctl and
# enabling ufw; everything else (packages, Docker from its apt repository, files, users, sshd -t) is real.
set -euo pipefail

BOOTSTRAP=/bootstrap/bootstrap.sh
WORK=/tmp/bootstrap-test
export SALUS_BOOTSTRAP_TEST_NO_SYSTEMD=1 SALUS_BOOTSTRAP_TEST_NO_UFW_ENABLE=1
# The container sees the host's /proc/swaps: the test's own, with no swap unless a case adds a line.
export SALUS_BOOTSTRAP_TEST_PROC_SWAPS=/tmp/bootstrap-test-swaps
SWAPS_HEADER=$'Filename\t\t\t\tType\t\tSize\t\tUsed\t\tPriority'
printf '%s\n' "$SWAPS_HEADER" >"$SALUS_BOOTSTRAP_TEST_PROC_SWAPS"

# Synthetic public keys; their private halves were never kept.
KEY1='ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPjQKB4oBfeVcoQEh0NFarmldEGPWl2HAEyCzJzEunRQ synthetic-test-operator'
KEY2='ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMXaRu6jQm7Pqn0SuCMzRZ9c4zC45YmDD+dLOOqXQOwx synthetic-test-ci'

pass() { printf 'ok: %s\n' "$*"; }
fail() {
	printf 'FAIL: %s\n' "$*" >&2
	exit 1
}
expect() { # expect DESCRIPTION COMMAND...
	local desc=$1
	shift
	if "$@"; then pass "$desc"; else fail "$desc"; fi
}

mkdir -p "$WORK"
write_conf() { # write_conf SSH_PORT [EXTRA-LINE]
	cat >"$WORK/bootstrap.conf" <<EOF
# Test settings, synthetic.
DEPLOY_SSH_KEY=$KEY1
DEPLOY_SSH_KEY="$KEY2"
SSH_PORT=$1
UFW_EXTRA_PORTS=8443/tcp
BACKUP_HOST=storagebox.invalid
BACKUP_USER=u000000-sub1
BACKUP_KNOWN_HOST=[storagebox.invalid]:23 $KEY2
BACKUP_AGE_RECIPIENT=age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
${2:-}
EOF
}

run() { # run LOGNAME ARGS...: runs the bootstrap, keeps its output
	local name=$1
	shift
	"$BOOTSTRAP" "$@" >"$WORK/$name.log" 2>&1
}

echo "== refuses to run without a deploy key"
printf 'SSH_PORT=22\n' >"$WORK/nokey.conf"
if run nokey --config "$WORK/nokey.conf"; then fail "ran without a deploy key"; fi
expect "says why" grep -q 'no DEPLOY_SSH_KEY' "$WORK/nokey.log"
printf 'DEPLOY_SSH_KEY=ssh-ed25519 AAAA... operator@laptop\n' >"$WORK/placeholder.conf"
if run placeholder --config "$WORK/placeholder.conf"; then fail "accepted the example's placeholder key"; fi
expect "the example's placeholder key is refused" grep -q 'not a public key line' "$WORK/placeholder.log"
expect "nothing created" bash -c '! id deploy 2>/dev/null && [[ ! -e /opt/salus ]]'

echo "== refuses automatic reboots (the encrypted volume is unlocked by hand)"
write_conf 22 UPGRADE_AUTO_REBOOT=yes
if run reboot --check --config "$WORK/bootstrap.conf"; then fail "accepted UPGRADE_AUTO_REBOOT=yes"; fi
expect "says why" grep -q 'UPGRADE_AUTO_REBOOT=yes is refused' "$WORK/reboot.log"

echo "== refuses the test flags on a host with systemd"
write_conf 22
mkdir -p /run/systemd/system
if run flags --check --config "$WORK/bootstrap.conf"; then fail "ran with test flags under systemd"; fi
expect "says why" grep -q 'for the container test only' "$WORK/flags.log"
rmdir /run/systemd/system

# The server's own unattended-upgrades list, with a pocket beyond security: replaced, with a warning.
mkdir -p /etc/apt/apt.conf.d
printf 'Unattended-Upgrade::Allowed-Origins { "${distro_id}:${distro_codename}-updates"; };\n' >/etc/apt/apt.conf.d/51local-origins

echo "== --check on a fresh server"
write_conf 22
run check1 --check --config "$WORK/bootstrap.conf" || {
	cat "$WORK/check1.log"
	fail "--check failed"
}
expect "reports changes" grep -q '^would: create user deploy' "$WORK/check1.log"
expect "reports Docker" grep -q "^would: install Docker Engine" "$WORK/check1.log"
expect "changes nothing" bash -c '! id deploy 2>/dev/null && [[ ! -e /opt/salus && ! -e /etc/ssh/sshd_config.d/00-salus.conf ]] && ! command -v ufw'

echo "== first run"
run run1 --config "$WORK/bootstrap.conf" || {
	cat "$WORK/run1.log"
	fail "first run failed"
}
grep -E '^(change|warning|error)' "$WORK/run1.log" || true
expect "warns that the server's -updates origin is replaced" grep -q "^warning: unattended-upgrades: the server's own Allowed-Origins entry '.*-updates' is replaced" "$WORK/run1.log"
expect "made changes" grep -Eq '^[1-9][0-9]* changes made' "$WORK/run1.log"
expect "deploy user exists, no usable password" bash -c '[[ $(getent shadow deploy | cut -d: -f2) == "*" ]]'
expect "deploy is in the docker group" bash -c 'id -nG deploy | tr " " "\n" | grep -qx docker'
expect "authorized_keys holds exactly the two keys, 600" bash -c \
	"[[ \$(cat /home/deploy/.ssh/authorized_keys) == \$(printf '%s\n' '$KEY1' '$KEY2') ]] && [[ \$(stat -c '%a %U' /home/deploy/.ssh/authorized_keys) == '600 deploy' ]] && [[ \$(stat -c '%a' /home/deploy/.ssh) == 700 ]]"
expect "sshd accepts the configuration" sshd -t
eff=$(sshd -T -C user=deploy,host=localhost,addr=127.0.0.1)
for want in 'permitrootlogin no' 'passwordauthentication no' 'kbdinteractiveauthentication no' 'port 22'; do
	expect "sshd effective: $want" grep -qx "$want" <<<"$eff"
done
rules=$(ufw show added)
for spec in 22/tcp 80/tcp 443/tcp 443/udp 3478/udp 7881/tcp 7882/udp 8443/tcp; do
	expect "ufw allows $spec" grep -qE "^ufw allow $spec( |$)" <<<"$rules"
done
expect "ufw denies incoming by default" grep -qx 'DEFAULT_INPUT_POLICY="DROP"' /etc/default/ufw
expect "fail2ban watches sshd" grep -qx 'enabled = true' /etc/fail2ban/jail.d/salus-sshd.conf
origins=$(apt-config dump | grep -A0 '^Unattended-Upgrade::Allowed-Origins::' || true)
expect "unattended upgrades: security pockets only" bash -c \
	"[[ \$(grep -c . <<<'$origins') -eq 4 ]] && ! grep -q -- '-updates' <<<'$origins' && grep -q -- '-security' <<<'$origins'"
expect "no automatic reboot" bash -c "apt-config dump | grep -qx 'Unattended-Upgrade::Automatic-Reboot \"false\";'"
expect "upgrade time set" grep -qx 'OnCalendar=\*-\*-\* 04:00' /etc/systemd/system/apt-daily-upgrade.timer.d/salus.conf
expect "Docker and compose installed" bash -c 'docker --version && docker compose version >/dev/null'
expect "Docker's repository is signed-by" grep -qx 'Signed-By: /etc/apt/keyrings/docker.asc' /etc/apt/sources.list.d/docker.sources
expect "no daemon.json written" test ! -e /etc/docker/daemon.json
expect "/opt/salus is deploy's, 750" bash -c '[[ $(stat -c "%a %U:%G" /opt/salus) == "750 deploy:deploy" && $(stat -c "%a %U:%G" /opt/salus/deployments) == "750 deploy:deploy" ]]'
expect "backup staging is root's, 700" bash -c '[[ $(stat -c "%a %U" /opt/salus/backups) == "700 root" ]]'
installed=$(cat /etc/salus/bootstrap.conf)
expect "the settings are installed for root only" test "$installed" = "$(cat "$WORK/bootstrap.conf")"
expect "mode 600, root, not under /opt/salus" bash -c "[[ \$(stat -c '%a %U' /etc/salus/bootstrap.conf) == '600 root' && ! -e /opt/salus/bootstrap.conf ]]"
expect "the backup script is installed" cmp -s /bootstrap/backup.sh /usr/local/sbin/salus-backup
expect "the backup timer is nightly and persistent" bash -c 'grep -qx "OnCalendar=\*-\*-\* 02:30" /etc/systemd/system/salus-backup.timer && grep -qx "Persistent=true" /etc/systemd/system/salus-backup.timer'
expect "the backup unit has a timeout, hardening and a failure hook" bash -c 'f=/etc/systemd/system/salus-backup.service; grep -qx "TimeoutStartSec=2h" $f && grep -qx "OnFailure=salus-backup-failed.service" $f && grep -qx "ProtectHome=read-only" $f && grep -q "systemd-cat -t salus-backup -p err" /etc/systemd/system/salus-backup-failed.service'
expect "the backup has its own SSH key" test -s /root/.ssh/salus-backup.pub
expect "the Storage Box host key is pinned" grep -qF '[storagebox.invalid]:23 ssh-ed25519' /etc/salus/backup_known_hosts
expect "cryptsetup is installed" command -v cryptsetup
expect "salus-volume is installed, root's" bash -c 'cmp -s /bootstrap/volume.sh /usr/local/sbin/salus-volume && [[ $(stat -c "%a %U" /usr/local/sbin/salus-volume) == "755 root" ]]'
expect "says the volume is not set up yet" grep -q '^warning: the encrypted volume is not set up' "$WORK/run1.log"
expect "no crypttab line, nothing opened at boot" bash -c '! grep -qs salus /etc/crypttab'

echo "== second run changes nothing"
run run2 --config "$WORK/bootstrap.conf" || {
	cat "$WORK/run2.log"
	fail "second run failed"
}
expect "0 changes" grep -qx '0 changes made' "$WORK/run2.log"
expect "no change lines" bash -c "! grep -q '^change:' '$WORK/run2.log'"
run check2 --check || fail "--check with the installed settings failed"
expect "--check agrees" grep -qx '0 changes would be made' "$WORK/check2.log"

echo "== active swap, or a swap line in fstab, is reported and fails the run"
printf '%s\n' "$SWAPS_HEADER" $'/swap.img                               file\t\t1048572\t\t0\t\t-2' >"$SALUS_BOOTSTRAP_TEST_PROC_SWAPS"
if run swap1 --check; then fail "--check passed with swap active"; fi
expect "names the swap" grep -q '^error: swap is active (/swap.img)' "$WORK/swap1.log"
if run swap2 --config "$WORK/bootstrap.conf"; then fail "a real run passed with swap active"; fi
expect "a real run fails too" grep -q '1 problems need attention' "$WORK/swap2.log"
printf '%s\n' "$SWAPS_HEADER" >"$SALUS_BOOTSTRAP_TEST_PROC_SWAPS"
cp /etc/fstab "$WORK/fstab.orig"
printf '/swap.img none swap sw 0 0\n' >>/etc/fstab
if run swap3 --check; then fail "--check passed with swap in fstab"; fi
expect "names the fstab line" grep -q '^error: /etc/fstab has a swap entry' "$WORK/swap3.log"
cp "$WORK/fstab.orig" /etc/fstab
run swap4 --check || fail "--check without swap failed"
expect "no swap, no error" bash -c "! grep -q swap '$WORK/swap4.log'"

echo "== a public listener ufw would block"
/usr/sbin/sshd -p 2222 -o PidFile=/run/sshd-test.pid
if run check3 --check; then fail "--check passed although ufw would block a listener"; fi
expect "--check names it" grep -q '^error: 2222/tcp (sshd) listens on a public address and ufw would block it' "$WORK/check3.log"
before=$(ufw show added)
if run refuse --config "$WORK/bootstrap.conf"; then fail "a real run went on although ufw would block a listener"; fi
expect "a real run refuses" grep -q 'refusing to enable ufw' "$WORK/refuse.log"
expect "and leaves ufw as it was" test "$(ufw show added)" = "$before"
write_conf 22 UFW_ACCEPT_BLOCKING=yes
run accept --check --config "$WORK/bootstrap.conf" || fail "--check with UFW_ACCEPT_BLOCKING=yes failed"
expect "UFW_ACCEPT_BLOCKING=yes turns it into a warning" grep -q '^warning: 2222/tcp (sshd) listens on a public address; ufw will block it' "$WORK/accept.log"
write_conf 22 UFW_EXTRA_PORTS=2222/tcp
run extra --check --config "$WORK/bootstrap.conf" || fail "--check with the port in UFW_EXTRA_PORTS failed"
expect "an allowed extra port is not blocked" bash -c "! grep -q '2222/tcp (sshd)' '$WORK/extra.log'"
kill "$(cat /run/sshd-test.pid)"
write_conf 22
mkdir -p "$WORK/fake"
cat >"$WORK/fake/docker" <<'EOF'
#!/bin/sh
# A fake Docker CLI with three running containers.
case "$1" in
ps) echo c1 ;;
inspect) printf '%s\n' '/other-app-web-1||8080|8080/tcp' '/salus-caddy-1|0.0.0.0|443|443/udp' '/other-db-1|127.0.0.1|5432|5432/tcp' ;;
compose) echo "Docker Compose version v0" ;;
*) echo "Docker version 0" ;;
esac
EOF
chmod +x "$WORK/fake/docker"
PATH=$WORK/fake:$PATH run check4 --check || fail "--check with the fake Docker failed"
expect "warns about another project's public port" grep -q '^warning: container other-app-web-1 publishes 8080/tcp' "$WORK/check4.log"
expect "silent about allowed and loopback ports" bash -c "! grep -Eq 'salus-caddy-1|other-db-1' '$WORK/check4.log'"
expect "reports, changes nothing" grep -qx '0 changes would be made' "$WORK/check4.log"

echo "== an sshd setting that would lock the deploy user out"
printf 'DenyUsers deploy\n' >/etc/ssh/sshd_config.d/10-test-deny.conf
before=$(cat /etc/ssh/sshd_config.d/00-salus.conf)
write_conf 2201
if run deny --config "$WORK/bootstrap.conf"; then fail "went on although DenyUsers shuts out deploy"; fi
expect "names the problem" grep -q 'sshd: DenyUsers includes deploy' "$WORK/deny.log"
expect "puts the previous drop-in back" test "$(cat /etc/ssh/sshd_config.d/00-salus.conf)" = "$before"
rm /etc/ssh/sshd_config.d/10-test-deny.conf

echo "== a new SSH port"
write_conf 2200
run run3 --config "$WORK/bootstrap.conf" || {
	cat "$WORK/run3.log"
	fail "run with a new port failed"
}
expect "sshd moves to 2200" bash -c 'sshd -T | grep -qx "port 2200"'
expect "ufw allows 2200 and keeps 22" bash -c 'ufw show added | grep -q "^ufw allow 2200/tcp" && ufw show added | grep -q "^ufw allow 22/tcp"'
expect "tells the operator about the old rule" grep -q "ufw delete allow 22/tcp" "$WORK/run3.log"
expect "fail2ban follows" grep -qx 'port = 2200' /etc/fail2ban/jail.d/salus-sshd.conf
run run4 --config "$WORK/bootstrap.conf" || fail "fourth run failed"
expect "then nothing again" grep -qx '0 changes made' "$WORK/run4.log"

echo "== the units are valid"
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends systemd >/dev/null
units=(/etc/systemd/system/salus-backup.service /etc/systemd/system/salus-backup.timer /etc/systemd/system/salus-backup-failed.service)
expect "systemd-analyze verify" systemd-analyze verify "${units[@]}"

echo "bootstrap tests passed"
