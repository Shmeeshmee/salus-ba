# Patient history: SAL-39, SAL-40, SAL-41, SAL-42

Epic SAL-38. Status: planned, 2026-11-12. The access model and the owner's decisions are ADR 0013
(docs/adr/0013-patient-history-access.md); this plan holds the contract, the packages and how they run.
PROGRESS: docs/plans/patient-history.PROGRESS.md.

## Tickets

| Ticket | Branch | What |
| --- | --- | --- |
| SAL-39 | `sal-39-timeline` | The patient's timeline in `mobile/` across every clinic: visits, signed documents, completed questionnaire answers and their files, newest first; filters by clinic, type and date; open any item; cursor pagination. Never a draft. Chats left out with an extension point (`history.TypeCommunication`) |
| SAL-40 | `sal-40-doctor-history` | The practitioner's view of the patient's history in `web/`, opened from a booking and from the call page: their own visits and the visits shared with the clinic, with the pre-visit summary on top (allergies, current medications, last visit, open items). Every view audited |
| SAL-41 | `sal-41-allergies` | Allergies and current medications in the patient's profile (`mobile/`), terminology-backed search, the practitioner records and confirms entries during a visit (`web/`), an allergy warning on the call page and in drafting |
| SAL-42 | `sal-42-sharing` | Cross-clinic sharing: the toggle at booking, off by default, with the per-visit checkbox preview (select all, select none, exactly what the doctor will see); manage per clinic in settings, add or remove visits, revoke; immediate effect |

## Decisions

Recorded in ADR 0013 "Decisions" (owner, 2026-11-12): the minimal profile; sharing off by default and per visit;
own visits only; doctor entries visit-scoped. Not to be reopened by a package; a package that finds them in conflict
with the code stops and reports.

Settled by this plan, within those decisions:

| Topic | Decision |
| --- | --- |
| A practitioner's shared visits | Seen while the care and history consents with their clinic are active, the history consent lists the visit, and the practitioner has an appointment with the patient at the clinic (`access.Profile`'s rule), as ADR 0013 says |
| What a shared visit brings | Its encounter, its signed documents (Composition, DocumentReference, PDF), its completed questionnaire responses and their clean files, the entries recorded in it. Never the transcript, suggestions, drafts or the appointment |
| Only finished visits | A visit is shareable once its encounter is finished; row-level security refuses another |
| An empty selection | An active history consent lists at least one visit: saving none revokes it (`PUT /history-sharing/{organizationId}` with `[]`) |
| Proof timestamps | `date_time`, `revoked_at`, `added_at` and `removed_at` are set by the service from its injected clock, as for the care and transcription consents; the database does not pin them to `now()` (a review asked; declined for consistency with CLAUDE.md's clock rule) |
| Changing a selection | Optimistic locking: `PUT /history-sharing/{organizationId}` with If-Match of the sharing's version (none for a first grant); a removal and a revocation are final, adding again adds a row |
| Revoking the care consent | Also revokes the history consent with that clinic, in the same transaction (as the transcription consent) |
| The visit's title in the preview | The title of its first signed document, else none (the app shows the specialty); the same fields the receiving clinic sees, so the preview is exactly what the doctor will see |
| Removing an entry | The patient's own unconfirmed entry is deleted; anything confirmed or recorded by a clinic becomes entered-in-error, kept with the record and shown to no one |
| A patient's change to a confirmed entry | Makes it unconfirmed again (a change of status too): the service clears the confirmation in the same update, and the database refuses an edit that keeps another's confirmation |
| Who a confirmation vouches for | The confirming clinic: a colleague at that clinic may correct the entry and keep it confirmed; the recording clinic corrects its own entry by clearing another clinic's confirmation. A removed entry stays removed |
| Coding | A substance or medication picked from the terminology carries its ATC code (`http://www.whocc.no/atc`) and its label as text; one typed by hand is text only (no free-text code). Units and periods are UCUM |
| Diagnoses in the doctor's view (SAL-40) | Shown from the signed documents' coded fields. The analytics epic is adding a `condition` domain (docs/plans/analytics.md, SAL-19): once it is on main, SAL-40 reads diagnoses through `condition.Service` instead, without touching its code |
| AI | No part of the history is sent to the language model (ADR 0013 "No AI"). The suggestions and the drafting keep reading only the consultation's transcript; SAL-42 adds a test that a shared or unshared visit's content never reaches their input |

## Contract (step 0, this PR, migration 00042)

- **Migration `00042_patient_history.sql`**: `allergy_intolerance`, `medication_statement`, `history_consent`,
  `history_consent_encounter`, the function `history_shared(encounter, patient)`, the `*_shared` policies, the
  profile tables' policies, the `profile_entry_update_check` trigger, 00031's transcript and suggestion policies
  narrowed to the clinic's own encounters, and the grants. Its policies call `history_shared` for every encounter row
  a clinic's scope reads: SAL-39 and SAL-40 measure their reads (PROGRESS) and say whether it needs more than the
  partial index on `history_consent_encounter (encounter_id)`. Tests: `migrations/patient_history_test.go` (every scope, table,
  removal, revocation, insert check and constraint). Ownership: `allergyintolerance`, `medicationstatement`,
  `consent`; retention `keep` for all four.
- **`internal/access`**: `Consent.History`, `Consent.Shared`; `Record`; `HistoryVisit`, `ProfileEntry`, `Attributed`
  (`history.go`, tested over viewer, consent and visit).
- **`internal/consent`**: `KindHistory` (not yet accepted by `Kind.Valid` or the `/consents` endpoints: SAL-42 adds it
  with its service, and adds `history` to the OpenAPI `ConsentKind`), `Consent.Shared` and `SharedVisit`, the FHIR
  mapping of a history consent (golden `testdata/history_consent.json`), `Candidate` (the preview's row), and
  `State`, which now gives `History` and `Shared` (the open selections) to every caller of `consent.Service.State`.
- **`internal/allergyintolerance`, `internal/medicationstatement`**: the model, `Input` (the JSON write body),
  `Seen` (an entry with `Attributed`), and the FHIR mapping `toFHIR(entry, attributed)` with golden files. SAL-41
  adds the repository, the service, the handlers, the FHIR routes and their CapabilityStatement entries.
- **`internal/history`**: `Item`, `Page`, `Filter`, `Cursor` (opaque, keyset on date then reference, newest first),
  `Summary` and its rows, and the ports `Allergies` and `Medications` that SAL-41's services implement with exactly
  that method (`Seen(ctx, appointment.Caller, patientID)`), with their fakes in `fakes_test.go`.
- **`api/openapi.yaml`**: the paths and schemas of all four tickets (tag `history`), the generated clients, and
  `api/contract_history_test.go` keeping the schemas equal to the Go types and enums (`EntryWritten`'s Go type comes
  with SAL-41's handlers, which add it to that test). A ticket that needs to change a
  schema changes the Go type with it (the contract test fails otherwise) and says so in its PR.

## Packages

Each package: strict TDD, Docker only, CLAUDE.md and archcheck. New user-facing strings in every language of their
app (web five, mobile ten), `legal`-tagged where they carry legal weight, listed in the PR as copy waiting for the
owner.

### SAL-39, the patient's timeline

- `history.Service.Timeline(ctx, appointment.Caller, Filter, Cursor, limit) (Page, error)` in
  `internal/history/service_patient.go`: the patient's own items from every clinic. A source per item type, each a
  port on the owning domain's service (encounter, composition, questionnaireresponse, binary) returning the
  patient's items after a cursor, already filtered by its own access rule and audited. The service merges them by
  `Cursor.Before` and cuts the page. Chats plug in later as another source of `TypeCommunication` through `internal/communication` (SAL-27, on main since 2026-11-12), not in this package.
- The clinic name of each item: the patient's scope cannot read `organization` today; add what is needed (a method
  on `organization.Service` for the clinics of the patient's own appointments, and if row-level security needs it, a
  `SECURITY DEFINER` lookup like `encounter_organization`, 00033) in this package's own migration.
- `GET /me/history` (`handler_http.go`). Benchmark: a patient with 2,000 items, p95 of a page under 50 ms in the
  numbers of PROGRESS.
- `mobile/`: the history screen (filters by clinic, type and date; infinite scroll on `next`), each item opening the
  existing screens (SAL-17's documents, the questionnaire answers, the file viewer).
- E2E: a patient with visits at two clinics sees both, newest first, and never a draft.

### SAL-40, the practitioner's view and the pre-visit summary

- `history.Service.ForPractitioner(ctx, appointment.Caller, patientID, Cursor, limit) (Page, error)` and
  `Summary(ctx, appointment.Caller, patientID, appointmentID) (Summary, error)` in
  `internal/history/service_practitioner.go`. The facts for the caller's clinic come from `consent.Service.State`
  (now with the shared visits) and the patient's appointments there; every source filters with `access.HistoryVisit`
  before it lists or counts. Each owning domain gets a read for a practitioner's visits and for shared encounter
  ids, applying `access.HistoryVisit` (not its own appointment rule, which refuses another clinic's visit).
- The questions of a shared questionnaire response: 00042 does not open another clinic's `questionnaire` rows. A
  `questionnaire_shared` policy chained on `questionnaire_response` made the planner spin (an UPDATE of
  `questionnaire_translation` never finished), so read the versions a shared completed response answers through a
  `SECURITY DEFINER` lookup that returns only those versions (as `encounter_organization`, 00033), in this
  package's migration, with a test that `migrations/language_test.go` still runs in its usual time.
- Allergies and medications through the `Allergies` and `Medications` ports. Until SAL-41 is on main, the fakes;
  SAL-41 merges first, so this branch merges main and wires the real services in `internal/api/app.go`.
- `GET /patients/{id}/history`, `GET /patients/{id}/history/summary?appointment=`. Every view audited.
- `web/`: the history panel from the appointment detail and the call page, shared items marked with their clinic.
- Tests: service tests over every viewer (no care relationship, admin, colleague, other clinic, revoked) and no
  cross-clinic data without sharing, also not in a count; handler tests; an AuditEvent per view.

### SAL-41, allergies and medications

- `allergyintolerance` and `medicationstatement`: repository (Postgres, its sqlc package), service (`Add`, `Record`,
  `Update`, `Remove`, `Confirm`, `AddToProfile`, `Seen` with the port's exact signature), JSON handlers as
  api/openapi.yaml says, the FHIR read and search (`?patient=`) with golden files, and their entries in
  `api/fhir/CapabilityStatement.json`. Validation against the value sets and the table's lengths.
- Terminology search for substances and medications through the existing `/terminology/search` (medications: the
  drug category; substances: drugs, with free text for anything else).
- `mobile/`: the profile's allergies and medications, add, edit, remove, "Add to my profile" on a clinic's entry.
- `web/`: record and confirm during a visit on the call page; an allergy warning on the call page and in the
  document drafting editor.
- E2E: the patient adds an allergy, the practitioner sees the warning on the call page.

### SAL-42, cross-clinic sharing

- `consent`: the history kind's grant, change and revocation (`KindHistory` in `Valid`), `Candidates(ctx, Caller,
  organizationID)` (the patient's finished visits at other clinics, as `Candidate`), the care consent's revocation
  ending the history consent, the audit of every grant, change, removal and revocation, and the endpoints of
  api/openapi.yaml.
- `mobile/`: the toggle at booking (off by default) expanding into the preview with a checkbox per visit, select
  all, select none, sending only what is ticked; the settings screen per clinic (add or remove visits, revoke).
  The consent and revocation strings are `legal`.
- Tests: removal and revocation take effect at once (service and row-level security); the suggestions and the
  drafting never receive a shared or unshared visit's content.
- E2E: the patient shares 2 of 3 previous visits; the practitioner sees exactly those 2 in their view (SAL-40's
  endpoint, so this branch merges after SAL-40).

## Waves and merge order

```
Step 0 (this PR): plan, ADR 0013, the contract (00042)
Wave 1, in parallel:  SAL-39   SAL-40   SAL-41   SAL-42
Merge order:          SAL-41, SAL-39, SAL-40, SAL-42
```

- SAL-40 wires SAL-41's services, so SAL-41 merges first. SAL-42's journey reads SAL-40's endpoint, so it merges
  after SAL-40. SAL-39 and SAL-40 both add to `internal/history`, in separate files; the second resolves the rest.
- One at a time: main merged into the branch first (no force-push), migrations renumbered if main took their numbers,
  generated code (sqlc, the clients) regenerated after the merge, never hand-merged, checks rerun.
- Two other epics run at the same time (chat SAL-26, analytics SAL-18): their code is not touched. Shared files
  (`api/openapi.yaml`, the clients, `ownership.yaml`, `config.yaml`, `internal/retention/rules.go`,
  `internal/api/app.go`, the router) are merged by hand.

Each package runs in its own git worktree, branch and compose project (`-p salus-hist39`, `salus-hist40`,
`salus-hist41`, `salus-hist42`; step 0 `salus-hist0`) with an override that publishes no host ports and tags its own
app image, so the sessions never share a stack.

## Migrations

Step 0 is `00042`. A package that needs one takes the next free number when it merges main before its own merge (a
migration not yet on main may be renumbered; `scripts/check-migrations.sh` guards only main's).

## Checks per PR

`make lint`, `make test`, `make archcheck`, `make e2e`, `make cover`, `make fhir-validate` when a golden FHIR file
changed, plus the web and mobile checks of what changed. Copilot's quota is out until 2026-12-01: each PR is
reviewed by a fresh subagent that has not seen the work, given the diff; valid findings are fixed before the merge,
declined ones listed in the PR with the reason.

## Personal data

Every field is health data (Art. 9); the patient is the data subject.

| Table | Fields | Why |
| --- | --- | --- |
| `allergy_intolerance` | patient, the recording clinic, visit and role (for a clinic's entry), the term, the substance's text, category, reaction, severity, clinical and verification status, who confirmed and when, in the profile or not | The allergies every treating doctor must know (safety); the source and confirmation for the record and for the visit-scoped rule |
| `medication_statement` | patient, the source as above, the term, the medication's text, the dose and its unit, how often, as needed, status, the confirmation | The current medications, for safe prescribing |
| `history_consent` | patient, the receiving clinic, granted, revoked, status, version | Proof of the patient's decision (Art. 7(1)) and the access check |
| `history_consent_encounter` | the consent, the visit, when added, when removed | Which visits were shared when: the access check and its proof |

No other field is collected. The DPIA notes are in docs/privacy/dpia.md "Patient history".

## Copy waiting for the owner

Every new user-facing string goes to the owner first (CLAUDE.md "Copy"), listed in each PR's description and in
PROGRESS when merged: SAL-39's timeline (ten languages), SAL-40's view and summary (five), SAL-41's profile screens
(ten) and the warning (five), SAL-42's toggle, preview and settings (ten, `legal`).
