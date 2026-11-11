# Restore drills

One row per drill (docs/deploy/backup-restore.md "Restore drill"). Counts and versions only, never contents. A
restore onto a new server ("Restoring onto a new server") also creates and unlocks a new LUKS volume and writes the
key files from Bitwarden: note how long that took in Notes, and that `salus-volume check` passed.

| Date | Dump (size) | Objects archive (size) | Fetch to restored | Migration version | Objects restored | Rows without object / objects without row | Sample matched | Run by | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
