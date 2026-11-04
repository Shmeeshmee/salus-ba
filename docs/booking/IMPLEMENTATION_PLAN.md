# Booking implementation plan

Read `SPEC.md` first. Packages run in order; each is one commit `booking(<n>): <summary>` on `feat/booking`,
made with `/run-package docs/booking <n>`.

## Rules for every package

- Test first: write the failing test, run it and record the failure in `PROGRESS.md`, then the least code to
  pass, then refactor. Service tests with hand-written fakes and `clock.Fixed`; repository tests on
  `pgtest.NewDatabase` (and `NewAppDatabase` for every RLS check); handlers with `httptest`, FHIR handlers
  against golden files.
- Every new domain is added to `docs/architecture/archcheck/config.yaml`, every new table to `ownership.yaml`
  with `personal_data` and a retention rule, every domain with queries to `sqlc.yaml`, in the commit that adds
  it. The archcheck allowlist never grows.
- Every migration is a new file. Grants to `salus_app` (and to `salus_retention` for a `delete` rule), `ENABLE
  ROW LEVEL SECURITY` and the policies sit in the migration that creates the table. Policies read settings as
  `NULLIF(current_setting('…', true), '')`.
- Every read or write of `Patient`, `Consent` or `Appointment` audits in the same transaction, refusals
  included; a test per service method asserts the event.
- A change to `api/openapi.yaml` is followed by `make gen-clients` in the same commit.
- No personal data in a log line, error, panic, golden file, fixture or e-mail beyond SPEC §4. A test asserts
  the log output of each new handler and job holds no name, birth date, phone or e-mail.
- Start-up and timing tests wait for a condition (as in `4c62e68`), never on a longer sleep.
- Pin every version. Update `PROGRESS.md` in the same commit.

## Check sets

Every check runs through `make`, in Docker. Never `go`, `npm` or `flutter` on the host.

- **Go**: `make lint`, `make test`, `make archcheck`, `make cover` (every package at 80% or more).
- **FHIR**: `make fhir-validate`.
- **Clients**: `make gen-clients` leaves no diff.
- **Web**: `make web-lint`, `make web-typecheck`, `make web-test`, `make web-build`.
- **Mobile**: `make mobile-analyze`, `make mobile-test`.
- **Stack**: `make up`, then the package's manual check, then `make down`.

## Packages

### 1. Retention job

Migration: role `salus_retention` (SPEC §2 "Retention privileges"; same `DO` block pattern as `salus_app`,
`BYPASSRLS`), `GRANT SELECT, DELETE ON session, invite`. `cmd/migrate up` sets its password from
`RETENTION_DATABASE_PASSWORD` (`config`); compose and CI get a synthetic value and `RETENTION_DATABASE_URL`
for `worker`; `pgtest` gains a pool helper for the role. `internal/retention`: `Service.Apply` with the
`Purger` port and the overdue check for `remove-range` (port `AuditAges` → `auditevent.Service.OldestRecorded`).
`auth/service_retention.go`: purgers for `session` (30 days after `revoked_at` or `expires_at`, whichever is
first) and `invite` (30 days after `accepted_at` or `expires_at`), SQL in `auth`'s `query.sql`.
`internal/config`: `RETENTION_DATABASE_URL` in a loader of its own, called only by `cmd/worker`.
`internal/worker`: the `retention_apply` job, the only one built on the retention pool; refuses to start when
a `delete` or `remove-range` rule has no applier. `cmd/scheduler`: River client with the periodic job (daily 03:00 UTC),
replacing today's placeholder; `cmd/scheduler/main_test.go` keeps its start-up wait.
Tests: `salus_retention` has `BYPASSRLS`, is not a superuser, owns nothing, can delete an old session and an
old invite of any clinic, cannot `INSERT` or `UPDATE` them, and cannot `SELECT`, `DELETE` or `TRUNCATE`
`audit_event`; purgers delete exactly the rows past their period at a fixed clock (boundary ± 1 s) and keep
the rest; `Apply` calls each applier once, does nothing for `keep` and `restrict`, fails with a count-only
error when an audit row is overdue; every `delete` and `remove-range` rule has an applier; the job's log holds
only table names and counts; nothing else can use the retention credentials: the api, notifier and scheduler
configurations do not read `RETENTION_DATABASE_URL` (set in the environment, absent from what they load),
only the `worker` service has it in `deployments/docker-compose.yml`, and every other worker in
`internal/worker` is built on the `salus_app` pool; the scheduler enqueues the periodic job once with two clients running.
Checks: Go; Stack (the job runs in the worker and logs its counts).

