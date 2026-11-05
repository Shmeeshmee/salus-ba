# Booking progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Retention job | done | `booking(1)` |
| 2 Patient | done | `booking(2)` |
| 3 Consent and access | done | `booking(3)` |
| 4 Directory additions | done | `booking(4)` |
| 5 Schedule and slot | done | `booking(5)` |
| 6 Appointment | done | `booking(6)` |
| 7 FHIR | done | `booking(7)` |
| 8 Web | done | `booking(8)` |
| 9 Mobile | done | `booking(9)` |
| 10 End-to-end | done | `booking(10)` |

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
- 2026-11-05, owner (review of PR #18, recorded in `SPEC.md` §2 "Changed on review of PR #18"):
  - Paediatrics: SNOMED CT 394537008 "Pediatric specialty", outside `c80-practice-codes`; the preferred
    binding on `PractitionerRole.specialty` allows it. An informational validator note would be acceptable.
  - The notifier's e-mail jobs read the one appointment they name after a revocation: approved.
  - After a revocation the practitioner marks past appointments `fulfilled` or `noshow`; cancelling them stays
    refused. RLS, `access` and the table tests changed to match.
  - `GET /specialties` and `practitionerId` on `/auth/me`: approved, in `SPEC.md` §2.
  - Retention audit: one `AuditEvent` per run per table (system agent, row count), not per row.
  - Public practitioner listing: the service checks the clinic is listed and the practitioner active; RLS second.
  - FHIR canonical base stays `https://salus.invalid/fhir`, marked "needs the real domain before production".
  - Retention periods, the lawful-basis sign-off and `PATIENT_APP_URL` stay open; they do not block the merge.

## Waiting on people

- Before production: a legal decision on the retention of `appointment` and `consent`, and legal
  confirmation of the lawful basis split (`SPEC.md` §3, §4).
- Before production: `PATIENT_APP_URL`, the link in appointment e-mails, once the app has a link domain, and
  `PRIVACY_NOTICE_URL`, the privacy notice the consent screen links to (a placeholder).
- Before production (owner, 2026-11-05, `SPEC.md` §7): legal review of the consent and privacy text; whether
  booking should require consent at all, given care rests on Art. 9(2)(h); localising the patient app
  (bs/hr/sr) before any real clinic uses it.
- Before production (ops): the owner role that runs `migrate` needs `CREATEROLE` for `salus_retention`, as for
  `salus_app` (accounts), and, since `salus_retention` has `BYPASSRLS`, to be a superuser or to hold
  `BYPASSRLS` itself (Postgres 16+); otherwise the role is provisioned outside `migrate`. Production needs
  `RETENTION_DATABASE_URL` for the worker only, and `RETENTION_DATABASE_PASSWORD` for `migrate` (or neither,
  when provisioned elsewhere).
- Before 2036-11: removal of audit ranges (`SPEC.md` §6).
- Before production (package 7): the canonical base of our FHIR definitions, `https://salus.invalid/fhir` (the
  `organization-listed` and `organization-timezone` extensions and the `listed` SearchParameter), a placeholder
  like `PATIENT_APP_URL`. **Needs the real domain before production** (owner, 2026-11-05): a changed extension
  URL breaks every client.

Answered on review (2026-11-05, see "Decisions"): the notifier's one-appointment read, marking past visits
after a revocation, the public listing check, the paediatrics code, ENT and orthopaedics (confirmed),
`GET /specialties` and `practitionerId`.

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

### 2026-11-04 — package 2: patient

**Built.**
- `migrations/00015_patient.sql`: table `patient` (SPEC §2 "Patient table", §4): unique `account_id`
  referencing `account`, `birth_date` not before 1900-01-01, non-blank names, `phone` E.164 or NULL; RLS
  policy `patient_own` with the patient branch only (`id = app.patient_id` or `account_id = app.account_id`,
  through `NULLIF`); `GRANT SELECT, INSERT` and a column-level `UPDATE` (names, birth date, phone,
  `updated_at`, `version_id`) to `salus_app`; no `DELETE`. The consent branch comes with package 3.
- `internal/patient` (context `patient` in `config.yaml`; `ownership.yaml` with `personal_data: true`;
  `sqlc.yaml`): `Service.Me`, `Service.SaveMe` (create when no version is given, update of the given
  version otherwise; names trimmed; validation of SPEC §2 "Profile"), `Service.Read` (the patient's own
  profile; staff get not found until package 3); every call audited in its transaction, refusals included
  (`PATRQT` for the patient, `TREAT` for staff). Repository on `app.account_id`.
- `GET`/`PUT /me/patient` (patients only, staff 403, `Cache-Control: no-store`), wired in `internal/api`;
  `api/openapi.yaml` (`Patient`, `SavePatientRequest`, error codes `invalid_birth_date`, `invalid_phone`,
  `not_found`, `version_conflict`); `make gen-clients`.
- `internal/auditevent`: the purpose codes `PATRQT` and `TREAT`. `internal/retention`: rule `patient`
  `keep`. `docs/privacy/dpia.md`: the patient profile.

**Red first.** migrations: `relation "patient" does not exist` (42P01). retention: `no rule for patient`.
patient: `undefined: patient.Service`, then `undefined: patient.NewPostgresRepository`; the handler test was
written before the handler, and first ran red on the field order of the response (fixed in the test: the
struct is laid out for `fieldalignment`). api: `PUT /me/patient` = `404 page not found` before the wiring.

**Checks.** `make lint` 0 issues, migrations unchanged; `make test` passes; `make archcheck` ok; `make cover`
passes, every package at 80% or more (patient 98.3%, api 92.3%, retention 100%; total 95.2%); `make sqlc`
and `make gen-clients` leave no diff. Also run, as the generated clients changed: `make web-typecheck` and
`make mobile-analyze`, both clean.

**Readings and decisions.**
- **Versioning in the JSON API.** The plan asks for 409 on a stale save, so the version travels in the body
  (`versionId`; absent creates) rather than in `If-Match` (which answers 412 on the FHIR side). A create when
  a profile exists is the same 409 `version_conflict`: the client reads the profile and retries.
  `PUT` answers 201 on a create and 200 on an update.
- **Create without aborting the transaction.** The insert is `ON CONFLICT (account_id) DO NOTHING`, so a
  second create returns no row instead of a unique violation, and the refusal's audit event is written in the
  same, still usable, transaction.
- **Scope.** Patient transactions set `app.account_id` (the profile is looked up by account; SPEC §2 "Patient
  table" allows both settings). Nothing sets `app.patient_id` yet; the policy has it for the later clinic
  and appointment work, and its test covers both settings, a clinic and a staff account (no rows).
- **Who is a patient.** A caller without a clinic: `account` guarantees that a clinic is set exactly for
  staff, and the session carries it. No role lookup.
- **Validation.** Not audited when refused: nothing is read or written (the organization rename does the
  same). "In the past" is read as before today's date in UTC (a birth date of today is refused); names are
  counted in characters after trimming; the phone is strict (`+` and 8–15 ASCII digits, no spaces), the
  mobile app normalises it. A birth date that is not `YYYY-MM-DD` is 400 `invalid_birth_date`. The database
  repeats the cheap checks (blank names, 1900, E.164) as a backstop.
- **Errors and logs.** Domain errors carry no value; handlers log only through `httpx.InternalError`. Tests
  at the handler and through the whole app (access log included) send a synthetic profile and check that the
  log holds none of its fields.

**Deferred.** Staff access to a profile through consent and `access` (package 3); the FHIR `Patient` read
(package 7).

**Dev stack.** Untouched: migration 00015 is not applied to the dev database (no stack check in this
package; the running images do not use it); `make migrate` applies it. Tests ran in throwaway databases.

### 2026-11-04 — package 3: consent and access

**Built.**
- `migrations/00016_consent.sql`: table `consent` (SPEC §2 "Consent shape", §4) with `status` `active` |
  `inactive`, a check that `revoked_at` is set exactly when inactive, the partial unique index
  `consent_one_active` on `(patient_id, organization_id) WHERE status = 'active'` and an index on
  `(patient_id, organization_id)`; RLS `consent_own` (patient scope, all commands) and `consent_clinic`
  (clinic scope, `SELECT` only); the `patient` policy's consent branch `patient_consent` (clinic scope,
  `SELECT` only, an active consent for the clinic). `GRANT SELECT, INSERT` and a column-level `UPDATE`
  (`status`, `revoked_at`, `updated_at`, `version_id`) to `salus_app`; no `DELETE`.
- `internal/access` (context `privacy`): `Level` (`None`, `Name`, `Full`), `Role`, `Viewer`, `Consent`,
  `Visit`, `Facts`; `Profile` and `Appointment` as SPEC §2 "Access rule". Imports nothing from `internal/`.
- `internal/consent` (context `privacy`; `ownership.yaml` with `personal_data: true`; `sqlc.yaml`):
  `Grant`, `Revoke` (state, `revoked_at` and audit; no job yet), `Mine`, `Read`, `State`; every call but
  `State` audited, refusals included. Handlers `POST /consents`, `POST /consents/{id}/revoke`,
  `GET /me/consents` (patients only, staff 403, `Cache-Control: no-store`); `api/openapi.yaml` (`Consent`,
  `ConsentList`, `GrantConsentRequest`; codes `consent_active`, `consent_inactive`, `profile_required`);
  `make gen-clients`.
- `internal/patient`: `Caller` gains `Admin` and `PractitionerRoleID`; the repository gains `SetClinic`;
  `Service.Read` for staff sets the clinic scope, asks the `Consents` port (`consent.Service.State`) and the
  `Appointments` port for the facts, and reads the profile only when `access.Profile` is `Full`; otherwise
  404, audited. `internal/api`: `noAppointments` (no facts until package 6) and `consentCaller`.
- `internal/retention`: rule `consent` `keep`. `docs/privacy/dpia.md`: consent and access.

**Red first.** access: `no non-test Go files in /src/internal/access`. migrations: `relation "consent" does
not exist` (42P01). retention: `no rule for consent`. consent: `undefined: consent.NewPostgresRepository`,
then `undefined: consent.Service`, then `undefined: consent.NewHandler`. patient: `too many arguments in call
to patient.NewService` (the `Consents` and `Appointments` ports). api: `not enough arguments in call to
patient.NewService` before the wiring.

**Checks.** `make lint` 0 issues, migrations unchanged; `make test` passes; `make archcheck` ok; `make cover`
passes, every package at 80% or more (access 100%, consent 96.6%, patient 98.0%, api 91.0%, retention 100%;
total 95.4%); `make sqlc` and `make gen-clients` leave no diff. Also run, as the generated clients changed:
`make web-typecheck` and `make mobile-analyze`, both clean.

**Readings and decisions.**
- **Who the patient is, for consent.** `consent` needs the calling patient's id; `patient.Service.Read`
  needs `consent.Service.State`. A `Patients` port in `consent` made the two services need each other at
  construction. So the composition root resolves the id into `consent.Caller.PatientID`
  (`consentCaller`, through `patient.Service.Me`, audited there as a `Patient` read), as
  `organizationCaller` resolves the admin flag. `consent` imports only `access` and `auditevent`; the
  dependency stays one-way for package 6 (appointment → consent). No function value is bound for the
  revocation.
- **The clinic branch needs both patient settings unset.** SPEC §2 says the `patient` consent branch applies
  only when `app.patient_id` is unset. Patient transactions set `app.account_id` (package 2), so a patient's
  transaction with a clinic set but no `app.patient_id` would have seen the clinic's other patients. Both
  clinic branches (`patient`, `consent`) therefore require `app.patient_id` and `app.account_id` unset
  (fail closed; tested with each setting beside a clinic). Staff transactions never set either.
- **`State` is not audited.** It is the access check inside the caller's audited access (the staff
  `Patient` read audits, success or refusal), runs in the caller's transaction and scope, and discloses no
  consent to anyone. Every disclosure of a consent (`Read`, `Mine`, grant, revoke) is audited.
