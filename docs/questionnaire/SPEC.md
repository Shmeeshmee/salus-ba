# Questionnaires

Status: approved, 2026-11-06, with the owner's answers to D1 to D9 (§3) and the approved text (`COPY.md`).
Branch `feat/questionnaire` from `main`, one commit per package (`questionnaire(<n>): <summary>`), one PR at
the end.

## 1. Goal

A clinic keeps a library of questionnaires. Practitioners and admins build them in the web app with a
drag-and-drop builder: sections, instructions and questions of a small set of types, each question optionally
required, with help text and one condition that shows it only after a given earlier answer. A questionnaire is
published before it is used, and a published one never changes: editing it makes a new version. A practitioner
role can have a default questionnaire, requested from the patient with every booking with that role, and the
appointment's practitioner can add another to a single appointment. The patient fills it in on mobile before
the appointment starts; the answers are saved on the server as they go, and once the patient submits, the
practitioner reads them in the web app. Answers are health data: the access rule decides who sees them,
consent is checked before they are saved or read, and every read and write is audited, refusals included.
Questionnaires are FHIR R4 `Questionnaire` and `QuestionnaireResponse`, read over FHIR and written through the
JSON API, as in booking. File and video uploads are a goal of questionnaires and come in the next feature,
"questionnaire attachments" (D1); the item model reserves the `attachment` type now.

## 2. Settled decisions

Given by the owner. Do not reopen them.

| Topic | Decision |
| --- | --- |
| Who builds | Practitioners and admins, in the web app, with a drag-and-drop builder |
| Who fills in | Patients, on mobile only. No patient questionnaire screens on the web |
| Look | Simple but robust: it can do a lot and looks straightforward. Each screen shows only the information needed; the rest sits behind a tooltip or a dropdown |
| Uploads | Requesting file and video uploads is a goal of questionnaires. They come in a separate feature, "questionnaire attachments", right after this one (D1). The `attachment` item type is reserved in the model now |
| FHIR | R4 `Questionnaire` and `QuestionnaireResponse`. Reads over FHIR, writes through the JSON API, as in booking |
| Copy | Plain and professional (CLAUDE.md "Copy"). `COPY.md` holds the approved text (2026-11-06), word for word; a change goes back to the owner first |
| Drag and drop | `@dnd-kit` (pinned exact), plus "Move up" and "Move down" buttons (D7, §3 "Why `@dnd-kit`") |
| Answers after a revocation | The appointment's practitioner keeps reading the submitted answers of an appointment that started before the revocation, as part of the care record, as for `Encounter`; nobody else (D9) |

Decisions taken in this plan (reopen only with the owner). Those that rest on one of D1 to D9 name it; all
nine were answered on 2026-11-06 (§3).

