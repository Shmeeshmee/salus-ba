# Backups and restore

The production database and Garage's objects (the attachments) are backed up every night to a Hetzner Storage
Box, encrypted before they leave the server (docs/adr/0011-single-server-deployment.md); a failed or missing backup
is mailed to the operator. This page covers how that works, how to set up the Storage Box, the
restore drill to run before the first real data (and after any change to the backup), and the restore onto a new
server.

## What is and is not backed up

- **Backed up:** the Postgres database of the `salus` compose project, every night, as one `pg_dump` in custom
  format (`-Fc`); and right after it every object of Garage's bucket (`S3_BUCKET`: the questionnaire attachments
  and the signed documents' PDFs under `composition/`) as one tar. Both are kept 14 days (`BACKUP_RETENTION_DAYS`);
  how long backups are kept is still open (ADR 0011 point 7).
- **The objects are encrypted twice.** Each object is already envelope-encrypted by the api (its data key wrapped by
  `BINARY_KEK`, which the backup never reads); the archive is then `age`-encrypted like the dump. A restored object is
  readable only with the restored database's wrapped key and the lost server's `BINARY_KEK`.
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
- **Erasure.** A dump holds what the database held that night (an archive the objects of that night), so data erased since is still in the dumps for up
  to 14 days (unreadable without the age private key). A restore brings it back, and the erasures completed after
  the dump's time must be applied again before the stack is opened to users. The record of those erasures would be
  in the same database and is lost with it, so the owner has to keep a list outside the server of completed erasures
  (request id, patient id, date) for the last 14 days. **Today there is nothing to re-apply them with:** erasure is
  part of the subject-rights feature, which is not built yet (`internal/retention` leaves erasure to it, and
  `salusctl` has no erasure command). Until it is, an erasure would be done by hand and would have to be redone by
  hand after a restore. Building the re-application (for example a `salusctl` command that takes the list) and
  keeping the list are open items for the owner, with point 7 of ADR 0011.

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
5. Deletes the local encrypted file, unless `BACKUP_KEEP_LOCAL=yes` (then local copies of the dumps are pruned the
   same way, never the last one), and writes the time of the last success to `/opt/salus/backups/last-success`.

Then `salus-backup-objects.service` (`salus-backup objects`) backs up Garage's bucket the same way: it waits for the
`objectstore` container to be healthy, reads `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY` and `S3_BUCKET` from
`/opt/salus/.env` (`BACKUP_ENV_FILE`, parsed, never sourced), and runs a pinned `rclone` image on that container's
compose network as nobody, read-only, with no capabilities and the key pair in its environment only. `rclone archive
create --format tar` streams every object under its key to stdout, through `age`, into a second container of the
same image that writes the stream over SFTP (`rclone rcat`, the backup's SSH key and the pinned host key copied in
with `docker cp`, so they are in no command line) to `BACKUP_OBJECTS_PATH/.partial-salus-objects-YYYYMMDDTHHMMSSZ.tar.age`
on the Storage Box, renamed once complete. Nothing of the archive is written to the server's disk, whatever the
bucket's size, and `BACKUP_KEEP_LOCAL` does not apply to it. When any stage fails (Garage refuses the key, the
upload breaks) the partial file is removed and nothing that looks like an archive is left. The archives are pruned
like the dumps (14 days, never the last one); the success goes to `last-success-objects`. It reads Garage through
its S3 API only, so it does not depend on where Garage keeps its files on the host.

**Order and consistency.** The timer starts `salus-backup.service`, which `Wants=` the objects unit; the objects
unit is ordered `After=` it, so the archive starts once the dump has ended, whatever its result. The two are close
together but not one snapshot, and the api keeps writing in between:

- An upload stores its object before its row, and a row is never written without its object, so every row in the
  dump has its object in the archive taken after it, except for a file deleted (by the patient, a discard or the
  retention job, which runs at 03:00 UTC) between the dump and the archive: after a restore that row points to no
  object and its download fails, as it would have failed on the old server a moment later.
- Objects written after the dump (uploads of those minutes) are in the archive without a row. Nothing refers to
  them; the retention job's purger deletes every object older than 24 hours that has no row, so they go by
  themselves after a restore.

The drill below counts both.

