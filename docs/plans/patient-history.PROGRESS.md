# Patient history: progress

Plan: docs/plans/patient-history.md. ADR 0014.

## Step 0: plan, ADR, contract (2026-11-12)

Branch `history-base`. Migration 00043 (four tables, `history_shared`, the shared and profile policies), the access
rules `HistoryVisit`, `ProfileEntry`, `Attributed`, the history consent's model, state and FHIR mapping, the
allergy and medication models and FHIR mapping, the history package's types, cursor and shared ports with fakes,
the OpenAPI paths and schemas of all four tickets with the generated clients and their contract tests, ownership,
archcheck contexts and retention rules. No user-facing string.

## Fix: the free-slot search under JIT's optimise cost (2026-11-12)

Branch `fix-slot-search`. 00043's `practitioner_role_shared` and `organization_shared` were correlated EXISTS
subqueries, planned once per row: a directory read of slots carried them per slot (slot → schedule → role → clinic),
each pulling in encounter's, consent's and appointment's policies. On a directory of production size the free-slot
search (`SearchFreeSlots`, `GET /fhir/R4/Slot?schedule=…&status=free`) was planned above `jit_optimize_above_cost`
(500 000) and spent almost all its time in JIT compilation; two e2e tests hit the 10 s client timeout on it in CI.

Migration 00046 recreates both policies over `history_shared_organizations()` and
`history_shared_practitioner_roles()`: the set of shared visits' clinics and roles, read once per query in an
InitPlan, behind history_shared's own scope conditions (no patient, no account, a clinic set), so directory and patient
reads never run it. Invoker's rights, so the access is exactly 00043's (test
`TestHistorySharing_ThePoliciesAdmitWhat00043Admitted` compares both conditions row for row in ten scopes and seven
sharing states).

Measured with `seedSlotSearch` (`migrations/history_shared_lookup_test.go`: 20 listed clinics, 200 roles and schedules,
about 22 000 slots, 1 400 free in the searched schedule over 14 days, 1 200 finished visits, 300 history consents),
as `salus_app`, JIT on (server defaults), EXPLAIN ANALYZE, three runs each, 14-day window, 1 276 rows:

| Scope | Before: cost | Before: planning / execution (JIT) | After: cost | After: planning / execution |
| --- | --- | --- | --- | --- |
| directory | 2 989 681 | 100–174 ms / 80.7–83.8 s (JIT 77.9–80.9 s) | 18 979 | 0.4–1.3 ms / 3.7–4.3 ms, no JIT |
| directory + patient | 3 005 984 | 105–106 ms / 82.4–83.3 s (JIT 79.9–80.9 s) | 18 979 | 0.4–0.7 ms / 3.6–5.9 ms, no JIT |
| a clinic (no match) | 2 989 681 | 106–109 ms / 84.9–85.1 s (JIT 82.1–82.5 s) | 18 979 | 0.5–0.9 ms / 5.1–20.2 ms, no JIT |

Where the lookup does run, in the scope of the clinic with the most history consents (17): `SELECT id FROM
practitioner_role` cost 507, 3.8–27.1 ms (20 rows); `organization` cost 2, 1.7–2.9 ms; `schedule` cost 2 197, 3.7 ms.
`TestHistorySharing_TheFreeSlotSearchStaysUnderJITsOptimizeCost` keeps the search under a fifth of
`jit_optimize_above_cost`, with no scan of encounter, consent, history_consent(_encounter) or appointment in its plan.

## SAL-40: the practitioner's view and the pre-visit summary (2026-11-12)

Branch `sal-40-doctor-history`. `history.Service.ForPractitioner` and `Summary` (`service_practitioner.go`), on
`encounter.Service.History` (the visits `access.HistoryVisit` lets the practitioner see, gated on `access.Profile` at
Full) and one read per owning domain for those visits (`composition.Service.History`,
`questionnaireresponse.Service.History`), each domain auditing the items a page shows (`Disclosed`) and the history
auditing the view itself (`Patient/<id>`, TREAT). A shared visit's signed documents, completed responses, their files
and the questions they answer open for that practitioner through `encounter.Service.Shared`/`SharedAppointment`; the
questions through migration 00050's `history_questionnaire` (`SECURITY DEFINER`, owned by `salus_history_lookup`),
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
