# Deploying Salus: first-time setup

One Hetzner Cloud server with Docker Compose, deployed from GitHub Actions (docs/adr/0011-single-server-deployment.md,
docs/plans/deploy.md). This page is the order of work for the first deploy; the files it points to hold the detail:

| File | What |
| --- | --- |
| `deployments/production/README.md` | The stack: hostnames, ports, TURN, healthchecks, memory budget |
| `deployments/production/.env.production.example` | Every variable, how to generate each secret |
| `deployments/production/bootstrap/README.md` | Preparing the server (`bootstrap.sh`, `salus-volume`) |
| `docs/deploy/backup-restore.md` | Backups to the Storage Box and the restore drill |
| `.github/workflows/images.yml`, `deploy.yml` | Building the images; deploying them |

Nothing is deployed until step 8 sets `DEPLOY_ENABLED`. Until then every Deploy run stops at "not enabled".

## 0. Decide first

- **The domain** (`SALUS_DOMAIN`). The stack serves the apex (landing page), `app.` (web app and its API),
  `api.` (the API, for the mobile app), `livekit.` (video signalling) and `turn.` (TURN over TLS).
- **The server type** (`SALUS_SERVER_TYPE`): at least 4 vCPU and 8 GB (the memory budget in
  `deployments/production/README.md` is sized for that); x86 (the images are built for amd64 only).