- **Grant to an unknown clinic.** The foreign key aborts the transaction, so the refusal is audited in a
  transaction of its own (404 `not_found`). Grant does not check `listed` (package 4 adds the flag; booking
  requires a listed clinic in package 6). A patient without a profile gets 409 `profile_required`;
  revoking or reading then is 404, the list is empty.
- **Revoke** reads the consent first, so another patient's or an unknown id is 404 and a revoked one is 409
  `consent_inactive`; the update is `WHERE status = 'active'`, so a concurrent second revocation is 409 too.
  Revoking is `update` in the audit (FHIR `restful-interaction`), listing is `search-type` (action `E`).
- **Staff reads of a profile.** `Read` returns the profile only at `Full`; `Name` is given with an
  appointment (package 7), as SPEC §2 "FHIR reads and searches" says. With no appointment facts until
  package 6, no staff read succeeds yet; the service tests cover every level with faked facts, and the
  repository test shows the clinic scope sees the row with an active consent and loses it right after the
  revocation commits. `Caller.Admin` and `PractitionerRoleID` are not set by any route yet (no staff route
  reads a profile before package 7).
- **Access table tests.** Every combination of viewer (patient, other patient, practitioner, admin) ×
  consent (active, revoked, never) × appointment (none, past, future, other practitioner) × clinic (match,
  other) is checked against an explicit list of the non-`None` levels, with a guard against an expectation
  that names no combination; plus the boundary of "past" (a second before, at, after the revocation),
  re-granting, and incomplete viewers or facts. "Past" is strictly before the latest `revoked_at`; with an
  active consent it does not matter.

**Deferred.** The `consent_revoked` job, the cancellation of future appointments, the past-appointment
branch of the `patient` policy and the appointment facts (package 6); the FHIR `Consent` and `Patient` reads
(package 7).

**Dev stack.** Untouched: migration 00016 is not applied to the dev database (no stack check in this
package; the running images do not use it); `make migrate` applies it. Tests ran in throwaway databases.

### 2026-11-04 — package 4: directory additions

**Built.**
- `migrations/00017_directory_listing.sql`: `organization.listed` (default false), `organization.time_zone`
  (default `Europe/Sarajevo`, not blank), `practitioner_role.specialty_code` (nullable); a partial index on
  listed clinics; `GRANT UPDATE (listed, time_zone) ON organization` and `GRANT UPDATE (specialty_code,
  updated_at, version_id) ON practitioner_role` to `salus_app`; the directory scope's `SELECT`-only policies
  (`app.directory = 'public'`, through `NULLIF`): listed clinics, and the active practitioners and roles of
  listed clinics.
- `organization`: `Listed` and `TimeZone` in the model; `Update(c, id, Changes, version)` (the name, and the
  listing and zone when given; nil keeps them), the zone validated with `time.LoadLocation` (`""` and `Local`
  refused) before any access, `ErrInvalidTimeZone` (FHIR 400 `value`, golden `invalid_time_zone.json`);
  `ListListed` in the directory scope, not audited. The repository's `Rename` became `Update`.
