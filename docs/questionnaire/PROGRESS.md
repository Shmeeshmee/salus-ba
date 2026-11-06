# Questionnaires progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 questionnaire | done, 2026-11-06 | `questionnaire(1)` |
| 2 questionnaire API | done, 2026-11-06 | `questionnaire(2)` |
| 3 questionnaireresponse and the access rule | done, 2026-11-06 | `questionnaire(3)` |
| 4 Answers | done, 2026-11-06 | `questionnaire(4)` |
| 5 Booking and cancellation | done, 2026-11-06 | `questionnaire(5)` |
| 6 Responses API | done, 2026-11-06 | `questionnaire(6)` |
| 7 E-mail mention | done, 2026-11-06 | `questionnaire(7)` |
| 8 Web: library and builder | done, 2026-11-06 | `questionnaire(8)` |
| 9 Web: appointment and working hours | done, 2026-11-06 | `questionnaire(9)` |
| 10 Mobile | done, 2026-11-06 | `questionnaire(10)` |
| 11 End-to-end | not started | |

## Decisions

- 2026-11-06, owner (recorded in `SPEC.md` §2 and §3): D1 uploads in a separate feature, "questionnaire
  attachments", right after this one, with `attachment` reserved in the model; D7 `@dnd-kit`, pinned, plus
  Move up and Move down; D2 to D6 and D8 as recommended; D9 the appointment's practitioner keeps reading the
  submitted answers of past appointments after a revocation, nobody else. `COPY.md` approved with W11, W35 and
  M13 changed.
- 2026-11-06, plan: `@dnd-kit` speaks to screen readers during a drag, and the approved text has nothing for
  it; W41 and W42 are proposed in `COPY.md`, and package 8 is held until the owner approves them.
- 2026-11-06, owner: W41 and W42 approved as written; the hold on package 8 is lifted.

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

## Package 5: Booking and cancellation (done, 2026-11-06)

Built:
- `internal/appointment`: the `Jobs` port gains `EnqueueQuestionnaireRequest`, enqueued by `Book` in the booking
  transaction, and `EnqueueQuestionnaireDiscard`, enqueued in the shared `cancelled` step, so the patient's, staff's
  and the revocation job's cancellations all enqueue it. Job args in `model.go`: `QuestionnaireRequestArgs`
  (`questionnaire_default_request`) and `QuestionnaireDiscardArgs` (`questionnaire_responses_discard`), the
  appointment's and the clinic's ids only, on River's default queue (the worker's), as `consultation_access_end`.
- `internal/questionnaire`: `Service.ActiveDefault(ctx, agent, org, roleID)`, the active version of a role's
  default for a job, in the clinic's scope; `ErrNotFound` without a default, `ErrNotActive` without an active
  version. The read of the default (`PractitionerRole/<id>`) and of the version are audited with the job as agent
  (`HOPERAT`).
