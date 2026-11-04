# Booking progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Retention job | done | `booking(1)` |
| 2 Patient | done | `booking(2)` |
| 3 Consent and access | done | `booking(3)` |
| 4 Directory additions | done | `booking(4)` |
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