- **Domains and context.** Two new domains, both in a new `clinical` context (CLAUDE.md lists it as planned;
  `config.yaml` gains `clinical: [questionnaire, questionnaireresponse]`, and the CLAUDE.md list
  `clinical` gains `questionnaire, questionnaireresponse`, in the commits that add each package):
  - `questionnaire`: the clinic's library (FHIR `Questionnaire`), its versions and lifecycle, and the default
    questionnaire of each practitioner role. Holds no patient data.
  - `questionnaireresponse`: the requests to a patient and their answers (FHIR `QuestionnaireResponse`),
    tied to an appointment.
  Dependencies, one way: `questionnaireresponse` → `questionnaire` (the version and its items, a role's
  default), `appointment` (`Attendance` and `Lookup`, for the access facts and the appointment's times),
  `access`, `auditevent`. `questionnaire` → `practitionerrole` (that a role is of the caller's clinic, and whose
  it is) and `auditevent`. `notification` → `questionnaireresponse` (whether one is outstanding).
  `appointment` learns no questionnaire type: its booking and cancellation transactions enqueue two more job
  kinds through its `Jobs` port, as they do `consultation_access_end`.
- **Tables.** Three:
  - `questionnaire` (owner `questionnaire`, `personal_data: false`): `id` (the version's FHIR id), `series_id`
    (the same for every version of one questionnaire; its canonical url), `organization_id`, `version`
    (1, 2, …), `status` (`draft` | `active` | `retired`), `title`, `description`, `items` (`jsonb`, the item
    tree), `published_at`, `retired_at`, `version_id`, timestamps. Unique (`series_id`, `version`); at most
    one `draft` and at most one `active` per series (partial unique indexes). No author column: who changed what
    is in the `AuditEvent`s. RLS: the clinic scope sees its own rows; the patient scope sees only the versions
    its own responses reference.
  - `questionnaire_default` (owner `questionnaire`, `personal_data: false`): `practitioner_role_id` (primary
    key), `organization_id`, `series_id`, `version_id`, timestamps. A setting of the role, not information about
    the practitioner. RLS: clinic scope.
  - `questionnaire_response` (owner `questionnaireresponse`, `personal_data: true`): `id`, `organization_id`,
    `appointment_id`, `patient_id`, `practitioner_role_id`, `questionnaire_id` (the exact version), `series_id`,
    `source` (`default` | `added`), `status` (`in-progress` | `completed` | `stopped`), `answers` (`jsonb`),
    `due_at` (the appointment's start, copied once), `authored_at` (set on submit), `version_id`, timestamps.
    Unique (`appointment_id`, `series_id`) where `status <> 'stopped'`: one live request per questionnaire per
    appointment. `patient_id`, `practitioner_role_id` and `due_at` are copied once from the appointment, as
    `encounter` does, because RLS and the retention purger need them and `salus_retention` reads only this
    table. RLS: the patient scope sees its own rows; the clinic scope sees the rows of its clinic whose patient
    has an active consent there, and the `completed` rows of appointments that started before the latest
    revocation (D9); the notice scope (`app.appointment_id` with `app.organization_id`, the notifier's) reads
    the rows of that one appointment; a `questionnaire_response_discard` `DELETE` policy lets the discard job
    delete the rows of the appointment named in `app.appointment_id`. Grants to `salus_app`; `SELECT, DELETE`
    to `salus_retention`.
- **Versions and lifecycle (D2).** A questionnaire belongs to its clinic; every practitioner and admin of the
  clinic may create, edit, publish and retire it. `draft` → `active` (published, never changed again) →
  `retired`. Only a draft is edited (`PUT`, `If-Match`); a draft that was never published may be deleted.
  "Edit" on an active questionnaire creates the next version as a draft, a copy with the same `linkId`s (409
  `draft_exists` if the series already has one). Publishing a version retires the active version of its series
  in the same transaction. A retired version is no longer requested and cannot be published again; requests
  already made keep the version they were made with, and the patient fills in that version.
- **Item model (D4).** The `item` tree of FHIR R4, these types only: `group` (a section), `display`
  (instructions), `string`, `text`, `boolean`, `choice` (one answer), `choice` with `repeats: true` (several
  answers), `integer`, `decimal`, `date`. Per item: `linkId`, `text`, `type`, `required` (questions only),
  `repeats` (choice only), `answerOption` (choice only: `valueString`, the clinic's own wording, so no code is
  invented), help text, and at most one `enableWhen`. Sections are top level only and hold questions and
  instructions, not other sections. A draft may keep a section without items; publishing refuses it (FHIR
  `que-1`: 422 `invalid_questionnaire`, the section's `linkId` in `linkIds`). Help text is the R4 core way: a child `display` item with the core
  extension `http://hl7.org/fhir/StructureDefinition/questionnaire-itemControl`, `http://hl7.org/fhir/questionnaire-item-control`
  `help`. `enableWhen`: `question` an earlier `boolean` or `choice` item (earlier in document order),
  `operator` `=`, `answerBoolean` or `answerString` (one of that item's options); for a choice with repeats,
  true when any of its answers matches. An item whose question is hidden is hidden too (R4: treated as not
  answered). A `group` may carry the condition, which hides the whole section. **Reserved:** `attachment` (D1) is
  declared in the model as an item type with its answer value (`valueAttachment`), and refused by the service
  (422 `invalid_questionnaire`) until the attachments feature enables it; the builder does not offer it, and the
  OpenAPI enum does not list it yet. Nothing in the model, the storage or the API assumes the list of types is
  closed.
- **Limits.** Checked by the service on every save; the builder shows them before they are reached. 200 items
  per questionnaire (sections and instructions included); title 200 characters, description 1,000, item text
  500, help text 500; 2 to 50 options per choice, each 1 to 200 characters, unique within the item; a `string`
  answer at most 255 characters, a `text` answer 4,000; `integer` in the 32-bit range; `decimal` at most 15
  significant digits; `date` a full date (`YYYY-MM-DD`). A request body is read up to 256 KiB.
- **linkIds.** Opaque and random: the builder makes one per new item (`crypto.randomUUID()`); the service
  checks the format (`[A-Za-z0-9-]`, 1–64 characters) and that each is unique within the questionnaire. A new
  version keeps the `linkId` of every item it carries over, and the service refuses (422
  `invalid_questionnaire`) a new version that gives an existing `linkId` another type (single and multiple choice count as different types);
  the builder gives an
  item a new `linkId` when its type changes. An `enableWhen` that points at a missing, later or non-boolean,
  non-choice item, or at an answer that is not one of its options, is refused the same way; the builder removes
  a condition when the question it depends on is deleted, and says so first.
- **Canonical url.** `https://salus.invalid/fhir/Questionnaire/<series_id>`, `version` the version number as
  a string, on the `salus.invalid` placeholder base like the existing definitions (needs the real domain
  before production, booking SPEC §7). `QuestionnaireResponse.questionnaire` is `<url>|<version>`.
- **Requests (D3).** A request is a `questionnaire_response` row, created `in-progress` with no answers. Two
  ways:
  - **The role's default.** The practitioner sets the default of their own role, an admin of any role of the
    clinic, to a questionnaire of the clinic that has an active version, or clears it (`PUT` and `DELETE
    /practitioner-roles/{id}/questionnaire`, `GET` to read it). It references the series, so a newly published
    version applies to the next booking. The booking transaction enqueues `questionnaire_default_request`
    (appointment id, clinic id); its worker calls `questionnaireresponse.Service.RequestDefault`, which reads the
    role's default and requests its active version (`source: default`), once (`ON CONFLICT DO NOTHING`); with
    no default, or none active, it does nothing. A default applies to bookings made after it is set.
  - **Added by the practitioner.** `POST /appointments/{id}/questionnaires` `{questionnaireId}`: the
    appointment's practitioner only (`access.RequestsQuestionnaire`), the appointment `booked` and not yet
    started, the questionnaire an active version of the clinic; 404 for anyone else, 409 `already_requested`,
    `not_active`, `appointment_started`, `appointment_closed` (no longer booked), `consent_required`.
    `source: added`. A request and the default request lock the appointment in their transaction against a
    cancellation: a cancellation that committed first refuses the request (the job then does nothing); one that
    comes later waits for it, and its discard job deletes the response.
  - **Withdrawn.** `POST /questionnaire-responses/{id}/withdraw`, the appointment's practitioner, while not
    submitted: `status` becomes `stopped` and the answers are cleared at once. The row stays as a record that
    the request was made and withdrawn, with no answers.
- **Patient flow (D5).** `PUT /questionnaire-responses/{id}/answers` `{answers}` (`If-Match`) replaces the
  saved answers of an `in-progress` response: the patient only, with an active consent, while the appointment
  is `booked`, before `due_at`. Each answer is checked against its item's type and limits; `required` is not
  checked yet. 409 `submitted`, `withdrawn`, `appointment_closed` (the appointment was cancelled; the discard
  job has not yet deleted the response), `past_due`, `consent_required`; 412 on a stale version; 422 `answers_invalid` with the
  offending `linkIds`. `POST /questionnaire-responses/{id}/submit` (`If-Match`) checks the saved answers in full:
  every enabled required question answered (422 `answers_invalid` with the `linkIds` otherwise), with the same
  409s as the save; it removes the
  answers of hidden items, sets `completed` and `authored_at`, and is final: no later save, no amendment. The
  mobile app saves when the patient leaves a field and 2 seconds after they stop typing, not on each keystroke.
  Due before the appointment starts; after `due_at`, an unsubmitted response is closed and its answers are
  never shown to the practitioner (staff see its status, "Not submitted before the appointment", W34).
- **What the practitioner sees.** That a questionnaire was requested and whether it was submitted, and when;
  the answers only once it is `completed`. Admins see the requests and their status for the clinic's
  appointments, never the answers.
- **Access rule.** In `access`, pure and table-tested:
  - `access.QuestionnaireResponse(viewer, facts, visit, submitted) Level`: the patient on their own → `Full`;
    with an active consent, the appointment's practitioner → `Full` when submitted, `Name` (it exists, its
    status) when not, and the clinic's admin → `Name`; without an active consent, for a past appointment (started
    before the latest revocation), the appointment's practitioner → `Full` when submitted (D9), and the admin →
    `Name`, as for the appointment itself; everything else → `None`. Nobody but the patient and the appointment's
    practitioner is ever `Full`: the table tests assert it for every combination.
  - `access.RequestsQuestionnaire(viewer, facts, visit) bool`: the appointment's practitioner, with an active
    consent; nobody else. The service checks that the appointment is `booked` and has not started.
  Table tests over role × consent (active, revoked, never) × appointment (none, past, future, other
  practitioner) × clinic × submitted.
- **Consent.** Saving or submitting answers needs the patient's active consent with the clinic: the answers
  are shared with it. A revocation refuses the next save at once. Reading answers as the practitioner needs an
  active consent, or D9's past-appointment rule. Requests are only made with an active consent.
- **Audit.** Every read and write of a `QuestionnaireResponse` writes an `AuditEvent` in the same transaction,
  refusals included: request, withdraw, each save, submit, read, each resource a search returns (or one event
  without an entity when it returns none), the default request and the discard by their jobs
  (`questionnaire_default_request`, `questionnaire_responses_discard` as agent, `HOPERAT`), and the notifier's
  check (its job as agent). Every authenticated read and write of a `Questionnaire` or a role's default is
  audited too, as directory records are: an admin or practitioner `HOPERAT`, a patient `PATRQT`. Purpose for a
  response: `PATRQT` for the patient, `TREAT` for staff. Codes as in booking SPEC §2 "Audit codes".
- **Cancellation and revocation (D8).** Every cancellation (the patient, staff, and the revocation job, all
  through `appointment`'s shared `cancelled` step) enqueues `questionnaire_responses_discard` (appointment id,
  clinic id) in the cancel's transaction; its worker deletes every response of that appointment, submitted or
  not, after checking through `appointment.Lookup` that it is cancelled. A revocation reaches it through the
  cancellations it causes; responses of appointments that started before the revocation stay, under D9.
- **Notifications (D6).** No e-mail of its own. The booking confirmation and the 24-hour reminder add one
  sentence (COPY E1) when the appointment has an outstanding questionnaire: a response `in-progress`, or a
  role default not yet requested (the confirmation can run before the default-request job; then the role's
  active default counts). Never the questionnaire's title, which can name a condition. The notifier asks
  `questionnaireresponse.Service.Outstanding(ctx, agent, org, appointmentID)` in its notice scope, audited.
- **FHIR.** `Questionnaire`: `url`, `version`, `title`, `status`, `subjectType` `Patient`, `date` (last
  change), `description`, `item`, `meta`. Read: staff of the clinic, and a patient for a version their own
  response references; 404 otherwise. Search by `url`, `version`, `status` (clinic scope for staff; the patient
  scope for patients). `QuestionnaireResponse`: `questionnaire`, `status`, `subject` and `author` `Patient/<id>`,
  `authored`, `item` (answers nested as the questionnaire's sections, only for `Full`; a `Name` view carries no
  `item`), `meta`, and the extension below. Read under the access rule; search by `appointment` (below),
  `patient`, `status`, with `_include=QuestionnaireResponse:questionnaire` so one request brings the versions
  the responses use: a patient only for themselves, a practitioner only for their own appointments, an admin for
  the clinic. No FHIR create or update: writes go through the JSON API. `CapabilityStatement.json` gains exactly
  these interactions and parameters.
- **The appointment extension.** R4's `QuestionnaireResponse` has no `Appointment` reference (`basedOn` is
  CarePlan or ServiceRequest, `encounter` exists only once the practitioner joins, `partOf` is Observation or
  Procedure). So, as `organization-listed`: `api/fhir/StructureDefinition-questionnaireresponse-appointment.json`
  (`https://salus.invalid/fhir/StructureDefinition/questionnaireresponse-appointment`, context
  `QuestionnaireResponse`, `valueReference` to `Appointment`, 1..1 in our responses) and
  `SearchParameter-questionnaireresponse-appointment.json` (`appointment`, type reference). `make
  fhir-validate` checks the golden files against them.
- **JSON API** (`api/openapi.yaml`, `make gen-clients`): `POST /questionnaires`; `PUT` and `DELETE
  /questionnaires/{id}`; `POST /questionnaires/{id}/publish`, `/retire`, `/revise`; `GET`, `PUT` and `DELETE
  /practitioner-roles/{id}/questionnaire`; `POST /appointments/{id}/questionnaires`; `PUT
  /questionnaire-responses/{id}/answers`; `POST /questionnaire-responses/{id}/submit`, `/withdraw`. Errors keep
  `{"error": "<code>"}`, plus `linkIds` on 422.
- **Storage of answers.** `answers` holds a flat list of `{linkId, values}`; the FHIR handler nests them under
  their sections from the questionnaire version. One row per response, one query to read it; a save replaces
  the list. The `items` of a version are read whole for the same reason. If the measurements (below) show a
  cost, it is recorded before anything changes.
- **Web builder (D7).** A page per questionnaire. On the left the item types; on the right the questionnaire.
  A type is dragged into place, or clicked to add at the end; items and sections are reordered by dragging,
  or with "Move up" and "Move down" for keyboard users. An item card shows its text, type and whether it is
  required; it opens to edit, and the rest (help text, the condition) sits behind "More options". Single and
  multiple choice are separate types in the palette. Drag and drop with `@dnd-kit/core` and `@dnd-kit/sortable`
  (pinned exact, versions in `PROGRESS.md`): pointer, touch and keyboard sensors, and its screen-reader
  instructions and announcements set to COPY W41 and W42 (no default English text of its own). `@dnd-kit` is imported only by the builder's
  drag-and-drop module. "Save draft" saves explicitly (`If-Match`); leaving with unsaved changes asks first. Library page: title and status, the version in a tooltip, filtered by
  status. Role default on the Working hours page. Appointment detail: the requests and their status, "Add
  questionnaire", "Withdraw", and the submitted answers.
- **Mobile.** My appointments shows a questionnaire to fill in on the appointment's card. The questionnaire
  screen is one scrolling page: sections as headings, help text behind an icon, the saved state, and "Submit"
  at the end with a confirmation. Native Flutter widgets (the date picker included), no new dependency. After
  submitting, the patient can read their answers.
- **Performance.** Measured, never assumed, numbers in `PROGRESS.md`: `BenchmarkPublish` and
  `BenchmarkReadVersion` (a 200-item questionnaire: validate and store, read and encode as FHIR);
  `BenchmarkSaveAnswers` and `BenchmarkSubmit` (200 answers: validate, evaluate conditions, store, audit) on the
  real database; `make bench-questionnaire-http`, a Go load test against the stack: 1,000 saves, 10 concurrent,
  and 1,000 `QuestionnaireResponse?appointment=…&_include=…` searches, p50, p95 and p99; in the browser, the
  builder with 200 items (first render, one drag) and on mobile the screen with 200 items (first frame), read
  from the dev tools and recorded by hand.

## 3. Answered decisions

Put to the owner in the draft and answered on 2026-11-06. §2 is written on these answers.

- **D1 Uploads.** A separate feature, "questionnaire attachments", right after this one. The `attachment` item
  type is reserved in the model now (§2 "Item model"); that feature's needs are in §7.
- **D2 Ownership and versions.** As recommended: a clinic-wide library; admins and practitioners both create and
  edit; `draft` → `active` (immutable) → `retired`; editing an active one creates a new version, `linkId`s stable.
- **D3 How a questionnaire reaches a patient.** As recommended: a default per practitioner role, requested with
  every booking with it, and the appointment's practitioner adds one to a single appointment. Admins set any
  role's default and do not add to single appointments.
- **D4 Question types.** As recommended: group, display, string, text, boolean, choice, choice with repeats,
  integer, decimal, date; required and help text; one `enableWhen` per item on an earlier boolean or choice
  answer, operator `=`; sections one level deep.
- **D5 Patient flow.** As recommended: saved on the server as they go, submitted once, read-only after; due
  before the start; the practitioner sees no answers until submitted.
- **D6 Notifications.** As recommended: no separate e-mail; the booking confirmation and the 24-hour reminder
  carry E1 when a questionnaire is outstanding, never its title.
- **D7 Drag and drop.** `@dnd-kit`, pinned, plus "Move up" and "Move down". See "Why `@dnd-kit`" below.
- **D8 Revocation, cancellation and retention.** As recommended: kept for past appointments; deleted, submitted
  or not, with a cancelled or revoked future appointment; unsubmitted ones deleted once due. The legal
  retention period is in §8.
- **D9 Answers after a revocation.** Yes: the appointment's practitioner only, as part of the care record, as
  for `Encounter`. `access`, the RLS policy of `questionnaire_response` and the table tests follow (§2 "Tables",
  "Access rule").

### Why `@dnd-kit`

CLAUDE.md allows a dependency only for what the standard library (here, the browser) cannot do in under ~50
lines. Native HTML5 drag and drop fails two requirements: it has no dependable touch support (touch input
does not start a drag in most browsers, and where it does the behaviour differs per browser), and it is not
accessible: it has no keyboard interaction and nothing for screen readers during a drag. Both pointer-and-touch and accessible use are required. Covering
them by hand means our own pointer-event drag with hit testing, auto-scroll, a keyboard drag mode and live-region
announcements, far beyond 50 lines and easy to get subtly wrong. `@dnd-kit/core` (sensors for pointer, touch and
keyboard, collision detection, announcements) and `@dnd-kit/sortable` (reordering lists, also between lists, as
our sections need) do exactly that, are small, maintained and MIT-licensed; their only runtime dependencies are
`tslib` and their own `@dnd-kit/accessibility` and `@dnd-kit/utilities` (checked and recorded in package 8). "Move up" and "Move down" stay as the plain path that needs no drag at all. The packages are
pinned exact in `web/package.json` (and `@dnd-kit/utilities` if `sortable` needs it as a direct import), the
versions recorded in `PROGRESS.md`, and only the builder's drag-and-drop module imports them (an oxlint
`no-restricted-imports` rule enforces it).

## 4. Lawful basis

- Answers are health data (Art. 9), and so is the fact that a patient was asked a given questionnaire (its
  title can name a condition). Care: Art. 9(2)(h), with the consent for sharing before care (booking SPEC §3):
  the patient answers to prepare an appointment, and the answers are shared with the clinic. Saving and
  submitting require the active consent. **Needs legal confirmation before production**, with booking's split
  and D9.
- Questionnaire definitions are the clinic's own documents, not personal data. The builder reminds staff not
  to put patient details in them (COPY W14).
- Purpose limitation: answers are used for the patient's care only; no reports, analytics or training on them
  without a separate basis and SPEC.
- Audit and retention: as in accounts (Art. 5(2), 32).

## 5. Personal data

| Where | Field | Why it is needed |
| --- | --- | --- |
| `questionnaire_response` | `appointment_id`, `patient_id`, `practitioner_role_id`, `organization_id` | Which appointment the questionnaire prepares, whose it is and who reads it; copied once for RLS and the purger |
| `questionnaire_response` | `questionnaire_id`, `series_id`, `source` | Which version was asked, so the answers can be read against its items, and whether it came with the role's default or was added |
| `questionnaire_response` | `status`, `authored_at` | Whether and when the patient submitted, so the practitioner sees only submitted answers |
| `questionnaire_response` | `answers` | The patient's answers: the purpose of the feature. Only the items the clinic asked; hidden items' answers removed on submit |
| `questionnaire_response` | `due_at` | The appointment's start, copied once: after it the response is closed, and an unsubmitted one is deleted by the purger, which may read only this table |
| `questionnaire_response` | `version_id`, timestamps | Optimistic locking of saves; `meta.lastUpdated` |
| `questionnaire`, `questionnaire_default` | none | Clinic documents and a role setting; no patient data, no author (who changed what is in the audit) |
| River jobs | `questionnaire_default_request`, `questionnaire_responses_discard`: appointment id, clinic id | To request the default and to discard after a cancellation; opaque ids only |
| E-mail | One sentence that a questionnaire is outstanding | So the patient knows to fill it in. No title, no question, no answer |

Logs carry the request id, route, status, duration, opaque ids and error codes only: never a questionnaire's
title, an item's text, an option or an answer, and never `linkId`s together with values. Golden files and
fixtures use synthetic questionnaires (placeholder medical wording, labelled as a placeholder) and synthetic
answers.

Retention (defaults, **need a legal decision before production**):

| Table | Rule |
| --- | --- |
| `questionnaire_response` | `delete` an unsubmitted (`in-progress` or `stopped`) response once `due_at` has passed (the daily job); a `completed` one is kept with the clinical record. Responses of cancelled appointments are deleted by the discard job |
| `questionnaire`, `questionnaire_default` | No personal data, no rule. Retired versions are kept while any response references them |

DPIA notes: a new section "Questionnaires (`docs/questionnaire`)" in `docs/privacy/dpia.md`, one subsection per
package that touches personal data (3, 4, 5, 6, 7, 10). Processors: none new; `docs/privacy/processors.md`
gains a line that questionnaires add no processor, and the SES entry gains the E1 sentence.

## 6. Out of scope

File and video uploads (D1), `open-choice`, scores and calculated items, units and `Quantity` answers,
conditions on more than one answer or other operators, nested sections, amending a submitted response,
questionnaires on the web for patients, patient-initiated questionnaires, questionnaires outside an
appointment, a separate questionnaire e-mail, push notifications, a preview in the builder, copying a
questionnaire between clinics, importing or exporting `Questionnaire` JSON, FHIR writes, SDC features beyond the
`help` item control, `Provenance` (answers are the patient's own; the first practitioner-authored clinical
write brings it), localisation (before production, as for the rest of the patient app).

## 7. Up next

- **Questionnaire attachments (D1), the next feature.** Enables the reserved `attachment` item type and
  `valueAttachment` answers: a patient attaches photos, documents and short videos. It needs:
  - an ADR for the object store, and the object store in compose (MinIO, pinned) for local development and the
    tests, tested against the real one like Redis and LiveKit;
  - envelope encryption: each file encrypted with its own data key, the data key encrypted with a key-encryption
    key that comes from configuration locally, like `TOTP_KEY` (`internal/config` only), and from the KMS in
    production (a before-production item);
  - malware scanning before a file is readable by staff (a scanner in compose, e.g. ClamAV, pinned), with a
    quarantined state;
  - size and type limits per item and per response, checked on the bytes, not the extension or the declared
    type;
  - video: the formats and lengths accepted, upload size, whether anything is transcoded or a thumbnail made, and
    playback in the web app; resumable or chunked upload from mobile;
  - upload and download through the api (streamed, or short-lived signed URLs), every access audited;
  - retention: files deleted with their response (cancellation, unsubmitted past due) and kept with a
    completed one; a purger for the object store;
  - mobile camera, photo library and file pickers, with permission text in a copy review;
  - `processors.md` (none while self-hosted; the hosting provider's object store once chosen) and DPIA notes.
- **A preview in the builder.** The questionnaire as the patient will see it, conditions working.
- **Amending a submitted response**, if clinics ask for it: R4 `amended`, with the history kept.
- **Export and erasure.** `Patient/$everything` (subject-rights feature) includes the patient's responses and
  the versions they reference.

## 8. Before production

- Legal decision on the retention of `questionnaire_response` (D8), on D9, and legal confirmation of the
  lawful basis (§4).
- The new web, mobile and e-mail text localised (bs/hr/sr) with the rest of the patient app; builder text with
  the web app.
- The canonical base `https://salus.invalid/fhir` replaced by the real domain (booking SPEC §7); the
  questionnaires' canonical urls follow it.
- The mobile questionnaire screen checked on a real Android device and an iOS build, with a screen reader.
- Decimals typed with a comma (bs/hr/sr) accepted by the mobile app, with the localisation.
- The mobile date picker's locale (owner, on review of PR #24): it still shows Flutter's US-English header date ("Wed, Nov 4"), month tooltips and screen-reader labels; `flutter_localizations` (Flutter SDK) with the bs/hr/sr localisation sets them.

## 9. Done when

All packages are committed on `feat/questionnaire`; `make lint`, `make test`, `make archcheck`, `make cover`
(every package at 80% or more), `make fhir-validate` and `make e2e` pass locally and in CI; `make gen-clients`
leaves no diff; the web and mobile checks pass; `docs/privacy/` describes what exists; `PROGRESS.md` records
every package with its failing tests and the measurements of §2 "Performance".
