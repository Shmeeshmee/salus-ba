# Patients, consent and booking

Status: approved, 2026-11-04, with the owner's changes (section 2, "Changed on approval"). Branch `feat/booking` from `main`, one commit per
package (`booking(<n>): <summary>`), one PR at the end.

## 1. Goal

A patient completes a short profile on mobile, browses the clinics that chose to be listed and their
practitioners, picks a free slot, consents once per clinic to share their profile and appointments with it,
and books. The appointment is booked at once. The patient, the practitioner and the clinic's admin can cancel
within their rules; the practitioner marks the visit fulfilled or no-show. E-mails confirm, cancel and remind.
A practitioner sees a patient's profile only with that patient's active consent and an appointment between
them; revoking consent takes the clinic's access to the profile and to anything future away immediately and
cancels the patient's future appointments there, while the clinic keeps minimal read access to past
appointments as its care records. Every access to a patient, a consent or an appointment is audited, and the retention
rules from the accounts feature are now applied by a scheduled job.

## 2. Settled decisions

Given by the owner. Do not reopen them.

| Topic | Decision |
| --- | --- |
| Patient | One `Patient` per patient account, not clinic-scoped. Fields: given name, family name, birth date, optional phone. Nothing else (no national ID, address or gender). Completed after registration, on mobile; booking requires a complete profile |
| Consent | FHIR `Consent`, one per patient per clinic: "share my profile and appointments with this clinic for care". Granted explicitly on the first booking with that clinic; revocable at any time from mobile |
| Revocation | Immediate: the clinic loses access to the profile, and the patient's future appointments with that clinic are cancelled, with an e-mail. Past appointments stay, as the clinic's records, under the retention rules |
| Access | A pure `access` package in the privacy context, table-tested, no I/O. A practitioner reads a patient's profile only with an active Consent for the practitioner's clinic and an appointment (now or in the past) with that practitioner. A clinic admin sees only the patient's name and the appointment times. Break-glass is out of scope |
| Audit | Every read or write of `Patient`, `Consent` or `Appointment` writes an `AuditEvent` in the same transaction; denied accesses too |
| Directory | `PractitionerRole.specialty` as a SNOMED CT coding, set by admins. `Organization` gains a `listed` flag and an IANA time zone (default `Europe/Sarajevo`). Anyone, without signing in, can browse listed clinics and their practitioners with specialty; nothing personal beyond the practitioner's name and specialty |
| Schedule | One `Schedule` per `PractitionerRole`: weekly availability rules (day, start and end in the clinic's time zone) and a slot length |
| Slots | The `scheduler` materialises `Slot` rows 4 weeks ahead in an idempotent job. Generation for 50 practitioners is benchmarked and recorded |
| Double booking | Prevented by the database: an exclusion constraint over (practitioner role, `tstzrange`) with `btree_gist` (installed by `00001`). A clash is 409. No Redis slot hold |
| Time | Stored in UTC, shown in the clinic's time zone |
| Booking | From mobile, a free slot, `booked` at once, no approval step |
| Cancellation | The patient up to 24 hours before the start; practitioners and admins at any time |
| Status | FHIR `booked`, `cancelled`, `fulfilled`, `noshow`. The practitioner marks `fulfilled` and `noshow` from the web app |
| E-mail | Confirmation, cancellation and a reminder 24 hours before: River jobs enqueued in the same transaction, sent by the notifier through the existing `Sender` port. Clinic name, date and time, a link. Never the specialty, a reason for the visit, or anything clinical. Job arguments are opaque ids |
| Retention | Package 1 is the retention job (accounts SPEC §6). Every new table with personal data gets a rule; `appointment` and `consent` are "kept with the clinical record", **needs a legal decision before production** |
| Domains | `patient` (context `patient`); `consent` and `access` (context `privacy`); `schedule`, `slot`, `appointment` (context `scheduling`). `config.yaml`, `ownership.yaml` and `sqlc.yaml` change in the commit that adds each one |

Decisions taken in this plan (reopen only with the owner):

- **Retention privileges.** The scheduler and worker connect as `salus_app`, which cannot delete sessions or
  invites (00011, 00013), and `invite` is row-level secured, so a job without a clinic sees no rows. A new
  role `salus_retention` (`LOGIN`, `BYPASSRLS`, not a superuser, no `CREATEROLE`/`CREATEDB`, owner of
  nothing) is granted `SELECT, DELETE` on exactly the tables whose rule is `delete`, and nothing else. It is
  created like `salus_app` (a `DO` block that tolerates a concurrent create and refuses an existing role with
  other attributes), its password is set by `cmd/migrate up` from `RETENTION_DATABASE_PASSWORD`. It has its
  own credentials, `RETENTION_DATABASE_URL`, loaded only for the retention job: `internal/config` reads it in
  a loader of its own that only `cmd/worker` calls, and only the `retention_apply` worker gets the pool built
  from it. Tests prove the api, notifier and scheduler configurations never read it, that only the `worker`
  service has it in compose, and that every other job in `internal/worker` is built on the `salus_app` pool. A
  test proves the role cannot `INSERT`, `UPDATE`, or touch `audit_event`. A later migration that adds a
  `delete` rule grants it in the same file.
- **Applying the rules.** `retention.Service.Apply(ctx)` walks `Rules()`. `keep` does nothing (closure and
  erasure are the subject-rights feature). `restrict` does nothing yet: no practitioner can leave a clinic
  (deactivation is out of scope), so no row can be in the restricted state; the applier arrives with
  deactivation. `delete` calls the owning domain's purger: `retention` declares the port
  `Purger{Purge(ctx, before time.Time) (int64, error)}` and each owning domain implements it in a
  `service_retention.go` whose constructor takes the domain's repository (so the SQL stays in its owner), built
  on the `salus_retention` pool in `internal/worker`. `remove-range` (only `audit_event`) is not removable by
  any role today (the triggers fire for the owner too) and nothing is due before 2036-11; the job checks the
  oldest `recorded` through `auditevent.Service` and fails, with the count only, if any row is overdue. The
  mechanism that removes old audit ranges (partitioning `audit_event` by year, dropped by `salusctl` as the
  owner) is recorded under "Up next". The job refuses to start when a `delete` or `remove-range` rule has no
  applier (a test over `Rules()`). It logs the table and row count per rule, nothing else.
- **Periodic jobs.** `cmd/scheduler` runs a River client that only enqueues River periodic jobs (leader
  elected, so several replicas enqueue once): `retention_apply` daily at 03:00 UTC and `slots_generate` every
  hour. `cmd/worker` works them on the default queue. Both are idempotent.
- **Row-level security scopes.** Every transaction is in one scope, set with `postgres.SetLocal`:
  - **clinic** (`app.organization_id`, as in accounts): staff, and the notifier jobs (the job carries the
    clinic id, like `invite_email`);
  - **patient** (`app.patient_id`): a patient's own transactions. A patient's booking or cancellation also
    sets the appointment's clinic, after the service has resolved and checked the slot or appointment; the
    `patient` table's consent branch (below) applies only when `app.patient_id` is unset, so a patient's
    transaction never sees another patient;
  - **directory** (`app.directory = 'public'`): anonymous and patient reads of listed clinics. `SELECT`-only
    policies show `organization` rows with `listed`, and the active `practitioner`, `practitioner_role`,
    `schedule` and `slot` rows of listed clinics. Unset settings match nothing (`NULLIF(…, '')`, accounts
    package 1), so a forgotten scope fails closed.
- **Patient table.** `patient` is row-level secured: a row is visible to its own patient scope
  (`id = app.patient_id`, or `account_id = app.account_id` while the profile is looked up by account), or to a
  clinic scope when an active `consent` for that clinic exists or the clinic has a past appointment with the
  patient (below). RLS decides the row; the service decides the fields through `access` (the full profile
  only with active consent and an appointment with that practitioner; otherwise the name). `patient.account_id`
  links the profile to its account (unique, references `account`); auth knows nothing about patients.
- **Profile.** Created by the first `PUT /me/patient` with all required fields; there is no incomplete
  `Patient` row, so "complete" means "exists". Given and family name 1–100 characters after trimming; birth
  date in the past and not before 1900-01-01; phone optional, E.164 (`+` and 8–15 digits). `GET /me/patient`
  returns it or 404. FHIR `Patient.name` (one `HumanName`, `use: official`), `birthDate`, `telecom` (`system:
  phone`, `use: mobile`).
- **Consent shape.** Table `consent`: `patient_id`, `organization_id`, `status` (`active` | `inactive`),
  `date_time` (granted), `revoked_at`, `version_id`, timestamps. At most one `active` per patient and clinic
  (partial unique index). Granting again after a revocation creates a new row, so the history is kept. FHIR R4:
  `status`; `scope` `patient-privacy` (`http://terminology.hl7.org/CodeSystem/consentscope`); `category` LOINC
  `59284-0`; `patient`; `dateTime`; `performer` the patient; `organization` the clinic; `policyRule`
  `http://terminology.hl7.org/CodeSystem/v3-ActCode` `OPTIN`; `provision` `type: permit`, `purpose`
  `v3-ActReason` `TREAT`, `actor` the clinic with role `v3-ParticipationType` `IRCP`. JSON: `POST /consents`
  `{organizationId}` (patient; 409 if one is active), `POST /consents/{id}/revoke`, `GET /me/consents` (the
  mobile list). Booking without an active consent answers 409 `consent_required`; mobile then shows the
  consent screen, grants, and books.
- **Revocation (changed on approval).** `consent.Service.Revoke` sets `inactive` and `revoked_at`, audits,
  and enqueues `consent_revoked` (arguments: the patient id and the clinic id) with `InsertTx`, all in one
  transaction. Access is gone when that transaction commits: RLS and `access` read the consent's state, not
  the appointments'. The `consent_revoked` worker (`cmd/worker`, default queue) calls
  `appointment.Service.CancelForRevokedConsent`, which, in the patient's scope with the clinic set, cancels
  the patient's appointments in that clinic that start after the revocation (reason `pat`, inside the 24-hour
  window too), frees their slots and enqueues one cancellation e-mail each, in one transaction; a retry finds
  nothing left to cancel. The dependency is one-way, appointment → consent: `consent` knows nothing about
  appointments.
