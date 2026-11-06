# Questionnaires progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 questionnaire | done, 2026-11-06 | `questionnaire(1)` |
| 2 questionnaire API | done, 2026-11-06 | `questionnaire(2)` |
| 3 questionnaireresponse and the access rule | done, 2026-11-06 | `questionnaire(3)` |
| 4 Answers | done, 2026-11-06 | `questionnaire(4)` |
| 5 Booking and cancellation | not started | |
| 6 Responses API | not started | |
| 7 E-mail mention | not started | |
| 8 Web: library and builder | held: W41 and W42 not approved | |
| 9 Web: appointment and working hours | not started | |
| 10 Mobile | not started | |
| 11 End-to-end | not started | |

## Decisions

- 2026-11-06, owner (recorded in `SPEC.md` §2 and §3): D1 uploads in a separate feature, "questionnaire
  attachments", right after this one, with `attachment` reserved in the model; D7 `@dnd-kit`, pinned, plus
  Move up and Move down; D2 to D6 and D8 as recommended; D9 the appointment's practitioner keeps reading the
  submitted answers of past appointments after a revocation, nobody else. `COPY.md` approved with W11, W35 and
  M13 changed.
- 2026-11-06, plan: `@dnd-kit` speaks to screen readers during a drag, and the approved text has nothing for
  it; W41 and W42 are proposed in `COPY.md`, and package 8 is held until the owner approves them.

## Package 1: questionnaire (done, 2026-11-06)

