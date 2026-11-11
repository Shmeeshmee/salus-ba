# Server bootstrap

`bootstrap.sh` prepares an Ubuntu 24.04 server for the `salus` compose project (docs/adr/0011-single-server-deployment.md):
a fresh Hetzner Cloud server or an existing one that already runs other software. It is idempotent: run it again
after changing `bootstrap.conf` and it changes only what differs; a second run with the same settings changes
nothing. `--check` reports what it would change and changes nothing.

## Running it

Copy the directory and your settings to the server, from a checkout:

```sh
cp deployments/production/bootstrap/bootstrap.conf.example bootstrap.conf   # fill it in
ssh root@SERVER rm -rf /root/salus-bootstrap
scp -r deployments/production/bootstrap root@SERVER:/root/salus-bootstrap
scp bootstrap.conf root@SERVER:/root/salus-bootstrap/bootstrap.conf
```

Then **open an interactive root session and run the script inside it**, so a mistake in the SSH settings cannot
lock you out (sshd keeps established sessions when it reloads):

```sh
ssh root@SERVER                      # terminal 1: stays open until step 3
/root/salus-bootstrap/bootstrap.sh --check --config /root/salus-bootstrap/bootstrap.conf
/root/salus-bootstrap/bootstrap.sh --config /root/salus-bootstrap/bootstrap.conf
```

1. Read the `--check` output first: it names every change, and fails if ufw would block a service already listening
   on a public address (add its port to `UFW_EXTRA_PORTS`, or set `UFW_ACCEPT_BLOCKING=yes` to block it on purpose).
2. From a **second terminal**, check that `ssh deploy@SERVER` (with `-p SSH_PORT` if you changed it) works.
3. Only then close terminal 1. From now on root cannot log in over SSH.

If step 2 fails, fix it from terminal 1 (`sshd -T`, `journalctl -u ssh`, `/etc/ssh/sshd_config.d/00-salus.conf`).

The settings are installed as `/etc/salus/bootstrap.conf` (root, mode 600: the root backup job reads them, so they
are not under the deploy-owned `/opt/salus`), and later runs need no `--config`.

### Later runs and root access

`DEPLOY_SUDO` defaults to `no`, so after the first run there is no root login and no sudo. A later bootstrap run (a
changed setting, a new version of the script) needs root, through one of:

- **Hetzner's console** (the Cloud Console's "Console" button, after "Reset root password" if no root password is
  set). This is the intended way.
- `DEPLOY_SUDO=yes`, set in the first run: passwordless sudo for `deploy`.

Why `no` by default: the same `deploy` account and keys are used by the deploy workflow in GitHub Actions. Sudo gives
`deploy` nothing the `docker` group does not already give (see below), but a general-purpose root shell on the CI
account would be one more thing to rely on; the bootstrap is run rarely and by hand, and the console works without
any key at all. Choose `yes` if you prefer to run the bootstrap over SSH; the risk is the same either way.

## What it does