- **Past and future after revocation (changed on approval).** "Past" means the appointment started before
  the latest `revoked_at` of the patient's consents with that clinic. Without an active consent, the clinic
  scope sees only the patient's past appointments with that clinic (RLS: `start < max(revoked_at)`), and
  through them the patient's name, the times and the status; never the profile, never a future or cancelled-by-
  revocation appointment. Granting consent again restores access to everything.
- **Access rule.** Pure functions in `access`, with `Level` one of `None`, `Name`, `Full`.
  `Viewer`: role, clinic, practitioner role id, own patient id. `Facts`: the patient id, whether an active
  consent exists for the viewer's clinic, the latest revocation time, and the patient's appointments in that
  clinic (practitioner role id, start; any status).
  - `access.Profile(viewer, facts) Level`: patient on themselves → `Full`; practitioner with active consent
    and an appointment with them → `Full`; admin with active consent → `Name`; without an active consent, a
    practitioner with a past appointment with them, or an admin of a clinic with a past appointment → `Name`;
    everything else → `None`.
  - `access.Appointment(viewer, facts, appt) Level`: the patient on their own → `Full`; with active consent,
    the appointment's practitioner and the clinic's admin → `Full` for the appointment, the name per
    `Profile`; without an active consent, the same viewers → `Name` (name, times, status) for a past
    appointment and `None` for anything else; other practitioners and other clinics → `None`.
  Table tests cover every combination of role × consent (active, revoked, never) × appointment (none, past,
  future, other practitioner) × clinic match. `access` imports nothing from `internal/`.
