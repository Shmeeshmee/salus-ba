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
  (`/opt/salus/.env`) are in Bitwarden and the GitHub `production` environment, not in the backup.
- **Erasure.** A dump holds what the database held that night, so data erased since is still in the dumps for up
  to 14 days (unreadable without the age private key). A restore brings it back: after any restore, the erasures
  completed after the dump's time must be applied again before the stack is opened to users. The record of those
  erasures (`datarequest`) is in the same database and is lost with it, so the owner keeps a list outside the server
  of completed erasures (request id, patient id, date) for the last 14 days. How that list is kept is part of open
  point 7.

## How the backup works

`salus-backup.timer` starts `salus-backup.service` every night at `BACKUP_TIME` (plus up to `BACKUP_RANDOM_DELAY`;
a run missed while the server was off runs at the next boot). The service runs `/usr/local/sbin/salus-backup`
(`deployments/production/bootstrap/backup.sh`) as root:

1. Finds the running container of the `postgres` service in the compose project `salus` by its compose labels.
2. Runs `pg_dump -Fc` in it (as the image's superuser, over the local socket) and pipes the output through
   `age -r <BACKUP_AGE_RECIPIENT>`: the plaintext dump never touches a disk. Only the encrypted file is written, to
   `/opt/salus/backups/.partial-…` (root only).
3. Uploads it over SFTP to `BACKUP_PATH` on the Storage Box as `.partial-…` and renames it to
   `salus-YYYYMMDDTHHMMSSZ.dump.age` (UTC), so an interrupted upload never looks like a dump. SFTP runs with the
   backup's own key (`/root/.ssh/salus-backup`) and the Storage Box's pinned host key (`/etc/salus/backup_known_hosts`,
   `StrictHostKeyChecking=yes`): a changed host key stops the backup.
4. Deletes the dumps on the Storage Box whose name is older than 14 days; other files there are left alone.
5. Deletes the local encrypted file, unless `BACKUP_KEEP_LOCAL=yes` (then local copies are pruned the same way), and
   writes the time of the last success to `/opt/salus/backups/last-success`.

Any failure exits non-zero and the unit fails. Its log (file names, sizes, timings, errors; never rows) is in the
journal:

```sh
systemctl list-timers salus-backup.timer      # next and last run
systemctl status salus-backup.service         # last result
journalctl -u salus-backup.service -n 50
sudo systemctl start salus-backup.service     # a backup now
```

Nothing alerts on a failed backup yet; until monitoring exists, check `last-success` or the unit's status weekly.

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

**Record the result** in docs/deploy/restore-drills.md (one row each): date, the dump's name, its size, the time
from fetch to restored, the migration version, who ran it, and anything that went wrong. Synthetic data only until
real data exists; afterwards the counts, never contents.

## Restoring onto a new server

When the server is lost, or to move to another:

1. **Create the server** (ADR 0011: Ubuntu 24.04, Helsinki) and run the bootstrap with the same `bootstrap.conf`
   (README in `deployments/production/bootstrap/`). The new server has a new backup key: authorise it on the
   Storage Box (step 7 above) and remove the lost server's key there.
2. **Point the deploy at it**: the server's address in the deploy workflow's settings (docs/deploy/README.md), and
   the DNS records to the new IP (lower their TTL beforehand if you can).
3. **Bring up Postgres only**, after the deploy wrote `/opt/salus/.env`, `release.env` and the compose files (run the
   deploy workflow with the release that was running; let it fail at `migrate` if need be, or copy the files by
   hand), so nothing writes to the empty database:

   ```sh
   C="docker compose -p salus --env-file /opt/salus/.env --env-file /opt/salus/release.env -f /opt/salus/deployments/docker-compose.prod.yml"
   $C up -d --wait postgres
   ```

4. **Create the roles** as in step 3 of the drill (from a checkout of the running release, through `$C exec -T
   postgres psql …` on the server), then **restore**, streaming from your machine so the private key never reaches
   the server (the key file as in step 4 of the drill):

   ```sh
   # on your machine, with the encrypted dump fetched as in the drill
   C="docker compose -p salus --env-file /opt/salus/.env --env-file /opt/salus/release.env -f /opt/salus/deployments/docker-compose.prod.yml"
   docker run --rm -v "$PWD/drill.dump.age:/in.age:ro" -v /dev/shm/salus-drill/age.key:/key:ro ubuntu:24.04 sh -c \
     'apt-get update -qq && apt-get install -y -qq age >/dev/null && age -d -i /key /in.age' |
     ssh deploy@NEW-SERVER "$C exec -T postgres sh -c 'pg_restore -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" --exit-on-error'"
   rm -rf /dev/shm/salus-drill
   ```

5. **Re-apply the erasures** completed after the dump's time (the owner's list, see "Erasure" above) before anyone
   can sign in.
6. **Deploy** the running release normally: `migrate` sets the roles' passwords and applies any newer migration, then
   the rest of the stack starts. Check the healthchecks and sign in.
7. **Attachments**: the files in Garage are gone (not backed up); the rows that point to them stay. Until open point
   7 is decided, tell the clinics which uploads were lost.
8. Run a backup by hand on the new server and check it reaches the Storage Box.

Note the time each step took the first time, alongside the drill's, so the next restore can be planned.
