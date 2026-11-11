# Backups and restore

The production database is backed up every night to a Hetzner Storage Box, encrypted before it leaves the server
(docs/adr/0011-single-server-deployment.md). This page covers how that works, how to set up the Storage Box, the
restore drill to run before the first real data (and after any change to the backup), and the restore onto a new
server.

## What is and is not backed up

- **Backed up:** the Postgres database of the `salus` compose project, every night, as one `pg_dump` in custom
  format (`-Fc`), kept 14 days (`BACKUP_RETENTION_DAYS`).
- **Not backed up: Garage's attachments** (questionnaire uploads, documents and any other file in the object store).
  Losing the server loses those files, while the restored rows still point to them. This is open point 7 of ADR 0011
  and needs the owner's decision before real patient data.
- **Not in the dump:** the cluster-wide Postgres roles (`salus_app` and the others the migrations create) and their
  passwords; the restore creates the roles first and the deploy's `migrate` sets the passwords again. The secrets
  (`/opt/salus/.env`) are in Bitwarden and the GitHub `production` environment, not in the backup; `BINARY_KEK` and
  `TOTP_KEY` are in the key files on the encrypted volume and in Bitwarden only, and the volume's LUKS passphrase in
  Bitwarden only.
- **Keys the restored data needs.** `BINARY_KEK` wraps every attachment's data key and `TOTP_KEY` seals the MFA
  secrets stored in the database. A restore must run with **exactly the values the lost server had** (Bitwarden,
  written into the new volume's key files by `salus-volume setup`):
  with any other value the restored MFA secrets cannot be opened (every user with MFA is locked out until an operator
  resets it, `salusctl reset-mfa`) and any surviving attachment cannot be decrypted. Never generate new ones for a
  restore.
- **Erasure.** A dump holds what the database held that night, so data erased since is still in the dumps for up
  to 14 days (unreadable without the age private key). A restore brings it back, and the erasures completed after
  the dump's time must be applied again before the stack is opened to users. The record of those erasures would be
  in the same database and is lost with it, so the owner has to keep a list outside the server of completed erasures
  (request id, patient id, date) for the last 14 days. **Today there is nothing to re-apply them with:** erasure is
  part of the subject-rights feature, which is not built yet (`internal/retention` leaves erasure to it, and
  `salusctl` has no erasure command). Until it is, an erasure would be done by hand and would have to be redone by
  hand after a restore. Building the re-application (for example a `salusctl` command that takes the list) and
  keeping the list are open items for the owner, with open point 7 of ADR 0011.

## How the backup works

`salus-backup.timer` starts `salus-backup.service` every night at `BACKUP_TIME` (plus up to `BACKUP_RANDOM_DELAY`;
a run missed while the server was off runs at the next boot). The service runs `/usr/local/sbin/salus-backup`
(`deployments/production/bootstrap/backup.sh`) as root, with its settings from `/etc/salus/bootstrap.conf`, at most
one at a time and for at most 2 hours:

1. Finds the container of the `postgres` service in the compose project `salus` by its compose labels, and waits up
   to `BACKUP_WAIT_SECONDS` (600) for it to be healthy: after a boot the missed run can start before the stack is up.
2. Runs `pg_dump -Fc` in it (as the `postgres` OS user, connecting over the local socket as the image's superuser,
   `POSTGRES_USER`, to `POSTGRES_DB`) and pipes the output through `age -r <BACKUP_AGE_RECIPIENT>`: the plaintext
   dump never touches a disk. Only the encrypted file is written, to `/opt/salus/backups/.partial-…` (root only).
3. Uploads it over SFTP to `BACKUP_PATH` on the Storage Box as `.partial-…` and renames it to
   `salus-YYYYMMDDTHHMMSSZ.dump.age` (UTC), so an interrupted upload never looks like a dump. SFTP runs with the
   backup's own key (`/root/.ssh/salus-backup`) and the Storage Box's pinned host key (`/etc/salus/backup_known_hosts`,
   `StrictHostKeyChecking=yes`): a changed host key stops the backup.
4. Deletes the dumps on the Storage Box whose name is older than 14 days and the `.partial-…` files earlier failed
   uploads left; other files there are left alone. It never deletes the last dump: if no dump newer than 14 days is
   there (the backups have failed for two weeks), the old ones stay and the log says so.
5. Deletes the local encrypted file, unless `BACKUP_KEEP_LOCAL=yes` (then local copies are pruned the same way), and
   writes the time of the last success to `/opt/salus/backups/last-success`.

Any failure exits non-zero and the unit fails. Its log (file names, sizes, timings, errors; never rows) is in the
journal:

```sh
systemctl list-timers salus-backup.timer      # next and last run
systemctl status salus-backup.service         # last result
journalctl -u salus-backup.service -n 50
journalctl -t salus-backup -p err             # the failed nights
sudo systemctl start salus-backup.service     # a backup now
```

**A full Storage Box** shows as a failed upload. The script then still prunes (old dumps and partial uploads, never
the last dump), so space older than the retention is freed, and fails; that night has no backup. Check the space with
`echo 'df -h' | sftp -b - -P 23 USER@HOST` and order a bigger box or shorten the retention.

### Alerting

On a failure, `OnFailure=` starts `salus-backup-failed.service`, which writes "salus-backup FAILED" to the journal at
err priority (`journalctl -t salus-backup -p err`). Nothing outside the server is told yet. To be told, add a drop-in
to that unit, e.g. a ping to a dead-man's-switch service that alerts when it hears nothing or a failure:

```sh
sudo systemctl edit salus-backup-failed.service
# [Service]
# ExecStart=/usr/bin/curl -fsS -m 10 --retry 3 https://ALERT-SERVICE/FAILURE-URL
```

The ping carries no personal data, but the service is still a third party: add it to docs/privacy/processors.md
first. A drop-in survives bootstrap runs (the bootstrap rewrites only the unit itself). Until then, check
`last-success` or the unit's status weekly.

## Setting up the Storage Box

1. **Order a Storage Box** in Hetzner's console in the same region as the server (Helsinki), the smallest size is
   plenty for a pilot's dumps. Hetzner's DPA covers it (docs/privacy/processors.md).
2. **Turn on SSH support** for it (in the Storage Box's settings), which serves SFTP on **port 23**.
3. **Create a sub-account** for the backups, with its own base directory, SSH support on and external reachability
   on. Its user (`uXXXXXX-subN`) and host (`uXXXXXX-subN.your-storagebox.de` or the main box's
   host) go into `BACKUP_USER` and `BACKUP_HOST`; `BACKUP_PATH` is relative to the sub-account's base directory.
4. **Pin the host key.** `ssh-keyscan -p 23 -t ed25519 HOST`, compare its fingerprint (`ssh-keygen -lf -`) with
   the one Hetzner publishes for Storage Boxes, then put the line in `BACKUP_KNOWN_HOST`.
5. **Create the age key pair** on your own machine, never on the server:
   `docker run --rm -it ubuntu:24.04 sh -c 'apt-get update -qq && apt-get install -y -qq age >/dev/null && age-keygen'`.
   Store the whole output (the `AGE-SECRET-KEY-…` line is the private key) in Bitwarden as "Salus backup age key";
   put the `age1…` public key in `BACKUP_AGE_RECIPIENT`. Without the private key no dump can be restored.
6. **Run the bootstrap** with the settings complete. It prints the backup's public key at the end.
7. **Authorise that key on the sub-account:** add it in the console's SSH keys for the Storage Box, or from the
   server: `ssh-copy-id -p 23 -s -i /root/.ssh/salus-backup.pub USER@HOST` (password of the sub-account, once).
   Run the bootstrap again: it enables the timer once the settings are complete.
8. **Run a backup by hand** (`sudo systemctl start salus-backup.service`), check the log, then run the drill below.

For the drill, also authorise your own key on the sub-account (read access is all it needs); the server's key stays
on the server.

## Restore drill

Run before the first real patient data, then every quarter and after any change to the backup. It proves a dump can
be fetched, decrypted and restored, and measures how long it takes. It runs on your own machine with Docker and a
checkout of the repository; the decrypted dump only ever streams from one process to the next. Use the newest dump.

```sh
start=$(date +%s)
# 1. Fetch the newest dump (encrypted).
echo 'ls -1 salus/postgres' | sftp -q -b - -P 23 USER@HOST | grep '\.dump\.age$' | sort | tail -1   # its name
sftp -P 23 USER@HOST:salus/postgres/salus-YYYYMMDDTHHMMSSZ.dump.age ./drill.dump.age

# 2. A throwaway Postgres: the dev compose file's postgres (the production major version), in a project of its own,
#    no host port.
DRILL="docker compose -p salus-restore-drill -f deployments/docker-compose.yml"
$DRILL up -d --wait postgres

# 3. The roles the migrations create: the dump grants to them but does not create them.
grep -ho 'CREATE ROLE [a-z_]* [A-Z ]*' migrations/*.sql | sort -u | sed 's/ *$/;/' |
  $DRILL exec -T postgres psql -U salus -d salus -v ON_ERROR_STOP=1

# 4. The private key from Bitwarden into a RAM-backed file (/dev/shm on Linux), never onto a disk: paste the
#    AGE-SECRET-KEY-... line, then Ctrl-D.
(umask 077 && mkdir -p /dev/shm/salus-drill && cat >/dev/shm/salus-drill/age.key)

# 5. Decrypt and restore in one stream. If production's database owner is not `salus`, add --no-owner.
docker run --rm -v "$PWD/drill.dump.age:/in.age:ro" -v /dev/shm/salus-drill/age.key:/key:ro ubuntu:24.04 sh -c \
  'apt-get update -qq && apt-get install -y -qq age >/dev/null && age -d -i /key /in.age' |
  $DRILL exec -T postgres pg_restore -U salus -d salus --exit-on-error
rm -rf /dev/shm/salus-drill

# 6. Sanity queries: the migration version and the row counts.
$DRILL exec -T postgres psql -U salus -d salus \
  -c "SELECT max(version_id) AS migration FROM goose_db_version WHERE is_applied" \
  -c "ANALYZE" -c "SELECT relname, n_live_tup FROM pg_stat_user_tables ORDER BY relname"
echo "restored in $(( $(date +%s) - start )) s"

# 7. Clean up: the throwaway database and the encrypted file.
$DRILL down -v
rm -f drill.dump.age
```

The migration version must equal the release's latest migration on the night of the dump, and the counts must be
plausible against production's (compare `SELECT count(*)` of a few tables on the server, read-only). Do not open any
patient's record: counts and the version are enough.

**The keys and the passphrase, on the server** (as root): `salus-volume keys`, pasting `BINARY_KEK` and `TOTP_KEY`
from Bitwarden, must answer `binary_kek unchanged` and `totp_key unchanged`. Any other answer means Bitwarden does not
hold the keys a restore needs (it refuses to change a key file without `--replace`, so nothing changes): find the
right copies before anything else. The LUKS passphrase is proved at every unlock; after a planned reboot, unlock
with Bitwarden's copy, not from memory.

**Record the result** in docs/deploy/restore-drills.md (one row each): date, the dump's name, its size, the time
from fetch to restored, the migration version, who ran it, and anything that went wrong. Synthetic data only until
real data exists; afterwards the counts, never contents.

## Restoring onto a new server

When the server is lost, or to move to another. Nothing may reach the database before the restore, and no user may
reach the stack until the erasures are re-applied, so the stack is brought up in stages: Postgres alone, the restore,
everything but Caddy, then Caddy.

1. **Create the server** (ADR 0011: Ubuntu 24.04, Helsinki) and run the bootstrap with the same `bootstrap.conf`
   (README in `deployments/production/bootstrap/`). The new server has a new backup key: authorise it on the
   Storage Box (step 7 above) and remove the lost server's key there.

   **A new encrypted volume**, before anything else runs: create a Hetzner Volume in Helsinki, attached to the new
   server, mount option manual (docs/deploy/README.md step 2.4), and as root on the server run
   `salus-volume setup /dev/disk/by-id/scsi-0HC_Volume_<id>` (README step 4) with a new LUKS passphrase (store it in
   Bitwarden in place of the lost volume's) and **the lost server's `BINARY_KEK` and `TOTP_KEY` from Bitwarden**,
   unchanged. `salus-volume check` must say ok.

   If only the server was lost and its volume survived, attach that volume instead, write `/etc/salus/volume.conf`
   by hand (`DEVICE=/dev/disk/by-uuid/<cryptsetup luksUUID of the device>`, `NAME=salus-data`,
   `MOUNTPOINT=/mnt/salus-data`) and run `salus-volume unlock --no-start`: the data is all there, and no restore is
   needed; go on with step 3, then step 7.
2. **Keep the deploy workflow away** until step 9: do not approve a `production` deployment, so nothing runs
   `migrate` or starts the stack against an empty database.
3. **Put the release's files in place by hand**, as the deploy job would (docs/deploy/README.md):
   - the contents of `deployments/production/` of the release that was running, into `/opt/salus/deployments/`;
   - `/opt/salus/.env` (owner `deploy`, mode 600) from Bitwarden's env note (without `BINARY_KEK` and `TOTP_KEY`:
     they are in the key files of step 1);
   - `/opt/salus/release.env` with `SALUS_TAG=<that release's commit SHA>`;
   - a copy of both in `/opt/salus/releases/<that SHA>/` (the deployments files and `.env`, mode 600), so the next
     deploy can roll back to this release (`deployments/production/deploy.sh` keeps each release there).

   On the server, as `deploy`:

   ```sh
   C="docker compose -p salus --env-file /opt/salus/.env --env-file /opt/salus/release.env -f /opt/salus/deployments/docker-compose.prod.yml"
   $C pull
   $C up -d --wait postgres          # Postgres only, on the encrypted volume: no migrate, no app, no Caddy
   ```

4. **Start from an empty database.** On a new volume the image created it empty. If anything ran against it, drop
   and recreate it:

   ```sh
   $C exec -T -u postgres postgres sh -c 'dropdb -U "$POSTGRES_USER" --if-exists --force "$POSTGRES_DB" &&
     createdb -U "$POSTGRES_USER" -O "$POSTGRES_USER" "$POSTGRES_DB"'
   ```

5. **Create the roles**, as in step 3 of the drill, from a checkout of the running release on your machine:

   ```sh
   grep -ho 'CREATE ROLE [a-z_]* [A-Z ]*' migrations/*.sql | sort -u | sed 's/ *$/;/' |
     ssh deploy@NEW-SERVER "$C exec -T postgres sh -c 'psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -v ON_ERROR_STOP=1'"
   ```

   (Here and below `C` is set on your machine to the same compose command, so it expands there.) The new cluster has
   none of these roles yet; Postgres's own superuser is the image's `POSTGRES_USER`.
6. **Restore**, streaming from your machine so the private key never reaches the server (the key file as in step 4
   of the drill, the encrypted dump fetched as in step 1):

   ```sh
   docker run --rm -v "$PWD/drill.dump.age:/in.age:ro" -v /dev/shm/salus-drill/age.key:/key:ro ubuntu:24.04 sh -c \
     'apt-get update -qq && apt-get install -y -qq age >/dev/null && age -d -i /key /in.age' |
     ssh deploy@NEW-SERVER "$C exec -T postgres sh -c 'pg_restore -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" --exit-on-error'"
   rm -rf /dev/shm/salus-drill
   ```

7. **Everything but Caddy.** `migrate` sets the roles' passwords (and applies nothing new: the release is the one the
   dump came from), then the rest starts; without Caddy nothing is reachable from outside:

   ```sh
   $C run --rm migrate
   $C up -d --wait $($C config --services | grep -vx caddy)
   ```

8. **Re-apply the erasures** completed after the dump's time, from the owner's list ("Erasure" above), while Caddy is
   still down. There is no tool for this yet: do it the same way the erasure was done, and record it.
9. **Open it:** point the DNS records at the new IP (lower their TTL beforehand if you can) and the deploy
   workflow's settings at the new server (docs/deploy/README.md), then `$C up -d --wait caddy`, or approve a deploy
   of the same release, which brings up everything. Check the healthchecks and sign in.
10. **Attachments**: the files in Garage are gone (not backed up); the rows that point to them stay. Until open point
    7 is decided, tell the clinics which uploads were lost.
11. Run a backup by hand on the new server and check it reaches the Storage Box.

Note the time each step took the first time, alongside the drill's, so the next restore can be planned.