Any failure exits non-zero and the unit fails. Its log (file names, sizes, timings, errors; never rows or object
contents) is in the journal:

```sh
systemctl list-timers 'salus-backup*'                         # next and last runs
systemctl status salus-backup.service salus-backup-objects.service
journalctl -u salus-backup.service -u salus-backup-objects.service -n 50
journalctl -t salus-backup -p err                             # the failed nights
sudo systemctl start salus-backup.service                     # both backups now, the dump first
sudo systemctl start salus-backup-objects.service             # the objects alone
```

**While the encrypted volume is locked** (after a reboot, before the owner unlocks it and starts the stack), no
`postgres` or `objectstore` container runs: both backups wait `BACKUP_WAIT_SECONDS`, fail and send an alert. That is
the intended reminder; unlock, start the stack and run `sudo systemctl start salus-backup.service`.

**A full Storage Box** shows as a failed upload. The script then still prunes (old dumps and partial uploads, never
the last dump), so space older than the retention is freed, and fails; that night has no backup. Check the space with
`echo 'df -h' | sftp -b - -P 23 USER@HOST` and order a bigger box or shorten the retention.

### Alerting

Two things send an e-mail to `ALERT_EMAIL` (bootstrap.conf):

- **A failed backup.** Each backup unit has `OnFailure=salus-backup-failed@%n.service`, which writes "salus-backup
  FAILED" to the journal at err priority and runs `salus-backup alert <unit>`: subject "Salus backup failed on
  HOST: UNIT", with the time, systemd's result and the exit status.