Built:
- Migration `00025_questionnaire.sql`: `questionnaire` (one row per version; `series_id`, `version`, `status`
  `draft` | `active` | `retired`, `items` jsonb, `published_at`, `retired_at`) with unique (`series_id`,
  `version`), the partial unique indexes `questionnaire_one_draft` and `questionnaire_one_active`, checks that
  a draft has no publish time and only a retired version a retire time, and `UNIQUE (id, organization_id)`
  for package 3's reference; `questionnaire_default` (primary key the role, referencing `practitioner_role (id,
  organization_id)`, so the role and the default are always one clinic). RLS: the clinic scope only (also
  closed to the patient settings `app.patient_id` and `app.account_id`, as `appointment` is); a restrictive
  `DELETE` policy deletes drafts only. Grants to `salus_app`: `SELECT, INSERT, DELETE` and `UPDATE` of the
  lifecycle and content columns only (clinic, series and version number never change). Ownership entries
  (`personal_data: false`), `config.yaml` context `clinical: [questionnaire]`, CLAUDE.md's `clinical` line,
  `sqlc.yaml` entry (`make sqlc` also adds the two tables to every domain's generated `models.go`).
- `internal/questionnaire`: `model.go`, `errors.go` (`ErrNotFound`, `ErrForbidden`, `ErrConflict`,
  `ErrNotDraft`, `ErrNotActive`, `ErrDraftExists`, `*InvalidError` with the offending `LinkIDs`, which is
  `ErrInvalid`), `repository.go`, `repository_postgres.go`, `query.sql`; `service.go` (`Create`,
  `UpdateDraft`, `DeleteDraft`, `Read`, `Search`), `service_lifecycle.go` (`Publish`, `Retire`, `Revise`),
  `service_default.go` (`Default`, `SetDefault`, `ClearDefault`), `validate.go`.
  Every call is audited in its transaction, refusals included: `Questionnaire/<id>` (search: one event per
  version returned, or one without an entity), defaults as `PractitionerRole/<role id>`; staff `HOPERAT`,
  anyone without a clinic `PATRQT` and `ErrForbidden`.
- Not wired into `internal/api/app.go` yet: nothing calls the service before package 2's handlers.

Failing first (red), each for the right reason before the code:
- `migrations`: the five `TestQuestionnaire_*` tests, `relation "questionnaire" does not exist`.
- `internal/questionnaire`: every service test against a stub returning `todo` (Create, UpdateDraft, the
  lifecycle refusals, Publish, Retire, Revise, Read, Search, the defaults, the validation table); the
  repository tests did not build (`undefined: questionnaire.NewPostgresRepository`). One test bug found on the
  way: a validation case mutated the shared options slice of another; fixed in the test.

Checks: `make lint` 0 issues, migrations unchanged; `make test` pass; `make archcheck` pass (a first draft of
the real-database tests used `auditevent.NewPostgresRepository` and was refused as `foreign-repository`; they
count events through a test auditor instead, auditevent's own tests cover the storage); `make cover` pass,
`internal/questionnaire` 93.9%, total 94.7%; `make sqlc` regenerated, `make migrate` applied 00025 to the dev
database.

Measurements (`go test -run '^$' -bench 'BenchmarkPublish|BenchmarkReadVersion' -benchmem -benchtime=2s
-count=3 ./internal/questionnaire/` in the `tools` container, compose Postgres, i7-14700HX; a 200-item
questionnaire: 10 sections of 19 items, every type, help text and conditions):

| Benchmark | ns/op (3 runs) | B/op | allocs/op |
| --- | --- | --- | --- |
| `BenchmarkPublish` (validate and store a draft, then publish it; two transactions, each audited) | 10.8 M, 10.5 M, 11.1 M | ~970 k | ~2,725 |
| `BenchmarkReadVersion` (read one version and decode its items; one transaction, audited) | 3.47 M, 3.31 M, 3.21 M | ~186 k | ~890 |

Both are dominated by the transaction and its commit (each call commits an audit event), not by validation
or JSON. Encoding as FHIR comes with package 2's handler and is measured there. Nothing is changed on these
numbers.

Readings:
- `Publish` takes the version the caller last read (`If-Match`), like `UpdateDraft`, so a draft is published
  as it was seen; `Retire`, `Revise` and `DeleteDraft` take none (they change no content).
- `Publish` locks the draft first (`FOR NO KEY UPDATE`), then retires the active version and publishes: two
  concurrent publishes of one draft serialise and the second gets `ErrNotDraft` (tested on the real database).
  Two drafts of one series cannot exist (partial unique index), so that is the only concurrent publish.
- `Revise` only from the active version (`not_active` otherwise); the new version number is the series'
  highest plus one, and the insert is `ON CONFLICT DO NOTHING`, so a series with a draft, or a concurrent
  revise, gives `draft_exists`. The active version stays active until the new draft is published.
- A linkId may not change kind against any other version of the series (not only the previous one), so a
  linkId dropped in v2 and reused in v3 keeps its type.
- Item text is required on every item, sections included (COPY W28 "Add text to this item."); the title is
  required (the builder sends W8 "Untitled questionnaire"). Help text is allowed on questions and sections,
  not on instructions (a FHIR display item has no children, que-1). A malformed linkId fails the save
  without being named in `LinkIDs`; a duplicate is named at its second occurrence.
- The help text, the condition and the options are stored as the model's JSON (`help`, `enableWhen`,
  `answerOption`, nested `item`); package 2 maps the help text to the core `questionnaire-itemControl` child.
- `SetDefault` takes any version id of the series and stores the series; a series with no active version is
  `not_active`; a questionnaire of another clinic, or a role the caller may not set (another practitioner's,
  another clinic's), is `ErrNotFound`. `Default` of a role without one returns no series at version 0, as
  `schedule` does for a role without availability. Roles are read through `practitionerrole.Service.Lookup`.
- Patients get `ErrForbidden` from every method here (package 2's 403); their read of the versions their own
  responses reference comes in package 6.

Deferred: the handlers, `openapi.yaml`, the FHIR mapping and the wiring in `app.go` (package 2); the patient
scope's policy on `questionnaire` (package 3).

## Package 2: questionnaire API (done, 2026-11-06)

Built:
- `internal/questionnaire/handler_http.go`: `POST /questionnaires` (201), `PUT` (`If-Match`) and `DELETE`
  (204) `/questionnaires/{id}`, `POST /questionnaires/{id}/publish` (`If-Match`), `/retire`, `/revise` (201),
  and `GET`, `PUT` (`{"questionnaireId"}`) and `DELETE` (204) `/practitioner-roles/{id}/questionnaire`. Bodies
  are read up to 256 KiB, known fields only (the items too), one object; anything else is 400
  `invalid_request` before the service is called. Errors: 403 `forbidden`, 404 `not_found`, 409 `not_draft`,
  `not_active`, `draft_exists`, 412 `version_conflict`, 422 `invalid_questionnaire` with `linkIds` (an empty
  list when none is named). Versions carry their `versionId` as the ETag.
- `internal/questionnaire/handler_fhir.go`: `Questionnaire` read and search by `url`, `version`, `status`
  (each at most once; anything else 400 `not-supported`). `url`, `version` (a string), `title`, `status`,
  `subjectType` `Patient`, `date` (the last change), `description`, `item`, `meta`. Conditions as
  `enableWhen` with operator `=`; help text as a child `display` item `<linkId>.help` with the R4 core
  `questionnaire-itemControl` extension, code `help` (`.` is never in one of our linkIds, so it cannot clash).
- `internal/api/app.go`: the service and both handlers (`questionnaireRoutes`), and `questionnaireCaller`:
  the session's clinic only for its admin or a practitioner (from `auth.ClinicMember`), so a patient is a
  caller without a clinic and gets `ErrForbidden`.
- `api/openapi.yaml`: the nine operations (tag `questionnaires`), `SaveQuestionnaireRequest`,
  `QuestionnaireItem` (the type enum without `attachment`), `Questionnaire`, `QuestionnaireDefault`,
  `SetRoleQuestionnaireRequest`, the four new error codes and `Error.linkIds`; `make gen-clients` (TypeScript
  `web/src/api/schema.d.ts`, Dart `mobile/packages/salus_api`).
- `api/fhir/CapabilityStatement.json`: `Questionnaire` with `read`, `search-type` and `url` (uri), `version`
  and `status` (token), the core `conformance-*` definitions; `embed_test.go` and the router's `metadata.json`
  golden follow.
- Golden files (`internal/questionnaire/testdata/`): `questionnaire_active.json` (every item type, help text
  on a question and a section, conditions on a boolean and a choice, one hiding a section),
  `questionnaire_draft.json`, `search.json`, `not_found.json`, `unsupported_search.json`, and also
  `forbidden.json` and `unauthenticated.json`. Synthetic, placeholder wording.

Failing first (red), each for the right reason before the code:
- `internal/questionnaire`: the handler tests did not build (`undefined: questionnaire.NewHTTPHandler`,
  `undefined: questionnaire.NewHandler`); then the FHIR tests failed on the missing golden files until they
  were written with `-update` and reviewed.
- `api/fhir`: `TestCapabilityStatement_ListsExactlyTheInteractionsServed`, `Questionnaire = ""`.
- `internal/api/router`: `metadata.json` differed from the new CapabilityStatement (rewritten with `-update`,
  diff reviewed: the new resource, the date and one sentence of the documentation).
- `internal/api`: `TestNew_ServesTheQuestionnaireLibraryToStaffOfTheClinicAndAuditsIt` was written after the
  wiring; checked red by taking the routes out again (`404 page not found`), then put back.

Tests: every route's success and each error code through `httptest`; 401 and 400 without calling the
service; the 256 KiB limit at and one byte past it; the log holds nothing on success and only the error on a
failure, never a title, text, option or help text. FHIR: read and search golden files, the filter passed, each
unreadable parameter 400 without a call. Through the app on the real database as `salus_app`: the lifecycle
(edit, 412, 422 with `linkIds` for the reserved `attachment` type, publish, `not_draft`, revise,
`draft_exists`, retire, discard of the new draft), the default (set, read by the admin, cleared, `not_active`
for a retired series), a patient (403 on the JSON API and the FHIR search, 404 on the FHIR read), another
clinic's admin (404, an empty search), and the audit: one event per call and per version a search returned,
refusals included, counted against a baseline.

Checks: `make lint` 0 issues, migrations unchanged; `make test` pass; `make archcheck` pass; `make cover` pass,
`internal/questionnaire` 95.8%, total 94.9%; `make fhir-validate` 0 errors (only the dom-6 narrative warning
every resource has, and an information that the item-control code system is draft in R4); `make gen-clients`
run twice, no diff the second time; `make web-typecheck` pass (not in this package's check sets, run because
`schema.d.ts` changed).

Measurements (`go test -run '^$' -bench 'BenchmarkFHIRReadVersion|BenchmarkReadVersion' -benchmem
-benchtime=2s -count=3 ./internal/questionnaire/` in the `tools` container, compose Postgres, i7-14700HX; the
200-item questionnaire of package 1):

| Benchmark | ns/op (3 runs) | B/op | allocs/op |
| --- | --- | --- | --- |
| `BenchmarkFHIRReadVersion` (encode one version as FHIR JSON through the read handler, service stubbed) | 532 k, 542 k, 545 k | ~205 k | 805 |
| `BenchmarkReadVersion` (package 1's, run again beside it) | 1.13 M, 1.16 M, 1.17 M | ~189 k | ~867 |

A FHIR read of a 200-item version costs about 1.7 ms in all, a third of it the encoding. `BenchmarkReadVersion`
ran faster than package 1's run (3.2 to 3.5 M), on the same machine with a less loaded database; both are
recorded. Nothing is changed on these numbers.

Readings:
- A patient's FHIR read is 404, as SPEC.md §2 "FHIR" says for anyone who may not see a version; a patient's
  search is 403 (`forbidden.json`), since they may not search the library yet. Package 6 gives patients the
  versions their own responses reference, in the patient scope.
- A search `url` must be a canonical url of this server, without `|version` (use `version`); anything else
  is 400, documented in the CapabilityStatement, rather than an empty searchset.
- The JSON API carries the model's own item JSON (`linkId`, `text`, `type`, `help`, `answerOption`, `item`,
  `required`, `repeats`, `enableWhen` without an operator), so the builder writes what it reads; FHIR is mapped
  in `handler_fhir.go`. Versions are named by `url` (the canonical url), not by a series id.
- A role's default has no `If-Match`: it is one setting, replaced whole, and the service takes no version.
- An over-limit body is 400 `invalid_request`, as the other JSON endpoints answer it.

Deferred, for a person:
- FHIR R4 `que-1` requires a `group` to have items; the service accepts a section with none (package 1), so
  an empty section would be invalid FHIR. Whether publishing should refuse one is a product question; nothing
  is changed here.

## Package 3: questionnaireresponse and the access rule (done, 2026-11-06)

Built:
- `internal/access`: `QuestionnaireResponse(viewer, facts, visit, submitted)` and `RequestsQuestionnaire(viewer,
  facts, visit)`, table-tested over role × consent × appointment × clinic × submitted, with the assertion that
  nobody but the patient and the appointment's practitioner is ever `Full`, and the revocation boundary to the
  second.
- Migration `00026_questionnaire_response.sql`: `questionnaire_response` (SPEC §2 "Tables"), foreign keys to
  the appointment, the patient, the role and the version in the same clinic (`(questionnaire_id,
  organization_id)`), checks (`authored_at` exactly when `completed`, no answers when `stopped`, the statuses and
  sources), the partial unique index `questionnaire_response_one_live` (appointment, series) where not stopped,
  and indexes on the patient and on the unsubmitted rows' due time. RLS: the patient's own (select, update); the
  clinic's with an active consent (select, insert, update) and, after a revocation, the `completed` rows of
  appointments that started before it (select only, on the copied `due_at`); the notice scope (select); the
  discard `DELETE` policy (the appointment named in `app.appointment_id`, of the clinic in
  `app.organization_id`, and cancelled). The patient-scope policy on `questionnaire` (the versions their own
  responses reference). Grants: `salus_app` `SELECT, INSERT, DELETE` and `UPDATE` of the status, answers,
  submit time, `updated_at` and `version_id` only; `salus_retention` `SELECT, DELETE`.
- Ownership entry (`personal_data: true`), retention rule `delete` (period zero: an unsubmitted response once
  due) and its purger `questionnaireresponse.Purger` (`service_retention.go`), registered in
  `internal/worker/retention.go`. `config.yaml` `clinical: [questionnaire, questionnaireresponse]`, CLAUDE.md's
  `clinical` line, `sqlc.yaml` entry (`make sqlc` also adds the table to every domain's generated `models.go`).
- `internal/questionnaireresponse`: `model.go` (`Response`, `Answer`, `Value`, `Status`, `Source`, `Seen`,
  `Filter`), `errors.go`, `repository.go`, `repository_postgres.go`, `query.sql`, `service.go` (`Request`,
  `Withdraw`), `service_read.go` (`Read`, `Search`), `service_retention.go`. Every call audited in its
  transaction, refusals included: `QuestionnaireResponse/<id>`, `TREAT` for staff and `PATRQT` for the patient;
  a refused call without disclosure has no entity; a search one event per response returned or one without an
  entity.
- `appointment.Attendance` gains `RevokedAt` (the latest revocation of the patient's consent with the clinic),
  so a domain can run its own `access` rule on the attendance; `TestAttendance_GivesTheInactiveConsentAfterARevocation`
  asserts it.
- `docs/privacy/dpia.md`: a "Questionnaires" section with this package's subsection.
- Not wired into `internal/api/app.go` yet: no handler calls the service before package 6.

Failing first (red), each for the right reason before the code:
- `internal/access`: the new tests did not build (`undefined: access.QuestionnaireResponse`,
  `access.RequestsQuestionnaire`).
- `migrations`: the eight `TestQuestionnaireResponse_*` tests, `relation "questionnaire_response" does not
  exist`; then `TestRetention_RoleIsGrantedSelectAndDeleteOnExactlyTheTablesWhoseRuleIsDelete` failed on the new
  grant until the retention rule existed. Two of the new tests were wrong and were fixed in the test: the notice
  scope sets the clinic too, so the clinic's own view shows beside the notice's (as for `appointment`); and the
  insert-for-another-clinic case read the version through row-level security and so inserted nothing without an
  error (it now inserts the ids directly and gets `42501`).
- `internal/retention`: `TestRules_DeleteQuestionnaireResponsesOnceDue` (no rule), the `Apply` and `NewService`
  tests (no purger, then `ErrNoApplier` naming `questionnaire_response`).
- `internal/appointment`: `got.RevokedAt undefined`.
- `internal/questionnaireresponse`: every service test against stubs returning `todo`; the repository tests did
  not build (`undefined: questionnaireresponse.NewPostgresRepository`), then one failed only on the time's
  location (pgx gives `Local`), fixed in the test.

Checks: `make lint` 0 issues, migrations unchanged; `make test` pass; `make archcheck` pass; `make cover` pass,
`internal/questionnaireresponse` 93.5%, `internal/access` 100%, total 94.9%; `make sqlc` regenerated; `make
migrate` applied 00026 to the dev database.

Readings:
- After a revocation the admin sees that a response exists (`Name`) only for a `completed` one of a past
  appointment, like the practitioner's `Full`: SPEC §2 "Tables" lets the clinic scope see only those rows then,
  so the rule matches row-level security rather than giving `Name` to rows the database hides.
- `Request` reads the version through `questionnaire.Service.Read` (CLAUDE.md: wire the sibling's method
  rather than an adapter), so a request also writes the practitioner's audited read of the version. Its
  refusals map to this domain's errors (a version of another clinic or unknown is `ErrNotFound`).
- A request to an appointment that is no longer booked (cancelled, fulfilled, no-show) is `ErrAppointmentClosed`
  (409 `appointment_closed`, the code `appointment` already answers), which SPEC §2's list of 409s does not name;
  a booked one at or after its start is `appointment_started`. `consent_required` is what the appointment's
  practitioner gets when the consent is not active (in practice a past appointment after a revocation, whose
  future ones were cancelled); anyone else gets `ErrNotFound`. The checks run in that order: who, consent,
  status, start, version, duplicate.
- `Withdraw` is the appointment's practitioner's with an active consent (`access.RequestsQuestionnaire`);
  someone who sees the response but may not withdraw it (the patient, the admin) gets `ErrForbidden`, audited
  with the entity; a withdrawn one again is `ErrWithdrawn`. It locks the row (`FOR NO KEY UPDATE`) so it
  serialises with package 4's submit.
- The retention rule's period is zero (SPEC §5: deleted once `due_at` has passed). `TestRules_AreCompleteAndNameEachTableOnce`
  required a positive period for every delete rule; it now requires none for keep and restrict, a positive one
  for remove-range and zero or more for delete, and `TestRules_DeleteQuestionnaireResponsesOnceDue` pins this
  rule.
- `appointment.Attendance` gains `RevokedAt` rather than the whole `access.Facts`: a slice in it would make the
  struct uncomparable, which the existing attendance tests rely on; with `ConsentActive`, `Patient`,
  `Practitioner` and the appointment it is all the rule needs.
- The `Lookup` port from `appointment` named in the plan is not declared yet: nothing in this package calls it
  (package 5's discard does). `Value` holds the answer kinds of SPEC §2 (`valueDecimal` as `json.Number`, so no
  digit is lost); package 4 checks them.

Deferred: answers and submit (package 4); `RequestDefault` and `Discard`, and the `Lookup` port (package 5);
the handlers, the FHIR mapping, the wiring in `app.go` and the patient's read of the referenced versions in the
`questionnaire` service (package 6).


## Package 4: Answers (done, 2026-11-06)

Built:
- `internal/questionnaireresponse/service_answers.go`: `SaveAnswers` (`If-Match`) replaces the saved answers;
  `Submit` (`If-Match`) checks them in full, removes the answers of hidden items and completes the response at
  the clock's time, final. Both lock the response (`FOR NO KEY UPDATE`, so they serialise with a withdrawal),
  run in the patient's scope and share one set of checks, in this order: who (the patient; staff who see it
  `ErrForbidden`, anyone else `ErrNotFound`), `submitted`, `withdrawn`, `past_due` (at or after `due_at`),
  `consent_required`, then the version (412 `ErrConflict`), then the answers (`*AnswersError`, 422
  `answers_invalid` with the `linkIds` in document order). Every call audited in its transaction as an update,
  refusals included (with the entity when the caller sees the response).
- `validate.go`: the answer checks per type and limit (SPEC §2 "Limits"): at least one value, several only
  for a choice with repeats, each value exactly one `value[x]` of the item's type; `string` 1 to 255 and
  `text` 1 to 4,000 characters, not blank; a choice one of the item's options, each once; `decimal` written
  plainly (FHIR's form without an exponent), at most 15 significant digits; `date` a full date that exists, from
  year 1. A section, an instruction or the reserved `attachment` has no valid answer; an answer to no item of
  the version is refused unnamed; an item answered twice is named.
- `enabled.go`: the condition evaluation, one pass in document order: an item is enabled when its section is
  and its condition holds on the answer to an enabled item (a hidden one counts as unanswered, R4); a choice with
  repeats matches when any answer does.
- `repository_postgres.go` and `query.sql`: `SaveAnswers` and `Complete`, each guarded by `status =
  'in-progress'` and a new `version_id`; an empty kept list is stored as `[]`. `make sqlc` regenerated.
- `internal/questionnaire`: `Repository.SetPatient` and `Service.Referenced(ctx, caller, patientID, id)`, the
  patient's read of a version in their own scope (row-level security from package 3 shows only the versions
  their responses reference), audited `PATRQT`; the `Questionnaires` port of `questionnaireresponse` gains it.
- `docs/privacy/dpia.md`: the package 4 subsection.

Failing first (red), each for the right reason before the code:
- `internal/questionnaire`: `repo.SetPatient undefined`, `f.svc.Referenced undefined`.
- `internal/questionnaireresponse`: `f.svc.SaveAnswers undefined`, `undefined: questionnaireresponse.AnswersError`,
  `ErrAnswersInvalid`, `ErrPastDue`, `ErrConflict`, `f.svc.Submit undefined`. The repository's two methods were
  written to satisfy the port before their own repository tests; those tests were checked red by mutation
  (without the `[]` guard, `Complete` with no answers breaks `questionnaire_response_answers_check`). The service
  tests were checked to bite by mutation too: a 16-digit decimal limit fails three limit cases, and dropping the
  section from the condition evaluation fails eight condition cases.

Tests: each type's valid and invalid answers at the limits, table-driven (53 cases, including multibyte strings
counted in characters and an integer outside 32 bits refused by decoding); every refusal for both calls (submitted,
withdrawn, at and after the start, without consent, stale version, the appointment's practitioner, the admin,
another practitioner, another patient, a patient without a profile, another clinic's admin, an unknown response),
each unchanged and audited once without a read of the version; the save right after a revocation refused; a save
one second before the start; submit at the clock's time, then again (`submitted`) and a save after it; the
conditions on a boolean, a single choice, a choice with repeats, a hidden section, a chain of two through a
section, the missing linkIds in document order, the answers of hidden items removed; a saved answer that breaks
its item refused on submit; error text without a linkId or an answer; backend failures roll back. Repository, as
`salus_app`: save and complete in the patient's scope, neither on a completed response nor another patient's.
Questionnaire: `Referenced` per patient and version, audited; the patient scope on the real database shows only the
referenced version.

Checks: `make lint` 0 issues, migrations unchanged; `make test` pass; `make archcheck` pass; `make cover` pass,
`internal/questionnaireresponse` 95.6%, `internal/questionnaire` 95.7%, total 95.0%; `make sqlc` regenerated.

Measurements (`go test -run '^$' -bench 'BenchmarkSaveAnswers|BenchmarkSubmit' -benchmem -benchtime=2s -count=3
./internal/questionnaireresponse/` in the `tools` container, compose Postgres, i7-14700HX; 200 required questions,
every type, each shown on the first one's answer, about 30 KB of answers):

| Benchmark | ns/op (3 runs) | B/op | allocs/op |
| --- | --- | --- | --- |
| `BenchmarkSaveAnswers` (lock and read the response, check 200 answers, store them; one transaction, audited) | 4.25 M, 4.86 M, 5.04 M | ~590 k | ~1,350 |
| `BenchmarkSubmit` (the same, plus the conditions and the required check, store completed) | 4.81 M, 5.61 M, 4.82 M | ~695 k | ~1,390 |

The attendance and the version are stubbed (they are other domains' reads, measured there: a 200-item version
read is 1.1 to 3.5 ms, package 1 and 2), and the audit is counted, not stored (auditevent's repository is not this
domain's to build in a test, as in package 1). The full path is measured by package 6's `make
bench-questionnaire-http`. Nothing is changed on these numbers.

Readings:
- The patient needs the version's items to have their answers checked, and package 1 refuses patients every
  questionnaire read. `questionnaire.Service.Referenced` reads it in the patient's scope instead, relying on
  package 3's row-level security for the reference and on the caller, which has checked that the patient's own
  response references the version; package 6 can build the patient's FHIR read on it. It is audited like any
  read of a `Questionnaire` (SPEC §2 "Audit"), so a save writes two events.
- SPEC §2 lists 409 `submitted`, `withdrawn`, `past_due`, `consent_required` and 412 without an order; the checks
  run status, due time, consent, version, answers, so a closed response says why rather than 412.
- An answer must hold at least one value: the app clears a field by leaving its answer out of the list. A blank
  `string` or `text` answer is refused rather than stored.
- `decimal` is refused with an exponent: FHIR allows one, but the app never sends it and an unbounded exponent is
  not storable as a Postgres `numeric`.
- The integer range is held by `Value.Integer` (`int32`): a value outside it fails to decode (package 6's 400),
  so the service never sees one; a test pins the decoding.
- `required` is not checked on save (SPEC §2); submit re-checks every saved answer, so the rules hold even for
  answers saved before a rule tightened.
- The answers are kept in the order the app sent them; the FHIR handler nests them by the version (package 6).
- "Right after a revocation commits" is tested with the service fakes: `appointment.Attendance` reads the consent
  state in the caller's transaction (not cached), and a real-database test across `consent`, `appointment` and
  this domain belongs to the composition root (package 6's `internal/api` tests), since archcheck refuses another
  domain's repository in this package's tests.

Deferred: the handlers, the FHIR mapping and the wiring in `app.go` (package 6); `RequestDefault`, `Discard` and the
`Lookup` port (package 5).
