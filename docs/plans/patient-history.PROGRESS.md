# Patient history: progress

Plan: docs/plans/patient-history.md. ADR 0014.

## Step 0: plan, ADR, contract (2026-11-12)

Branch `history-base`. Migration 00043 (four tables, `history_shared`, the shared and profile policies), the access
rules `HistoryVisit`, `ProfileEntry`, `Attributed`, the history consent's model, state and FHIR mapping, the
allergy and medication models and FHIR mapping, the history package's types, cursor and shared ports with fakes,
the OpenAPI paths and schemas of all four tickets with the generated clients and their contract tests, ownership,
archcheck contexts and retention rules. No user-facing string.

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