### 2. Patient

Migration: `patient` (SPEC §2 "Patient table", §4), unique `account_id` referencing `account`, RLS with the
patient branch only (the consent branch is added in package 3 with `consent`), grants `SELECT, INSERT,
UPDATE`. `internal/patient` (context `patient`): model, errors, `Service.Me`, `Service.SaveMe` (create on
first save, update with `version_id` after), validation (SPEC §2 "Profile"), `Service.Read` for later callers
(staff access arrives in 3), audit (`PATRQT`). `pkg/postgres` settings for the scopes if helpers are needed.
Handlers `GET`/`PUT /me/patient` (patients only; staff 403); `openapi.yaml`; `make gen-clients`. Retention rule
`keep`. `docs/privacy/dpia.md` gains the patient profile.
Tests: validation table (names, birth date bounds, E.164); a second save with a stale version is 409; a staff
caller gets 403 and an audit event; as `salus_app` with `app.patient_id` set, another patient's row is
invisible, and with no scope nothing is; the log holds no profile field.
Checks: Go, Clients.

### 3. Consent and access

Migration: `consent` (partial unique index on `(patient_id, organization_id) WHERE status = 'active'`, RLS:
patient scope or clinic scope, grants), and the `patient` policy's consent branch (a new `CREATE POLICY` in
this migration; SPEC §2; the past-appointment branch comes with `appointment` in package 6).
`internal/access` (context `privacy`): `Profile` and `Appointment` (SPEC §2 "Access rule"), pure, exhaustive
table tests over role × consent (active, revoked, never) × appointment (none, past, future, other
practitioner) × clinic match. `internal/consent`: `Grant`, `Revoke` (state, `revoked_at` and audit; the
`consent_revoked` job is added in package 6, when there are appointments to cancel), `Mine`, `Read`,
`State(patientID, organizationID)` (active, and the latest `revoked_at`). `consent` imports nothing from
scheduling. `patient.Service.Read` for staff now uses `access` (the appointment facts come from a port that
returns none until package 6). Handlers `POST
/consents`, `POST /consents/{id}/revoke`, `GET /me/consents`; `openapi.yaml`; `make gen-clients`. Retention
rule `keep`. DPIA: consent.
Tests: grant twice is 409; revoke of another patient's consent is 404; grant after revoke creates a new row;
as `salus_app` in a clinic scope, the patient row is visible with an active consent and invisible right after
the revoke commits; a patient's scope never sees another patient even with a clinic set; audit events on each
call, refusals included.
Checks: Go, Clients.

### 4. Directory additions

Migration: `organization.listed boolean NOT NULL DEFAULT false`, `organization.time_zone text NOT NULL
DEFAULT 'Europe/Sarajevo'`, `practitioner_role.specialty_code text`, grants for the new updatable columns,
and the directory-scope `SELECT` policies on `organization`, `practitioner`, `practitioner_role`.
`practitionerrole`'s twelve specialties (SPEC §2 "Specialty"), codes and displays copied from
`valueset-c80-practice-codes` in the R4 core package and recorded in `PROGRESS.md`.
`organization`: `listed` and `time_zone` in the model and update (admin, `If-Match`; zone validated),
`ListListed`. `practitionerrole`: the specialty list (SNOMED code and display, from `c80-practice-codes`),
`SetSpecialty` (admin, `If-Match`), `PublicByOrganization` (listed clinic, active roles). `practitioner`:
`PublicNames` for those roles. `time/tzdata` imported by `cmd/api`, `cmd/worker`, `cmd/notifier`,
`cmd/scheduler`. Handler `PUT /practitioner-roles/{id}/specialty`; `openapi.yaml`; `make gen-clients`.
Tests: as `salus_app` in the directory scope, an unlisted clinic and its practitioners are invisible and no
column beyond the public ones is returned by the public queries; an unknown time zone or specialty is 400; a
practitioner cannot set a specialty; staff reads stay audited, public reads write nothing.
Checks: Go, Clients.