- **Who may do what with an appointment.** Patient: book, cancel their own until 24 h before (later: 409
  `cancellation_window_closed`), read their own. Practitioner: read and cancel their own appointments, mark
  them `fulfilled` or `noshow` once the start has passed (earlier: 409). Admin: read and cancel every
  appointment of the clinic, not change other statuses. A cancelled, fulfilled or no-show appointment does not
  change again (409). Cancelling frees the slot when the start is still ahead. Cancellation reason
  (`http://terminology.hl7.org/CodeSystem/appointment-cancellation-reason`): `pat` by the patient, `prov` by
  staff; no free text.
- **Booking rules.** The slot is free, starts in the future, and belongs to a listed clinic; the profile exists;
  the consent is active. In one transaction: `UPDATE slot SET status = 'busy' … WHERE status = 'free'` (no row:
  409), insert the appointment (an exclusion violation, `23P01`: 409), enqueue the confirmation and, when the
  start is more than 24 h ahead, the reminder (`ScheduledAt` = start − 24 h), audit. Every change of a slot's
  status happens in the same transaction as the appointment change that causes it (book, cancel, the
  revocation job), so slot status and the appointments never disagree; a test books and cancels concurrently
  and checks after every round that a slot is `busy` exactly when a non-cancelled appointment holds it.
