# Restore drills

One row per drill (docs/deploy/backup-restore.md "Restore drill"). Counts and versions only, never contents. A
restore onto a new server ("Restoring onto a new server") also creates and unlocks a new LUKS volume and writes the
key files from Bitwarden: note how long that took in Notes, and that `salus-volume check` passed.

| Date | Dump (size) | Objects archive (size) | Fetch to restored | Migration version | Objects restored | Rows without object / objects without row | Sample matched | Run by | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |

No production drill has run yet: there is no production data and no Storage Box. Run the first before the first real
patient data.

## Local drills

`scripts/restore-drill.sh` on a worktree's own compose stack (backup-restore.md "The local rehearsal"): synthetic
data, the real `backup.sh` to a throwaway SFTP server, the project's database and object store volumes removed and
restored. The times are a developer machine's, with two other stacks running.

| Date | Project, migration | Dump (size) | Objects archive (size) | Backup | Fetch to restored | Rows | Objects restored | Rows without object / objects without row | Attachment decrypted | Receipt PDF | Run by | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 2026-11-25 | salus-cms-rd4, 00086 | salus-20261125T230407Z.dump.age (905 526 B) | salus-objects-20261125T230410Z.tar.age (5 832 B) | 15 s (dump 1 s, objects 3 s; the rest stops the app and counts the rows) | 43 s | 95 tables, 5 018 rows, every count equal | 3 (2 457 B) | 0 / 0 | yes, SHA-1 equal through the api | 200, byte-identical to the one before the backup | RD4 (unattended run) | Third run. The first found the drill's own journey wrong (a practitioner cannot open an unsubmitted response's file; the journey now submits it) and that the restored database reopens every session of the dump (the runbook gained the sessions step, "Restoring onto a new server" step 9); the second proved the old session opens (200); the third closes every session and the old one is refused (401). Staff signed in again with password and TOTP (TOTP_KEY opens the restored secrets). The runbook's steps 1 to 9 worked as written; its erasure paragraph was out of date (fixed) |

## Migration drills

`scripts/migration-drill.sh` on the compose Postgres, in databases of its own.

| Date | Empty to head | From a 00059 dump to head | Checks | Run by |
| --- | --- | --- | --- | --- |
| 2026-11-25 | 00086, 2 s (17 s with the first build) | goose to 00059, `migration-drill/seed-00059.sql` (2 clinics, 6 accounts, 3 patients and consents, 4 slots, 3 visits booked, fulfilled and cancelled, an encounter, a questionnaire answered, a chat message, 20 audit events), `pg_dump -Fc`, `pg_restore --exit-on-error` into a new database, `migrate up` to 00086: 6 s | every count equal; every visit, slot and rule video (00059); every clinic has its first location (00080); visits on their slots, the cancelled one and the revoked consent unchanged; row-level security as salus_app: the other clinic sees none of the visits, the clinic all 3 | RD4 |
