# Server bootstrap

`bootstrap.sh` prepares an Ubuntu 24.04 server for the `salus` compose project (docs/adr/0011-single-server-deployment.md):
a fresh Hetzner Cloud server or an existing one that already runs other software. It is idempotent: run it again
after changing `bootstrap.conf` and it changes only what differs; a second run with the same settings changes
nothing. `--check` reports what it would change and changes nothing.

## Running it

From a checkout, as root on the server (the first time over the root login Hetzner set up with your key):

```sh
cp deployments/production/bootstrap/bootstrap.conf.example bootstrap.conf   # fill it in
ssh root@SERVER rm -rf /root/salus-bootstrap
scp -r deployments/production/bootstrap root@SERVER:/root/salus-bootstrap
scp bootstrap.conf root@SERVER:/root/salus-bootstrap/bootstrap.conf
ssh root@SERVER /root/salus-bootstrap/bootstrap.sh --check --config /root/salus-bootstrap/bootstrap.conf
ssh root@SERVER /root/salus-bootstrap/bootstrap.sh --config /root/salus-bootstrap/bootstrap.conf
```

The settings are then installed as `/opt/salus/bootstrap.conf`, and later runs (as `deploy` with `sudo`, or from the
Hetzner console) need no `--config`. Keep the root session open until `ssh deploy@SERVER` (with `-p SSH_PORT` if you
changed it) works from a second terminal: from then on root cannot log in over SSH.

## What it does

| Step | What | Left alone |
| --- | --- | --- |
| Packages | `ufw`, `fail2ban`, `unattended-upgrades`, `age`, `openssh-*`, `iproute2`, only the missing ones | Everything installed already |
| `deploy` user | SSH key only (no usable password); `authorized_keys` set to exactly the `DEPLOY_SSH_KEY` lines; in the `docker` group; passwordless sudo only with `DEPLOY_SUDO=yes`. Refuses to run with no key configured | Other users |
| sshd | `/etc/ssh/sshd_config.d/00-salus.conf`: `Port`, `PermitRootLogin no`, `PasswordAuthentication no`, `KbdInteractiveAuthentication no`. Written only after the deploy key is in place, validated with `sshd -t` (put back on failure), checked with `sshd -T` for the deploy user (it sorts before cloud-init's `50-cloud-init.conf`, so its values win), then reloaded. A new port restarts `ssh.socket` (Ubuntu 24.04's socket activation reads the port at `daemon-reload`); established sessions stay | The rest of `sshd_config` |
| ufw | Default deny incoming, allow outgoing; allows `SSH_PORT`/tcp, 80/tcp, 443/tcp, 443/udp, 3478/udp, 7881/tcp, 7882/udp and `UFW_EXTRA_PORTS`. Before enabling ufw on a server where it is off, it warns about every port listening on a public address that would then be blocked | Existing rules (an old SSH port rule is reported, never deleted) |
| fail2ban | A jail for sshd on `SSH_PORT` (journal backend) | Other jails |
| Upgrades | `unattended-upgrades` for the security pockets only (Docker's repository is not among them), at `UPGRADE_TIME`, no automatic reboot unless `UPGRADE_AUTO_REBOOT=yes` | |
| Docker | Docker Engine and the compose plugin from Docker's apt repository (key checked by fingerprint, `Signed-By`), only when Docker is absent. An existing Docker is never reconfigured or restarted, and no `daemon.json` is written; if it lacks the compose plugin, the script says so and fails at the end | Other compose projects: no `prune`, no `down` |
| `/opt/salus` | `/opt/salus` and `deployments/` (`deploy:deploy`, 750); `backups/` (`root:root`, 700: only the root backup job writes there); the settings as `bootstrap.conf` | Other directories under `/opt` |
| Backup | `/usr/local/sbin/salus-backup` (`backup.sh`), its own SSH key `/root/.ssh/salus-backup` (generated once; the public half is printed at the end), the Storage Box's pinned host key, `salus-backup.service` and `salus-backup.timer` (nightly, `Persistent=true`, random delay). The timer is enabled only once the `BACKUP_*` settings are complete. See docs/deploy/backup-restore.md | |

## Security notes

- **The `docker` group is root-equivalent.** Whoever can log in as `deploy` (the operator and the deploy workflow)
  can start a container that mounts `/` and so has root on the host. Treat the deploy keys as root keys.
- **Docker-published ports bypass ufw.** Docker writes its own iptables rules ahead of ufw's, so a port a container
  publishes on `0.0.0.0` is open whatever ufw says. The production compose file is the real boundary: only Caddy's
  and LiveKit's ports in the list above are published there. At the end of every run (and in `--check`) the
  bootstrap lists the ports any container publishes on a public address and warns about each one outside the list,
  other projects' included; it never changes them.
- With root login off, root on the server is reached with `sudo` (`DEPLOY_SUDO=yes`, or another user that already
  had it) or through Hetzner's console.

## Tests

`make deploy-check` (or `test/run.sh [shellcheck|bootstrap|backup]`), all in Docker:

- shellcheck (pinned image) on every script;
- `bootstrap.sh` in a fresh `ubuntu:24.04` container: refuses a missing or placeholder key; `--check` reports and
  changes nothing; a real run (packages, Docker from its repository, the user, sshd's configuration through `sshd
  -t` and `sshd -T`, ufw's rules and defaults, fail2ban, upgrades, the layout, the backup's files); a second run and
  a `--check` report 0 changes; the warnings about a public listener and about another project's published port;
  a new SSH port, then 0 changes again;
- `backup.sh` against a throwaway Postgres with synthetic rows and an SFTP-only server: a wrong host key, a missing
  database and a failing `pg_dump` exit non-zero and leave nothing behind; a dump is uploaded encrypted, fetched,
  decrypted and restored into a second Postgres with the same row counts and migration version; faked dumps older
  than 14 days are deleted, newer ones and other files kept; the optional local copy.

Two flags exist for the container test only and default off; never set them on a server:
`SALUS_BOOTSTRAP_TEST_NO_SYSTEMD=1` (skips every `systemctl` call) and `SALUS_BOOTSTRAP_TEST_NO_UFW_ENABLE=1` (skips
`ufw enable`, which needs the kernel's netfilter). What only a real server shows: ufw active with the rules, sshd
reloaded and the port change through `ssh.socket`, fail2ban banning, the timers firing, and Docker's service
starting at boot.
