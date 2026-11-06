# Questionnaires implementation plan

Read `SPEC.md` first (approved 2026-11-06, D1 to D9 answered in §3). Packages run in order; each is one commit
`questionnaire(<n>): <summary>` on `feat/questionnaire`, made with `/run-package docs/questionnaire <n>`.

## Rules for every package

- Test first: write the failing test, run it and record the failure in `PROGRESS.md`, then the least code to
  pass, then refactor. Service tests with hand-written fakes and `clock.Fixed`; repository tests on
  `pgtest.NewDatabase` (and `NewAppDatabase` for every RLS check); handlers with `httptest`, FHIR handlers
  against golden files.
- Every new domain is added to `docs/architecture/archcheck/config.yaml` (context `clinical`) and to the
  `clinical` line of CLAUDE.md "Go architecture", every new table to `ownership.yaml` with `personal_data` (and
  a retention rule when true), every domain with queries to `sqlc.yaml`, in the commit that adds it. The
  archcheck allowlist never grows.
- Every migration is a new file, with its grants, `ENABLE ROW LEVEL SECURITY` and policies. Policies read
  settings as `NULLIF(current_setting('…', true), '')`.
- Every read or write of a `QuestionnaireResponse`, a `Questionnaire` or a role's default audits in the same
  transaction, refusals included; a test per service method asserts the event.
- No personal data, questionnaire title, item text, option or answer in a log line, error, panic or job
  argument. A test asserts the log output of each new handler and job holds none of them. Golden files and
  fixtures are synthetic, medical wording labelled as a placeholder.
- A change to `api/openapi.yaml` is followed by `make gen-clients` in the same commit. A change to the FHIR
  interactions updates `api/fhir/CapabilityStatement.json` in the same commit.
- Timing tests wait for a condition (as in `4c62e68`), never on a longer sleep; a flaky one is fixed, not
  slowed or skipped.
- No new Go, npm or Flutter dependency except `@dnd-kit/core` and `@dnd-kit/sortable` in package 8 (SPEC §3
  "Why `@dnd-kit`"), pinned exact. Pin every version.
  Update `PROGRESS.md` in the same commit.

## Check sets

Every check runs through `make`, in Docker. Never `go`, `npm` or `flutter` on the host.

- **Go**: `make lint`, `make test`, `make archcheck`, `make cover` (every package at 80% or more).
- **FHIR**: `make fhir-validate`.
- **Clients**: `make gen-clients` leaves no diff.
- **Web**: `make web-lint`, `make web-typecheck`, `make web-test`, `make web-build`.
- **Mobile**: `make mobile-analyze`, `make mobile-test`.
- **Stack**: `make up`, then the package's manual check, then `make down`.

## Packages

### 1. questionnaire

