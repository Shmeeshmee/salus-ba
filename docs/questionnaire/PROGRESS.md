# Questionnaires progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 questionnaire | done, 2026-11-06 | `questionnaire(1)` |
| 2 questionnaire API | not started | |
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
