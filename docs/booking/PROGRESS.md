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
- Owner (package 6): the notifier's e-mail jobs read in the clinic's scope narrowed to the one appointment the
  job names (`app.appointment_id`), so a cancellation after a revocation still reaches the patient; confirm
  this reading of SPEC §2 "Row-level security scopes" (readings, package 6).
- Owner (package 6): after a revocation, staff see a past appointment at the name, times and status only and
  cannot change it (403), so a practitioner cannot mark a past visit fulfilled or no-show once the patient has
  revoked; confirm, or say whether the care record needs that change (readings, package 6).
- Before production (package 7): the canonical base of our FHIR definitions, `https://salus.invalid/fhir` (the
  `organization-listed` and `organization-timezone` extensions and the `listed` SearchParameter), a placeholder
  like `PATIENT_APP_URL`. It must be the product's real domain before any client depends on it: a changed
  extension URL breaks every client.
- Owner (package 7): `practitionerrole.Service.PublicByOrganization` cannot check that the clinic is listed (a
  role row does not carry it), so for roles the listing rests on row-level security alone; run on a login that
  bypasses it, an unlisted clinic's roles show. Production runs as `salus_app`, so nothing leaks; a service-level
  check would need the organization service in the role domain. Say whether to add it.
- Specialty codes (package 4): the R4 `c80-practice-codes` value set has no paediatrics concept; paediatrics
  is `394733009` "Medical specialty--OTHER--NOT LISTED" with the text "Paediatrics" until someone confirms a
  code (a concept from outside this value set would need the SPEC changed). ENT
  (`418960008` "Otolaryngology") and orthopaedics (`394801008` "Surgery-Trauma and orthopedics") were chosen
  among the value set's options; confirm.

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