### 5. Schedule and slot

Migrations: `schedule` and `slot` (SPEC §2 "Schedule shape", "Double booking"), RLS (clinic scope, plus the
directory scope for listed clinics), grants; `GRANT SELECT, DELETE ON slot TO salus_retention`.
`internal/schedule`: `Availability`, `SaveAvailability` (practitioner own role, admin any; `If-Match`;
enqueues `slots_generate` with the schedule and clinic ids), `Read`, `ByActor`. `internal/slot`:
`Generate(ctx, scheduleID, now)` (SPEC §2 "Slot generation"), `Search`, `Hold` (free → busy), `Release`,
purger (`free` slots 30 days after their end). `internal/worker`: `slots_generate` (one schedule, or all when
periodic); `cmd/scheduler`: the hourly periodic job. Handlers `GET`/`PUT /practitioner-roles/{id}/availability`;
`openapi.yaml`; `make gen-clients`. Retention rules: `schedule` `restrict`, `slot` `delete`.
`BenchmarkGenerate` for 50 schedules × 28 days (first run and an idempotent re-run), numbers in `PROGRESS.md`.
Tests: generation across both 2027 DST changes in `Europe/Sarajevo` (skipped and first-occurrence times), a
re-run inserts nothing, a narrowed rule deletes only future free slots, a busy slot survives; two overlapping
slots in one schedule are rejected by the database; invalid rules (overlap, end before start, length not a
multiple of 5) are 400.
Checks: Go, Clients.

### 6. Appointment

