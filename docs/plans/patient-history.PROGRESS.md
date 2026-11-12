# Patient history: progress

Plan: docs/plans/patient-history.md. ADR 0014.

## Step 0: plan, ADR, contract (2026-11-12)

Branch `history-base`. Migration 00043 (four tables, `history_shared`, the shared and profile policies), the access
rules `HistoryVisit`, `ProfileEntry`, `Attributed`, the history consent's model, state and FHIR mapping, the
allergy and medication models and FHIR mapping, the history package's types, cursor and shared ports with fakes,
the OpenAPI paths and schemas of all four tickets with the generated clients and their contract tests, ownership,
archcheck contexts and retention rules. No user-facing string.

## SAL-40: the practitioner's view and the pre-visit summary (2026-11-12)

Branch `sal-40-doctor-history`. `history.Service.ForPractitioner` and `Summary` (`service_practitioner.go`), on
`encounter.Service.History` (the visits `access.HistoryVisit` lets the practitioner see, gated on `access.Profile` at
Full) and one read per owning domain for those visits (`composition.Service.History`,
`questionnaireresponse.Service.History`), each domain auditing the items a page shows (`Disclosed`) and the history
auditing the view itself (`Patient/<id>`, TREAT). A shared visit's signed documents, completed responses, their files
and the questions they answer open for that practitioner through `encounter.Service.Shared`/`SharedAppointment`; the
questions through migration 00044's `history_questionnaire` (`SECURITY DEFINER`, owned by `salus_history_lookup`),
not a chained policy. `migrations/language_test.go` runs in its usual time with it (about 17 s per test, parallel,
as before). Diagnoses: the ICD-10 codes of the signed documents' coded fields (`HistoryItem.diagnoses`, added to
the schema and the Go type), for own and shared visits alike: `condition.Service` has no read of a visit's
Conditions and the plan says not to change its code. `GET /patients/{id}/history`, `.../history/summary`. Web: the
history panel on the appointment detail (its own practitioner) and as a tab of the call page's live panels, read only
once opened. Allergies and medications through the ports, stood in by `noAllergies`/`noMedications` in
`internal/api/history.go` until SAL-41's services are wired at the merge.

Readings (`make bench-history`, compose Postgres on the shared dev machine, other stacks running): a patient with
300 visits with the practitioner and 100 shared from another clinic, each with a signed document, and 500 more
visits there not shared (read by the clinic's scope through `history_shared`, left out by the rule).

| Read | p50 | p95 | p99 |
| --- | --- | --- | --- |
| First page of 30, documents and responses read per visit (first cut) | 63.7 ms | 116.6 ms | 170.6 ms |
| First page of 30, one query for every visit's documents, one search per patient for the responses | 19.8 ms | 55.8 ms | 93.9 ms |
| Pre-visit summary | 16.7 ms | 46.9 ms | 94.8 ms |

Every page of 100 to the last (9 pages, 804 items): the last 28 ms, the slowest 43 ms. The cost was the per-visit
queries, not `history_shared`: the partial index on `history_consent_encounter (encounter_id)` is enough for these
reads; nothing more was added.
