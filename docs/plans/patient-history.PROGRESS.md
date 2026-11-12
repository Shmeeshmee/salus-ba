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

The full timeline gains in both runs (first page p50 1.5 to 3 times faster); a read of one source (the documents)
gains nothing measurable under this noise, as expected with nothing to run beside it. The plan's target (p95 under
50 ms) is not shown on this machine: it needs a run on a quiet one. Each source costs 30 to 150 ms here for 31 rows,
which points at the row-level security of the patient's scope rather than at the audit; that is the next thing to
profile (EXPLAIN ANALYZE per source) if the quiet run misses the target.