- `internal/questionnaireresponse`: `service_jobs.go` with `RequestDefault` and `Discard` (SPEC §2 "Requests",
  "Cancellation and revocation"); the `Appointments` port gains `Lookup`, the `Questionnaires` port
  `ActiveDefault`; the repository gains `SetNotice` and `DeleteOfAppointment` (`DELETE … RETURNING id`, bound by
  package 3's `questionnaire_response_discard` policy). Each request is audited as a create and each deleted
  response as a delete, with the job as agent (`HOPERAT`). `make sqlc` regenerated.
- `internal/worker/questionnaire.go`: the `questionnaire_default_request` and `questionnaire_responses_discard`
  workers on the default queue and their service; registered in `addDefaultWorkers`, so the scheduler knows the
  kinds. They log the appointment's id only.
- `docs/privacy/dpia.md`: the package 5 subsection.

Failing first (red), each for the right reason before the code:
- `internal/appointment`: `jobs.EnqueueQuestionnaireRequest undefined`, `jobs.EnqueueQuestionnaireDiscard
  undefined`; with the port and the River implementation in place, the booking and cancellation tests failed on
  the missing enqueues (`TestBook_*`, `TestCancel_*`, `TestCancelForRevokedConsent_*`).
- `internal/questionnaire`: `f.svc.ActiveDefault undefined`; then one case failed on a test bug (the "without a
  default" case had the shared default set), fixed in the test.
- `internal/questionnaireresponse`: `r.SetNotice undefined`, `r.DeleteOfAppointment undefined`,
  `f.svc.RequestDefault undefined`, `f.svc.Discard undefined`; with stubs returning nil, every new service test
  failed on behaviour. The repository test first failed on its own SQL (two statements in one prepared
  statement), fixed in the test.
- `internal/worker`: `TestScheduler_KnowsEveryKindTheWorkerAndTheNotifierWork` and both job tests, `job kind is
  not registered in the client's Workers bundle`.

Tests: booking enqueues the request in its transaction with the two ids (fake jobs, the rollback cases of
`TestBook_FailsOnABackendError…`, and River's table: a rolled-back transaction leaves no job); each cancellation
path (patient, practitioner, admin, revocation) enqueues the discard. `ActiveDefault` with a default, without one,
the series retired, a draft only, another clinic, a backend failure, and its audit. `RequestDefault` with a
default (one row, in progress, due at the start, source `default`, audited once), twice (one row), already
requested by the practitioner (no second row), without a default, with the series retired, a draft only, a
default of another clinic, a cancelled, started, unknown or other clinic's appointment, a consent revoked since the
booking, and backend failures (rolled back, error without the id). `Discard` deletes the in-progress, completed
and withdrawn responses of a cancelled appointment and keeps another appointment's, once (a repeat changes
nothing), in the clinic's scope with the notice; does nothing for a booked, unknown or other clinic's appointment;
after a revocation deletes the future (cancelled) appointment's and keeps the past one's completed response.
Repository, as `salus_app`: the delete removes both responses of the cancelled appointment, also after a
revocation, and nothing of a booked appointment, without the notice, with another appointment's notice, or for an
id that is not a UUID. Worker, on the real database: the request job twice gives one row and one audit event, and
nothing for the started appointment; the revocation job cancels the future appointment and enqueues its discard
(the two ids on the default queue); the discard job (twice, then for the past appointment) leaves the past
appointment's completed response only, with one audit event; no log line holds a name, the e-mail, the title or a
question.

Checks: `make lint` 0 issues, migrations unchanged; `make test` pass; `make archcheck` pass; `make cover` pass,
`internal/appointment` 92.9%, `internal/questionnaire` 95.4%, `internal/questionnaireresponse` 95.0%,
`internal/worker` 93.7%, total 94.8%; `make sqlc` regenerated.

Readings:
- The two jobs run on the default queue (the worker's), as `consultation_access_end`: they write the database
  and send nothing.
- `RequestDefault` checks the appointment through `appointment.Lookup` in the clinic's scope, which shows a
  future appointment only while the patient's consent is active: after a revocation it finds nothing and does
  nothing, rather than failing on the insert policy and being retried. A booked appointment that has started by
  the time the job runs gets no request (it could never be answered).
- A repeated request writes no second row and no second create event; the questionnaire's service still audits
  the job's read of the default on every run, as every read of a role's default is audited.
- `ActiveDefault` audits the job's read as one event per entity disclosed: the role's default always, the version
  when there is an active one. A job has no account, so the agent is its kind, purpose `HOPERAT`.
- `Discard` sets the clinic and the appointment's notice (`app.appointment_id`), so the appointment's own notice
  policy shows the cancelled appointment after a revocation, and checks it is cancelled before deleting; the
  delete policy checks it again. Each deleted response is one delete event; nothing deleted, nothing audited.
- "A default applies to bookings made after it is set" (SPEC §2 "Requests") is met by the job running right after
  the booking; a default set in the seconds between the booking and the job applies to that booking too. Not
  guarded: a guard would need the default's history.

Deferred: the handlers, the FHIR mapping, the wiring of the service in `app.go` (package 6); `Outstanding` and the
e-mail sentence (package 7).

Dev stack: no migration in this package; the running `api`, `worker` and `scheduler` containers still run the
code before this package until they are rebuilt (`make up`).

## Package 6: Responses API (done, 2026-11-06)

Built:
- `internal/questionnaireresponse/handler_http.go`: `POST /appointments/{id}/questionnaires` (201), `PUT
  /questionnaire-responses/{id}/answers` (`If-Match`), `POST /questionnaire-responses/{id}/submit` (`If-Match`),
  `/withdraw`. Bodies read up to 256 KiB, known fields only, one object; anything else (an integer outside 32 bits
  included) 400 `invalid_request` before the service is called. Errors: 403 `forbidden`, 404 `not_found`, 409
  `already_requested`, `not_active`, `appointment_started`, `appointment_closed`, `consent_required`,
  `submitted`, `withdrawn`, `past_due`, 412 `version_conflict`, 422 `answers_invalid` with `linkIds`. The answer
  is the `QuestionnaireResponse` schema with its `versionId` as the ETag, `Cache-Control: no-store`.
- `internal/questionnaireresponse/handler_fhir.go`: `QuestionnaireResponse` read and search by `appointment`,
  `patient`, `status` (each at most once) with `_include=QuestionnaireResponse:questionnaire`; anything else 400
  `not-supported`. `questionnaire` is `<url>|<version>`, `subject` and `author` the patient, `authored` once
  submitted, the `questionnaireresponse-appointment` extension, and for a `Full` view only `item`: the answers
  nested under the version's sections in document order (sections without an answer, instructions and
  unanswered questions left out). Each version the responses answer is read once per request through
  `questionnaire.Service.Read` as the caller (audited), and included once.
- `internal/questionnaire`: `Caller.PatientID`; `Read` and `Search` give a patient with a profile the versions
  their own responses reference, in the patient scope (`Referenced`; `searchReferenced`), audited `PATRQT`; a
  patient without a profile is still `ErrForbidden`. `ToFHIR` and `CanonicalURL` exported for the response
  handler (as `practitioner.ToFHIR`).
- `internal/api/app.go`: the response service and both handlers on `appointmentCaller`; `questionnaireCaller`
  reads a patient's profile id through `patient.Service.Me`, as `appointmentCaller` does.
- `api/fhir/`: `StructureDefinition-questionnaireresponse-appointment.json` (context `QuestionnaireResponse`,
  required `valueReference` to `Appointment`), `SearchParameter-questionnaireresponse-appointment.json`
  (`appointment`, reference, target `Appointment`); `CapabilityStatement.json` gains `QuestionnaireResponse`
  (`read`, `search-type`, the three parameters, `searchInclude`) and the patient's access in the `Questionnaire`
  documentation; `embed_test.go` and the router's `metadata.json` golden follow.
- `api/openapi.yaml`: the four operations, `RequestQuestionnaireRequest`, `SaveAnswersRequest`, `Answer`,
  `AnswerValue`, `QuestionnaireResponse`, the six new error codes and `linkIds` for `answers_invalid`; `make
  gen-clients`.
- Golden files (`internal/questionnaireresponse/testdata/`): `response_in_progress.json`,
  `response_completed.json`, `response_name.json`, `search_include.json` (two responses of one version, the
  version included once), `not_found.json`, `unsupported_search.json`, `unauthenticated.json`. Synthetic,
  placeholder wording.
- `test/e2e/questionnaire_latency_test.go` and `make bench-questionnaire-http`.
- `docs/privacy/dpia.md`: the package 6 subsection.

Failing first (red), each for the right reason before the code:
- `internal/questionnaire`: `unknown field PatientID in struct literal of type questionnaire.Caller`.
- `internal/questionnaireresponse`: `undefined: questionnaireresponse.NewHTTPHandler`, then `undefined:
  questionnaireresponse.NewHandler`; the FHIR tests then needed the golden files, written with `-update` and
  reviewed.
- `api/fhir`: `QuestionnaireResponse = ""` and the missing `SearchParameter-questionnaireresponse-appointment.json`.
- `internal/api/router`: `metadata.json` differed (rewritten with `-update`, diff reviewed: the new resource and two
  documentation sentences).
- `internal/api`: `TestNew_ServesQuestionnaireResponsesUnderTheAccessRuleAndAuditsIt`, `404 page not found` before
  the wiring. Then package 2's `TestNew_ServesTheQuestionnaireLibraryToStaffOfTheClinicAndAuditsIt` counted 10
  refusals without a clinic instead of 5: each of the patient's 5 calls is now preceded by the caller's audited
  lookup of their profile (they have none), as for appointments. The expected count is 10, with that reason in
  the test.

Tests: every route's success and each error code through `httptest`; 401 and 400 without a call; the 256 KiB limit
at and one byte past it; the log holds nothing on success and refusals and only the error on a 500, never an
answer. FHIR: the golden files, the version read once per request as the right caller (patient or staff), a
`Name` view without `item`, `_include` bringing each version once, each parameter and each unreadable one 400
without a call. Through the app on the real database as `salus_app` (`newMatrix`): requests and their refusals
(`already_requested`, `not_active` for a draft, `appointment_started`, 404 for another practitioner, the admin, the
patient and another clinic, 401), the patient's save, 412, 422 with `linkIds`, 400 for an integer outside 32 bits,
submit and its 422, `submitted` after it, staff's save 403 or 404; the withdrawal and its refusals; the FHIR read
and search matrix with the consent active (patient `Full`; the practitioner `Name` before the submit and `Full`
after; the admin `Name`; another practitioner and another clinic 404 and an empty search), the patient's
`Questionnaire` read and search of the versions their responses reference; after a revocation the practitioner
`Full` on a completed response of the past appointment (D9), the admin `Name`, another practitioner 404, the
future appointment's responses 404 for the clinic, the patient's save `consent_required`; the audit per request,
save, submit, read and returned resource, refusals included; the log holds no title, question or answer.

Checks: `make lint` 0 issues, migrations unchanged; `make test` pass; `make archcheck` pass; `make cover` pass,
`internal/questionnaireresponse` 96.1%, `internal/questionnaire` 94.9%, `api/fhir` 100%, total 94.9%; `make
fhir-validate` 0 errors (the dom-6 narrative warning every resource has; "the questionnaire could not be resolved"
on the `salus.invalid` placeholder canonical, a warning and an information per response; the draft item-control
code system information); `make gen-clients` run twice, no diff the second time; `make web-typecheck` and `make
mobile-analyze` pass (not in this package's check sets, run because the clients changed).

Measurements (`make bench-questionnaire-http`: the api container on the compose Postgres, i7-14700HX; one
synthetic clinic, one patient, 10 appointments, one 200-item questionnaire (10 sections of 19 questions, every
answer type) requested on each; a save's body 17,708 bytes, 190 answers; each of 10 workers saves its own
response 100 times, If-Match chained; then all submitted, and the practitioner runs 1,000
`QuestionnaireResponse?appointment=…&_include=QuestionnaireResponse:questionnaire` round the 10 appointments, each
answer one completed response with 190 nested answers and the 200-item version; 10 at a time):

| Run | Saves p50 / p95 / p99 (max), throughput | Searches p50 / p95 / p99 (max), throughput |
| --- | --- | --- |
| 1 | 9.63 / 17.95 / 24.03 ms (45.7), 928/s | 9.09 / 13.52 / 14.68 ms (16.2), 1,039/s |
| 2 | 10.16 / 15.01 / 23.29 ms (40.1), 923/s | 21.8 / 28.63 / 35.97 ms (57.6), 430/s |
| 3 | 10.38 / 15.1 / 19.48 ms (45.2), 915/s | 22.33 / 29.14 / 34.88 ms (47.1), 421/s |
| 4 | 9.49 / 16.27 / 22.32 ms (45.0), 949/s | 9.8 / 13.04 / 16.3 ms (18.8), 1,016/s |

Saves are steady at about 10 ms p50. Searches were bimodal: runs 2 and 3 took twice as long as runs 1 and 4 on
the same data (run 4 came after them, so it is not the tables growing; the dev database held about 25,000 audit
events). Probably other load on the machine or Postgres's autovacuum between runs; not investigated further.
Every run is recorded and nothing is changed on these numbers. A search costs a transaction for the responses
(one audit event each) and one for the version (one audit event), plus the session check and the caller's
lookups.

Readings:
- The FHIR handler reads each version through `questionnaire.Service.Read` (the port `Versions`), as
  `practitionerrole`'s handler reads practitioners: the canonical `<url>|<version>` needs the version number and
  the nesting needs its sections, and `questionnaire_response` may not read the `questionnaire` table. So every
  FHIR read or search of a response also writes one audited read per version, even without `_include` (the
  version is used, only its url is shown). For a `Name` view too, since `questionnaire` names the version.
- The patient's `Questionnaire` read goes through `Read` (delegating to `Referenced`); the patient's search runs in
  the patient scope, where row-level security shows only the versions their responses reference. A patient
  without a profile keeps `ErrForbidden` (403 on the search, 404 on the read), as in package 2.
- `questionnaireCaller` now reads the patient's profile (`patient.Service.Me`, audited) for every patient call,
  as `appointmentCaller` does; staff callers are unchanged.
- The JSON answer's `answers` follow the timestamps: `govet` fieldalignment ordered the struct, and JSON order
  carries no meaning. A `Name` view never reaches the JSON API: its four calls return the caller's own response
  (patient) or one without answers (request, withdraw).
- `QuestionnaireResponse.item` carries only `linkId` and `answer`, not the question text: the SPEC lists `item` as
  the answers, and the text is in the included version. Answers to a choice are `valueString`, as the options
  are (R4 allows `string` in `answerOption`).
- The request's 403 is CSRF only: anyone but the appointment's practitioner gets 404 (package 3). Withdraw gives
  403 to the patient and the admin, who see the request.
- Not in the CapabilityStatement's interactions: nothing beyond read and search-type, as the SPEC says.

Deferred: `Outstanding` and the e-mail sentence (package 7); the web screens (8, 9) and mobile (10).

Dev stack: no migration. `make bench-questionnaire-http` ran `make up`, so the api, worker, scheduler, notifier and
agent containers were rebuilt with this package and left running, as found. The four load-test runs left their
synthetic clinics, patients, appointments, one questionnaire each, 10 responses each and their audit events on the
dev database (synthetic data only, as `bench-join-http` does).

## Package 7: E-mail mention (done, 2026-11-06)

Built:
- `questionnaireresponse.Service.Outstanding(ctx, agent, org, appointmentID)` (`service_notice.go`): in one
  transaction in the clinic's scope with the appointment's notice, the appointment's responses (any of another
  clinic dropped, not on row-level security alone); outstanding when one is in progress, or else when the role's
  active default (`ActiveDefault`, with the job as agent) is of a series none of them is of. An appointment the
  scope does not show, or of another clinic, has nothing outstanding. Audited with the e-mail job as agent
  (`HOPERAT`): a `search-type` event per response read, one without an entity when there are none.
  `recordJob` takes an empty id for that event.
- `notification`: the port `Questionnaires` (that one method); the booking confirmation and the reminder ask it
  after the clinic read and before the account read, and add COPY E1 as their second paragraph when it says yes;
  the cancellation never asks. A failed check sends nothing (the job retries). Golden files
  `booked_questionnaire.txt` and `reminder_questionnaire.txt`; `booked.txt` and `reminder.txt` unchanged.
- `internal/worker`: the notifier gets the questionnaireresponse service from `newQuestionnaireResponses`.
- `docs/privacy/processors.md`: the SES entry gains the E1 sentence. `docs/privacy/dpia.md`: "Package 7: e-mail
  mention".

Test first (failures seen before the code):
- `questionnaireresponse`: `f.svc.Outstanding undefined` (the three new tests).
- `notification`: `unknown field Questionnaires in struct literal of type notification.Ports`.
- `worker`: `TestNotifier_MentionsAnOutstandingQuestionnaireWithoutItsTitle` and the existing
  `TestNotifier_SendsTheAppointmentEmailsToThePatient` left the job `available` (the unwired port). Once wired,
  the latter counted 8 reads by the e-mail jobs instead of 6: each now also reads the role's default (audited by
  the questionnaire's service, `PractitionerRole/<id>`). The expected count is 8 with that reason, and a second
  assertion counts the 2 default reads.

Tests: `Outstanding` table-driven (nothing, in progress, submitted, withdrawn, the default not yet requested, the
default submitted, the default retired, a default of another clinic, another appointment's response), each with
the scope and the audit events; an unknown appointment and another clinic's; each backend failure rolls back the
audit. The e-mails with and without a questionnaire outstanding (goldens, E1 the second paragraph, the job as
agent), the cancellation not asking, a failed check sending nothing, a reminder that no longer applies not
asking. On the real database through the notifier as `salus_app`: the confirmation before the default's request
holds E1, the reminder with the default in progress holds it, with it submitted does not, the confirmation without
a default does not; no e-mail holds the synthetic title, the question or the version's id; 4 `search-type` audit
events by the jobs.

Checks: `make lint` 0 issues, migrations unchanged; `make test` pass; `make archcheck` pass; `make cover` pass,
`internal/notification` 91.6%, `internal/questionnaireresponse` 95.6%, `internal/worker` 93.7%, total 94.8%.

Readings:
- "A role default not yet requested" is read as: the role has an active default and the appointment has no
  response of its series, in any status. A submitted or withdrawn response of that series (default or added)
  counts as requested, so a withdrawn default is not mentioned again.
- The check is audited as a search of the appointment's responses by the job, as package 6 audits a search, and
  reads no answer. The appointment's `Lookup` is unaudited, as in the jobs of package 5; the appointment is read
  and audited by the e-mail's `Notice` already.
- The check comes before the account's e-mail is read, so a failed check reads no address.

Deferred: nothing of this package. The web screens (8, 9), mobile (10), end to end (11).

Dev stack: no migration, nothing written to the dev database (the tests use their own databases and the compose
Mailpit, whose test messages they delete). The running notifier container still has package 6's build; `make
up` rebuilds it.


## Package 8: Web: library and builder (done, 2026-11-06)

Built:
- `web/package.json`: `@dnd-kit/core` 6.3.1 and `@dnd-kit/sortable` 10.0.0, exact (the current releases,
  `npm view` on 2026-11-06). Runtime dependencies checked: `core` needs `tslib` ^2.0.0, `@dnd-kit/accessibility`
  ^3.1.1 and `@dnd-kit/utilities` ^3.2.2; `sortable` needs `@dnd-kit/utilities` ^3.2.2 and `tslib`; peers
  `react`/`react-dom` >=16.8. Installed: `@dnd-kit/accessibility` 3.1.1, `@dnd-kit/utilities` 3.2.2 (both MIT,
  only `tslib` below them), `tslib` 2.8.1 was already in the tree. Four packages added to the lockfile, no
  advisory. `@dnd-kit/utilities` is not a direct import (the transform string is written by hand).
- `web/.oxlintrc.json`: `no-restricted-imports` for `@dnd-kit/*`, off only for `src/builder-dnd.tsx`. Checked
  with a throwaway file importing `@dnd-kit/sortable`: 1 error; deleted.
- `web/src/messages.ts`: `questionnaires`, W1 to W29, W41 and W42 exactly as `COPY.md` (a script compared all
  104 quoted strings, placeholders matched against the template parameters: none missing); "Yes" and "No" of W38
  for a condition's answer.
- `builder.ts`: the item list as pure functions: `rowsOf` (document order, an end row per section; the top level
  only while a section moves), `move` (as a sortable list: down lands after the target, up before it), `insert`,
  `step` and `canStep` (Move up and Move down, one row at a time, so an item steps into and out of sections),
  `remove` (and the conditions on it), `dependents`, `changeKind` (a new `linkId`, dependents pointed at it),
  `placement`, `at` and `cancelled` (the W42 sentences), `earlierQuestions`, `problems` (W27 and W28 per item,
  checked as the API checks). A section never lands inside another: `insert` refuses it (`nested-section`), and a
  moved section only sees the top level.
- `builder-dnd.tsx` (the only `@dnd-kit` import): `DragArea` (pointer sensor for mouse and pen, touch sensor with
  a 250 ms delay, keyboard sensor with `sortableKeyboardCoordinates`; instructions W41 and announcements W42 for
  start, over, end and cancel; `@dnd-kit`'s `aria-roledescription` dropped, so none of its English is left),
  `SortableList`, `SortableRow`, `PaletteSource` (pointer and touch only; the keyboard adds by the button),
  `ListEnd` (the drop area after the last row, W12 while empty).
- `questionnaire.ts`: FHIR `Questionnaire` and the JSON API's answer to the builder's model (help text from the
  help item control, single and multiple choice apart), the save body, and `libraryOf`.
- `Questionnaires.tsx` (`/questionnaires`): W1 to W8, the status filter, the version and last change in a
  tooltip on hover and focus, "New questionnaire" creating an untitled draft and opening it.
- `Builder.tsx` (`/questionnaires/<id>`) and `ItemCard.tsx`: palette, title (W14 under it), description, the
  cards closed by default (text or type, type, Required, handle, Move up, Move down, Delete), opened to edit
  (type, text, Required, options, "More options" with help text and the condition); "Save draft" (`If-Match`,
  CSRF), "Publish" with W22 (and the retire line when the series has an active version; unsaved changes saved
  first), "Discard draft" with W21, an active version read only with W23, "Edit as new version" and "Retire"
  with W24, a retired one with W25; W29 per error (412 with "Reload", `draft_exists` with "Open draft",
  `not_draft`, 422 marking the named items, not found); limits before they are reached (item count always,
  palette off at 200, character counts from 80% of a limit with `maxLength`, "Add option" off at 50); W26 on
  leaving by a link or the browser's back, `beforeunload` for closing or reloading; Move up and Move down
  announced with W42 in a polite live region; an added item opens with the focus in its text.
- `route.ts`: `guardLeaving`; `navigate` and `usePath` ask the guard. `api/result.ts`: a failed outcome carries
  `linkIds`. `App.tsx`: the "Questionnaires" link and both routes (the builder wide). `vite.config.ts`: the dev
  server proxies `/questionnaires` writes to the API and serves the app on `GET`. `api/fhir.ts`:
  `Questionnaire`. `icons.tsx`, `app.css`.

Test first (failures seen before the code):
- `builder.test.ts`: `Failed to resolve import "./builder"`. `Questionnaires.test.tsx` and `Builder.test.tsx`
  likewise. `questionnaire.test.ts` was written before `questionnaire.ts`, but its first run came after the
  module, so no red run of it was recorded.
- `Builder.test.tsx` first run: 3 of 29 failed, each for its reason: the card helper also caught "More options"
  (the test), the start-of-drag "over" replaced "Moving …" (fixed: the place a drag starts over is not
  announced), the leave prompt said "Cancel" instead of W26's "Stay". The keyboard drag then needed the sensor's
  own next-task listener (`sensorReady`, a wait for that task, not a sleep) and W41's portal (waited for as a
  condition).
- `App.test.tsx`: the nav link and the builder's path, 2 failed before the wiring.
- The focus of an added item: the assertion failed (focus on `body`) before `autoFocus`.

Found in the browser walk (each reproduced in the walk first; jsdom cannot show the first two):
- A palette type dragged into the list was added twice and the drop lost: the dragged button followed the pointer,
  so the release was also a click on it, whose "add at the end" overwrote the drop. Fixed: the palette is dragged
  as a copy (`DragOverlay`, mounted only for palette drags, as a mounted overlay broke keyboard drags of items).
- The browser's back left the builder without W26: the browser renders between `popstate` listeners, so the
  screen was gone before a separate guard ran, also as a capturing listener. Fixed: `usePath`'s own listener
  asks the guard first. A jsdom test of the back path was added.
- A keyboard move of an open (tall) card never landed: by centre or corners it stays nearest its own place.
  Fixed: `closestTop`, the row whose top is nearest (the keyboard lines tops up).
- Two runs early on lost one typed character right after a drop; with the field checked as focused first,
  never in the six later runs. Probably Vite's first-run dependency optimisation reloading the page after the new
  packages; not proven.

Tests (Vitest, 311 in all, 99 new): `builder.ts` moves (into, within and out of sections, sections as blocks),
inserts (each target, a section inside a section refused, an empty section), every Move up and Move down step and
the ends, placement, the W42 sentences, delete with dependents, the type change's new `linkId`, each problem;
the mapping both ways; the library (rows, tooltip, each filter, both empty texts, create, failures); the builder
(each card state, the palette, the type change saved with a new `linkId`, Move up and Move down with W42, the
handle's W41 and no role description, a keyboard drag space–arrow–space with W42, escape with W42, delete with and
without W19, save with `If-Match` chained and CSRF, 412, 409 `not_draft`, 422 with `linkIds` on the right items,
the three limits, publish's confirmation, the retire line, saving before publishing, revise, `draft_exists` with
"Open draft", retire, discard, not found, W26 on a link and on the back button, leaving when saved).

Checks: `make web-lint` 0 warnings, 0 errors; `make web-typecheck` passes; `make web-test` 20 files, 311 tests;
`make web-build` passes, with Vite's chunk-size warning (916 kB minified, 258 kB gzip; 835 kB before this package,
about 80 kB more with `@dnd-kit` and the screens); `make gen-clients` no diff (no `openapi.yaml` change). Not in
this package's check sets, run as CLAUDE.md asks before done: `make lint` 0 issues, migrations from `main`
unchanged; `make archcheck` passes; `make test` passes.

Browser walk (`make up` stack; Chromium 1.55 in the `playwright` service with the Vite dev server inside it, as
the consultation journey runs; a temporary e2e test made the synthetic clinic, practitioner and a 200-item draft
through the API and started a temporary spec; both deleted before the commit; 6 full passes in a row at the end):
- Desktop 1280 px, pointer: the nav link, the library, "New questionnaire", a type clicked, Section dragged to the
  end, Short answer dragged into the section's end, Move up into the section (W42 heard in the live region), a
  condition, a mouse drag moving the question below its dependent (W28 shown), back with Move up (gone), save,
  publish with W22, W23, the library with the tooltip (W5) and the Active filter.
- Keyboard only: open from the library, "Edit as new version", a type added with Enter and typed into, the handle
  reached by Tab with W41 as its description, space, arrow up, space (W42 at each step), space, arrow up, escape
  (W42's cancel), Move up, "Save draft", "Discard draft" and "Discard".
- 390 px with touch emulation (`hasTouch`, `isMobile`): library, new draft, three types tapped, a touch drag
  (touch start held past the 250 ms delay, moved, ended) reordering them, delete, discard with W21, the published
  version read only; no horizontal scroll on any of them.
- Unsaved changes: W26 on the Calendar link (Stay keeps the page), on the library link after the measured
  drags, on the browser's back (the URL restored, the changes kept, Leave leaves), and the browser's
  `beforeunload` prompt on closing the tab.

Measurements (SPEC §2 "Performance"; the walk above, i7-14700HX, Vite dev server, so React's development
build: upper bounds for production). The 200-item draft (10 sections of 19 short answers), first render from
navigation start to the 200th card in the page, which includes the session check and the two FHIR reads: 283,
305, 327; 302, 331, 307; 311, 309, 301; 300, 317, 336; 333, 321, 329 ms. One mouse drag of a question within its
section, from the release to the list in its new order (one CDP round trip and an animation-frame poll included):
208, 99, 96; 212, 97, 102; 219, 100, 100; 207, 98, 95; 210, 100, 99 ms (the first drag of each run about 210 ms).
Read by script from `performance.now()`, not the dev tools' profiler. Nothing is changed on these numbers.

Readings:
- The library lists, per questionnaire, its draft and its active version, or its latest retired one when it has
  neither (older retired versions are history, reached through the series). A questionnaire with both shows two
  rows, so neither status is hidden; the filter applies to the rows. One FHIR search, filtered in the browser.
- The builder reads the version and its series over FHIR (the JSON API has no read): the series gives the active
  version W22 names and the draft "Open draft" opens.
- W42 after a move: "[Item] is in [Section], at position…" when the item lands in a section, "[Item] moved to
  position…" when it lands on the top level; the same after a drop. Move up and Move down are disabled at the
  ends, so W42's cancel follows only a cancelled drag. A palette drop announces where the new item lands; a
  cancelled palette drag says nothing (it was nowhere).
- Positions are counted within the item's list: the top level, or its section.
- Deleting an item others depend on asks with W19 also for a section (counting the conditions on its items);
  without dependents it deletes at once.
- An option left empty counts as text missing (W28 "Add text to this item."): COPY.md has no line for it.
- A condition whose question is moved below stays and shows W28; it is removed only when the question is
  deleted (SPEC §2 "linkIds"). A type change points the conditions on the item at its new `linkId`; one that no
  longer fits shows W28.
- The type select has no label in COPY.md: it is named by the item's text (`aria-label`), no new text added
  (question below).
- The builder shows the status as a tag without the W5 tooltip (the clinic's time zone is not read there).
- Load failures other than not found use the shared "Something went wrong. Try again.", as elsewhere.

Deferred: the appointment detail and the role's default (package 9); mobile (10); the Playwright journey in `make
e2e` (11). Splitting the bundle is not measured as a problem and not done.

Questions for a person: a label for the item type select in the builder (proposal: "Type"), and whether an empty
option should have its own W28 line.

Dev stack: no migration. The walk runs left about 15 synthetic clinics on the dev database, each with an admin, a
practitioner, a role with working hours and free slots, a 200-item draft, and from the full runs a published
"Placeholder walk questionnaire" and their audit events (synthetic data only, as earlier walks). The running
`node` dev server (Vite) picks up the new config and packages from the mounted `web/` on its own; nothing else on
the stack changed. Screenshots of the walk are in the ignored `.claude/runs/questionnaire/walk-8/`.

## Package 9: Web: appointment and working hours (done, 2026-11-06)

Built:
- `web/src/messages.ts`: W30 to W40 exactly as `COPY.md` (a script compared the 31 quoted strings of those sections,
  placeholders against the template parameters: none missing); W38's "Yes" and "No" were already there (package 8).
- `responses.ts`: the requests as pure functions. `requestsOf` reads the `QuestionnaireResponse` search with the
  included versions (each response's title from its `url|version`); `answered` lists a submitted response's
  enabled questions by section, with its answers (an item is enabled when its section is and its condition's
  question is enabled and answered as the condition says, any answer of a choice with repeats; instructions and
  hidden items left out; an enabled question without an answer kept, with no values); `valueText` writes each
  answer type; `statusOf` is W34.
- `AppointmentQuestionnaires.tsx`, on `AppointmentDetail.tsx` after the consultation: W32, W33, each request with
  its title and W34; for the appointment's own practitioner (the account's practitioner is the appointment's
  `Practitioner` participant) "View answers" and "Hide answers" (W37) on a submitted one, the answers by section
  in a `dl` (W38, multiple choice as a list, long answers with their line breaks), "Withdraw" with W36 while in
  progress, and "Add questionnaire" with the picker and W35 while the appointment is booked and has not started;
  W40 per error. An admin sees the requests and their status only, W39 in a tooltip on the status (hover and
  focus).
- `Availability.tsx`: under the hours, the role's default (W30): "None" and the clinic's active questionnaires,
  saved as soon as one is picked (`PUT`, or `DELETE` for "None"), the picker disabled while it saves; W31 on a
  409 `not_active`, and also when the role's default has no active version any more.
- `ui.tsx`: `TipTag`, the library's status tooltip (package 8's `StatusTag`) made shared for its second use.
  `api/fhir.ts`: `QuestionnaireResponse`. `vite.config.ts`: the dev server proxies `POST
  /appointments/<id>/questionnaires` and `/questionnaire-responses`. `app.css`. `test/responses.ts`: synthetic
  responses and the search bundle.

Test first (failures seen before the code):
- `responses.test.ts`: `Failed to resolve import "./responses"`.
- `AppointmentQuestionnaires.test.tsx`: `Failed to resolve import "./AppointmentQuestionnaires"`. Its first run
  after the code: 1 of 27 failed, the test's own fixture (a Name view pointing at another version than the one
  included).
- `AppointmentDetail.test.tsx`: the three new tests, no `Questionnaires` region before the wiring.
- `Availability.test.tsx`: the nine new tests, no "Questionnaire for every booking" before the code.

Tests (Vitest, 372 in all, 61 new): `answered` by section, unanswered, a hidden section, a chain of two, a choice
with repeats; `valueText` per type; `requestsOf` pairing and an empty search; `statusOf` each status and the
start. The section: the search's parameters; each status (before and after the start, submitted in the clinic
zone, withdrawn); a failed load; the answers of every type by section, hidden items and instructions left out,
"Not answered", hide; no "View answers" before submit or once withdrawn; the admin's status only with W39 and a
Name view of a submitted response; add (only active questionnaires without a live request, a withdrawn one
offered again, the body, CSRF), cancel, each W40 code, not offered to an admin, another practitioner, after the
start or on a cancelled appointment; withdraw with W36 and CSRF, kept when not confirmed, W40 `submitted`, not
offered when submitted, withdrawn or to an admin. The detail: the section for the own practitioner, the admin and
another practitioner. The default: read among the actives (`status=active`), None, set with CSRF, clear, 409
`not_active` and a failure keep the old value, a default without an active version, an admin's choice of
practitioner, a failed load.

Checks: `make web-lint` 0 warnings, 0 errors; `make web-typecheck` passes; `make web-test` 22 files, 372 tests;
`make web-build` passes, with Vite's chunk-size warning (926 kB minified, 261 kB gzip; 916 kB after package 8);
`make gen-clients` no diff (no `openapi.yaml` change). Not in this package's check sets, run as CLAUDE.md asks
before done: `make lint` 0 issues, migrations unchanged; `make archcheck` passes; `make test` passes.

Browser walk (`make up` stack; Chromium 1.55 in the `playwright` service with the Vite dev server inside it, as in
package 8; a temporary e2e test made a synthetic clinic with its admin and practitioner, a patient with consent, a
future and a past appointment, three published questionnaires (one with every type, a section, instructions, a
condition and an optional question), two requests on the future appointment, one of them answered and submitted
by the patient, and one on the past appointment, then ran a temporary spec; both deleted before the commit; 3 of 3
passed on the final run):
- Desktop 1280 px and 390 px, the practitioner: the requests with W34 (submitted with its time, not submitted
  yet), the answers of every type by section with "Not answered" and the hidden question and instructions absent,
  hide; "Add questionnaire" offering only the one not requested, added, withdrawn with W36, offered again after a
  reload, the picker opened and cancelled by keyboard (Enter); the past appointment with W34's "Not submitted
  before the appointment" and no "Add questionnaire"; Working hours: the default set, kept after a reload, cleared
  with None. No horizontal scroll on any of them.
- The admin: statuses only, no button in the section, W39 shown when the status has the focus; Working hours: the
  practitioner's default changed with the arrow key on the focused picker (saved on change, kept after a reload),
  then cleared.
- The walk found that the dev server did not proxy the two new JSON endpoints (`vite.config.ts` above).

Readings:
- The requests come from one FHIR search, `QuestionnaireResponse?appointment=Appointment/<id>&_include=…`: the
  practitioner's view is `Full` once submitted and `Name` before (package 3), so the answers are in the same
  answer; the version included gives the title and the sections. An admin's view never has answers; the screen
  does not offer them by role, not by their absence.
- "Withdraw" is offered while the response is in progress, as the plan says, also once the appointment has
  started (the service allows it then).
- The picker leaves out the questionnaires the appointment already has an in-progress or submitted request of
  (409 `already_requested` otherwise); a withdrawn one is offered again. It reads the actives when opened, not
  with the page, so a plain view of the appointment audits no `Questionnaire` search.
- The default is saved on change, without a "Save" button or a success line: COPY.md has neither for it, and the
  picker showing the new value (and the error otherwise) is the feedback. A default whose questionnaire has no
  active version shows "None" with W31, the closest approved sentence; nothing is requested for it until another
  version is published or the default changes.
- A request whose version is missing from the search answer is titled W35's "Questionnaire".
- Dates of `date` answers are written as "15 January 2026" (as the patient's birth date), not in the clinic zone:
  they are calendar dates.

Deferred: mobile (10); the end-to-end journeys (11). Splitting the bundle is not measured as a problem and not done.

Questions for a person: none new. Package 8's (a label for the builder's type select; an empty option's W28 line)
are still open.

Dev stack: no migration. `make up` rebuilt and restarted the api, worker, scheduler, notifier and agent with the
branch as it was (no Go change in this package). The walk runs left 3 synthetic clinics on the dev database, each
with an admin, a practitioner, a patient, two appointments (one moved into the past), three published
questionnaires, their requests (one submitted with synthetic answers, one added and withdrawn by the walk), and
their audit events (synthetic data only, as earlier walks). Screenshots are in the ignored
`.claude/runs/questionnaire/walk-9/`.

## Package 10: Mobile (done, 2026-11-06)

Built:
- `mobile/lib/messages.dart`: M1 to M16 exactly as `COPY.md`, placeholders as parameters (`savedAsYouGo`,
  `upToCharacters`, `submitBody`, `submittedAt`), and `answersMessage` mapping the save and submit codes to M16.
  `test/copy_test.dart` checks every quoted M string of `docs/questionnaire/COPY.md` against the file (47 strings;
  checked to fail on a changed full stop).
- `questionnaire.dart`: the FHIR `Questionnaire` (items, sections, help from the `help` item control, options, one
  `enableWhen`; an unknown type such as the reserved `attachment` kept as `unsupported` and not shown) and the
  patient's `QuestionnaireResponse` (appointment from the extension, answers flattened out of their sections, each
  value in its item's type); `enabled`, `missing` and `shownAnswers`, the server's condition evaluation
  (`enabled.go`) in one function; `readTyped`, each typed answer checked against the server's limits with the M10
  sentence; `toApi` and `fromApi` for the JSON API.
- `fhir.dart`: `questionnaireResponses` (one search, `patient` with `_include=QuestionnaireResponse:questionnaire`,
  the withdrawn left out, the versions by canonical) and `questionnaireResponse` (a read, for a 412). `booking.dart`:
  `saveAnswers` and `submitAnswers` through the generated `QuestionnairesApi`, `If-Match: W/"<versionId>"`.
  `session.dart`: `ApiFailure.linkIds` from a 422's body.
- `appointments_screen.dart`: per questionnaire on the appointment's card, its title and M1 status, M3 and "Fill in"
  or "Continue" while it can be filled in, M4 once the appointment has started without it, "View answers" once
  submitted; nothing on a cancelled appointment; the list read again after the screen.
- `questionnaire_screen.dart` and `questionnaire_fields.dart`: one scrolling page (M5, sections as headings, help
  behind the M8 icon, "Required", each type's native widget: text fields, radio buttons, check boxes, the date
  picker with M9 and M10 as its texts), the saved state (M6) and the last refusal in a footer that stays in view,
  "Submit" with M11, M12 and the field errors, M13 and the read-only view (M14) after submitting, M15 on leaving
  with an answer not saved, M16 per code.

Save behaviour: a typed answer is checked at once and saved 2 s after typing stops (the timer restarts on each
change) or when the field loses focus; a choice, yes or no, or a date is saved at once. One save runs at a time;
changes made during it are saved after it. A lost connection or a 5xx shows M6's "Not saved…" and is tried again
after 5 s (and on the next change). A 412 reads the response again over FHIR and shows the latest answers with
M16; a 422 marks the questions it names; a 404 or 409 shows M16 and ends filling in (no more saves, no Submit).

Test first (failures seen before the code):
- `questionnaire_test.dart`: `Error: Couldn't resolve the package 'salus' in 'package:salus/questionnaire.dart'`.
- `fhir_test.dart`: `The method 'questionnaireResponses' isn't defined for the type 'Fhir'` (and
  `questionnaireResponse`).
- `questionnaire_screen_test.dart`: the same compile failure, then 2 of 24 failed on the first run, both the
  tests' own: one did not answer the saves (a 404 closed the screen), and the leave test relied on the field
  keeping its focus while the dialog opened (it loses it, and the save that follows made the second back press
  leave at once; rewritten with a failing save). After the lazy list (below), 10 failed until the tests scrolled to
  the rows they assert, which showed that the saved state and the messages scrolled out of view: they moved to the
  footer.
- `copy_test.dart`: the new test failed with a changed full stop in M9, restored.

Tests (Flutter, 208 in all, 72 new): the conditions against the 13 cases of package 4's
`TestSubmit_EvaluatesTheConditions` (missing and kept), a hidden section's items; each typed type at and one past its
limit, table-driven (27 cases); the FHIR mapping both ways and the JSON API's values; the two FHIR calls. Widgets
(`MockClient`): the card's states (to fill in, in progress, submitted, closed; withdrawn and cancelled not shown); the
page (M5, sections as headings, help behind the icon, Required); a condition showing and hiding; a choice saved at
once and typing saved 2 s after it stops, not per keystroke (none at 1.999 s), `If-Match` chained; a field left
saving at once; each typed check on its field and an invalid answer left out of the save; several choices and a date
from the picker; saved answers shown again; a failed save shown and tried again; 409 `consent_required`, `past_due`,
`withdrawn`, `submitted` and a 412 with their messages; a 422 on its question; M15 with Stay and Leave, and none when
saved; submit with a required question unanswered (no request), with M11 cancelled and confirmed, M13 and the
read-only view; a 422 and a 409 on submit; a submitted response read only with every type and "Not answered"; no
CSRF header; nothing printed (no title, question or answer in the log).

Checks: `make mobile-analyze` no issues; `make mobile-test` 208 tests pass; `make gen-clients` no diff (no
`openapi.yaml` change). Not in this package's check sets, run as CLAUDE.md asks before done: `make lint` 0 issues,
migrations unchanged; `make archcheck` passes; `make test` passes (no Go change).

Measurements (SPEC §2 "Performance", the first frame of the screen with 200 items: 10 sections of 19 questions of
every type, a third with help text; `flutter test` in the `flutter` container, i7-14700HX, a 390 × 844 point
surface; `pumpWidget` timed with a `Stopwatch`, six runs per process, three processes; a temporary test, not
committed). The test VM runs debug JIT code on a software renderer, so these are upper bounds and comparisons, not
device numbers:

| Layout | First run | Warm runs (5 per process, 3 processes) |
| --- | --- | --- |
| A `Column` in a `SingleChildScrollView` (as `ui.Page`) | 1,401 ms | 617 to 872 ms, settling at about 620 |
| A lazy `ListView.builder` (committed) | 415 to 423 ms | 44 to 56 ms |

The lazy list builds only the rows in view and is about 13 times faster warm, so the screen uses it (with
`ui.Page`'s look) instead of `ui.Page`. The first frame on a real device, read from the dev tools, is the manual step
(SPEC §8 "Before production").

Readings:
- The card's status: "To fill in" while in progress with no saved answer, "In progress" with one, "Submitted", and
  "Closed" once the appointment has started without a submit (the response's `due_at` is the start, and the FHIR
  resource carries the appointment, not `due_at`). A withdrawn request is not shown: the patient no longer needs to
  fill it in; a cancelled appointment shows none (its responses are deleted by the discard job).
- A typed answer that fails its check is shown on its field and left out of the save (it is not an answer), so a
  previously saved value of that question is replaced by none; Submit is refused while a shown field has an error.
- Decimals are sent as Dart numbers by the generated client, which writes an exponent below 0.000001; the server
  refuses an exponent, so the app refuses such a number with "Enter a number." A trailing zero (`1.50`) is sent as
  `1.5`. Only `.` is accepted as the decimal separator (localisation is out of scope, SPEC §6).
- M10's "This number is too large." is shown for a whole number outside 32 bits and a decimal with more than 15
  digits before the point; other decimal errors are "Enter a number."
- A 422 on a save shows the type's check on each named question (M10 for typed ones, M9's "Select one." or "Select
  all that apply." for choices); on submit, M12 and "This question is required.".
- The date picker gets M9 and M10 as its title and errors and "Cancel" (M11); its "OK" and its input-mode labels are
  Flutter's own (no approved text); the range is 1900 to 2100.
- The page is not `ui.Page` but a lazy list with its look (measured above); the saved state and the refusals sit in a
  footer so they stay in view on a long questionnaire. The read-only view is in the same list.
- Help text opens under its item when the icon is tapped (no dialog).
- Instructions are not repeated in the read-only view (as the web app's answers view).

Deferred: the end-to-end journeys (11). The 200-item first frame on a real Android device and an iOS build, and the
screen with a screen reader (SPEC §8).

Questions for a person: none new. Package 8's are still open. For the localisation before production: whether
decimals typed with a comma should be accepted.

Dev stack: no migration, nothing written to the dev database (the Flutter tests use `MockClient`; the Go tests their
own databases). The stack was found running and left running; the api, worker, scheduler, notifier and agent show a
restart during the session that no target of this package makes (none runs `up`), with the branch's code (no Go
change in this package).