- **Double booking, twice guarded.** `appointment` has
  `EXCLUDE USING gist (practitioner_role_id WITH =, period WITH &&) WHERE (status <> 'cancelled')`, the
  owner's settled guarantee. `slot` has `EXCLUDE USING gist (schedule_id WITH =, period WITH &&)`, so
  generated slots never overlap and generation can insert with `ON CONFLICT DO NOTHING`.
- **Schedule shape.** Table `schedule`: `organization_id`, `practitioner_role_id` (unique), `slot_minutes`
  (5 to 240, a multiple of 5), `availability` as `jsonb`, an array of `{day, start, end}` (ISO weekday 1–7,
  `HH:MM` local, start before end, no overlap within a day, validated by the service), `version_id`. Created
  empty with the role's first availability edit. JSON: `GET` and `PUT /practitioner-roles/{id}/availability`
  (the practitioner for their own role, an admin for any role of the clinic; `If-Match` on `version_id`).
  Saving enqueues `slots_generate` for that schedule in the same transaction, so the change shows within
  seconds rather than at the next hourly run.
- **Slot generation.** For each active schedule, from now to 28 days ahead in the clinic's zone: insert the
  slots the rules give (`ON CONFLICT DO NOTHING`), delete future `free` slots the rules no longer give. `busy`
  slots and their appointments are never touched; an appointment outside new availability stays booked. A
  local time that does not exist (the spring DST change) is skipped; an ambiguous one (autumn) takes the first
  occurrence. Slot `status`: `free` | `busy` (FHIR `SlotStatus`).
- **Time zones.** `organization.time_zone` is validated with `time.LoadLocation`. The binaries import
  `time/tzdata`, so zone data never depends on the runtime image.
- **Specialty.** `practitioner_role.specialty_code`, nullable, one of twelve from the R4
  `c80-practice-codes` value set (SNOMED CT), held in `practitionerrole` with its display (owner's list on
  approval): general practice, internal medicine, cardiology, dermatology, gynaecology, paediatrics,
  psychiatry, neurology, ENT, ophthalmology, orthopaedics, endocrinology. Package 4 takes each code and
  display from the value set in the R4 core package, not from memory, and records them in `PROGRESS.md`. JSON
  `PUT /practitioner-roles/{id}/specialty` `{code}` (admin of the clinic, `If-Match`). FHIR
  `PractitionerRole.specialty` with `system` `http://snomed.info/sct`.