- **A missing backup.** `salus-backup-check.timer` runs `salus-backup check` every day at `BACKUP_CHECK_TIME` (08:00,
  `Persistent=true`). It lists both directories on the Storage Box and mails "Salus backup missing on HOST" when the
  newest dump or archive (by the time in its name) is older than `BACKUP_MAX_AGE_HOURS` (26) or there is none, or
  the box cannot be listed (sftp's own error is in the journal). This catches what no unit failure reports: a timer
  that never ran, a server that was off, a file deleted on the box. Once it has sent that mail the check unit
  succeeds (the problems are logged at err priority), so there is one mail, not two; it fails, and its
  `OnFailure=salus-backup-failed@%n.service` tries to mail, only when the check could not run or could not send.

The mail goes through the same SMTP relay as the notifier, configured in `/opt/salus/.env`: `SMTP_ADDR` (host:port),
`SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_TLS` (`starttls`, the default when unset, requires STARTTLS; `tls` is
implicit TLS; `none` only for a local test relay, and refused with a login, as the notifier refuses it) and
`SMTP_FROM` (the envelope sender is the address in its angle brackets). The file is read as compose reads these
values, never sourced: an unquoted value ends before ` #`, a line that is no `KEY=value` is skipped, and a value
with `$`, `#`, `"` or `\` in it (a generated password often has one) must be single-quoted, as compose interpolates
unquoted and double-quoted values. It is sent with the host's `curl`, the login passed on curl's standard input, never its command line. It
carries the host name, the unit or backup kind, the time and the exit status only: no file contents, no personal
data. A send that fails is logged at err priority (`journalctl -u 'salus-backup-failed@*' -u salus-backup-check`).
With `ALERT_EMAIL` empty nothing is mailed (the bootstrap warns); check the journal weekly instead.

Send a test alert after setting it up (its subject says "failed"; it is a test):

```sh
sudo salus-backup alert salus-backup.service
sudo systemctl start salus-backup-check.service && journalctl -u salus-backup-check.service -n 5
```

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
8. **Run a backup by hand** (`sudo systemctl start salus-backup.service`, which runs both), check both logs, then
   run the drill below.

For the drill, also authorise your own key on the sub-account (read access is all it needs); the server's key stays
on the server.

## Restore drill

Run before the first real patient data, then every quarter and after any change to the backup. It proves a dump and
an objects archive can be fetched, decrypted and restored, that the restored rows find their objects, and measures
how long it takes. It runs on your own machine with Docker and a
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

# 7. The newest objects archive of the same night (its time just after the dump's), fetched and decrypted. The tar
#    holds the objects as Garage stored them, still envelope-encrypted (unreadable without BINARY_KEK); keep it in
#    the drill's directory and delete it at the end. Paste the key again as in step 4.
echo 'ls -1 salus/objects' | sftp -q -b - -P 23 USER@HOST | grep '\.tar\.age$' | sort | tail -1   # its name
sftp -P 23 USER@HOST:salus/objects/salus-objects-YYYYMMDDTHHMMSSZ.tar.age ./drill.objects.tar.age
(umask 077 && mkdir -p /dev/shm/salus-drill && cat >/dev/shm/salus-drill/age.key)
docker run --rm -v "$PWD/drill.objects.tar.age:/in.age:ro" -v /dev/shm/salus-drill/age.key:/key:ro ubuntu:24.04 sh -c \
  'apt-get update -qq && apt-get install -y -qq age >/dev/null && age -d -i /key /in.age' >drill.objects.tar
rm -rf /dev/shm/salus-drill

# 8. Into the dev compose file's Garage (the production image) in the drill's project, with rclone (the backup's
#    pinned image) on its network. The dev Garage's synthetic key pair and bucket are in deployments/docker-compose.yml.
$DRILL up -d --wait objectstore
RCLONE="docker run --rm --network salus-restore-drill_default -v $PWD/drill.objects.tar:/in.tar:ro
  -e RCLONE_CONFIG_D_TYPE=s3 -e RCLONE_CONFIG_D_PROVIDER=Other -e RCLONE_CONFIG_D_REGION=garage
  -e RCLONE_CONFIG_D_ENDPOINT=http://objectstore:3900 -e RCLONE_CONFIG_D_ACCESS_KEY_ID=GK5a1a50000000000000000001
  -e RCLONE_CONFIG_D_SECRET_ACCESS_KEY=5a1a500000000000000000000000000000000000000000000000000000000001
  rclone/rclone:1.75.2@sha256:2687085f718d3c628f7fdfb77c52a1d344332aed543110bb88088f2c70d43eb5 --config="
$RCLONE archive extract /in.tar D:salus-binaries
$RCLONE size D:salus-binaries                    # objects and bytes: compare with production's (below)

# 9. Rows and objects: the attachments' rows without an object, and the objects without a row (composition/ PDFs
#    are another table's and are left out). Both should be zero or a handful from the minutes between the dump and
#    the archive ("Order and consistency" above). Rejected files have no object by design.
$DRILL exec -T postgres psql -U salus -d salus -Atc "SELECT id FROM \"binary\" WHERE status <> 'rejected' ORDER BY 1" >drill.rows
$RCLONE lsf --files-only D:salus-binaries | grep -v / | sort >drill.objects
echo "rows without an object: $(comm -23 drill.rows drill.objects | wc -l)"
echo "objects without a row: $(comm -13 drill.rows drill.objects | wc -l)"

# 10. A sample attachment downloads: one restored object's bytes equal production's. Pick an id from drill.rows,
#     then on the server (read-only, the same rclone on the salus network with the env file's key pair):
$RCLONE md5sum D:salus-binaries/SAMPLE-ID
#   ssh deploy@SERVER 'set -a; . <(grep -E "^S3_(ACCESS_KEY_ID|SECRET_ACCESS_KEY|BUCKET)=" /opt/salus/.env); set +a;
#     docker run --rm --network salus_default -e RCLONE_CONFIG_P_TYPE=s3 -e RCLONE_CONFIG_P_PROVIDER=Other
#       -e RCLONE_CONFIG_P_REGION=garage -e RCLONE_CONFIG_P_ENDPOINT=http://objectstore:3900
#       -e RCLONE_CONFIG_P_ACCESS_KEY_ID="$S3_ACCESS_KEY_ID" -e RCLONE_CONFIG_P_SECRET_ACCESS_KEY="$S3_SECRET_ACCESS_KEY"
#       rclone/rclone:1.75.2@sha256:2687085f718d3c628f7fdfb77c52a1d344332aed543110bb88088f2c70d43eb5 --config=
#       md5sum "P:${S3_BUCKET:-salus-binaries}/SAMPLE-ID"; ...same with size "P:${S3_BUCKET:-salus-binaries}"'
#     The two checksums must match. That the file also opens (decrypted with BINARY_KEK) is checked on a real restore
#     ("Restoring onto a new server", steps 8 and 10), where the production key is in place.
echo "restored in $(( $(date +%s) - start )) s"

# 11. Clean up: the throwaway database and store, the decrypted tar and the encrypted files.
$DRILL down -v
rm -f drill.dump.age drill.objects.tar.age drill.objects.tar drill.rows drill.objects
```

The migration version must equal the release's latest migration on the night of the dump, and the counts must be
plausible against production's (compare `SELECT count(*)` of a few tables on the server, read-only). Do not open any
patient's record: counts and the version are enough.

**The keys and the passphrase, on the server** (as root): `salus-volume keys`, pasting `BINARY_KEK` and `TOTP_KEY`
from Bitwarden, must answer `binary_kek unchanged` and `totp_key unchanged`. Any other answer means Bitwarden does not
hold the keys a restore needs (it refuses to change a key file without `--replace`, so nothing changes): find the
right copies before anything else. The LUKS passphrase is proved at every unlock; after a planned reboot, unlock
with Bitwarden's copy, not from memory.

**Record the result** in docs/deploy/restore-drills.md (one row each): date, the dump's and the archive's names and
sizes, the time from fetch to restored, the migration version, the objects restored, the rows without an object and
the objects without a row, whether the sample matched, who ran it, and anything that went wrong. Synthetic data only
until real data exists; afterwards the counts, never contents.

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
   needed; go on with step 3, then steps 7 and 10 (the objects and the erasures are still there).
2. **Keep the deploy workflow away** until step 10: do not approve a `production` deployment, so nothing runs
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

8. **Attachments**, while Caddy is still down: restore the objects of the same night. Fetch
   and decrypt the archive on your machine as in drill step 7, and stream the tar onto the encrypted volume
   (`SALUS_DATA_DIR`, /mnt/salus-data; never the server's own disk), then extract it into Garage with the stack's own
   key pair, on the server as `deploy` (set `R` on both machines):

   ```sh
   R=rclone/rclone:1.75.2@sha256:2687085f718d3c628f7fdfb77c52a1d344332aed543110bb88088f2c70d43eb5
   # on your machine:
   ssh deploy@NEW-SERVER "docker run --rm -i -v /mnt/salus-data:/d --entrypoint sh $R -c 'cat >/d/objects.tar'" <drill.objects.tar
   # on the server:
   set -a; . <(grep -E '^S3_(ACCESS_KEY_ID|SECRET_ACCESS_KEY|BUCKET)=' /opt/salus/.env); set +a
   docker run --rm --network salus_default -v /mnt/salus-data/objects.tar:/in.tar:ro \
     -e RCLONE_CONFIG_P_TYPE=s3 -e RCLONE_CONFIG_P_PROVIDER=Other -e RCLONE_CONFIG_P_REGION=garage \
     -e RCLONE_CONFIG_P_ENDPOINT=http://objectstore:3900 -e RCLONE_CONFIG_P_ACCESS_KEY_ID="$S3_ACCESS_KEY_ID" \
     -e RCLONE_CONFIG_P_SECRET_ACCESS_KEY="$S3_SECRET_ACCESS_KEY" "$R" --config= \
     archive extract /in.tar "P:${S3_BUCKET:-salus-binaries}"
   docker run --rm -v /mnt/salus-data:/d --entrypoint rm "$R" /d/objects.tar
   ```

   In step 10, once the stack is open, sign in as staff and download one attachment of a record you are entitled to see
   (with synthetic data before real data exists): it proves row, object and `BINARY_KEK` fit together. Files
   uploaded after the archive's time are lost; tell the clinics which (their rows point to no object).
9. **Re-apply the erasures** completed after the dump's time, from the owner's list ("Erasure" above), while Caddy is
   still down. There is no tool for this yet: do it the same way the erasure was done, and record it.
10. **Open it:** point the DNS records at the new IP (lower their TTL beforehand if you can) and the deploy
    workflow's settings at the new server (docs/deploy/README.md), then `$C up -d --wait caddy`, or approve a deploy
    of the same release, which brings up everything. Check the healthchecks, sign in and download a sample
    attachment (step 8).
11. Run a backup by hand on the new server and check both reach the Storage Box; send a test alert.

Note the time each step took the first time, alongside the drill's, so the next restore can be planned.