Migration: `questionnaire` and `questionnaire_default` (SPEC §2 "Tables"), the partial unique indexes (one
draft, one active per series), RLS (clinic scope; the patient scope's policy on `questionnaire` comes in package
3 with the table it reads), grants to `salus_app`. Ownership entries (`personal_data: false`). `config.yaml`:
context `clinical: [questionnaire]`; CLAUDE.md's `clinical` line. `sqlc.yaml` entry.
`internal/questionnaire`: `model.go` (`Questionnaire`, `Item`, `ItemType`, `Option`, `Condition`, `Status`,
`Default`, `Caller`), `errors.go`, `repository.go`, `repository_postgres.go`, `service.go`: `Create`,
`UpdateDraft` (`If-Match`), `DeleteDraft`, `Publish` (retires the series' active version in the same
transaction), `Retire`, `Revise` (copies the active version to a new draft, same `linkId`s), `Read`,
`Search`; `service_default.go`: `SetDefault`, `ClearDefault`, `Default` (the practitioner for their own role, an
admin for any role of the clinic, through `practitionerrole`); `validate.go` beside the model: the item types,
limits, `linkId`s, conditions (SPEC §2 "Item model", "Limits", "linkIds"). Every method audited.
`BenchmarkPublish` and `BenchmarkReadVersion` (200 items, real database), numbers in `PROGRESS.md`.
Tests: the reserved `attachment` type is declared and refused (422 `invalid_questionnaire`); each lifecycle
step and each refusal (edit an active or retired version, publish a retired one, revise
with a draft existing: `draft_exists`, delete a published one); publishing v2 retires v1 in one transaction (a
rollback leaves v1 active); two concurrent publishes of drafts of one series leave one active (real database);
every validation rule, table-driven (each limit at and one past its boundary; duplicate, malformed and
type-changed `linkId`s; a condition on a later item, a non-boolean non-choice item, a missing option, a hidden
section's items); the default: own role, another practitioner's role (404), admin any role, another clinic,
a questionnaire without an active version (409 `not_active`); every method's audit event; as `salus_app`, the
clinic scope sees only its clinic's rows and an unset scope sees nothing. The log holds no title or item text.
Checks: Go.

### 2. questionnaire API

`questionnaire/handler_fhir.go`: `Questionnaire` read and search by `url`, `version`, `status` (staff of the
clinic; patients come in package 6). `questionnaire/handler_http.go`: `POST /questionnaires`, `PUT` and
`DELETE /questionnaires/{id}`, `POST /questionnaires/{id}/publish`, `/retire`, `/revise`, and `GET`, `PUT`,
`DELETE /practitioner-roles/{id}/questionnaire` (SPEC §2 "JSON API"), body limit 256 KiB, 422 with `linkIds`.
`openapi.yaml`, `make gen-clients`; `CapabilityStatement.json` gains `Questionnaire` with `read`, `search-type`
and the three parameters. Golden files: a draft, an active version with every item type, help text and a
condition, a search `Bundle`, and the `OperationOutcome`s (404, 400). The help text uses the R4 core
`questionnaire-itemControl` extension, validated by `make fhir-validate`.
Tests: each endpoint's success and error codes through HTTP (403 for patients, 404 for another clinic, 409,
412, 422); the FHIR read and search access per role; an unknown search parameter is 400; audit per read and
per returned resource; the handler log holds no title or item text.
Checks: Go, FHIR, Clients.

### 3. questionnaireresponse and the access rule