- **Public browse.** FHIR, unauthenticated: `Organization?listed=true`, the existing
  `PractitionerRole?organization=…&_include=PractitionerRole:practitioner` opened to anonymous callers for a
  listed clinic (active roles only; `Practitioner` with name only; `PractitionerRole` with specialty), and
  `Schedule?actor=PractitionerRole/…` so the app can find the schedule to search slots in (search by `actor`
  is not in the owner's FHIR list; it is the standard R4 parameter and the smallest way to get there). Public
  reads are **not audited**: they hold only what a clinic chose to publish, and auditing anonymous traffic would
  turn every request into a write. Every authenticated read stays audited as in accounts.
- **Listed in FHIR.** `listed` is an extension (`api/fhir/StructureDefinition-organization-listed.json`,
  `valueBoolean`) with a documented `SearchParameter` `listed` (token) in `api/fhir/`, not `Organization.active`:
  `active` already means "this record is in use" (accounts), and a clinic in use is often not yet listed;
  reusing it would make an unlisted clinic look closed to every FHIR client. The time zone is an extension too
  (`organization-timezone`, `valueCode`). Both are set through the existing `Organization` update with
  `If-Match` (admin).
- **FHIR reads and searches.** `Patient` read: the patient themselves and a practitioner with `Full` access;
  everyone else 404. `Consent` read: the patient, and staff of its clinic. `Schedule` read, search by `actor`.
  `Slot` search by `schedule`, `start` (`ge`, `lt`), `status` (anonymous and patients: `status=free` required,
  future slots only). `Appointment` read; search by `patient`, `practitioner`, `date` (`ge`, `lt`), `status`:
  a patient only for themselves, a practitioner only for themselves, an admin for the clinic.
  `Appointment.participant`: the patient, the practitioner and the practitioner role, each `status: accepted`.
  The patient participant carries `display` (the name) only at `Name` or `Full` access, and that read writes a
  `Patient` audit event too; an admin gets the name this way and 404 on `Patient` read.
- **Audit codes.** As in accounts (`rest` + `restful-interaction`). `purposeOfEvent`: `TREAT` for staff,
  `PATRQT` for a patient's own data, `HOPERAT` for jobs. A search audits each resource returned, or one event
  without an entity when it returns none. A notifier job's reads are audited with the job as agent (like
  `salusctl`).
- **Jobs.** Default queue (`cmd/worker`): `retention_apply`, `slots_generate`, `consent_revoked`. **E-mails.**
  Queue `email`, three jobs: `appointment_booked_email`, `appointment_cancelled_email`,
  `appointment_reminder_email`; arguments the appointment id and the clinic id. The worker loads the
  appointment and does nothing when it no longer applies (a reminder for a cancelled appointment). Recipient:
  the patient's account e-mail, through `auth`. Body: the clinic name, the date and time in the clinic's zone
  with the zone name, and the link `<PATIENT_APP_URL>/appointments`. Plain text, never logged.
- **Who gets the cancellation e-mail.** The patient, for every cancellation (including their own, as a
  receipt). Staff see cancellations in the calendar; no staff e-mail.
- **JSON API** (`api/openapi.yaml`, generated clients): `GET`/`PUT /me/patient`; `POST /consents`,
  `POST /consents/{id}/revoke`, `GET /me/consents`; `POST /appointments` `{slotId}`,
  `POST /appointments/{id}/cancel`, `POST /appointments/{id}/status` `{status}`;
  `GET`/`PUT /practitioner-roles/{id}/availability`; `PUT /practitioner-roles/{id}/specialty`. Errors keep
  `{"error": "<code>"}`. Reads of FHIR data go through FHIR, writes through these actions.
- **Web and mobile read FHIR.** Both keep a small hand-written FHIR layer for the elements they use, as the
  web app does today; the JSON actions go through the generated clients.

Changed on approval (2026-11-04, owner):

- Revocation enqueues `consent_revoked` instead of cancelling through a function value bound in `app.go`; the
  dependency is appointment → consent only. Access ends in the revocation's transaction.
- Lawful basis: care records under Art. 9(2)(h), consent for pre-care sharing and new bookings (§3). After a
  revocation the clinic keeps read access to the minimal data of past appointments (name, times, status) and
  loses the profile and anything future; RLS, `access` and its table tests follow.
- `salus_retention` has its own credentials, loaded only for the retention job, with tests that nothing else
  can use them.
- Slot status changes in the appointment's transaction, proven by a concurrent book-and-cancel test.
- The specialty list is the owner's twelve.

## 3. Lawful basis

- Patient profile and appointments reveal that a person seeks care (and with whom, which hints at the
  condition), so they are treated as health data (Art. 9). **Needs legal confirmation before production.**
  - **Care records: Art. 9(2)(h)** (provision of health care), with Art. 6(1)(b) and the clinic's
    record-keeping obligations. An appointment that has taken place, or was due, is part of the clinic's
    record and is kept and readable by the clinic in minimal form (name, times, status) after a revocation.
  - **Consent: Art. 9(2)(a)**, for sharing before care (the profile with the clinic) and for new bookings.
    Revoking it ends the clinic's access to the profile and to every future appointment, and cancels those
    appointments.
