# Questionnaires progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 questionnaire | done, 2026-11-06 | `questionnaire(1)` |
| 2 questionnaire API | done, 2026-11-06 | `questionnaire(2)` |
| 3 questionnaireresponse and the access rule | not started | |
| 4 Answers | not started | |
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