`internal/access`: `QuestionnaireResponse` and `RequestsQuestionnaire` (SPEC §2 "Access rule") with their table
tests over every combination. Migration: `questionnaire_response` (SPEC §2 "Tables"), RLS (patient scope; clinic
scope with active consent, and `completed` rows of appointments started before the latest revocation; the
notice scope; the discard `DELETE` policy), the patient-scope policy on `questionnaire` (versions referenced by
the patient's own responses), grants to `salus_app`, `SELECT, DELETE` to `salus_retention`. Ownership entry
(`personal_data: true`), retention rule `delete` (SPEC §5) and its purger in `service_retention.go`.
`config.yaml`: `questionnaireresponse` in `clinical`; CLAUDE.md's line. `sqlc.yaml` entry.
`internal/questionnaireresponse`: model, errors, repository, `service.go`: `Request` (the practitioner adds a
questionnaire to an appointment), `Withdraw`, `Read`, `Search` (by appointment, patient, status), with the
ports it consumes from `appointment` (`Attendance`, `Lookup`), `questionnaire` (a version) and `auditevent`.
Tests: the access table, including D9 (after a revocation the appointment's practitioner is `Full` on a
submitted response of a past appointment, `None` on an unsubmitted one and on a future one; the admin is never
`Full`; nobody but the patient and the appointment's practitioner is ever `Full`); `Request` by the appointment's practitioner, another practitioner, an admin, the
patient, another clinic; on a cancelled, started, fulfilled appointment; without consent; twice
(`already_requested`; a withdrawn one may be requested again); a draft or retired version (`not_active`);
`Withdraw` before and after submit (submit arrives in package 4: the test sets the row); a withdrawn response
has no answers; `Read` and `Search` per role and level (a `Name` view carries no answers); the purger deletes
unsubmitted rows past `due_at` and keeps `completed` ones and those not yet due (± 1 s); as `salus_app`: the
patient scope sees only its own rows, the clinic scope after a revocation sees only `completed` rows of past
appointments, the notice scope only the one appointment's rows, the patient scope only the referenced
`questionnaire` versions; `salus_retention` can delete only through the purger's query; audit per call,
refusals included.
Checks: Go.

### 4. Answers

`questionnaireresponse/service_answers.go`: `SaveAnswers` and `Submit` (SPEC §2 "Patient flow"), the condition
evaluation (`enabled.go`: an item is enabled when its condition holds and the item it depends on is enabled;
repeats match any answer), answer checks per type and limit. `BenchmarkSaveAnswers` and `BenchmarkSubmit`
(200 answers, real database), numbers in `PROGRESS.md`.
Tests: each type's valid and invalid answers at the limits, table-driven; a save on a submitted, withdrawn,
past-due response, by staff, by another patient, without consent (`consent_required`, also right after a
revocation commits), with a stale version (412); submit with a missing required answer (422 with its
`linkId`), with a required question that is hidden (submits), with answers to hidden items (removed), twice
(409 `submitted`); conditions on a boolean, a single choice, a choice with repeats, a hidden section, a chain of
two; `authored_at` from the clock; audit per save and submit, refusals included; the log holds no answer.
Checks: Go.

### 5. Booking and cancellation

`appointment`: the `Jobs` port gains `EnqueueQuestionnaireRequest` (enqueued in the booking transaction) and
`EnqueueQuestionnaireDiscard` (in the shared `cancelled` step, so the patient's, staff's and the revocation
job's cancellations all enqueue it); job args in `model.go`, opaque ids only. `questionnaireresponse`:
`RequestDefault` and `Discard` (SPEC §2 "Requests", "Cancellation and revocation"), audited with the job as
agent. `internal/worker`: the `questionnaire_default_request` and `questionnaire_responses_discard` workers.
`TestScheduler_KnowsEveryKindTheWorkerAndTheNotifierWork` lists both kinds.
Tests: booking enqueues the request in its transaction (a rollback leaves none) with the two ids only; every
cancellation path enqueues the discard; `RequestDefault` with a default, without one, with the series
retired, twice (one row); `Discard` deletes every response of a cancelled appointment, submitted or not, and
does nothing for one that is not cancelled; the revocation of a consent with a future and a past appointment
discards the future one's responses and keeps the past one's `completed` response; repeating either job changes
nothing; the scheduler test fails without the two kinds.
Checks: Go.

### 6. Responses API

`questionnaireresponse/handler_http.go`: `POST /appointments/{id}/questionnaires`, `PUT
/questionnaire-responses/{id}/answers`, `POST /questionnaire-responses/{id}/submit`, `/withdraw` (SPEC §2
"JSON API"). `questionnaireresponse/handler_fhir.go`: read, search by `appointment`, `patient`, `status`, with
`_include=QuestionnaireResponse:questionnaire`; answers nested under their sections. `questionnaire`: the
patient's read and search (versions their own responses reference). `api/fhir/`: the
`questionnaireresponse-appointment` extension and its `SearchParameter`. `openapi.yaml`, `make gen-clients`;
`CapabilityStatement.json` gains `QuestionnaireResponse` with `read`, `search-type`, the three parameters and
the `searchInclude`. Golden files: an in-progress, a completed and a `Name` view `QuestionnaireResponse`, a
search `Bundle` with an included `Questionnaire`, the `OperationOutcome`s. Load test `make
bench-questionnaire-http` (SPEC §2 "Performance"), p50, p95, p99 in `PROGRESS.md`.
Tests: every error code through HTTP; the access matrix for read and search (patient, practitioner with and
without access, admin, other clinic, after a revocation, submitted and not); `_include` brings each version once;
an unknown search parameter is 400; audit per read and per returned resource; `make fhir-validate` passes with
the new golden files against the extension; the handler log holds no answer.
Checks: Go, FHIR, Clients.

### 7. E-mail mention

Copy approved (`COPY.md`, E1). `questionnaireresponse.Service.Outstanding`
(SPEC §2 "Notifications"), read in the notice scope, audited with the job as agent. `notification`: the booking
confirmation and the reminder add E1 when one is outstanding; a port `Questionnaires` with that one method.
`docs/privacy/processors.md`: the SES entry gains the sentence.
Tests: the confirmation with an in-progress response, with a role default not yet requested, with nothing
outstanding, with a submitted one; the reminder likewise; the body never holds the title (a synthetic title is
asserted absent); the audit event of the check.
Checks: Go.

### 8. Web: library and builder

Copy approved, W41 and W42 included (2026-11-06). The strings go into `web/src/messages.ts` exactly as `COPY.md` has them. `web/package.json`:
`@dnd-kit/core` and `@dnd-kit/sortable`, exact versions (the current releases, their runtime dependencies
checked, all recorded in `PROGRESS.md`); an oxlint `no-restricted-imports` rule so only `builder-dnd.tsx` imports
them. `web/src`: the "Questionnaires" link in the main navigation; the library page (W1 to W8:
title and status, version in a tooltip, the status filter, "New questionnaire"); the builder page
(`Builder.tsx`: the type palette, the questionnaire, item cards closed by default, "More options" for help text
and the condition; drag and drop with `@dnd-kit` (pointer, touch and keyboard sensors) from the
palette and between positions, into and out of sections, its screen-reader instructions and announcements set
to W41 and W42, no default English text left; "Move up" and "Move down", announced with W42 through a live
region; delete, with the warning when a condition depends on the item;
"Save draft", "Publish" with its confirmation, "Edit" on an active version (revise), "Retire", "Discard draft";
the limits shown before they are reached; the unsaved-changes prompt). The list logic lives in one small module
(`builder.ts`) as pure functions over the item list (insert, move, move into a section), so it is tested
without the DOM; `builder-dnd.tsx` maps `@dnd-kit`'s events onto it. The builder's measurements (SPEC §2 "Performance") in `PROGRESS.md`.
Tests (Vitest): `builder.ts` moves, inserts, rejects a section inside a section, removes a condition whose
question is deleted; a type changed gets a new `linkId`; each card state; Move up and Move down at the ends, and their W42 announcement;
a keyboard drag (space, arrows, space) reorders and announces W42; escape cancels; the instructions are W41; save sends `If-Match` and the CSRF header;
a 409 `draft_exists`, a 412 and a 422 with `linkIds` show their messages on the right items; publish asks to
confirm.
Checks: Web, Clients; the builder walked in a browser against the stack at desktop width, with the keyboard only,
and with touch emulation.

### 9. Web: appointment and working hours

Copy approved. `AppointmentDetail.tsx`: a "Questionnaires" section (the requests with their status
and submitted time; "Add questionnaire" with a picker of active questionnaires, for the appointment's
practitioner while it is booked and ahead; "Withdraw" while not submitted; the answers of a submitted one,
by section, hidden items left out, unanswered optional questions as "Not answered"). Admins see the requests and
their status only. `Availability.tsx`: the role's default questionnaire (picker, clear).
Tests: each state of the section per role; add and withdraw call their endpoints; the answers render each type;
a `Name` view shows the status only; the default picker reads, sets and clears.
Checks: Web, Clients; walked in a browser against the stack.

### 10. Mobile

Copy approved. The strings go into `mobile/lib/messages.dart` exactly as `COPY.md` has them.
`mobile/lib`: the questionnaire line and button on the card in my appointments; `questionnaire_screen.dart`
(one scrolling page, sections as headings, help text behind an icon, each type's widget, the saved state,
"Submit" with its confirmation, the field errors on 422, the read-only view after submitting); the save
behaviour (on leaving a field and 2 s after typing stops; a failed save retried and shown); conditions evaluated
on the device the same way as the server, from one function tested against the same cases as package 4. The
200-item first-frame measurement in `PROGRESS.md`.
Tests (widget, `MockClient`): each type's widget and validation; a condition shows and hides; saving after the
pause and on leaving a field, not per keystroke; a failed save; submit asks to confirm, then shows the read-only
view; a 409 `consent_required`, `past_due`, `withdrawn` and a 412 show their messages; the card's states (to
fill in, continue, submitted, closed).
Checks: Mobile, Clients. Running on a device is the manual step in `README.md`.

### 11. End-to-end

`test/e2e/questionnaire_test.go` (build tag `e2e`): synthetic data fresh per run. A practitioner creates and
publishes a questionnaire through the API (every type, one condition) and sets it as their role's default; a
patient books; the request appears (waited for as a condition); the confirmation e-mail in Mailpit holds E1 and
not the title; the patient saves answers, then submits; the practitioner reads the `QuestionnaireResponse` over
FHIR with the included version and the answers nested. Second journey: the practitioner adds a second
questionnaire, the patient saves part of it, the patient cancels; the responses are gone (waited for). A
Playwright spec (`test/e2e/web/questionnaire.spec.ts`) builds a three-item questionnaire in the builder by drag
and drop and with "Move up", publishes it, and finds it in the library; the move by drag uses the mouse, and one more by keyboard.
Checks: all of the above, `make e2e` twice in a row.

## Definition of done

Every package committed, `PROGRESS.md` complete, all check sets pass locally and in CI on the PR into `main`.
