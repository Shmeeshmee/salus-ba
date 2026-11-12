# Patient history: progress

Plan: docs/plans/patient-history.md. ADR 0013.

## Step 0: plan, ADR, contract (2026-11-12)

Branch `history-base`. Migration 00041 (four tables, `history_shared`, the shared and profile policies), the access
rules `HistoryVisit`, `ProfileEntry`, `Attributed`, the history consent's model, state and FHIR mapping, the
allergy and medication models and FHIR mapping, the history package's types, cursor and shared ports with fakes,
the OpenAPI paths and schemas of all four tickets with the generated clients and their contract tests, ownership,
archcheck contexts and retention rules. No user-facing string.
