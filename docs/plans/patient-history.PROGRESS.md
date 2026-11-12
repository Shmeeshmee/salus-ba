# Patient history: progress

Plan: docs/plans/patient-history.md. ADR 0014.

## Step 0: plan, ADR, contract (2026-11-12)

Branch `history-base`. Migration 00043 (four tables, `history_shared`, the shared and profile policies), the access
rules `HistoryVisit`, `ProfileEntry`, `Attributed`, the history consent's model, state and FHIR mapping, the
allergy and medication models and FHIR mapping, the history package's types, cursor and shared ports with fakes,
the OpenAPI paths and schemas of all four tickets with the generated clients and their contract tests, ownership,
archcheck contexts and retention rules. No user-facing string.

## SAL-42: cross-clinic sharing (2026-11-12)

Branch `sal-42-sharing`. `consent`: the history kind in `Kind.Valid` and OpenAPI `ConsentKind` (granted only through
`PUT /history-sharing/{organizationId}`; `POST /consents` with it is `invalid_request`, since an active history consent
lists at least one visit), `Candidates`, `SetSharing` (If-Match, 412), `StopSharing`, `MySharing`, the care consent's
revocation ending the history consent in its transaction, every call audited; `ByID` and `ByPatient` know the third
kind. `history.Sharing` (consent's `Visits` port) builds the preview from `encounter.Service.Search`,
`composition.Service.Signed`, `practitionerrole.Service.Lookup` and the new `organization.Service.Name`, each in the
patient's own scope and audited by its owner. No migration. Tests: service (fakes), repository (pgtest: removal and
revocation take effect for the clinic's scope at once, row-level security refuses a visit at the receiving clinic, an
unfinished one and another patient's), handlers, the drafting and the suggestions never receiving another visit's
content. Mobile: the booking toggle with the preview, Shared history per clinic, My consents' history label; ten
languages, `legal` where the plan says. E2E: two of three visits shared, read by the clinic's practitioner as the
history consent; the read through SAL-40's view is added once SAL-40 is on main.
