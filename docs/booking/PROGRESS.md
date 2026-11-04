# Booking progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Retention job | done | `booking(1)` |
| 2 Patient | to do | |
| 3 Consent and access | to do | |
| 4 Directory additions | to do | |
| 5 Schedule and slot | to do | |
| 6 Appointment | to do | |
| 7 FHIR | to do | |
| 8 Web | to do | |
| 9 Mobile | to do | |
| 10 End-to-end | to do | |

## Decisions

- 2026-11-04, owner (changes on approval, recorded in `SPEC.md` §2 "Changed on approval" and §3):
  - Revocation enqueues `consent_revoked` (patient id, clinic id) in its transaction; the worker cancels the
    future appointments and sends the e-mails. Access ends at the revocation's commit. Dependency
    appointment → consent only; no function value in `app.go`.
  - Lawful basis: care records under Art. 9(2)(h), consent for pre-care sharing and new bookings. After a
    revocation the clinic keeps minimal read access to past appointments (name, times, status) and loses the
    profile and anything future. **Needs legal confirmation before production.**
  - `salus_retention` has its own credentials, `RETENTION_DATABASE_URL`, loaded only for the retention job,
    with tests that the api and the other jobs cannot use them.
  - Slot status changes in the appointment's transaction; a concurrent book-and-cancel test proves status
    and the exclusion constraint agree.
  - Specialties: general practice, internal medicine, cardiology, dermatology, gynaecology, paediatrics,
    psychiatry, neurology, ENT, ophthalmology, orthopaedics, endocrinology (c80 practice codes).
  - Everything else in `SPEC.md` approved as written.

## Waiting on people

- Before production: a legal decision on the retention of `appointment` and `consent`, and legal
  confirmation of the lawful basis split (`SPEC.md` §3, §4).
- Before production: `PATIENT_APP_URL`, the link in appointment e-mails, once the app has a link domain.
- Before production (ops): the owner role that runs `migrate` needs `CREATEROLE` for `salus_retention`, as for
  `salus_app` (accounts), and, since `salus_retention` has `BYPASSRLS`, to be a superuser or to hold
  `BYPASSRLS` itself (Postgres 16+); otherwise the role is provisioned outside `migrate`. Production needs
  `RETENTION_DATABASE_URL` for the worker only, and `RETENTION_DATABASE_PASSWORD` for `migrate` (or neither,
  when provisioned elsewhere).
- Before 2036-11: removal of audit ranges (`SPEC.md` §6).

## Log

### 2026-11-04 — plan

`SPEC.md`, this plan and `RUN.tsv` written.

### 2026-11-04 — plan approved

Owner approved with the changes under Decisions; `SPEC.md` and the plan updated. Committed as
`booking(0): spec and plan`.

### 2026-11-04 — package 1: retention job

**Built.**
- `migrations/00014_retention_role.sql`: role `salus_retention` (`LOGIN`, `BYPASSRLS`, no superuser,
  `CREATEROLE` or `CREATEDB`, owner of nothing; the `salus_app` `DO` block, refusing an existing role with
  other attributes), `GRANT SELECT, DELETE ON session, invite`. No new table.
- `cmd/migrate up` sets the passwords of `salus_app` and `salus_retention` (each when its variable is set).
- `internal/config`: `LoadRetention()` (`RETENTION_DATABASE_URL`, `RETENTION_DATABASE_PASSWORD`), separate
  from `Load`; called only by `cmd/worker` and `cmd/migrate`.
- `internal/retention`: `Service.Apply` (the `Purger` and `AuditAges` ports, `Result`, `OverdueError`,
  `ErrNoApplier`); `NewService` refuses a `delete`/`remove-range` rule without an applier and a purger
  without a `delete` rule.
- `internal/auth/service_retention.go`: `SessionPurger`, `InvitePurger` on the repository's new
  `DeleteSessionsEndedBefore`, `DeleteInvitesEndedBefore` (`least(revoked_at, expires_at)`,
  `least(accepted_at, expires_at)` `< before`). `internal/auditevent`: `Service.CountRecordedBefore`.
- `internal/worker`: `retention_apply` (logs `table`, `deleted`, `overdue` per rule); `New(pool,
  retentionPool, log)` hands the retention pool to the purgers only; `NewScheduler` (periodic job daily at
  03:00 UTC); the worker and notifier share `workConfig`, which never leads.
- `cmd/scheduler`: the River client, replacing the placeholder; its test waits for "started" like the worker's.
- Compose: `RETENTION_DATABASE_URL` on `worker` only, `RETENTION_DATABASE_PASSWORD` on `migrate` and `tools`,
  `TEST_RETENTION_DATABASE_PASSWORD` on `tools`; `configs/.env.example`; `pgtest.NewRetentionDatabase`,
  `RetentionURL`, `AppURL`. `docs/privacy/dpia.md`: the retention job.