- `practitionerrole`: `Specialties()` (the owner's twelve, below), `SpecialtyByCode`, `SystemSNOMED`;
  `SpecialtyCode` on the model, `Admin` on `Caller`; `SetSpecialty` (admin of the clinic, version-checked,
  audited as an update, refusals included; `ErrInvalidSpecialty` before any access; `ErrConflict`);
  `PublicByOrganization` (directory scope, active roles, not audited). `NewService` takes a clock. JSON
  `PUT /practitioner-roles/{id}/specialty` in `handler_http.go`, wired in `internal/api` with
  `roleAdminCaller` (the admin flag read from `account` per request).
- `practitioner`: `PublicNames(ids)` in the directory scope: id, name, meta and active only (no clinic), not
  audited.
- `cmd/api`, `cmd/worker`, `cmd/notifier`, `cmd/scheduler` import `_ "time/tzdata"`.
- `api/openapi.yaml`: the path, `SetSpecialtyRequest`, `Specialty`, `RoleSpecialty`, error code
  `invalid_specialty`; `make gen-clients`. `docs/privacy/dpia.md`: the directory additions.

**Specialties** (owner's list; code and display copied from `ValueSet-c80-practice-codes.json` in
`hl7.fhir.r4.core#4.0.1`, read from the `fhir-validator` cache volume, sha256 `4a0a295c…434fac893`; system
`http://snomed.info/sct`):

| Owner's name (text) | Code | Display in the value set |
| --- | --- | --- |
| General practice | `394814009` | General practice |
| Internal medicine | `419192003` | Internal medicine |
| Cardiology | `394579002` | Cardiology |
| Dermatology | `394582007` | Dermatology |
| Gynaecology | `394586005` | Gynecology |
| Paediatrics | `394733009` | Medical specialty--OTHER--NOT LISTED |
| Psychiatry | `394587001` | Psychiatry |
| Neurology | `394591006` | Neurology |
| ENT | `418960008` | Otolaryngology |
| Ophthalmology | `394594003` | Ophthalmology |
| Orthopaedics | `394801008` | Surgery-Trauma and orthopedics |
| Endocrinology | `394583002` | Endocrinology |

**Red first.** migrations: `column "listed" of relation "organization" does not exist` (42703).
practitionerrole: the handler test `undefined: practitionerrole.NewHTTPHandler`; the model, service and
repository tests were written before their code but first run after it was in, so their red run was not
observed. practitioner: `repo.SetDirectory undefined`, `PublicNames undefined`. organization:
`undefined: organization.Changes`. api: `PUT /practitioner-roles/{id}/specialty` = `404 page not found`
before the wiring. cmd: `main.go does not import _ "time/tzdata"` (four binaries).

**Red run, observed afterwards (owner's request, 2026-11-04).** A `git worktree` on `d554191` (package 4's
parent) with only package 4's 20 test files and golden files from `ba5597d` copied in, no production code;
`go test -count=1` on the nine packages they touch, in the `tools` container. History not rewritten. Every
package fails, each for the reason its tests state (output trimmed; the compiler stops at ten errors per
package):

```
# internal/practitionerrole_test — build failed
model_test.go:13:33: undefined: practitionerrole.Specialty
model_test.go:30:29: undefined: practitionerrole.Specialties
model_test.go:33:22: undefined: practitionerrole.SystemSNOMED
service_specialty_test.go:16:114: unknown field Admin in struct literal of type practitionerrole.Caller
handler_http_test.go:36:24: undefined: practitionerrole.NewHTTPHandler
handler_http_test.go:53:10: updated.SpecialtyCode undefined (type practitionerrole.PractitionerRole has no field or method SpecialtyCode)
handler_http_test.go:98:21: undefined: practitionerrole.ErrInvalidSpecialty
handler_http_test.go:101:21: undefined: practitionerrole.ErrConflict
# internal/organization_test — build failed
handler_fhir_test.go:44:23: undefined: organization.Changes
handler_fhir_test.go:192:22: undefined: organization.ErrInvalidTimeZone
repository_postgres_test.go:52:77: unknown field TimeZone in struct literal of type organization.Organization
repository_postgres_test.go:81:19: repo.Update undefined (type organization.Repository has no field or method Update)
# internal/practitioner_test — build failed
repository_postgres_test.go:124:20: repo.SetDirectory undefined
repository_postgres_test.go:129:20: repo.PublicByIDs undefined
service_public_test.go:40:38: newService(repo, audit).PublicNames undefined
--- FAIL: TestMain_EmbedsTheTimeZoneData   (cmd/api, cmd/notifier, cmd/scheduler, cmd/worker)
    tzdata_test.go:22: main.go does not import _ "time/tzdata"
--- FAIL: TestNew_TheClinicsAdminSetsARolesSpecialtyAuditedAndNobodyElseCan   (internal/api)
    PUT /practitioner-roles/{id}/specialty as the admin at version 1 = 404 404 page not found, want 200
    ERROR: column "specialty_code" does not exist (SQLSTATE 42703)
--- FAIL: TestDirectoryListing_*   (migrations, four tests)
    ERROR: column "listed" of relation "organization" does not exist (SQLSTATE 42703)
    ERROR: column "specialty_code" of relation "practitioner_role" does not exist (SQLSTATE 42703)
```

The practitionerrole model, service and repository tests whose red run was not seen fail on the missing
specialty list, the missing `Admin` caller field, the missing `SpecialtyCode` and the missing errors: the
behaviour package 4 then added.

**Checks.** `make lint` 0 issues, migrations unchanged; `make test` passes; `make archcheck` ok; `make cover`
passes, every package at 80% or more (organization 95.5%, practitioner 95.2%, practitionerrole 96.4%, api
90.7%; total 95.3%); `make sqlc` and `make gen-clients` leave no diff (hashes compared); `make fhir-validate`
passes (the new golden included); `make web-typecheck` and `make mobile-analyze` clean.

**Readings and decisions.**
- **Paediatrics has no concept in c80-practice-codes** (only paediatric sub-specialties). The closest honest
  choice is `394733009` "Medical specialty--OTHER--NOT LISTED"; a sub-specialty would mislabel a general
  paediatrician. To keep it findable, each specialty carries `Text`, the owner's name, for
  `CodeableConcept.text` (package 7) and the apps; the JSON answer has it. Listed under "Waiting on people".
- **ENT and orthopaedics.** ENT is `418960008` "Otolaryngology" (the specialty) rather than `394604002`
  "Surgery-Ear, nose and throat surgery"; orthopaedics has only `394801008` "Surgery-Trauma and orthopedics".
  Gynaecology is `394586005` "Gynecology", not "Obstetrics and gynecology".
- **Code only in the table.** The SPEC names the column `specialty_code`; the system is always SNOMED CT and
  the display comes from the list, so the database skill's "system + code (+ display)" is met by the code
  plus the fixed list. The database does not check the code (the list may grow without a migration); the
  service does.
- **Organization update.** The FHIR handler passes only the name until package 7 reads the `listed` and
  time-zone extensions, so `Changes` keeps the listing and zone when they are not given. Until then no
  endpoint lists a clinic: only the service can (and the owner, by SQL).
- **Specialty endpoint.** `If-Match` as the SPEC says (`W/"n"` or `"n"`): missing or unreadable is 400
  `invalid_request`, a stale version 412 `version_conflict` (HTTP's precondition status, as on the FHIR side;
  the patient profile's 409 is for a version in the body). The answer carries the new `ETag`. A practitioner
  gets 403, anyone outside the clinic 404; all audited. No way to clear a specialty (not in the SPEC).
- **Public reads.** `ListListed`, `PublicByOrganization` and `PublicNames` set `app.directory = 'public'`, are
  not audited (SPEC "Public browse"), and re-check what the rows carry (`listed`, `active`, the clinic id) so
  they do not rest on row-level security alone. A role search of an unlisted clinic gives none rather than
  not found; package 7 decides the HTTP form. The practitioner query selects no clinic; the repository test
  compares the whole returned value.
- **Consent grant** still does not check `listed` (as package 3 left it): the SPEC asks for a listed clinic
  at booking (package 6), not at consent.
- **Time zones.** `UTC` is accepted (an IANA name); `time.LoadLocation`'s `""` (UTC) and `Local` are refused;
  the four binaries embed `time/tzdata`, checked by a source test in each.

**Deferred.** FHIR: `Organization?listed=true`, the `listed` and time-zone extensions on read and update,
anonymous `PractitionerRole` search with `specialty`, the `StructureDefinition`s and `SearchParameter`
(package 7). The web specialty editor and listing toggle (package 8).

**Dev stack.** Untouched: migration 00017 is not applied to the dev database (no stack check in this
package; the running images do not use it); `make migrate` applies it. Tests ran in throwaway databases.
`alpine:3` was pulled to read the value set from the `salus_fhir` volume (read-only) and removed again.

### 2026-11-04 — package 5: schedule and slot

**Built.**
- `migrations/00018_schedule_slot.sql`: `schedule` (one per role, `slot_minutes` 5 to 240 in steps of 5,
  `availability` a JSON array, `(practitioner_role_id, organization_id)` referencing the role, for which
  `practitioner_role` gains `UNIQUE (id, organization_id)`) and `slot` (`period tstzrange`, half-open and
  bounded; `status` `free` | `busy`; `(schedule_id, organization_id)` referencing the schedule;
  `EXCLUDE USING gist (schedule_id WITH =, period WITH &&)`; a partial index on `upper(period)` of free slots
  for retention). RLS: clinic scope (all commands) on both; directory scope `SELECT` on schedules of roles the
  role policy shows (active, listed clinic) and on their `free` slots; a `RESTRICTIVE` `DELETE` policy lets the
  application delete free slots only. Grants: `salus_app` `SELECT, INSERT` and `UPDATE (slot_minutes,
  availability, updated_at, version_id)` on `schedule`; `SELECT, INSERT, DELETE` and `UPDATE (status,
  updated_at, version_id)` on `slot`; `salus_retention` `SELECT, DELETE` on `slot`.
- `internal/schedule` (context `scheduling`): `Availability`, `SaveAvailability` (create at version 0,
  update of the current version; validation; enqueues `slots_generate` with the schedule and clinic ids in
  its transaction), `Read` and `ByActor` (directory, not audited), `ForGeneration` (clinic scope, row locked
  `FOR NO KEY UPDATE`) and `Listed` (directory) for generation; staff reads and edits audited (`HOPERAT`),
  refusals included. `Ports` struct (the worker builds it without `Roles` and `Jobs`). River jobs port;
  `SlotsGenerateArgs`. JSON `GET`/`PUT /practitioner-roles/{id}/availability` in `handler_http.go`.
- `internal/slot` (context `scheduling`): `Generate(ctx, org, scheduleID)`, `GenerateAll`, `Search`
  (directory, free, from now on), `Hold` and `Release` (clinic scope set by the call, in the caller's
  transaction), `Purger` (`service_retention.go`). Generation (`service_generate.go`) in the clinic's zone,
  28 days ahead.
- `internal/auth`: `Account.PractitionerID` (read by `AccountByID`/`AccountByEmail`), `Member`,
  `Service.ClinicMember` (admin flag and practitioner of the session's clinic, not audited, like
  `IsClinicAdmin`). `internal/api`: the schedule service and handler, `scheduleCaller`.
- `internal/worker`: the `slots_generate` worker (one schedule, or every listed schedule when the arguments
  are empty; logs `schedules`, `inserted`, `deleted`; cancels the job of an unknown schedule); the scheduler
  registers the kind and enqueues it hourly with `RunOnStart`; the retention job gets the slot purger on the
  retention pool. `make bench-slots`.
- `internal/retention`: `schedule` `restrict`, `slot` `delete` after 30 days. `ownership.yaml`, `config.yaml`
  (`scheduling: [schedule, slot]`), `sqlc.yaml`. `api/openapi.yaml` (`Availability`, `AvailabilityRule`,
  `SaveAvailabilityRequest`; codes `invalid_slot_minutes`, `invalid_availability`; tag `scheduling`);
  `make gen-clients`. `docs/privacy/dpia.md`: schedules and slots.

**Red first.** migrations: `relation "schedule" does not exist` (42P01). retention (rules restored after the
red run): `no rule for schedule`/`slot`, `NewService() error = … has no applier … : slot`, and the grant test
`salus_retention holds map[… slot:[DELETE SELECT]], want map[invite:… session:…]` (the migration was in before
the rule). schedule: `undefined: schedule.Service`, `undefined: schedule.NewPostgresRepository`,
`undefined: schedule.NewRiverJobs`, `undefined: schedule.NewHTTPHandler`, then the handler's field order (fixed
in the test, as in package 2). slot: `undefined: slot.Service`, `undefined: slot.NewPostgresRepository`, then
an overlapping fixture of my own (fixed in the test). auth: `Account has no field or method PractitionerID`,
then `ClinicMember()` giving a practitioner of another clinic its practitioner id (fixed in the code: it read
the account through `clinicAdmin`, which overwrites the clinic). worker: `undefined: slotsSchedule` (the
helper was then dropped for a constant). api: the availability route answered 404 and wrote no audit or job
before the wiring.

**Checks.** `make lint` 0 issues, migrations unchanged; `make test` passes; `make archcheck` ok; `make cover`
passes, every package at 80% or more (schedule 90.5%, slot 93.3%, worker 98.4%, auth 95.6%, api 90.5%,
retention 100%; total 94.8%); `make sqlc` and `make gen-clients` leave no diff (hashes compared);
`make web-typecheck` and `make mobile-analyze` clean.

**Benchmark** (`make bench-slots`: `BenchmarkGenerate` in `internal/worker`, the services the worker builds,
as `salus_app` with row-level security; 50 schedules of one listed clinic, weekdays 08:00 to 16:00 in
quarter-hour slots, from Monday 09:00 Sarajevo: 31 950 slots; `-benchtime=5x -count=3`, i7-14700HX, the
compose Postgres):

| Run | Time per run (3 counts) | Per schedule |
| --- | --- | --- |
| First (no slots) | 1.60 s, 1.63 s, 1.33 s | about 30 ms for 639 slots |
| Re-run (nothing changes) | 0.84 s, 0.84 s, 0.85 s | about 17 ms |

`EXPLAIN ANALYZE` on one schedule: the delete of stale free slots takes 1.8 ms (hashed `NOT IN`); the insert
of 639 periods that all conflict takes 13 ms, the cost of `ON CONFLICT DO NOTHING` on the exclusion
constraint. Tried: skipping existing periods with `NOT EXISTS` before the insert; the re-run became slower
(1.66 s), as each probe pays the row-level security policies, so it was reverted. Under a second an hour for
50 practitioners, and about 17 to 30 ms for the one schedule an edit enqueues, needs nothing faster; if it
ever does, generating only the days not yet generated is the next step.

**Readings and decisions.**
- **Practitioner of a role.** The session holds neither the role nor the practitioner. `account.practitioner_id`
  (set at invite acceptance, accounts) is now read by `auth`, and `ClinicMember` gives the schedule handler the
  admin flag and the practitioner; the service reads the role through `practitionerrole.Service.Read` (audited
  there) and compares its practitioner. A staff account that is no practitioner and not the admin gets 403.
- **Before the first edit.** `GET` answers an empty availability at version 0 (no `id`, no `slotMinutes`),
  with `ETag: W/"0"`; a `PUT` with `If-Match: W/"0"` creates the schedule ("created with the role's first
  availability edit"), a create when one exists is 412 like any stale version. Version mismatches are 412
  `version_conflict`, as for the specialty (`If-Match`). An empty rule list is valid (no slots).
- **Validation.** `HH:MM` strictly, 00:00 to 23:59, so a window cannot end at midnight (23:59 is the latest);
  start before end; touching windows are allowed, overlapping or nested ones are not. Invalid input is 400 and
  not audited (nothing accessed), as in packages 2 and 4. Rules are stored sorted by day and start. Body limit
  64 KiB.
- **Generation.** A slot is generated when its start is strictly after now and strictly before now + 28 × 24
  h; slots step from the rule's start by the slot length while they end by the rule's end (wall-clock
  minutes). A local time is resolved against the zone's offsets a day before and after: a time neither gives
  is skipped (spring), and of two the earlier instant is taken (autumn; Go's `time.Date` takes the later one in
  Europe/Sarajevo). Tested on 28 March and 31 October 2027. Order: delete the future free slots not in the
  set, then insert (so a shorter slot length is not blocked by the old free slots); both in one transaction
  with the schedule row locked, so a generation and an edit, or two generations, of one schedule do not
  interleave. A new slot overlapping a busy one is skipped; an appointment outside new availability keeps its
  busy slot.
- **Which schedules the hourly run covers.** The scheduler and worker have no cross-clinic scope; the run lists
  schedules in the directory scope (active roles of listed clinics) and generates each in its clinic's scope,
  each in its own transaction (a failing one does not stop the others; the job then fails and River retries,
  which is idempotent). An unlisted clinic gets slots when an availability is saved, not hourly; when it is
  listed, the next hourly run (or `RunOnStart`) covers it. Nobody can book an unlisted clinic (package 6), so
  nothing is lost.
- **`RunOnStart`.** The hourly job is also enqueued when the scheduler starts leading, so slots do not wait
  up to an hour after a deploy; the leader-only rule (package 1) still holds.
- **Hold and Release.** Both set the clinic scope themselves and join the caller's transaction (tested: a
  rolled-back booking leaves the slot free). `Hold` is free to busy, else `ErrUnavailable` (unknown, other
  clinic, busy). `Release` frees a busy slot only while its start is ahead and reports whether it did; it is
  not an error otherwise. Package 6 adds the future-start and listed-clinic checks of booking in its own
  service.
- **Slots stay out of the audit.** A slot holds no patient data; the appointment that holds one is audited in
  package 6. Schedule reads and edits by staff are audited as directory operations (`HOPERAT`), like the role.
- **Busy slots in the directory.** The SPEC says the directory scope shows the slot rows of listed clinics; the
  policy shows only `free` ones (stricter, nothing in the product needs a busy slot outside the clinic). Package
  6 resolves a slot for booking: a busy one then reads as not found, answered 409 there.
- **Slot model.** `updated_at` and `version_id` on `slot` for `meta`; no practitioner role column (SPEC §4
  lists none): package 6 reaches the role through the schedule.
- **Benchmark place.** In `internal/worker`, where the real composition is: `internal/slot`'s tests may not
  build other domains' repositories (archcheck), so they use fakes for the schedule and the clinic.

**Deferred.** FHIR `Schedule` read and search by `actor`, `Slot` search (package 7); booking, cancelling and
the concurrent book-and-cancel test over `Hold`/`Release` (package 6); the web availability editor (package 8).

**Dev stack.** Untouched: migration 00018 is not applied to the dev database (no stack check in this package;
the running images do not use it); `make migrate` applies it. Tests and the benchmark ran in throwaway
databases. The `node` dev server kept running.

### 2026-11-04 — package 6: appointment

**Built.**
- `migrations/00019_appointment.sql`: `appointment` (SPEC §2 "Double booking", §4): patient, practitioner
  role with its clinic (foreign key on both), slot (no foreign key: a freed slot may be deleted while the
  cancelled appointment is kept), `period` (half-open, bounded), `status` (`booked`, `cancelled`, `fulfilled`,
  `noshow`), `cancelled_at`, `cancellation_reason` (`pat` | `prov`) and `cancelled_by_account_id`, consistent
  by a check; `EXCLUDE USING gist (practitioner_role_id WITH =, period WITH &&) WHERE (status <> 'cancelled')`.
  RLS: `appointment_own` (patient scope, all commands); `appointment_clinic` (`SELECT`: clinic scope with both
  patient settings unset, and an active consent or a start before the latest `revoked_at`);
  `appointment_clinic_update` (`UPDATE`: the same with an active consent only); `appointment_notice` (`SELECT`:
  the clinic scope with `app.appointment_id`, the one appointment). `patient` gains `patient_past_appointment`
  and `patient_notice` (`SELECT`). Grants: `SELECT, INSERT` and `UPDATE` of the status and cancellation columns,
  `updated_at`, `version_id`; no `DELETE`. Retention `keep`; `ownership.yaml`, `config.yaml`
  (`scheduling: [appointment, schedule, slot]`), `sqlc.yaml`.
- `internal/appointment` (context `scheduling`): `Book`, `Cancel`, `SetStatus`, `Read`, `Search`,
  `CancelForRevokedConsent`, `Notice` (the e-mail jobs' read), `Facts.Visits` (the patient domain's
  appointment facts); ports `Consents`, `Slots`, `Patients`, `Clinics`, `Roles`, `Jobs`; access through
  `access`; every call audited, refusals included, with a `Patient` read event whenever the name is given.
  JSON `POST /appointments`, `/appointments/{id}/cancel`, `/appointments/{id}/status` (`handler_http.go`);
  River jobs `appointment_booked_email`, `appointment_cancelled_email`, `appointment_reminder_email` (queue
  `email`, the appointment and clinic ids; the reminder `ScheduledAt` start − 24 h).
- `internal/slot`: `Bookable(id)` (directory scope: the free slot with its schedule's role), `ByID` and the
  `SlotByID` query; the `Schedules` port gains `Read`. `internal/patient`: `Lookup(id)` (the caller's
  transaction and scope, unaudited; the appointment service audits).
- `internal/consent`: `Revoke` enqueues `consent_revoked` (`RevokedArgs`: patient id, clinic id) with
  `InsertTx` in its transaction through the new `Jobs` port (`repository_river.go`). `consent` imports nothing
  of appointments.
- `internal/auth`: `AccountEmail(agent, id)`, audited with the job as agent.
- `internal/notification`: `Ports`; `SendAppointmentBooked`, `SendAppointmentCancelled`,
  `SendAppointmentReminder` (`service_appointment.go`), golden texts in `testdata/`.
- `internal/worker`: the `consent_revoked` worker (default queue, `revocation.go`, logs the count only); the
  three e-mail workers on the notifier (an unknown appointment cancels the job); the scheduler registers all
  four new kinds. `NewNotifier` takes `PATIENT_APP_URL`.
- `internal/config`: `PatientAppURL` (`PATIENT_APP_URL`); compose sets it on the notifier to the placeholder
  `https://patient.salus.invalid`; `configs/.env.example`. `cmd/notifier` passes it.
- `internal/api`: the appointment service and handler, `appointmentCaller`; `patient`'s facts are
  `appointment.NewFacts` (the `noAppointments` placeholder is gone); the consent service gets its jobs.
- `api/openapi.yaml`: the three paths, `BookAppointmentRequest`, `SetAppointmentStatusRequest`, `Appointment`,
  codes `consent_required`, `slot_unavailable`, `cancellation_window_closed`, `appointment_closed`,
  `appointment_not_started`, `invalid_status`; `make gen-clients`.
- `deployments/docker-compose.yml`: Postgres `max_connections` 200 → 400 (below). `docs/privacy/dpia.md`:
  appointments; `docs/privacy/processors.md`: appointment e-mails, no new processor.

**Red first.** migrations: `relation "appointment" does not exist` (42P01, seven tests). retention: `no rule
for appointment`. appointment: `undefined: slot.Bookable` (then `appointment.Service`, `NewPostgresRepository`,
`NewRiverJobs`, `NewHTTPHandler`, each before its code). slot: `Bookable undefined`, then `SlotByID undefined`
before `make sqlc`. patient: `Lookup undefined`. consent: `not enough arguments in call to
consent.NewService` (the jobs port), `undefined: consent.NewRiverJobs`. auth: `AccountEmail undefined`.
notification: `undefined: notification.Ports`. config: `cfg.PatientAppURL undefined`. worker: the build
failed on `notification.NewService` (the scheduler's kind test was not run red on its own). api: `POST
/appointments` = `404 page not found` before the wiring (both app tests). The appointment service tests passed
on their first run after the code; their red was the compile failure. The concurrent book-and-cancel test was
checked against a mutation (cancellation without the slot's release): it failed at round 1 with "6 slots
disagree"; the mutation was reverted.

**Checks.** `make lint` 0 issues, migrations unchanged; `make test` passes; `make archcheck` ok; `make cover`
passes, every package at 80% or more (appointment 92.3%, api 90.3%, notification 90.5%, worker 97.7%, consent
96.4%, slot 93.1%, patient 98.0%, auth 95.7%, config 100%, retention 100%; total 94.7%); `make sqlc` and
`make gen-clients` leave no diff (hashes compared); `make web-typecheck` and `make mobile-analyze` clean. The
concurrency tests repeated: the repository's 30 times, the api's 5 times, all green. Stack (dev stack, API on
18080, Mailpit on 18025): a synthetic patient registered, got 409 `profile_required`, saved a profile, got 409
`consent_required`, consented, booked (201; a second booking of the slot 409 `slot_unavailable`), and
"Your appointment is booked" arrived in Mailpit with the clinic, "Saturday 7 November 2026, 20:00
(Europe/Sarajevo)" and `https://patient.salus.invalid/appointments`; the patient cancelled (200, `pat`), the
slot was free again and "Your appointment is cancelled" arrived. River held the two ids only; the audit held
the create and update by the account and, per e-mail, the job's reads of the appointment, the patient and the
account; the api and notifier logs held no profile field or address.

**Postgres connections.** The first uncached `make cover` with this package failed with 53300 (too many
clients) in seven tests: the suite needs more than the 200 connections set in accounts package 5. Measured
with a 30 ms sampler of `pg_stat_activity` inside the Postgres container: 205 at the moment of failure with
the limit at 200; with the limit at 400, peaks of 229 and 222 over two uncached runs, both green. Compose (and
so CI) now sets 400.

**Readings and decisions.**
- **The notifier's scope.** SPEC §2 puts the notifier's jobs in the clinic scope, and "Past and future after
  revocation" hides a future appointment from that scope once the consent is revoked; the revocation job's
  cancellation e-mails would then find nothing to send, though the e-mail and the e2e journey require them. The
  e-mail jobs therefore set `app.appointment_id` beside the clinic, and two `SELECT` policies show that one
  appointment of the clinic and its patient's row (for the account to write to), whatever the consent. Staff
  transactions never set it, both policies need the patient settings unset, and the clinic still sees nothing
  else (tested). Read as the clinic scope narrowed to the job's appointment, not a new scope; listed under
  "Waiting on people" for confirmation.
- **Changes need full access.** A cancellation or status change needs `access` at `Full`: after a revocation a
  past appointment is at `Name`, so staff get 403 (row-level security agrees: a clinic updates only with an
  active consent). Listed under "Waiting on people".
- **Booking order.** The slot is resolved in the directory in a transaction of its own (`slot.Bookable`), so
  the directory setting never mixes into the patient's transaction; then, in the patient's scope with the
  slot's clinic: the clinic is read and checked listed (not on the directory's row-level security alone), the
  consent's state, the hold (free to busy), the insert, the e-mails, the audit. A slot taken in between is
  refused at the hold (409). A refusal is audited in a transaction of its own after the booking's (the
  exclusion violation aborts the booking's).
- **The exclusion constraint deadlocks.** Two concurrent inserts that overlap on an exclusion constraint can
  each wait for the other; Postgres aborts one with 40P01 and the other commits (seen once, in round 3 of 10, under
  the full suite). The repository maps 40P01 on the insert, like 23P01, to `slot_unavailable`. Overlapping slots
  of one role cannot exist (one schedule per role, slots never overlap within one), so the plan's "overlapping
  slots of one role" test runs at the repository, on the constraint itself; the HTTP test books one slot twice
  at once.
- **Concurrent book and cancel.** On the composed services as `salus_app`: two patients, six slots, 25 rounds,
  each round cancelling the holder of every slot while booking it again at once. After each round no slot is
  busy without a booked appointment or free with one, none is held twice, and the exclusion constraint never
  refused a booking (counted through a wrapping repository): the hold refused every taken slot first.
- **Revocation job.** It cancels the patient's booked appointments in the clinic that start at or after the
  latest `revoked_at` and were created at or before it, so an appointment booked after consenting again stays;
  reason `pat`, no canceller account, audited with `consent_revoked` as agent (`HOPERAT`); slots freed while
  ahead; one cancellation e-mail each; one transaction; a retry cancels nothing. `created_at` is set from the
  service's clock for this comparison.
- **Times.** The patient may cancel while the start is 24 h or more ahead (24 h + 1 s yes, 24 h − 1 s no, as
  tested); a status may be set from the start on. The reminder is enqueued only when start − 24 h is after now.
- **Who the practitioner is.** As in package 5: the account's practitioner, compared with the appointment's
  role's practitioner through `practitionerrole.Service.Read` (audited as a role read). A practitioner's search
  must name their own role; package 7 maps FHIR `practitioner` to it.
- **No cycle.** The patient service needs the appointment facts and the appointment service reads profiles
  through the patient service: the facts are `appointment.NewFacts(repository)`, built before both.
- **What the JSON answer holds.** The appointment without its patient and its canceller; reads with the name go
  through FHIR (package 7) on `Read` and `Search`, which give the name only where `access.Profile` allows and
  audit it as a `Patient` read. At `Name` level the appointment is reduced to its times, status and
  participants.
- **E-mails.** Plain text, golden files. The job reads the appointment first (audited) and sends nothing when
  it is no longer in the status the e-mail is about (a reminder or confirmation of a cancelled one, a
  cancellation of a booked one); an unknown appointment cancels the job. The recipient comes from
  `auth.AccountEmail`, audited with the job as agent. The time is "Monday 2 January 2006, 15:04" in the clinic's
  zone with the zone's name.
- **Search cost.** A staff search gathers each distinct patient's consent state and visits (two queries per
  patient). Not measured; the calendar of package 8 will show whether it needs batching.

**Deferred.** FHIR `Appointment` read and search, the `Patient` and `Consent` reads, the mapping of FHIR
`practitioner` to the role (package 7); the web calendar and status buttons (package 8); mobile booking and
cancelling (package 9); the e2e journeys (package 10).

**Dev stack.** Migrations 00015 to 00019 applied to the dev database. The Postgres container was recreated
with `max_connections=400` (its volume kept); api, worker, scheduler, notifier and agent rebuilt and
restarted; `node`, `landing`, `livekit`, `mailpit` and `redis` untouched, and the owner's "Demo clinic Bihać"
untouched. Left from the stack check, all synthetic: the unlisted clinic "Stack check clinic (booking package
6)" with a placeholder practitioner, role, schedule and one free slot; the patient account
`stack6.patient@example.test` with a profile, an active consent and one cancelled appointment; their audit
events; two messages in Mailpit; the reminder job, scheduled for 2026-11-06 19:00 UTC, which will complete
without sending (the appointment is cancelled). Two empty flag files of the connection sampler
(`/tmp/stop-sampler-6-1`, `-2`) sit in the Postgres container's `/tmp` until it is next recreated.


