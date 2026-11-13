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

## SAL-41: allergies and current medications (2026-11-13)

Branch `sal-41-allergies`. No migration of its own (00043's tables, policies and trigger).

As built: `allergyintolerance` and `medicationstatement` with their repositories (sqlc), services (`Add`, `Record`,
`Update`, `Remove`, `Confirm`, `AddToProfile`, `Get`, `Seen` with `history.Allergies` / `history.Medications`'
signature), the JSON handlers of api/openapi.yaml (tag history), the FHIR read and `?patient=` search of
AllergyIntolerance and MedicationStatement (golden files, CapabilityStatement), `EntryWritten` in
api/contract_history_test.go, both services wired in internal/api/app.go. Every entry read and every write is
audited, refusals included; a search that finds nothing names `Patient/<id>`. Terminology: `DrugCodes` and
`DrugLabelled` for the picked term's ATC code and label. `mobile/`: the profile's allergies and medications (add,
change, remove, "Add to my profile"). `web/`: the allergies and medications tab on the call page (record during the
visit, confirm), the warning of the active allergies on top of the call page in every phase and above the drafting
editor; a list still loading or one that could not be read says so there (with Try again), never "none"; the lists
are read again every 60 s. E2E: the patient adds an allergy, the practitioner sees the warning on the call page.

Decisions within the plan:
- A practitioner records an entry only while the visit is in progress ("during a visit"); a finished one is
  ErrForbidden.
- A change that changes neither the content nor the status keeps the confirmation; any other change by the patient,
  or by a clinic other than the confirming one, clears it (the trigger's rule).
- A term picked from the terminology must carry one of its labels as the text (any language); an entry keeps a term
  and text it already had though the term's labels changed since.
- A write that matches no row is told apart by reading the entry again in the transaction: gone from the scope is
  ErrNotFound, at another version ErrVersionConflict, visible but not writable ErrForbidden.

Measurements: none taken; the plan asks no benchmark of SAL-41 (`Seen` is one indexed read by patient, then the
consent state and the clinic's appointments).

Open item closed (follow-up PR, branch `fix-entry-provenance`, migration 00053, 2026-11-13): Provenance for the
entries' writes. A practitioner's `Record` (CREATE; the agent its author and, as recording confirms with the same
role, its verifier), `Update` (UPDATE, author), `Confirm` (UPDATE, verifier) and `Remove` (NULLIFY, author: kept as
entered-in-error), and the patient's `AddToProfile` (UPDATE) and `Remove` of a clinic's entry (NULLIFY), the patient
the agent, append one FHIR Provenance in the write's transaction, audited: target the entry's version written, a
clinic's agent the PractitionerRole on behalf of its Organization, for a change the revised version as an entity
(v3-DataOperation, provenance-participant-type). The patient's writes on their own entries write none (decided:
patient-reported, the entry names the patient as its source and the AuditEvent records the write). Stored
append-only in `allergy_intolerance_provenance` and `medication_statement_provenance`, owned by the entry's domain,
retention `keep`, never deleted with the entry (decided): the foreign key has no action, and `Remove` deletes the
patient's own entry only when it has no Provenance (`HasProvenance` in the patient's scope), else marks it
entered-in-error; the entry's delete policy does not read the Provenances (it would recurse). Row-level security:
the patient's scope those of their entries, a clinic's scope only its own Provenances of entries it sees, so another
clinic never learns the recording or confirming clinic (ADR 0014, `access.Attributed`, also the service's rule);
each insert is tied to the write it attests (the entry at that version; the recorder and confirmer for a recording,
the confirmer for a confirmation, the entry's clinic for a change or removal, the entry's patient for theirs). A
refused read is audited as a read of `Provenance/<id>`. `GET /fhir/R4/Provenance/{id}`: one route, several sources,
declared in the router (`Config.Provenances`, asked in turn): the entries' sources answer only ids in their store
(nothing audited otherwise), composition's signatures last (it audits any id); the answer's timing can tell which
store holds an id (opaque, accepted). Composition's store is unchanged. No search or `_revinclude`. A removal's
Provenance is read by no one (nobody sees an entered-in-error entry), so it has no golden file.

Copy waiting for the owner:

Mobile (ten languages, `mobile/lib/l10n/app_<language>.arb`): `allergies` Allergies; `currentMedications` Current medications; `allergiesIntro` Clinics you have given your consent to see the allergies in your profile when you have an appointment with them.; `medicationsIntro` Clinics you have given your consent to see the medications in your profile when you have an appointment with them.; `noAllergies` You have not added any allergies.; `noMedications` You have not added any medications.; `addAllergy` Add an allergy; `addMedication` Add a medication; `changeAllergy` Change allergy; `changeMedication` Change medication; `change` Change; `allergySubstance` Substance or food; `medicationName` Medication; `termSearchHint` Type to search, or write it in your own words.; `allergyCategory` Type (optional); `notSpecified` Not specified; `categoryFood` Food; `categoryMedication` Medication; `categoryEnvironment` Environment; `categoryBiologic` Biological product; `reactionOptional` Reaction (optional); `severity` Severity; `severityMild` Mild; `severityModerate` Moderate; `severitySevere` Severe; `entryStatus` Status; `allergyActive` Active; `allergyInactive` Inactive; `allergyResolved` Resolved; `reactionLine` Reaction: {reaction}; `reactionWithSeverity` Reaction: {reaction} ({severity}); `confirmedByClinician` Confirmed by a clinician; `recordedByClinic` Recorded by a clinic during one of your visits; `confirmedChangeNote` If you change this entry, it is no longer confirmed by a clinician.; `nameRequired` Enter a name, up to 200 characters.; `addToProfile` Add to my profile; `addToProfileTitle` Add to your profile?; `addToProfileBody` Every clinic you have given your consent to will then see it, without the name of the clinic that recorded it.; `addedToProfile` It is now in your profile.; `removeEntryTitle` Remove this entry?; `removeEntryBody` It will no longer be shown to you or to any clinic. If a clinic recorded or confirmed it, it stays in your medical record, marked as removed.; `entryRemoved` The entry is removed.; `entrySaved` Saved to your profile.; `entryChanged` This entry was changed in the meantime. The latest version is shown.; `entryInvalid` Some details are not valid. Check them and try again.; `entryNotAllowed` You cannot change this entry.; `entryNotFound` This entry is no longer available.; `doseHeading` Dose (optional); `doseAmount` Amount; `doseUnit` Unit; `unitMg` milligrams (mg); `unitG` grams (g); `unitUg` micrograms (µg); `unitMl` millilitres (mL); `unitIu` international units (IU); `unitTablet` tablets; `unitCapsule` capsules; `unitDrop` drops; `unitPuff` puffs (inhaler); `unitPatch` patches; `doseBoth` Enter both the amount and the unit, or neither.; `numberAboveZero` Enter a number above 0.; `howOftenHeading` How often (optional); `timesLabel` Times; `everyLabel` Every; `periodUnitLabel` Unit of time; `periodHours` hours; `periodDays` days; `periodWeeks` weeks; `periodMonths` months; `howOftenAll` Fill in all three, or leave them empty.; `asNeeded` As needed; `medicationActive` Taking it now; `medicationStopped` Stopped; `doseMg` {value} mg; `doseG` {value} g; `doseUg` {value} µg; `doseMl` {value} mL; `doseIu` {value} IU; `doseTablet` {count, plural, one{{value} tablet} other{{value} tablets}}; `doseCapsule` {count, plural, one{{value} capsule} other{{value} capsules}}; `doseDrop` {count, plural, one{{value} drop} other{{value} drops}}; `dosePuff` {count, plural, one{{value} puff} other{{value} puffs}}; `dosePatch` {count, plural, one{{value} patch} other{{value} patches}}; `doseLine` Dose: {dose}; `howOftenPerHour` {frequency}× an hour; `howOftenPerDay` {frequency}× a day; `howOftenPerWeek` {frequency}× a week; `howOftenPerMonth` {frequency}× a month; `howOftenEveryHours` {period, plural, one{{frequency}× every {every} hour} other{{frequency}× every {every} hours}}; `howOftenEveryDays` {period, plural, one{{frequency}× every {every} day} other{{frequency}× every {every} days}}; `howOftenEveryWeeks` {period, plural, one{{frequency}× every {every} week} other{{frequency}× every {every} weeks}}; `howOftenEveryMonths` {period, plural, one{{frequency}× every {every} month} other{{frequency}× every {every} months}}.

Web (five staff languages, `web/src/i18n/allergies/<language>.json`, keys under `allergies.`): `warningTitle` Active allergies; `reactionSeverity` {reaction} ({severity, select, mild {mild} moderate {moderate} severe {severe} other {}}); `warningLoading` Loading allergies…; `warningFailed` Allergies could not be loaded. Do not assume the patient has none.; `retry` Try again; `tab` Allergies and medications; `allergies` Allergies; `medications` Medications; `noAllergies` No allergies recorded.; `noMedications` No medications recorded.; `loading` Loading allergies and medications…; `loadFailed` Allergies and medications could not be loaded.; `confirmed` Confirmed; `unconfirmed` Not confirmed; `confirm` Confirm; `confirmLabel` Confirm {text}; `confirmChanged` This entry changed since it was read. The list is updated: check it and confirm again.; `confirmForbidden` You cannot confirm this entry.; `confirmFailed` The entry could not be confirmed. Try again.; `category.food` Food; `category.medication` Medication; `category.environment` Environment; `category.biologic` Biologic; `severity.mild` Mild; `severity.moderate` Moderate; `severity.severe` Severe; `clinicalStatus.active` Active; `clinicalStatus.inactive` Inactive; `clinicalStatus.resolved` Resolved; `status.active` Current; `status.stopped` Stopped; `dose.mg` {value, number} mg; `unit.mg` mg; `dose.g` {value, number} g; `unit.g` g; `dose.ug` {value, number} µg; `unit.ug` µg; `dose.mL` {value, number} mL; `unit.mL` mL; `dose.iU` {value, number} IU; `unit.iU` IU; `dose.tablet` {value, plural, one {# tablet} other {# tablets}}; `unit.tablet` Tablet; `dose.capsule` {value, plural, one {# capsule} other {# capsules}}; `unit.capsule` Capsule; `dose.drop` {value, plural, one {# drop} other {# drops}}; `unit.drop` Drop; `dose.puff` {value, plural, one {# puff} other {# puffs}}; `unit.puff` Puff; `dose.patch` {value, plural, one {# patch} other {# patches}}; `unit.patch` Patch; `timing.h` {frequency, plural, one {once} other {# times}} {period, plural, one {an hour} other {every # hours}}; `timing.d` {frequency, plural, one {once} other {# times}} {period, plural, one {a day} other {every # days}}; `timing.wk` {frequency, plural, one {once} other {# times}} {period, plural, one {a week} other {every # weeks}}; `timing.mo` {frequency, plural, one {once} other {# times}} {period, plural, one {a month} other {every # months}}; `periodUnit.h` Hours; `periodUnit.d` Days; `periodUnit.wk` Weeks; `periodUnit.mo` Months; `asNeeded` As needed; `addAllergy` Add an allergy; `addMedication` Add a medication; `substance` Substance; `substanceHint` Choose a term from the list, or type the substance.; `medication` Medication; `medicationHint` Choose a term from the list, or type the medication.; `fromTerms` From the medical terms list; `terms` Matching terms; `categoryLabel` Category; `notSpecified` Not specified; `reaction` Reaction; `severityLabel` Severity; `severityHint` Enter a reaction to choose its severity.; `statusLabel` Status; `doseLabel` Dose; `unitLabel` Unit; `noUnit` No unit; `frequency` Times; `period` Every; `periodUnitLabel` Unit of time; `save` Save; `saving` Saving…; `cancel` Cancel; `noVisit` Join the consultation before you record an entry.; `invalid` Check the entry: a value is missing or not allowed.; `doseBoth` Enter both the dose and its unit, or neither.; `timingAll` Fill in Times, Every and Unit of time, or leave all three empty.; `saveFailed` The entry could not be saved. Try again.; `saveForbidden` You cannot record entries in this consultation..

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

## SAL-40: the practitioner's view and the pre-visit summary (2026-11-12)

Branch `sal-40-doctor-history`. `history.Service.ForPractitioner` and `Summary` (`service_practitioner.go`), on
`encounter.Service.History` (the visits `access.HistoryVisit` lets the practitioner see, gated on `access.Profile` at
Full) and one read per owning domain for those visits (`composition.Service.History`,
`questionnaireresponse.Service.History`), each domain auditing the items a page shows (`Disclosed`) and the history
auditing the view itself (`Patient/<id>`, TREAT). A shared visit's signed documents, completed responses, their files
and the questions they answer open for that practitioner through `encounter.Service.Shared`/`SharedAppointment`; the
questions through migration 00051's `history_questionnaire` (`SECURITY DEFINER`, owned by `salus_history_lookup`),
not a chained policy. `migrations/language_test.go` runs in its usual time with it (about 17 s per test, parallel,
as before). Diagnoses: the ICD-10 codes of the signed documents' coded fields (`HistoryItem.diagnoses`, added to
the schema and the Go type), for own and shared visits alike: `condition.Service` has no read of a visit's
Conditions and the plan says not to change its code. `GET /patients/{id}/history`, `.../history/summary`. Web: the
history panel on the appointment detail (its own practitioner) and as a tab of the call page's live panels, read only
once opened. Allergies and medications through the ports, stood in by `noAllergies`/`noMedications` in
`internal/api/history.go` until SAL-41's services are wired at the merge.

Readings (`make bench-history-practitioner`, compose Postgres on the shared dev machine, other stacks running): a patient with
300 visits with the practitioner and 100 shared from another clinic, each with a signed document, and 500 more
visits there not shared (read by the clinic's scope through `history_shared`, left out by the rule).

| Read | p50 | p95 | p99 |
| --- | --- | --- | --- |
| First page of 30, documents and responses read per visit (first cut) | 63.7 ms | 116.6 ms | 170.6 ms |
| First page of 30, one query for every visit's documents, one search per patient for the responses | 19.8 ms | 55.8 ms | 93.9 ms |
| Pre-visit summary | 16.7 ms | 46.9 ms | 94.8 ms |
| First page of 30, after merging main (00046's once-per-query lookups), 2026-11-13 | 16.2 ms | 63.6 ms | 101.7 ms |
| Pre-visit summary, after merging main, 2026-11-13 | 13.2 ms | 22.1 ms | 24.9 ms |
| First page of 30, after the review fixes and SAL-41's wiring, load average 4.3, 2026-11-13 | 11.4 ms | 18.1 ms | 21.6 ms |
| Pre-visit summary, same run (now with SAL-41's real reads) | 18.1 ms | 21.9 ms | 25.5 ms |

Every page of 100 to the last (9 pages, 804 items): the last 28 ms, the slowest 43 ms. After the merge: the last 12.9 ms, the slowest 29.8 ms (the first page's p95 and p99 moved with
the load of the other stacks on the machine; p50 did not). The cost was the per-visit
queries, not `history_shared`: the partial index on `history_consent_encounter (encounter_id)` is enough for these
reads; nothing more was added.

Review (fresh subagent, 2026-11-13): eight findings. Fixed: the summary's allergies and medications are marked
unavailable on a failed read instead of empty, read after the summary's own transaction so a failure loses neither
the rest nor its audit, the empty text says "shown to you", and the SAL-41 stand-ins fail closed
(`TestHistory_TheProfileStandInsGoOnceSAL41IsHere` fails once SAL-41's services are on the branch); a diagnosis
carries its field's label and its Condition's confirmation, through `condition.Reads.Confirmed` for the clinic's own
and 00051's `history_confirmed_conditions` for a shared visit (a deviation from the plan's "without changing
condition's code", asked by the review: a read was added, nothing else changed); the history on the appointment
opens on request; the summary reads only the open responses; a file is audited as its Binary; the titles are one read
per page (`questionnaire.Service.Titles`); the shared-version read needs a practitioner with an appointment with the
patient (00051's lookups take the practitioner); a response opens with its questions; a shared visit without a clinic
name still says it is shared. Not done: ICD-10 names, no lookup by code exists; a visit has nothing to open
beyond the items listed with it. Audit rows are one INSERT each (about 30 per page): `auditevent` has no batch insert
on main, so none was added. The bench was rerun after SAL-41's wiring on a quieter host (load average 4.3 to 4.7):
every page of 100 to the last, 9 pages and 804 items, the last 9.0 ms, the slowest 21.3 ms.

Open for the owner: any appointment of the practitioner's with the patient at the clinic, of any status (cancelled,
years old), opens the history (`access.Profile`, as the plan says), and the summary accepts a cancelled appointment;
CLAUDE.md says "a care relationship or an active appointment". The rule is unchanged until the owner decides.

SAL-41 wired (2026-11-13, after #83): main merged, the stand-ins deleted, the summary reads
`allergyintolerance.Service.Seen` and `medicationstatement.Service.Seen` (their `Caller` is `appointment.Caller`, no
adapter); `TestHistory_ReadsTheProfileThroughSAL41sServices` keeps the stand-ins gone and the API test reads an
allergy the patient added. A failed read stays "unavailable"; "No allergies shown to you" follows only a read that
returned none. The medication's dosage is shown with SAL-41's strings (`allergies.dose.*`, `allergies.timing.*`,
`allergies.asNeeded`), no new copy. The call page has both tabs, SAL-41's allergies and medications and the history.

Second review (focused, 2026-11-13), after merging main (SAL-20): the empty texts say what is left out ("…
Entries recorded in visits you cannot see are not included; ask the patient.", and the same for the medications and
the last visit); the call page's summary is read again each time its tab is shown and when SAL-41's entries change,
and a failed summary or history read has "Try again"; `expectHTTP` fails on every missing string (`missing`, tested);
the dosage keeps only SAL-41's known units; a diagnosis is confirmed only by a Condition of that field and that code
(00051's `history_confirmed_conditions` returns the code). Accepted: `Seen` audits every entry it returns, inactive
allergies and stopped medications included, which the summary does not show: over-audit, the safe direction; an
active-only read would change SAL-41's services, so it was not added. Not adopted: deriving the practitioner in
00051's lookups from `app.practitioner_role_id` (as 00044 does): only the chat's repository sets that setting, the
questionnaire's and the condition's scopes set the clinic only, and the caller carries the practitioner, not a role
(a practitioner has one per clinic). The argument comes from the session's caller in the service, never the request.