**Red first.** pgtest: `undefined: setRolePassword` / `NewRetentionDatabase`. migrations: `permission denied
for table session` (42501) and `salus_retention holds map[], want map[invite:[DELETE SELECT]
session:[DELETE SELECT]]`. auth: `Repository has no field or method DeleteSessionsEndedBefore`, `undefined:
auth.NewSessionPurger`. auditevent: `undefined CountRecordedBefore`. retention: `undefined:
retention.Purger`, `OverdueError`. config: `undefined: config.LoadRetention`, then `LoadRetention is called by
[]`. migrate: `up did not set the salus_retention password`. worker: `undefined: workConfig`, `dailyAt`,
`newScheduler`, then `RETENTION_DATABASE_URL is set in [], want only the worker service`. cmd/worker: `not
enough arguments in call to worker.New`. cmd/scheduler: `TestRun_FailsWithoutConfig` blocked (the
placeholder waited for the signal).

**Checks.** `make lint` 0 issues, migrations unchanged; `make test` passes (run twice); `make archcheck` ok;
`make cover` passes, every package at 80% or more (retention 100%, worker 97.9%, auth 95.7%, auditevent
100%, config 100%, pgtest 83.3%; total 94.9%); `make sqlc` leaves no diff. Stack: 00014 applied by the
`migrate` service, both passwords set; one `retention_apply` enqueued by hand ran in the worker and logged
`audit_event deleted 0 overdue 0`, `session deleted 0`, `invite deleted 0`; `river_leader` held by the
scheduler.

**Readings and decisions.**
- **River leadership.** River elects one leader per database among every client that works a queue, and only
  the leader enqueues periodic jobs. With the periodic job only in `cmd/scheduler` (SPEC §2), the worker or
  notifier could win leadership and the job would never run. So `workConfig` sets `LeaderElectionDisabled` on
  the worker and the notifier; the scheduler is the only leader and also runs River's maintenance (scheduled
  jobs, retries, rescue, cleanup), so those need the scheduler running (it always is in compose). River's
  rescuer discards a stuck job of a kind its client does not know, so the scheduler registers every kind
  (`addDefaultWorkers`, `addEmailWorkers` with nil dependencies; it works only its empty `scheduler`
  queue). This also fixes a latent case from accounts: the worker, as leader, would have discarded a stuck
  `invite_email`.
- **Overdue audit check.** The plan names `auditevent.Service.OldestRecorded`; the SPEC wants the count in
  the error, so the method is `CountRecordedBefore`. It scans `audit_event` once a day (no index on
  `recorded`); partitioning by year (§6) makes it a partition lookup. Not measured; it runs at 03:00.
- **Applying.** `Apply` runs every rule; a purger's or the count's error stops it and River retries (the
  purges are idempotent); overdue audit rows do not stop the purges but fail the run. Cut-off: a row is
  deleted when it ended strictly before now minus the period (tested at ± 1 s).
- **Not audited per row.** The retention role cannot write `audit_event` (by design, tested); SPEC §2 says
  the job logs table and count per rule. Sessions and invites are not patient data.
- **Schedule.** `dailyAt{hour: 3}` (UTC); no `RunOnStart`, so a restart does not run the job again; a missed
  03:00 waits for the next day. No periodic job ID (durable periodic jobs need River Pro).
- **Isolation tests.** `Load` never holds the retention values (set in the environment); only
  `cmd/worker/main.go` and `cmd/migrate/main.go` call `LoadRetention`, and only `internal/config` reads the
  variables (source scan); compose sets `RETENTION_DATABASE_URL` only on `worker` (file scan);
  `retentionPool` is passed only to `newRetention` and `auth.NewPostgresRepository` (AST); the noop job runs
  with the retention pool beside it, so River runs on the app pool (the role has no grant on River's tables).
- **Two schedulers, one job.** The test's schedule fires on the leader's first call rather than at a fixed
  time, so a slow start cannot make it miss (the first version used a fixed time and failed once under the
  full suite); it then watches one second for a second leader's job.

**Deferred.** `slots_generate` and the `slot` purger (package 5). Audit range removal (§6).

**Dev stack.** Migration 00014 applied to the dev database (role `salus_retention`, password
`local-retention-password`); api, worker, scheduler, notifier and agent rebuilt and restarted; `node` and the
rest untouched. One completed `retention_apply` row in `river_job` (River's cleaner removes it). No data
deleted: no session or invite was due.