### 2026-11-04 — package 7: FHIR

**Built.**
- `pkg/fhir`: `Coding`, `CodeableConcept`, `Extension` (boolean or code value), `Reference.Display`,
  `ReferenceID` (a search parameter's `Type/id` or bare id) and `DateRange` (at most one `ge` and one `lt`, each
  an RFC 3339 instant with its offset).
- `organization`: the `listed` and time-zone extensions on read and update (a left-out extension keeps its
  value; one of ours repeated or without its value is 400); `Organization?listed=true` for anyone
  (`ListListed`). The handler takes the FHIR base URL for the Bundle.
- `practitionerrole`: `specialty` as a CodeableConcept (`http://snomed.info/sct`, code and display from the
  value set, `text` the owner's name); the search gives staff of the clinic its roles (audited, as before) and
  anyone else, signed in or not, the public directory (`PublicByOrganization`, practitioners through
  `practitioner.PublicNames`), an empty searchset for a clinic that is not listed. `Lookup(org, id)` and
  `OfPractitioner(org, practitionerID)`: the role in the clinic's scope, in the caller's transaction,
  unaudited; repository `ByPractitioner` (query `PractitionerRoleByPractitioner`).
- `patient/handler_fhir.go`: `Patient` read (`name` official, `birthDate`, `telecom` mobile phone); the caller
  (`patientReadCaller` in `internal/api`) carries the admin flag and a practitioner's role, read through
  `auth.ClinicMember` and `practitionerrole.OfPractitioner`.
- `consent/handler_fhir.go`: `Consent` read as SPEC §2 "Consent shape" (scope, LOINC category, patient and
  performer, organization, `OPTIN`, provision permit for `TREAT` with the clinic as `IRCP`).
- `schedule/handler_fhir.go`: `Schedule` read and search by `actor` (public, unaudited; actor and active).
  `slot/handler_fhir.go`: `Slot?schedule=…&status=free&start=lt…[&start=ge…]` (public, unaudited).
- `appointment`: `Seen.PractitionerID` (the role's practitioner, through `Roles.Lookup` once per role) and
  `Filter.PractitionerID` (staff: mapped to the role in the clinic, a practitioner may name only themselves;
  a patient: among their own); `handler_fhir.go`: read and search by `patient`, `practitioner`, `date` (`ge`,
  `lt`), `status`; participants the patient (with `display` only where the service gave the name), the
  practitioner and the role, each `accepted`; `slot`, `created` and `cancelationReason` when the level shows
  them.
- `api/fhir/`: `StructureDefinition-organization-listed.json` (`valueBoolean`),
  `StructureDefinition-organization-timezone.json` (`valueCode`, bound to the R4 `timezones` value set),
  `SearchParameter-organization-listed.json`; `CapabilityStatement.json` lists exactly the interactions and
  parameters served (tested), and documents the public reads. `scripts/fhir-validate.sh` validates the
  `api/fhir/` documents as well as the goldens.
- `Cache-Control: no-store` on `Patient`, `Consent` and `Appointment` answers. `docs/privacy/dpia.md`: package 7.

**Red first.** pkg/fhir: `undefined: fhir.CodeableConcept`, `fhir.Coding`, `fhir.Extension`, `fhir.ReferenceID`,
`fhir.DateRange`, `unknown field Display`. practitionerrole: `repo.ByPractitioner undefined`, `s.Lookup
undefined`, `s.OfPractitioner undefined`; the handler: the public search answered 401 to nobody signed in
(`want 200 from the public directory`), and the read golden lacked the specialty. organization: the update
ignored the extensions (`status = 200 after 1 service calls, want 400 and none`), `Organization?listed=true` =
`404 page not found`. patient, consent: `undefined: patient.NewFHIRHandler`, `consent.NewFHIRHandler`.
schedule, slot, appointment: `undefined: …NewHandler`; appointment service: `Seen has no field
PractitionerID`, `unknown field PractitionerID in … Filter`. api/fhir: `the CapabilityStatement does not
define listed by https://salus.invalid/fhir/SearchParameter/organization-listed` and the interaction list.
internal/api: the matrix test run with the new routes left out of `app.go` (restored after): `GET
/fhir/R4/Schedule?actor=… = 404`, `Slot?… = 404`, and so on; the directory test then failed on the public
search's new answer (below). The schedule, slot and organization handler tests' first failures were their
missing golden files; the goldens were written with `-update` and reviewed.

**Checks.** `make lint` 0 issues, migrations unchanged; `make test` passes; `make archcheck` ok; `make cover`
passes, every package at 80% or more (appointment 92.8%, consent 96.7%, organization 95.9%, patient 98.2%,
practitionerrole 96.3%, schedule 91.3%, slot 94.0%, api 90.9%, pkg/fhir 100%, api/fhir 100%; total 94.7%);
`make sqlc` leaves no diff; `make fhir-validate` passes with `-ig api/fhir` (0 errors in every file; the
warnings are `dom-6`, no narrative, and the notes that codes cannot be checked without a terminology server).
No `openapi.yaml` change, so the clients are untouched.

**Readings and decisions.**
- **Time-zone extension: ours.** The pinned validator's `hl7.fhir.r4.core#4.0.1` has `tz-code` (IANA code),
  but its context is `date`, `dateTime` and `instant` only; the extension pack it also loads
  (`hl7.fhir.uv.extensions.r4#5.2.0`) has `timezone` with the same contexts plus `Meta` and `time`. Neither may
  sit on an Organization, so `organization-timezone` is ours, `valueCode` bound (required) to the core
  `timezones` value set, which is what `tz-code` uses.
- **Canonical base.** `https://salus.invalid/fhir` for our definitions, a placeholder (Waiting on people).
- **Organization update.** An extension left out keeps its value (both columns are `NOT NULL`, so "remove" has
  no meaning); unknown extensions are ignored like the other elements the update does not set. The read answers
  both extensions always. Read stays staff-only; anyone lists through the search.
- **Public role search.** Staff of the clinic get its roles as before (inactive ones too, audited); everyone
  else the public directory, unaudited. A clinic that is not listed answers an empty searchset, not 404: a
  search that matches nothing is an empty Bundle in FHIR, and it discloses nothing. The accounts-era directory
  test asked 404 and 401 of that search; its expectation now follows SPEC "Public browse", and it runs on the
  `salus_app` login (as production does), which is what showed the listing check resting on row-level security
  (Waiting on people).
- **Slot search.** `status=free` is required of everyone (no staff slot search exists or is planned; the
  calendar reads appointments); `start=lt` is required to bound the answer, `start=ge` defaults to now.
  The self link is rebuilt from the parsed values. The Schedule resource carries its actor and `active` only:
  the weekly rules are not FHIR elements, the slots are what a client books.
- **Practitioner participant.** The appointment table holds the role; the practitioner comes from
  `practitionerrole.Lookup` in the read's transaction, in the appointment's clinic scope. In a patient's
  transaction that sets the clinic beside `app.patient_id`, as booking does; the clinic branches of
  `patient`, `consent` and `appointment` require the patient settings unset, so the patient still sees only
  their own. The lookups are not audited: a role is not patient data, and the read they serve is audited.
- **`practitioner` search parameter.** Staff: mapped to the practitioner's role in the caller's clinic
  (`OfPractitioner`); a practitioner may name only themselves (403, audited), the admin anyone (none for a
  practitioner of another clinic). A patient: filtered among their own before anything is audited. The
  `patient` parameter keeps package 6's rule: a patient always gets their own.
- **Dates.** Search dates are instants with an offset, `ge` and `lt` only (`DateRange`); a date without a time
  or another prefix is 400. Simplest unambiguous form; the web calendar sends instants.
- **Patient read for a practitioner.** The caller's role is resolved in the composition root, like the admin
  flag (`ClinicMember`), unaudited; an admin is at `Name` for a profile, so 404 (SPEC "FHIR reads and
  searches"). The name with an appointment is how an admin sees it.
- **Specialty.** Code and display from the value set, `text` from the list's `Text`. Paediatrics untouched and
  in no golden or fixture (cardiology is used).
- **Field layout.** `govet` fieldalignment orders the struct fields, so a few JSON keys come in a different
  order (for example `meta` before `extension`, `text` before `coding`); FHIR JSON does not depend on key
  order.

**Deferred.** The web calendar, availability editor, status buttons, specialty editor and listing toggle
(package 8); mobile (package 9); the e2e journeys (package 10).

**Dev stack.** Untouched: no migration in this package; tests ran in throwaway databases; the running images
were not rebuilt. The `node` dev server kept running.


### 2026-11-04 — package 8: web

**Built.**
- `web/src/Calendar.tsx`: the week, Monday to Sunday in the clinic's time zone, from
  `Appointment?practitioner=Practitioner/<id>&date=ge…&date=lt…` (instants of local midnight, UTC with `Z`); the
  practitioner's own, the admin's chosen practitioner (`PractitionerPicker.tsx`); previous, this and next week;
  each card the local times, the patient's name only where the appointment carries it, a status tag when not
  booked; the appointment under way gets the gradient and "Now", the only gradient in the app. At phone width the
  days stack and empty days are hidden.
- `web/src/AppointmentDetail.tsx` (`/appointments/<id>`): date and times in the zone, the status; for a
  practitioner the `Patient` read, its birth date and phone behind a "Patient details" disclosure, and on a 404
  the name from the appointment or none ("Appointment"); "Mark fulfilled" and "Mark no-show" (practitioner only,
  disabled before the start, with a hint), "Cancel appointment" with a confirmation (practitioner and admin);
  no actions once the appointment is no longer booked. Error codes map to fixed sentences.
- `web/src/Availability.tsx` ("Working hours", `/availability`): slot length and weekly rows (day, from, to,
  remove), "Add hours", save with `If-Match` of the version read (`W/"0"` before the first edit), reload after a
  412; the admin picks the practitioner.
- `web/src/Practitioners.tsx` (now `/practitioners`): each practitioner's specialty; for the admin a select per
  role, saved on change with `If-Match` of the role's version, the previous choice restored on a refusal. The
  options are `GET /specialties` as they come, in their order and with their names.
- `web/src/ClinicSettings.tsx`: the time zone (a select of the browser's IANA zones) and "List the clinic in the
  patient app", written as the `organization-timezone` and `organization-listed` extensions in the same
  `Organization` update (other extensions kept).
- Shared: `load.ts` (`useLoad`, a load per set of dependencies, stale answers dropped), `staff.ts` (roles with
  their practitioners, specialty and version; the clinic's zone), `time.ts` (days and instants in a zone without
  a library), `api/fhir.ts` (hand-written `Appointment`, `Patient`, `CodeableConcept`, `Extension`, the
  participant and settings helpers), `test/fixtures.ts` (synthetic accounts, clinic and staff), three icons.
- `App.tsx`: the calendar is the home screen; nav Calendar, Working hours, Practitioners, Clinic settings (admin).
  A practitioner on `/clinic` now gets "This page does not exist." instead of the practitioner list.
- API, two small additions the screens need (below): `GET /specialties` in `practitionerrole/handler_http.go`
  and `practitionerId` on the `Account` of `GET /auth/me`; `api/openapi.yaml` (`listSpecialties`,
  `SpecialtyList`, `Account.practitionerId`); `make gen-clients` (TypeScript and Dart).
- `web/vite.config.ts`: the dev server proxies `/practitioner-roles`, `/specialties` and only
  `^/appointments/<id>/(cancel|status)$`, since `/appointments/<id>` is a screen of the app.
- `docs/privacy/dpia.md`: package 8.

**Red first.** practitionerrole: `GET /specialties = 404 404 page not found`. auth: the `/auth/me` body without
`practitionerId`. internal/api: the app test passed at once (the handler test had already added the route); run
with the route removed it failed, and was restored. Web: `time.test.ts` and `Calendar`, `AppointmentDetail`,
`Availability` failed on the missing modules; `api/fhir.test.ts` 4 failed (`settingsOf`, `withSettings`,
`participant` undefined); `Practitioners` 6 failed (no specialty shown, no `Specialty of …` select);
`ClinicSettings` 3 failed (no `List the clinic…` or `Time zone`), then 2 when the hint sat inside the label;
`App` 7 failed (no Calendar heading, no routes).

**Checks.** `make lint` 0 issues, migrations unchanged; `make test` passes; `make archcheck` ok; `make cover`
passes, every package at 80% or more (auth 95.7%, practitionerrole 96.3%, api 90.9%; total 94.7%);
`make web-lint` 0 warnings, 0 errors; `make web-typecheck` ok; `make web-test` 113 passed (13 files);
`make web-build` ok (283.66 kB JS, 88.53 kB gzip; 8.88 kB CSS); `make gen-clients`
leaves no diff; `make mobile-analyze` no issues (the Dart client changed). No FHIR change, so no
`make fhir-validate` run.

**Browser walk.** The dev stack with `salus:dev` rebuilt from this branch (api, worker, scheduler, notifier
recreated; packages 6 and 7 were not in the running api before), the owner's Vite dev server on 5173 reused. A
scratch Playwright script (not in the repo; `mcr.microsoft.com/playwright:v1.56.1-noble`, Chromium, host network)
on a clinic made with `salusctl`. At 1280 × 800: the admin listed the clinic (time zone kept) and gave the
practitioner a specialty; the practitioner set 5-minute slots on three days; a synthetic patient (API only)
completed a profile, consented and booked three slots; the practitioner's week showed them in local time, the
detail showed the profile behind the disclosure with the outcome buttons disabled before the start; the admin's
week with the picker, the admin cancelled one after the confirmation (no `Patient` request from the admin's
pages); once the soonest started, its card carried the gradient and "Now" and the practitioner marked it
fulfilled. At 390 × 844 every screen for both roles. No horizontal overflow, no console error. The api,
notifier, worker and dev-server logs held no name, address or phone. Found and fixed on the way: the
confirmation's red button was invisible (`.actions button:not(.primary)` outranked `button.danger`); a direct
load of `/appointments/<id>` was proxied to the API (404) by the `/appointments` prefix; at 390 px the nav
labels wrapped inside their links and the hours rows broke mid-row.

**Readings and decisions.**
- **The specialty list comes from the API.** No endpoint listed the specialties, and a copy in the web app would
  drift from `practitionerrole.Specialties()` (paediatrics' code is still open). `GET /specialties` answers that
  list to anyone: it is fixed, public like the directory that shows it, and holds no personal data, so it is not
  audited. The editor shows it as it comes, paediatrics included, with no special case; no test fixture uses
  paediatrics.
- **`practitionerId` on `/auth/me`.** A practitioner must search appointments by themselves (package 7), and the
  web app had no way to know its own `Practitioner` id: the account holds it, `/auth/me` did not give it. Added
  as an optional field (opaque id, practitioner accounts only); the role for the working hours is found in the
  clinic's role search by that id.
- **The admin's detail does not ask for the `Patient`.** The API answers an admin 404 by rule, so asking would
  only write a refused-read audit event per view; the name comes with the appointment. A practitioner's 404
  (no full access) hides the profile, as the plan asks.
- **Outcome buttons for practitioners only.** An admin cannot set a status (SPEC); the buttons are not shown to
  them. Cancel is shown to both.
- **Specialty saved on change.** One control per row, no separate save; a refused save restores the previous
  choice. A role without a specialty shows "Choose a specialty" as a disabled first option (the API cannot clear
  one).
- **Time zones in the browser.** `Intl` only (no date library): the start of a local day is found by
  correcting a UTC guess with the zone's offset twice, tested across both 2027 changes in Europe/Sarajevo and
  in America/New_York. The time zone select lists `Intl.supportedValuesOf('timeZone')`; the API validates.
- **Cancelled appointments stay in the calendar**, muted and struck through, so staff see that the time was
  booked and cancelled.
- **`useLoad` and the hooks rule.** The hook takes its dependencies from the caller, which
  `react-hooks/exhaustive-deps` cannot check; one `oxlint-disable-next-line` on that line, with the reason.
- **Search cost** (package 6's open note): one practitioner-week per request; not measured, and nothing in the
  walk was slow.

**Deferred.** Mobile (package 9); the e2e journeys (package 10). The browser's own `<input type="time">`
follows the browser's locale (12-hour in an en-US browser); not changed.

**Dev stack.** No migration. `salus:dev` rebuilt from this branch; api, worker, scheduler and notifier recreated
with it; agent, landing, livekit, mailpit, redis, postgres untouched; the owner's `node` dev server kept running
(it reloaded `vite.config.ts` by itself). The owner's demo clinic and package 6's stack-check data untouched.
Left from the walk, all synthetic: the listed clinic "Synthetic walk clinic (booking 8)" with its admin
`walk8-admin@example.test` (MFA reset once with `salusctl` and enrolled again during the script's setup) and
practitioner `walk8-cy@example.test` (Cy Walkexample, cardiology, 5-minute slots on Wednesday, Thursday and
Friday), their open sessions (12 h); the patient `walk8-patient@example.test` (Ada Walksample) with a profile, an
active consent and three appointments (one fulfilled, one cancelled, one booked for Thursday 5 November 09:00
local); the slots generated for 28 days; the e-mail and reminder jobs; the audit events; the invite and
appointment e-mails in Mailpit.


### 2026-11-04 — package 9: mobile

**Built.**
- `mobile/lib/profile_screen.dart`: the profile, created on the first save and updated at the version read
  (`versionId` in the body); given and family name, date of birth (`YYYY-MM-DD`), optional phone (spaces, dashes,
  dots and brackets taken out, then E.164). The API's rules are checked before sending; a 409 `version_conflict`
  reads the profile again and asks the patient to check it. After sign-in or registration, a patient without a
  profile sees it first ("Not now" leads home with a reminder); a booking answered `profile_required` opens it.
- `home_screen.dart`: the profile read, then "Book an appointment", "My appointments", "My consents", "My
  profile" and "Sign out" (now a text button). A missing profile is read again when a pushed screen returns.
- `clinics_screen.dart`: `Organization?listed=true`; a clinic's practitioners with the specialty as the API gives
  it (`PractitionerRole?organization=…&_include=PractitionerRole:practitioner`).
- `slots_screen.dart`: `Schedule?actor=PractitionerRole/…`, then `Slot?schedule=…&status=free&start=ge…&start=lt…`
  for one local day at a time (the instants of local midnight, in UTC), today to 27 days ahead, times in the
  clinic's zone with the zone named. `BookScreen`: `POST /appointments`; `profile_required` and
  `consent_required` open the profile or the consent screen and book again; `slot_unavailable` says so and the
  times are read again; then "My appointments" with "Your appointment is booked.".
- `consent_screen.dart`: what is shared, who sees it, what for, how to revoke; "Agree and book" grants
  (`POST /consents`), "Not now" books nothing.
- `appointments_screen.dart`: `Appointment?patient=Patient/<own id>`, upcoming then past, each with the clinic's
  date, times and zone, practitioner, clinic and status; "Cancel appointment" with a confirmation while the start
  is 24 h or more ahead, disabled with a hint inside 24 h.
- `consents_screen.dart`: `GET /me/consents` with the clinic's name, given and revoked dates; "Revoke consent"
  with a confirmation that future appointments there are cancelled and past ones stay.
- Shared: `fhir.dart` (hand-written `Organization`, `PractitionerRole`, `Slot`, `Appointment` and the searches,
  through the generated client's `invokeAPI`, so they carry the bearer token), `booking.dart` (the JSON actions
  through the generated `PatientApi`, `ConsentsApi`, `AppointmentsApi`; `places`), `zones.dart`, `loading.dart`,
  fixed sentences in `messages.dart`, flat cards and a back button on pushed pages in `ui.dart`. `AuthFailure` is
  now `ApiFailure` (it covers every call) and `apiCall` is shared. The clock is injected (`SalusApp.now`).
- `pubspec.yaml`: `timezone` 0.11.2, pinned exact (below). `docs/privacy/dpia.md`: package 9.

**Red first.** zones: the test failed to compile, `Zone` undefined, then 3 of 5 failed on the 10-year data set
(below). fhir: the test failed to load without `package:salus/fhir.dart`. The widget tests (profile, book,
appointments, consents, and the changed home, register and sign-in tests) failed to compile without
`package:salus/booking.dart` and `SalusApp(booking:, now:)`; once the code was in, 12 failed because their taps
hit widgets below the 800 × 600 test view (the tap helpers now scroll the target into view first). Three tests
were written after their code (the home re-reading a missing profile, a grant answered `consent_active`, a failed
list with "Try again"); the first two were checked against a mutation of the code (both failed, then restored).

**Checks.** `make mobile-test` 77 passed; `make mobile-analyze` no issues; `make gen-clients` leaves no diff (no
`openapi.yaml` change); `make lint` 0 issues, migrations unchanged; `make archcheck` ok; `make test` passes;
`make cover` passes, every package at 80% or more (total 94.7%, no Go change). `dart format -l 120` (the width the
existing files are formatted at) over `lib` and `test`. Phone width: a widget test runs the booking at 390 × 844
and fails on any overflow. Smoke against the dev API, anonymous reads only: the app's exact query shapes for the
listed clinics, the role search with the include, the schedule and a day's free slots answered 200 with data;
the FHIR content type has no charset (below). Running the app on a device is the manual host step (README).

**Readings and decisions.**
- **Time zones need a package.** Dart's `DateTime` knows only UTC and the device's zone; showing a slot in the
  clinic's IANA zone needs the zone rules. `timezone` (dart-lang, pinned 0.11.2) with its `latest_all` data:
  `latest` and `latest_10y` leave out link zones, and `Europe/Sarajevo`, the API's default (accepted by Go's
  `time.LoadLocation`), is a link. An unknown zone (or a clinic no longer listed) falls back to UTC, named. Dates
  and month names are written out in `zones.dart`; no `intl`.
- **FHIR bodies are decoded as UTF-8.** The API sends `application/fhir+json` without a charset, so `http` would
  read the body as Latin-1; `fhir.dart` decodes the bytes (tested with a clinic name with "Č").
- **Which clinic an appointment is at.** FHIR `Appointment` names the role but no organisation, and no role read
  is public. Every booking needs a consent, so `GET /me/consents` names the clinics; the app reads the listed
  clinics and, for each consented one, its public role search, and maps role to clinic and practitioner. A clinic
  no longer listed shows as "A clinic no longer listed", its times in UTC. No API change.
- **The patient's own id.** `Appointment?patient=Patient/<id>` with the id from `GET /me/patient`; without a
  profile there are no appointments and no search.
- **Consent is asked only when the API says so.** The app books first; `consent_required` opens the consent
  screen, as the plan asks, so the app holds no copy of the consent rules. A grant answered `consent_active`
  (granted elsewhere in the meantime) books. The profile is asked the same way on `profile_required`, which the
  API checks before the consent.
- **24 hours.** Cancel is enabled while `start − now ≥ 24 h` on the device's clock (24 h − 1 s disabled, 24 h
  enabled, as the API); the API stays the judge and its 409 `cancellation_window_closed` is shown.
- **Revocation copy.** The cancellation runs in the `consent_revoked` job, so the app says future appointments
  "will be cancelled" and that an e-mail follows for each, not that they are cancelled already.
- **Status names for patients.** `fulfilled` is "Attended", `noshow` "Not attended".
- **Specialty** is shown as the API gives it (`text`, else the coding's display); no fixture uses paediatrics.
- **Copy** is plain, sentence case, no exclamation marks; nothing reads as a diagnosis. The sample phone in the
  hint is from the fake `+387 60 000 0xxx` range.

**Deferred.** The e2e journeys (package 10). A date picker for the birth date (a text field with the format, for
now).

**Dev stack.** Untouched: no migration, no image rebuilt; the smoke made anonymous FHIR reads only. The owner's
`node` dev server kept running; the owner's demo clinic and earlier packages' synthetic data untouched.


### 2026-11-04 — package 10: end-to-end

**Built.**
- `test/e2e/booking_test.go` (build tag `e2e`), the owner's two journeys, each on fresh synthetic data (random
  e-mail addresses and clinic name suffix, a new clinic per journey), so reruns never collide:
  - `TestBooking_PatientBooksAndPractitionerCancels`: a clinic made with `salusctl create-clinic`; its admin
    accepts the invite and lists the clinic through the FHIR `Organization` update (the `organization-listed`
    extension, `If-Match` of the version read) and invites a practitioner, who sets 30-minute slots on every day
    (`PUT /practitioner-roles/{id}/availability`, `If-Match: W/"0"`) and waits for the `slots_generate` job's free
    slots; the role is found as anyone finds it (`Organization?listed=true`, then the public
    `PractitionerRole?organization=…&_include=PractitionerRole:practitioner`). A patient registers, logs in, has
    no profile (404), completes it, finds the clinic, `Schedule?actor=…`'s free slots (`Slot?schedule=…&status=free&start=ge…&start=lt…`),
    is refused with 409 `consent_required`, consents (`POST /consents`) and books (201 `booked`); "Your
    appointment is booked" arrives in Mailpit with the clinic's name and the app link. The practitioner's
    `Appointment?practitioner=Practitioner/<id>&date=ge…&date=lt…` holds the appointment with the patient's name,
    and `Patient/<id>` answers the profile; the practitioner cancels (`POST /appointments/{id}/cancel`); the
    patient's `Appointment/<id>` read is `cancelled` with reason `prov`, and "Your appointment is cancelled"
    arrives.
  - `TestBooking_RevokedConsentEndsAccessAndCancelsTheFutureAppointment`: the same up to the practitioner
    reading the profile; the patient revokes (`GET /me/consents`, `POST /consents/{id}/revoke`); the
    practitioner's `Patient` read is 404 at once; the `consent_revoked` job's cancellation is waited for as a
    condition (the patient's read polled every 250 ms, up to 30 s), the reason is `pat`, and the cancellation
    e-mail arrives.
- `test/e2e/stack_test.go`: `send` with request headers, `mustWith` (for `If-Match`), `refusal` (status and
  `{"error"}` code), `practitionerId` on the account, and `mailText` (waits for the one e-mail to an address,
  optionally with a subject, deletes it so Mailpit is left as found, returns its text); `inviteToken` uses it.
  `test/e2e/accounts_test.go`: `enrolFromInvite` (accept and enrol, keep the enrolment's session), which
  `acceptInvite` now starts with before its login and verify.

**Red first.** The journeys passed on their first run against the stack rebuilt from this branch (the
behaviour was built in packages 1–9), so their red came from a mutation: with the `consent_revoked` worker made to
cancel nothing (`internal/worker/revocation.go`), the image rebuilt by `make e2e`, the second journey failed with
"the consent_revoked job cancels the future appointment: not within 30s" and the first passed; the mutation was
reverted and the image rebuilt by the next `make e2e`.

**Checks.** `make lint` 0 issues, migrations unchanged (lint caught a `time.Now` in the test; it uses
`clock.System` as `stack_test.go` does); `make test` passes; `make archcheck` ok; `make cover` passes, every package
at 80% or more (total 94.7%, no production change); `make fhir-validate` passes; `make gen-clients` leaves no
diff; `make web-lint` 0 warnings and errors, `make web-typecheck` ok, `make web-test` 113 passed, `make web-build`
ok; `make mobile-analyze` no issues, `make mobile-test` 77 passed. `make e2e` twice in a row after the change: both `ok` (4.4 s, 4.6 s), the login counter for the IP at 12 after the two runs (6 per run, as counted).

**Readings and decisions.**
- **The login rate limit and reruns.** The first two back-to-back `make e2e` runs after the journeys were written
  failed with 429 on `POST /auth/login`: the accounts SPEC allows 30 login attempts per client IP per 15-minute
  window (Redis fixed window), every e2e request comes from the one `tools` container's IP, and the journeys first
  logged in their admin, practitioner and patient (10 logins per suite run with the accounts journeys; with the
  runs made while writing the tests the window passed 30). The limit is a settled accounts decision and was not
  touched; the journeys now keep the staff sessions that the invite's enrolment opens (`enrolFromInvite`) and log
  in only the patient, so a suite run spends 6 of the 30 (the accounts journeys still test the login after
  enrolment and the verify step). Five full runs fit in one window; a sixth inside 15 minutes would get 429. Not a
  timing flake: a counter the suite shares with nothing else, now with headroom. The counter was waited out as a
  condition (its key gone from Redis), not by a sleep in a test.
- **Slots for the journeys.** 30-minute slots from 00:00 to 23:30 on all seven days, so a slot is always free
  whatever the hour of the run; the journey books the first free slot between one and 72 hours ahead, so it is
  never about to start. The slots' existence after the availability save is waited for as a condition (the
  `slots_generate` job runs in the worker).
- **Fresh data, not cleaned up.** Each journey makes its own clinic, staff and patient with random addresses, so
  reruns never meet earlier data; nothing is deleted from the database (append-only audit, `keep` rules). Mailpit
  is left as found: each e-mail a journey waits for is deleted once read.
- **What is asserted about the e-mails.** The subject (from SPEC §2 "Jobs"), the clinic's name and the link; the
  exact text is the notifier's golden files' job (package 6).

**Definition of done.**
- [x] Every package committed on `feat/booking` (1–9 before, this one now).
- [x] `PROGRESS.md` complete: every package with its red run, checks, readings and dev stack; the slot-generation
  benchmark in package 5.
- [x] All check sets pass locally: Go, FHIR, Clients, Web, Mobile, and `make e2e` twice in a row.
- [ ] CI on the PR into `main`: runs when the PR is opened; not visible from this session.

**Deferred.** Nothing in this package. The open questions stay under "Waiting on people".

**Dev stack.** `make e2e` (`make up --build`) rebuilt `salus:dev` from this branch and recreated api, worker,
scheduler and notifier with it (also after the mutation run, so the running images are this commit's); postgres,
redis, mailpit, livekit, landing, agent and the owner's `node` dev server on 5173 left running. No migration. Left
on the dev database from the runs, all synthetic and fresh per run: clinics named "E2E synthetic clinic <hex>"
(listed) with `e2e-admin-…@example.test` and `e2e-practitioner-…@example.test` staff, their schedules and
generated slots; `e2e-patient-…@example.test` patients ("Ena Syntheticova", born 1990-05-17, phone
+38760000001) with consents and cancelled appointments; the audit events; the accounts journeys' clinics and
accounts as before. From the mutation run, one appointment stays `booked` after its consent was revoked (the
mutated job cancelled nothing), in its "E2E synthetic clinic <hex>" for 2026-11-04 23:00 UTC; it was booked less
than 24 h ahead, so it has no reminder job, and the clinic sees it at name level only. Mailpit holds none of the journeys' e-mails. The
owner's demo clinic and earlier packages' synthetic data untouched.

### 2026-11-05 — review of PR #18

The owner's decisions on the open questions (see "Decisions"), built test first on `feat/booking` in one
commit. Each new test was run red before the code: the build failures for the missing symbols, and the
behaviour tests against the old code or with the new check stubbed out.

- **Paediatrics.** `practitionerrole.Specialties()` gives 394537008 "Pediatric specialty" (text "Paediatrics")
  instead of 394733009; 394733009 is no longer accepted. Red: the specialty table test and `SpecialtyByCode`
  (394537008 not found, 394733009 still found). New golden file `practitioner_role_paediatrics.json` from a
  handler test; `make fhir-validate` passes it with 0 errors, the same `dom-6` narrative warning as every
  other golden file and no binding note at all (`-tx n/a`, so SNOMED codes are not looked up).
- **Past visits after a revocation.** `access.RecordsOutcome(viewer, facts, appt)`: the appointment's
  practitioner, with an active consent or for a past appointment after a revocation; table-tested over every
  role × consent × appointment × clinic. `appointment.Service.SetStatus` asks it instead of requiring `Full`.
  Migration 00019 (not on `main`, so changed in place) gains `appointment_clinic_record_past`, an `UPDATE`
  policy for past appointments after a revocation whose `WITH CHECK` allows only `fulfilled` and `noshow`.
  Red: the service test (403) and the migration test (0 rows; no RLS error on setting `booked` or `cancelled`).
  Cancelling a past appointment after a revocation stays 403 in the service and 42501 in the database. The web
  app needs no change: it shows both buttons on a booked appointment, and a refused cancel already shows "You
  cannot change this appointment".
- **Retention audit.** Migration 00020 adds `audit_event.entity_name` and `entity_count` (both or neither, the
  count never negative; a migration test, red without the constraint). `auditevent.AuditEvent` gains
  `EntityName` and `EntityCount`, refused when a count has no table or is negative. `retention.Service.Apply`
  records one event per delete rule's table after its purge: agent `retention_apply`, action `D`, `rest` +
  `delete`, purpose `HOPERAT`, the table and the row count, zero included. The port `AuditAges` became `Audit`
  (count and record, `auditevent.Service`). The worker test checks the three events of a real run.
- **Public practitioner listing.** `practitionerrole.NewService` takes `Ports` with `Clinics`
  (`organization.Service.Read`, as booking already does) and `People` (`practitioner.Service.PublicNames`).
  `PublicByOrganization` reads the clinic first and returns nothing, without reading a role, when it is
  unlisted or unknown, then keeps only roles whose practitioner is active. Red with the two checks stubbed out.
  `internal/api` and the router's real-database directory tests pass with the new wiring.
- **`make cover` in CI.** It never left: `.github/workflows/go.yml`'s `test` job runs `make cover`, then
  `make bench-hash` (added by accounts package 2, `dd9bd94`, which only appended the benchmark step). PR #18's
  run printed every package and "total: 94.7% (floor 80% per package)". My earlier report that CI had no
  coverage job was wrong; nothing to restore.

**Checks.** `make lint` (0 issues; check-migrations: ok), `make archcheck`, `make test`, `make cover` (total
94.7%, every package at 80% or more), `make fhir-validate` (0 errors), `make e2e`: all pass.

**Dev stack.** The dev database was reset afterwards at the owner's request (`docker compose down` with
volumes, then `make up`), so none of the synthetic leftovers listed under package 10 remain.

### 2026-11-05 — consent text (after the merge of PR #18)

**Correction.** The 2026-11-05 review entry above, and the PR #18 merge, rest on the consent text having been
reviewed. It had not: the owner saw the consent screen's text for the first time after the merge. The question
asking to approve it came with the merge itself, as one choice, so it was never really a review. From now on the
text goes to the owner on its own, before anything that depends on it.

**Changes (owner, 2026-11-05), test first on `feat/consent-copy`:**
- The consent screen's "How to revoke" and the revoke dialog now say that the clinic no longer sees the
  profile but keeps the name and the details of appointments that have already taken place, as its care
  records, and that future appointments are cancelled.
- A "Who is responsible" section: "[Clinic] is responsible for your data. Salus processes it on the clinic's
  behalf.", and a "Read the privacy notice" link to `PRIVACY_NOTICE_URL`. That setting is a build-time
  `--dart-define`, like `API_BASE_URL`, and defaults to the placeholder `https://salus.invalid/privacy`.
- The link opens in the device's browser through `url_launcher` 6.3.3, Flutter's own plugin, pinned exact. When
  the device cannot open it, the screen says "The privacy notice could not be opened." and stays open.
- The setting reaches the screen through an `InheritedWidget` (`PrivacyNotice`) above the app's routes, so it is
  not threaded through six screens. Tests inject the opener and record the links opened.
- Red first: the harness change did not compile against the app; with the plumbing in place, the four new or
  changed tests failed on the missing text and link.
- `SPEC.md` gains §7 "Before production", which collects the open items and adds the owner's three new ones.
  `docs/privacy/dpia.md` and `README.md` are updated.

**Checks.** `make mobile-analyze` (no issues), `make mobile-test` (79 tests pass). No Go, web or API change.

