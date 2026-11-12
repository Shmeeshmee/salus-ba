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

## SAL-39: the patient's timeline (2026-11-12)

Branch `sal-39-timeline`. `history.Service.Timeline` over four sources (encounter, composition, questionnaireresponse,
binary for the files' state), `GET /me/history`, the mobile history screen, e2e over two clinics.

**Audit in one insert, sources at once.** Each source audited every item it returned with its own `INSERT` (up to
31 a source, about 120 a page), and the sources ran one after another. Now each source writes its page's events with
`auditevent.Service.RecordAll` (one `INSERT … SELECT FROM jsonb_to_recordset`, same columns and stamping as
`Record`, in the source's transaction), and `Timeline` asks the sources at the same time (each is its own
transaction in its domain).

Measured with `make bench-history` (`internal/api/history_load_test.go`: a patient with 2 000 items, 250 visits at
each of two clinics with a signed document, a completed response and a clean file each; 400 other patients with 20
visits each; pages of 30; 300 samples; the timeline as `api.New` composes it, every source real). The machine was
shared with other stacks throughout (load average 15 to 80 on 28 cores), so the absolute numbers are far above what a
quiet server gives; run 2 ran both versions at the same time to share the same noise.

| Read | Run 1 before (load 15) | Run 1 after (load 69) | Run 2 before | Run 2 after (concurrent, load 80) |
| --- | --- | --- | --- | --- |
| first page p50 / p95 | 195 / 512 ms | 133 / 220 ms | 310 / 793 ms | 105 / 404 ms |
| any page p50 / p95 | 170 / 289 ms | 173 / 358 ms | 283 / 427 ms | 148 / 297 ms |
| one clinic p50 / p95 | 122 / 179 ms | 165 / 486 ms | 183 / 233 ms | 129 / 322 ms |
| documents only p50 / p95 | 64 / 92 ms | 116 / 165 ms | 116 / 177 ms | 123 / 226 ms |
| binary's files check, mean | 37.4 ms | 13.5 ms | 19.9 ms | 13.8 ms |

After merging main (00046's lookups, 00047), with the batched and concurrent version, at load average 15 to 20:
first page p50 72 / p95 118 ms, any page 93 / 176 ms, one clinic 52 / 74 ms, documents only 60 / 95 ms; mean per
source: documents 60 ms, attachments 31 ms, responses 30 ms, visits 17 ms, the files check 9 ms.

The full timeline gains in both runs (first page p50 1.5 to 3 times faster); a read of one source (the documents)
gains nothing measurable under this noise, as expected with nothing to run beside it. The plan's target (p95 under
50 ms) is not shown on this machine: it needs a run on a quiet one. Each source costs 30 to 150 ms here for 31 rows,
which points at the row-level security of the patient's scope rather than at the audit; that is the next thing to
profile (EXPLAIN ANALYZE per source) if the quiet run misses the target.

**Review (fresh subagent), 2026-11-13:** ten findings, no access hole; 1, 2, 4 to 10 fixed test-first, 3 measured
and fixed differently (below).

- Audit (1, 2, 8): the sources read unaudited, in the one transaction `Timeline` opens, one after the other; the
  transaction then audits the page it returns (after the cut and binary's check), each item once with its own
  reference and clinic, in one `RecordAll`. The concurrent sources and the per-domain batches above are gone.
- 00050 (4): history_shared's own scope conditions in front of the call in `encounter_shared`,
  `document_draft_shared`, `questionnaire_response_shared`, `binary_shared`. A patient's read of the four tables
  called history_shared 44 times on the test fixture, now never
  (`TestHistorySharing_APatientsReadDoesNotCallHistoryShared`); the policies admit exactly 00043's rows in twelve
  scopes and seven sharing states (`TestHistorySharing_TheGuardedPoliciesAdmitWhat00043Admitted`).
- 3: reading every encounter of the patient costs 0.7 ms of execution (EXPLAIN ANALYZE, 500 encounters); the cost
  was planning, since `Own` used the FHIR search's query, re-planned on every call for its optional parameters. A
  read of the documents in the patient's scope without the encounter list was tried and dropped: its policy tree
  is costed at 390 000, above `jit_above_cost`, and each call spent 950 ms in JIT for a 5 ms scan. `Own` now has
  its own fixed query (`EncountersOfPatient`): 7.4 to 2.8 ms a call, documents 10.0 to 6.1 ms.
- 5: clinic names in one transaction, the roles in one `LookupAll`. 9: `FilesOf` lists a file once. 10: the
  cursor's length is checked before it is decoded; neither Caddy (no access log, URIs dropped) nor the API's request
  log (the route pattern only) records a query string. 6: the app opens a questionnaire's answers by reading that
  response and its version (`Questionnaire?url=&version=`), not every response of the patient. 7: an empty page
  with a next page shows Show more, not "nothing yet".

`make bench-history` (same seed as above), load average in brackets:

| Read, p50 / p95 | Before (one tx, audit fixed) [12 → 9] | After 00050, `Own`, 5 [12 → 9] |
| --- | --- | --- |
| first page | 125.5 / 168.4 ms | 21.3 / 35.0 ms |
| any page | 124.9 / 175.8 ms | 21.9 / 33.8 ms |
| one clinic | 79.6 / 99.2 ms | 19.2 / 31.1 ms |
| documents only | 73.6 / 124.2 ms | 15.0 / 24.1 ms |
| mean per source: visits, documents, responses, attachments | 13.6, 60.6, 26.7, 27.2 ms | 1.0, 6.4, 1.9, 2.2 ms |

The first page is under the plan's 50 ms at p95.