- Practitioner schedules, specialty and the public listing: the clinic's employment relationship and its
  legitimate interest in being found (Art. 6(1)(b), (f)); the clinic decides when it is listed.
- E-mails about an appointment: performance of the contract with the patient (Art. 6(1)(b)).
- Audit and retention: as in accounts (Art. 5(2), 32; Art. 6(1)(c), (f)).

## 4. Personal data

| Where | Field | Why it is needed |
| --- | --- | --- |
| `patient` | `given_name`, `family_name` | So the clinic knows who is coming, and the patient is addressed correctly |
| `patient` | `birth_date` | To tell apart patients with the same name, and because age matters to the practitioner preparing the visit |
| `patient` | `phone` (optional) | So the practitioner can reach the patient about a booked appointment when e-mail is too slow; never used for SMS or marketing |
| `patient` | `account_id`, `version_id`, timestamps | To find the profile of the signed-in account; optimistic locking |
| `consent` | `patient_id`, `organization_id`, `status`, `date_time`, `revoked_at` | The patient's decision, when it was made and withdrawn, as proof (Art. 7(1)) |
| `appointment` | `patient_id`, `practitioner_role_id`, `organization_id`, `slot_id`, `period`, `status` | The booking itself: who sees whom, where, when |
| `appointment` | `cancelled_at`, `cancellation_reason` (`pat` / `prov`), `cancelled_by_account_id` | Who cancelled and when, for the 24-hour rule and disputes. No free-text reason |
| `schedule` | `practitioner_role_id`, `slot_minutes`, `availability` | A practitioner's working hours, to offer slots |
| `slot` | `schedule_id`, `organization_id`, `period`, `status` | Bookable times; reveals a practitioner's availability |
| `practitioner_role` | `specialty_code` | So patients can choose the right practitioner; published only when the clinic is listed |
| `organization` | `listed`, `time_zone` | Not personal data |
| River jobs | appointment id, clinic id; for `consent_revoked` the patient id and clinic id | To send the e-mail, to cancel after a revocation; opaque ids only |
| E-mail | Recipient address, clinic name, date and time, link | To confirm, cancel and remind. No practitioner, specialty or reason |

Logs carry request id, route, status, duration, opaque ids and error codes only. Test data is synthetic
(`@example.test`, invented names, birth dates and `+387 60 000 0xxx`-style phone numbers marked as fake).

Retention rules. Defaults: **needs a legal decision before production.**

| Table | Rule |
| --- | --- |
| `patient` | Kept while the account exists (`keep`); erasure comes with the subject-rights feature |
| `consent` | Kept with the clinical record (`keep`), as proof of the decision |
| `appointment` | Kept with the clinical record (`keep`) |
| `schedule` | Kept while the role is active; restricted after the practitioner leaves (`restrict`), like `practitioner_role` |
| `slot` | `delete` free slots 30 days after their end; a busy slot is kept with its appointment |

## 5. Out of scope

Video and consultations, questionnaires, payments and billing, waitlists, recurring appointments, in-person
versus video types, rescheduling (cancel and book again), SMS and push, break-glass, practitioners on mobile,
AI features, staff booking for a patient, the patient's export and erasure (subject-rights feature), removing
audit ranges (see "Up next"), practitioner deactivation (and so the `restrict` applier).

## 6. Up next

- **Audit range removal.** Partition `audit_event` by year and let `salusctl` drop partitions past the
  10-year rule as the owner. Needed before 2036-11; the retention job alerts if a row is overdue earlier.
- **Subject rights.** `Patient/$everything` export, erasure under the retention rules.

## 7. Done when

All ten packages are committed on `feat/booking`; `make lint`, `make test`, `make archcheck`, `make cover`
(every package at 80% or more), `make fhir-validate` and `make e2e` pass locally and in CI; `make gen-clients`
leaves no diff; the web and mobile checks pass; `docs/privacy/` describes what exists; `PROGRESS.md` records
every package with its readings, the slot-generation benchmark included.