Migration: `appointment` (SPEC §2 "Double booking", §4), RLS (patient scope; clinic scope with an active
consent, or for an appointment that started before the latest revocation; SPEC §2 "Past and future after
revocation"), grants; the `patient` policy's past-appointment branch.
`internal/appointment`: `Book`, `Cancel`, `SetStatus`, `Read`, `Search`, `CancelForRevokedConsent`; the
`Consents`, `Slots`, `Patients`, `Clinics` and `Jobs` ports; access through `access`; audit, including the
`Patient` event when the name is shown. `consent.Service.Revoke` gains its `Jobs` port and enqueues
`consent_revoked` (patient id, clinic id) in its transaction; the `consent_revoked` worker in
`internal/worker` calls `CancelForRevokedConsent`. `app.go` wires `patient`'s appointment facts to
`appointment`; nothing in `consent` refers to `appointment`. `internal/notification`: the three e-mail texts
(time in the clinic's zone with its name). `internal/worker`: the three jobs on the `email` queue. Config
`PATIENT_APP_URL`. Handlers `POST /appointments`, `/appointments/{id}/cancel`, `/appointments/{id}/status`;
`openapi.yaml`; `make gen-clients`. Retention rule `keep`. DPIA and `processors.md` updated (no new processor).
Tests: two concurrent bookings of one slot, and of overlapping slots of one role, give one 201 and one 409
(real database); booking without a profile, without consent (`consent_required`), on an unlisted clinic, or in
the past is refused; the patient cancels at 24 h + 1 s and not at 24 h − 1 s; status changes only after the
start and only once; an admin cannot mark fulfilled; book and cancel run concurrently over a set of slots for
many rounds, and after each round a slot is `busy` exactly when a non-cancelled appointment holds it and the
exclusion constraint never fires on a free slot; revoking consent enqueues `consent_revoked` in the revoke's
transaction (a rollback leaves no job) and its arguments hold only the two ids; the job cancels exactly the
appointments in that clinic that start after the revocation, frees their slots and enqueues one e-mail each, in
one transaction, and a retry changes nothing; right after the revoke commits, before the job runs, the clinic
scope sees neither the profile nor the future appointment, and the practitioner still sees the past one with
name, times and status only; a booking 23 h ahead enqueues no reminder; job arguments hold only the two ids (stored JSON); a reminder
for a cancelled appointment sends nothing; e-mail bodies hold the clinic name, the local time and the link and
nothing else (golden text); the log holds no personal data.
Checks: Go, Clients, Stack (book through the API, the e-mail arrives in Mailpit).

### 7. FHIR

`handler_fhir.go` in `patient` (read), `consent` (read), `schedule` (read, search by `actor`), `slot`
(search by `schedule`, `start`, `status`), `appointment` (read; search by `patient`, `practitioner`, `date`,
`status`), `organization` (search by `listed`; `listed` and time-zone extensions on read and update),
`practitionerrole` (anonymous search of a listed clinic; `specialty`). `api/fhir/`: the two extension
`StructureDefinition`s and the `listed` `SearchParameter`; `CapabilityStatement.json` lists exactly the new
interactions and parameters. If the pinned validator's R4 package has a core time-zone extension, use it
instead of ours and record it. Golden files for every resource, Bundle and `OperationOutcome` (404, 400, 409,
412).
Tests: golden JSON; the access matrix through HTTP (patient, practitioner with and without access, admin,
other clinic, anonymous, and the clinic after a revocation: past appointments with the name only, no
`Patient` read, no future appointment); an unknown search parameter is 400; audit events per read and per returned
resource; `make fhir-validate` passes with `-ig api/fhir`.
Checks: Go, FHIR.

### 8. Web

`web/src`: practitioner week calendar (`Appointment?practitioner=…&date=ge…&date=lt…`, times in the clinic's
zone), availability editor, appointment detail (patient profile only when the `Patient` read answers; else the
name or nothing), mark fulfilled / no-show and cancel; admin: every practitioner's calendar in the clinic, the
specialty editor, the `listed` toggle and time zone in clinic settings. Visual direction from the accounts
SPEC §2 "Web": light, flat, mint accents, the gradient only on the current-appointment card, minimal inline
SVG icons, secondary details behind a tooltip or menu. FHIR types hand-written for the elements used.
Tests: Vitest for each screen's states, the CSRF header on every write, `If-Match` on edits, the profile
hidden on a 404, the status buttons disabled before the start.
Checks: Web, Clients; the screens walked in a browser against the stack at desktop width and 390 px.

### 9. Mobile

`mobile/lib`: profile completion (after registration and before the first booking), listed clinics,
practitioners of a clinic with specialty, free slots by day in the clinic's zone, the consent screen on the
first booking with a clinic (what is shared, with whom, for what, how to revoke), book, my appointments,
cancel (disabled inside 24 h), my consents with revoke (confirming that future appointments there are
cancelled). FHIR reads hand-written, JSON actions through the generated Dart client.
Tests: widget tests through `MockClient`: profile validation, `consent_required` leads to the consent screen
and then books, 409 on a taken slot, the 24-hour rule, revoke confirmation.
Checks: Mobile, Clients.

### 10. End-to-end

`test/e2e/booking_test.go` (build tag `e2e`), on a clinic created with `salusctl` and listed through the API,
with availability set and slots generated: a patient registers, completes the profile, finds the clinic,
gets `consent_required`, consents and books; the confirmation is in Mailpit; the practitioner finds the
appointment and reads the profile; the practitioner cancels; the patient sees `cancelled` and gets the e-mail.
Second journey: the patient books, revokes consent; the practitioner's `Patient` read is 404 at once, the future appointment
becomes `cancelled` once the job has run
(waited for as a condition), the e-mail arrives. Fresh synthetic data per run.
Checks: all of the above, `make e2e` twice in a row.

## Definition of done

Every package committed, `PROGRESS.md` complete, all check sets pass locally and in CI on the PR into `main`.