- **Before real patient data**: the two open points of ADR 0011 ("Before real patient data": the DPAs of Hetzner
  and the SMTP relay, point 1, and the backups' retention, point 7). Encryption at rest and the keys are decided
  there and set up below (the encrypted volume, step 4). The stack runs without the open points on synthetic data;
  it must not hold real patients' data until both are settled.

## 1. Keys, on a trusted machine, into Bitwarden first

| Key | How | Where it goes |
| --- | --- | --- |
| Your own SSH key | The one you already use (`ssh-ed25519`) | Hetzner project; `DEPLOY_SSH_KEY` in `bootstrap.conf` |
| The deploy workflow's SSH key | `ssh-keygen -t ed25519 -N '' -C salus-github-deploy -f salus-deploy` | Private half: GitHub secret `DEPLOY_SSH_KEY`; public half: a second `DEPLOY_SSH_KEY` line in `bootstrap.conf` |
| The backups' age key | `age-keygen -o salus-backup.age` | Private half: Bitwarden only, never on the server or in GitHub; public half (`age1…`): `BACKUP_AGE_RECIPIENT` |
| The stack's secrets | Fill in a copy of `.env.production.example` (each line says how to generate its value) | One Bitwarden secure note; GitHub secret `SALUS_ENV_FILE` (step 6) |
| The volume's LUKS passphrase | A long random passphrase from Bitwarden's generator (e.g. 6 words or more) | Bitwarden only. Typed into `cryptsetup` on the server at setup and after every reboot; never stored on the server or in GitHub |
| `BINARY_KEK`, `TOTP_KEY` | `openssl rand -base64 32`, once each | Two Bitwarden items of their own (not in the env file's note). Pasted into `salus-volume setup` (step 4), which writes them to the key files on the encrypted volume. Never in `SALUS_ENV_FILE`: the deploy refuses an env file that sets them |

`BINARY_KEK` and `TOTP_KEY` cannot be regenerated: losing them loses every attachment and every enrolled MFA, and a
restore needs the same values (docs/deploy/backup-restore.md). The LUKS passphrase cannot be recovered either:
without it the volume, and every record on it, is unreadable; a restore from the backups onto a new volume is the
only way back.

## 2. Hetzner

1. **Account and DPA.** In the Hetzner console, conclude the data processing agreement (GDPR Art. 28). Then
   record Hetzner as wired in in docs/privacy/processors.md (ADR 0011 open point 1).
2. **Project and SSH key.** Create a project (e.g. `salus`) and add your own public SSH key to it.
3. **Server.** Add a server: location **Helsinki** (`hel1`), image **Ubuntu 24.04**, type `SALUS_SERVER_TYPE`, a
   public IPv4 (IPv6 optional, see DNS), your SSH key. Note its IPv4 address: `LIVEKIT_NODE_IP` and the DNS
   records.
4. **Volume.** Volumes, Create volume: location **Helsinki**, attached to the server, a size for the database and
   the attachments with room to grow (it can be enlarged later; 50 GB is a start for a pilot), mount option
   **manual** (no automatic mount and no `/etc/fstab` line: the volume is encrypted on the server, step 4). If the
   console formats it anyway, `salus-volume setup` refuses it until you wipe it (step 4). Note the volume's id:
   on the server it is `/dev/disk/by-id/scsi-0HC_Volume_<id>`.
5. **Cloud Firewall (recommended).** Docker's published ports bypass `ufw`, so a Hetzner Cloud Firewall in front
   of the server is a second boundary: inbound TCP `SSH_PORT`, 80, 443, 7881 and UDP 443, 3478, 7882; nothing else.
   Apply it to the server.
6. **Storage Box.** Order a Storage Box (BX11 is plenty for nightly dumps) in Helsinki, enable SSH support, and
   create a sub-account for the backups with its own directory. Note its host, sub-account user and port 23:
   `BACKUP_HOST`, `BACKUP_USER`, `BACKUP_PORT` (docs/deploy/backup-restore.md "Storage Box").

## 3. DNS

At your DNS provider, for the domain `SALUS_DOMAIN`, all to the server's IPv4:

| Name | Type | Value |
| --- | --- | --- |
| `SALUS_DOMAIN` (apex, `@`) | A | server IPv4 |
| `app` | A | server IPv4 |
| `api` | A | server IPv4 |
| `livekit` | A | server IPv4 |
| `turn` | A | server IPv4 |
| `@` (optional) | CAA | `0 issue "letsencrypt.org"` |

No AAAA records: with Docker's default bridge, IPv6 clients would all reach the api from one address and share its
rate limits (`deployments/production/README.md`, "For DEP-3 and DEP-4"). A short TTL (300) for the first days helps
if the server changes. Caddy obtains the certificates once the names resolve.

## 4. Prepare the server

Follow `deployments/production/bootstrap/README.md`: fill in `bootstrap.conf` (both `DEPLOY_SSH_KEY` lines, the
`BACKUP_*` settings, `SSH_PORT`), copy it with the bootstrap directory to the server, and run `bootstrap.sh --check`
then `bootstrap.sh` from an interactive root session; check `ssh deploy@SERVER` from a second terminal before
closing it.

At the end the script prints the backup's public SSH key: add it to the Storage Box sub-account's
`.ssh/authorized_keys` (docs/deploy/backup-restore.md). It fails while swap is active (none on Hetzner's images) and
warns that the encrypted volume is not set up yet:

**The encrypted volume, once** (ADR 0011; deployments/production/README.md "Encrypted volume"). In the same root
session, with Bitwarden open:

```sh
ls -l /dev/disk/by-id/ | grep HC_Volume          # the volume of step 2.4, by its id
salus-volume setup /dev/disk/by-id/scsi-0HC_Volume_<id>
```

1. `cryptsetup` asks you to type `YES`, then for the LUKS passphrase twice (from Bitwarden, step 1), then once more
   to open it.
2. The script creates the filesystem, mounts it at `/mnt/salus-data`, lays out `postgres/`, `garage/` and `keys/`,
   and asks for `BINARY_KEK` and `TOTP_KEY` (paste each from Bitwarden; nothing is shown). It writes them to
   `/mnt/salus-data/keys/`, readable by the containers' uid only, and checks the result.
3. If it refuses because the device already holds a filesystem (the console formatted it): check in the console that
   the id is the new, empty volume's, then `wipefs -a /dev/disk/by-id/scsi-0HC_Volume_<id>` and run setup again.
4. `salus-volume check` must say ok. Nothing unlocks it at boot; see "Reboots" below.

A mount point other than `/mnt/salus-data` (`SALUS_DATA_DIR=... salus-volume setup ...`) also needs
`SALUS_DATA_DIR` in `SALUS_ENV_FILE`.

## 5. The server's host key

The deploy workflow pins the server's SSH host key. From your machine, once SSH works as `deploy`, scan exactly
the value you will put in `DEPLOY_HOST` (ssh matches the name it connects to; the IPv4 is simplest):

```sh
ssh-keyscan -p SSH_PORT -t ed25519 DEPLOY_HOST
```

Compare its fingerprint (`ssh-keygen -lf <(ssh-keyscan …)`) with the one you accepted on your first login (or the
server's `/etc/ssh/ssh_host_ed25519_key.pub`, `ssh-keygen -lf`). The line goes into the GitHub secret
`DEPLOY_KNOWN_HOSTS`. For a port other than 22 the line starts `[DEPLOY_HOST]:PORT`.

## 6. GitHub

In the repository's settings:

1. **Environment `production`** (Settings, Environments, New environment):
   - **Required reviewers**: yourself. Every deploy waits for your approval in the Actions tab.
   - **Deployment branches and tags**: "Selected branches and tags", `main` only.
   - The deploy refuses to run while this environment is missing or has no required reviewer.
   - **Environment secrets**:

     | Secret | Value |
     | --- | --- |
     | `SALUS_ENV_FILE` | The whole filled-in env file (step 1), pasted as is: without `BINARY_KEK` and `TOTP_KEY`, which the deploy refuses |
     | `DEPLOY_SSH_KEY` | The deploy workflow's private SSH key (`salus-deploy`, step 1), with its BEGIN and END lines |
     | `DEPLOY_KNOWN_HOSTS` | The host key line of step 5 |

2. **Repository variables** (Settings, Secrets and variables, Actions, Variables):

   | Variable | Value |
   | --- | --- |
   | `SALUS_DOMAIN` | The domain; must equal `SALUS_DOMAIN` in `SALUS_ENV_FILE` (the deploy checks) |
   | `DEPLOY_HOST` | The server's IPv4 (or a DNS name that is not behind a proxy) |
   | `DEPLOY_SSH_PORT` | Only if not 22 |
   | `DEPLOY_USER` | Only if not `deploy` |
   | `GHCR_OWNER` | Only if the images live in another namespace than `shmeeshmee` (then also in `SALUS_ENV_FILE`) |
   | `DEPLOY_ENABLED` | Leave unset until step 8 |

3. **Packages.** The Images workflow pushes `salus`, `salus-web`, `salus-landing` and `salus-caddy` to GHCR (the
   owner's packages). They are private by default. The deploy logs the server in to GHCR with the run's own
   read-only token, which works while each package grants this repository access: open each package's settings,
   "Manage Actions access", and check that `salus-ba` is listed with read access (packages pushed by this
   repository's workflow are linked to it). Making the packages public is not needed.

## 7. Images

Every merge to `main` runs the Images workflow after CI passes: it builds, scans (Trivy, critical vulnerabilities
fail it) and pushes the four images tagged with the commit SHA, and moves `main` when the commit is still main's
head. Check that the latest Images run of `main` succeeded with its `release` job, so there is a release to deploy.

## 8. The first deploy

1. In `SALUS_ENV_FILE`, set `SALUS_ACME_CA=https://acme-staging-v02.api.letsencrypt.org/directory` for the first
   run, so a mistake does not burn Let's Encrypt's production rate limits.
2. Set the repository variable `DEPLOY_ENABLED` to `true`.
3. Actions, Deploy, "Run workflow" on `main` (SHA empty: main's head; migrate on). Approve the `production`
   deployment when it asks.
4. The job uploads the release, pulls, migrates, waits for every healthcheck, and checks
   `https://SALUS_DOMAIN/healthz`, `https://app.SALUS_DOMAIN/healthz` and `https://api.SALUS_DOMAIN/healthz` from
   outside. With staging certificates the public check fails (they are not trusted); the rollback step finds no
   previous release and leaves the stack running. Look at it over SSH instead:
   `docker compose -p salus --env-file /opt/salus/.env --env-file /opt/salus/release.env -f /opt/salus/deployments/docker-compose.prod.yml ps`,
   and `curl -k --resolve app.SALUS_DOMAIN:443:127.0.0.1 https://app.SALUS_DOMAIN/healthz` on the server.
5. When everything is healthy: remove the `SALUS_ACME_CA` line from `SALUS_ENV_FILE` and run the deploy again
   **with the same SHA** (so a rollback could only go back to the same code). Caddy obtains the production
   certificates on start; the public check must pass. The staging certificates stay unused in Caddy's volume; to
   remove them:
   `docker compose -p salus --env-file /opt/salus/.env --env-file /opt/salus/release.env -f /opt/salus/deployments/docker-compose.prod.yml exec caddy rm -r /data/caddy/certificates/acme-staging-v02.api.letsencrypt.org-directory`.
6. **The first clinic and its admin**, on the server as `deploy` (`salusctl` runs as the database owner, so through
   the migrate service, which has the owner's connection):
   ```sh
   cd /opt/salus
   docker compose -p salus --env-file .env --env-file release.env -f deployments/docker-compose.prod.yml \
     run --rm --no-deps --entrypoint /usr/local/bin/salusctl migrate create-clinic -name "Clinic name" -admin-email admin@clinic.example
   ```
   The admin's invitation goes out by e-mail, which needs `SMTP_ADDR` (ADR 0011 open point 5).
7. **Backups**: on the next night, check `systemctl status salus-backup.service` and the Storage Box. Then run the
   restore drill (docs/deploy/backup-restore.md) and record it, before any real data.

From then on every merge to `main` deploys after your approval: Images, then Deploy. A deploy is refused while the
encrypted volume is locked (after a reboot, until you unlock it).

## Day to day

- **Reboots.** Automatic reboots are off (the bootstrap refuses them). When `/var/run/reboot-required` exists (a
  kernel or libc update), reboot by hand in a quiet hour and unlock in the same sitting: from Hetzner's console
  (or `ssh` as `deploy` with `DEPLOY_SUDO=yes`), as root, `reboot`, then after the boot `salus-volume unlock` and
  the passphrase from Bitwarden. It opens and mounts the volume, checks the layout and the key files, and starts
  the running release (`up -d --wait`, which runs `migrate` again: a no-op). Until then the stack is down, every
  deploy is refused, and a nightly backup that runs in between fails (run it by hand after the unlock:
  `systemctl start salus-backup.service`). After an unplanned reboot (Hetzner maintenance, a crash) the same applies:
  the stack waits for you.
- **A deploy takes the stack down briefly.** Every image changes with every release, so compose recreates every
  service; the app services wait for migrate. Expect a short outage per deploy (migrate plus start-up); deploy
  outside clinic hours.
- **A deploy failed.** `deploy.sh` brought the previous release back without running migrate (forward-only
  migrations: the previous code runs against the newer schema). The job's log (the server's
  `/opt/salus/releases/<sha>/deploy.log`) names the services that were not healthy or did not stay up. Fix forward
  with a new commit. The deploy runs detached on the server: a cancelled run or a dropped connection does not stop
  it half-way, and a second deploy started on the server meanwhile is refused until the first is done.
- **"Stopped without a result".** The detached deploy was killed or the server rebooted during it: the new
  release's files may be in place with nothing verified. Check `ps` and the deploy log, then deploy again or run
  `bash /opt/salus/releases/<sha>/deploy.sh rollback` as `deploy`. (It relies on logind's default
  `KillUserProcesses=no`, Ubuntu's default, so a process outlives the SSH session that started it.)
- **Go back to an older release by hand.** Actions, Deploy, "Run workflow" with its SHA and **migrate off**; the
  older code must accept the current schema (expand, then contract: ADR 0011).
- **Change a secret.** Update the Bitwarden note, paste the whole file into `SALUS_ENV_FILE`, run Deploy with the
  **same SHA** as the running release, on its own (not together with new code): a rollback restores the previous
  env file, so after `migrate` has set a rotated `APP_DATABASE_PASSWORD` or `RETENTION_DATABASE_PASSWORD`, a
  rollback would bring back the old password and fail. `POSTGRES_PASSWORD` cannot be changed this way (Postgres
  keeps the first one; `ALTER ROLE` first).
- **Look at the stack.** `ssh deploy@SERVER`, then the compose command above with `ps`, `logs --tail 100 <service>`.
  Logs carry no personal data by design; keep it that way when you paste them anywhere.
- **What the deploy keeps on the server.** `/opt/salus/releases/<sha>/` for the running and the previous release
  (each with its env file, mode 600), and their images; older ones are removed after a successful deploy.