| Step | What | Left alone |
| --- | --- | --- |
| Packages | `ufw`, `fail2ban`, `unattended-upgrades`, `age`, `cryptsetup`, `openssh-*`, `iproute2`, only the missing ones | Everything installed already |
| `deploy` user | SSH key only (no usable password); `authorized_keys` set to exactly the `DEPLOY_SSH_KEY` lines; in the `docker` group; passwordless sudo only with `DEPLOY_SUDO=yes`. Refuses to run with no key configured | Other users |
| ufw | Allow rules first (`SSH_PORT`/tcp, 80/tcp, 443/tcp, 443/udp, 3478/udp, 7881/tcp, 7882/udp and `UFW_EXTRA_PORTS`), then default deny incoming and allow outgoing, then enable. Before enabling ufw (or switching it to deny), it lists every port listening on a public address that no rule covers and **refuses** unless `UFW_ACCEPT_BLOCKING=yes` | Existing rules (an old SSH port rule is reported, never deleted) |
| sshd | `/etc/ssh/sshd_config.d/00-salus.conf`: `Port`, `PermitRootLogin no`, `PasswordAuthentication no`, `KbdInteractiveAuthentication no`, `PubkeyAuthentication yes`. Written only after the deploy key is in place and kept only if `sshd -t` accepts it and the effective configuration for `deploy` (`sshd -T`) still lets it in: no `DenyUsers`/`DenyGroups` match, `AllowUsers`/`AllowGroups` include it, `AuthorizedKeysFile` reads `~/.ssh/authorized_keys`, `AuthenticationMethods` needs no more than a key; otherwise the previous file is put back. It sorts before cloud-init's `50-cloud-init.conf`, so its values win. Reloaded on every run; when the port changed, `ssh.socket` and then `ssh.service` are restarted (Ubuntu 24.04's socket activation reads the port at `daemon-reload`). Established sessions stay | The rest of `sshd_config` |
| fail2ban | A jail for sshd on `SSH_PORT` (journal backend), reloaded on every run | Other jails |
| Upgrades | `unattended-upgrades` from the release and security pockets only (Ubuntu's default list without `-updates`; Docker's repository is not among them), at `UPGRADE_TIME`, never an automatic reboot: `UPGRADE_AUTO_REBOOT=yes` is refused, as after a reboot the encrypted volume waits for the owner (ADR 0011); `/var/run/reboot-required` says when to reboot by hand and unlock. Warns before replacing other origins the server's own apt configuration allowed | |
| Docker | Docker Engine and the compose plugin from Docker's apt repository (key checked by fingerprint, `Signed-By`), only when Docker is absent. An existing Docker is never reconfigured or restarted, and no `daemon.json` is written; if it lacks the compose plugin, the script says so and fails at the end | Other compose projects: no `prune`, no `down` |
| `/opt/salus` | `/opt/salus` and `deployments/` (`deploy:deploy`, 750); `backups/` (`root:root`, 700: only the root backup job writes there). The settings go to `/etc/salus/bootstrap.conf` | Other directories under `/opt` |
| Backup | `/usr/local/sbin/salus-backup` (`backup.sh`), its own SSH key `/root/.ssh/salus-backup` (generated once; the public half is printed at the end), the Storage Box's pinned host key, `salus-backup.service` (the Postgres dump) and `salus-backup-objects.service` (Garage's objects, right after the dump), both 2 h timeout, hardened, `OnFailure=salus-backup-failed@%n.service` (logs at err priority and mails `ALERT_EMAIL`), `salus-backup.timer` (nightly, `Persistent=true`, random delay), `salus-backup-check.timer` (daily at `BACKUP_CHECK_TIME`: mails `ALERT_EMAIL` when a backup on the Storage Box is older than `BACKUP_MAX_AGE_HOURS` or missing; its service has the same failure hook, for a check that cannot run or send). The timers are enabled only once the `BACKUP_*` settings are complete; a missing `ALERT_EMAIL` is a warning. A `salus-backup-failed.service` of an earlier release is removed. See docs/deploy/backup-restore.md | |
| Encrypted volume | `/usr/local/sbin/salus-volume` (`volume.sh`): `setup` once, `unlock` after every reboot, `keys`, `check` (deployments/production/README.md "Encrypted volume", docs/deploy/README.md). The bootstrap only installs it and warns while `/etc/salus/volume.conf` is missing; it never formats, opens or mounts anything | The volume |
| Swap | Checks `/proc/swaps` and `/etc/fstab`: active swap or a swap line **fails the run** (after the other steps), as swap would page personal data to the unencrypted disk. Hetzner's Ubuntu images have none; `swapoff -a` and removing the line fix it | Swap is never turned off by the script |

## Security notes

- **The `docker` group is root-equivalent.** Whoever can log in as `deploy` (the operator and the deploy workflow)
  can start a container that mounts `/` and so has root on the host. Treat the deploy keys as root keys.
- **Docker-published ports bypass ufw.** Docker writes its own iptables rules ahead of ufw's, so a port a container
  publishes on `0.0.0.0` is open whatever ufw says. The production compose file is the real boundary: only Caddy's
  and LiveKit's ports in the list above are published there. At the end of every run (and in `--check`) the
  bootstrap lists the ports any container publishes on a public address and warns about each one outside the list,
  other projects' included; it never changes them.

## Tests

`make deploy-check` (or `test/run.sh [shellcheck|bootstrap|backup|volume|deploy]`; `deploy` tests `../deploy.sh`), all in Docker:

- shellcheck (pinned image) on every script;
- `volume.sh` in a fresh `ubuntu:24.04` container against stubs of `cryptsetup`, `mount`, `findmnt`, `lsblk`,
  `blkid`, `mkfs.ext4` and `docker` (nothing is formatted, opened or mounted; ownership, modes and the key files are
  real): setup writes the layout, both key files (`65532` 0400) and `volume.conf`, with no fstab or crypttab line
  and no key echoed; it refuses a second setup, a device with a signature, a mounted device, a short key and two
  equal keys, and a setup that fails half-way unmounts and closes the volume and writes no `volume.conf`; unlock
  opens, mounts, lays the directories out again (another `deploy` gid), checks, then starts the stack holding
  deploy.sh's lock (or says there is no release yet, or prints the command with `--no-start`, or refuses while a
  deploy holds the lock), is idempotent, and stops on a wrong passphrase, a detached volume, something else at the
  mount point, a missing key file (naming `salus-volume keys`) or a wrong mode; `keys` rewrites a missing file and
  replaces another value only with `--replace`; `check` needs root and fails on a locked volume, a device that is
  not `crypt` and a host account with uid 65532;
- `bootstrap.sh` in a fresh `ubuntu:24.04` container: refuses a missing or placeholder key, `UPGRADE_AUTO_REBOOT=yes`,
  and the test flags on a host with systemd, and an `ALERT_EMAIL` that is not one bare address; active swap or a swap line in `/etc/fstab` fails `--check` and a real
  run; cryptsetup and `salus-volume` are installed; `--check` reports and changes nothing; a real run (packages, Docker from its repository, the
  user, sshd's configuration through `sshd -t` and `sshd -T`, ufw's rules and defaults, fail2ban, upgrades with the
  warning about a replaced origin, the layout, the backup's files); a second run and a `--check` report 0 changes;
  a public listener ufw would block makes `--check` fail and a real run refuse without touching ufw, unless
  `UFW_ACCEPT_BLOCKING=yes` or the port is in `UFW_EXTRA_PORTS`; the warning about another project's published
  port; a `DenyUsers deploy` elsewhere makes the run fail and puts the previous drop-in back; a new SSH port, then
  0 changes again; `systemd-analyze verify` on the units;
- `backup.sh` against a throwaway Postgres migrated with this checkout's migrations (real roles, grants, row-level
  security) plus synthetic rows, and an SFTP-only server: a wrong host key, a missing database (after waiting) and
  a failing `pg_dump` exit non-zero and leave nothing behind; a dump is uploaded encrypted, fetched, decrypted and
  restored with the drill's steps (roles first, `--exit-on-error`) with the same row counts, migration version,
  tables, policies, function owners and grants; faked dumps older than 14 days and a stale partial upload are
  deleted, newer dumps and other files kept; the optional local copy. The objects backup against a throwaway Garage
  (the production `garage.toml`) with synthetic objects: a missing objectstore container or key pair fails; the
  archive is uploaded encrypted, fetched, decrypted and extracted into a second Garage, which `rclone check` finds
  equal, a sample object byte for byte; its retention leaves the dumps alone. The alerts with `curl` and
  `systemctl` stubbed (no mail leaves): STARTTLS required by default, implicit TLS with `SMTP_TLS=tls`, an unknown
  value refused, the login through curl's config on stdin and in no argument, the envelope sender from a
  display-name `SMTP_FROM`, CRLF lines, nothing but host, unit, time and status in the mail, a failed send and a
  missing `ALERT_EMAIL` fail. The freshness check: fresh backups send nothing; a stale and a missing kind send one
  alert naming both; `BACKUP_MAX_AGE_HOURS` sets the limit.

Two flags exist for the container test only and default off; the script refuses them on a host that runs systemd:
`SALUS_BOOTSTRAP_TEST_NO_SYSTEMD=1` (skips every `systemctl` call), `SALUS_BOOTSTRAP_TEST_NO_UFW_ENABLE=1` (skips
`ufw enable`, which needs the kernel's netfilter) and `SALUS_BOOTSTRAP_TEST_PROC_SWAPS=FILE` (reads the swap from
FILE: a container sees the host's `/proc/swaps`). What only a real server shows: LUKS2 formatting, opening and mounting the real volume, the containers starting on it, ufw active with the rules, sshd
reloaded and the port change through `ssh.socket`, fail2ban banning, the timers and the failure hook firing, the
hardened unit reaching Docker, a mail reaching the real relay, and Docker's service starting at boot.
