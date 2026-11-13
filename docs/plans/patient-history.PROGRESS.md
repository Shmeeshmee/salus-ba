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

Open: Provenance for the clinical writes (`Record`, `Confirm`, a clinic's `Update` and `Remove`) is not written yet:
how to store it (composition keeps one Provenance id on its row; an entry has several clinical writes, and
`GET /fhir/R4/Provenance/{id}` is composition's route) was put to the owner, who decided (2026-11-13) that it comes in a follow-up PR after
SAL-41 merges.

Copy waiting for the owner:

Mobile (ten languages, `mobile/lib/l10n/app_<language>.arb`): `allergies` Allergies; `currentMedications` Current medications; `allergiesIntro` Clinics you have given your consent to see the allergies in your profile when you have an appointment with them.; `medicationsIntro` Clinics you have given your consent to see the medications in your profile when you have an appointment with them.; `noAllergies` You have not added any allergies.; `noMedications` You have not added any medications.; `addAllergy` Add an allergy; `addMedication` Add a medication; `changeAllergy` Change allergy; `changeMedication` Change medication; `change` Change; `allergySubstance` Substance or food; `medicationName` Medication; `termSearchHint` Type to search, or write it in your own words.; `allergyCategory` Type (optional); `notSpecified` Not specified; `categoryFood` Food; `categoryMedication` Medication; `categoryEnvironment` Environment; `categoryBiologic` Biological product; `reactionOptional` Reaction (optional); `severity` Severity; `severityMild` Mild; `severityModerate` Moderate; `severitySevere` Severe; `entryStatus` Status; `allergyActive` Active; `allergyInactive` Inactive; `allergyResolved` Resolved; `reactionLine` Reaction: {reaction}; `reactionWithSeverity` Reaction: {reaction} ({severity}); `confirmedByClinician` Confirmed by a clinician; `recordedByClinic` Recorded by a clinic during one of your visits; `confirmedChangeNote` If you change this entry, it is no longer confirmed by a clinician.; `nameRequired` Enter a name, up to 200 characters.; `addToProfile` Add to my profile; `addToProfileTitle` Add to your profile?; `addToProfileBody` Every clinic you have given your consent to will then see it, without the name of the clinic that recorded it.; `addedToProfile` It is now in your profile.; `removeEntryTitle` Remove this entry?; `removeEntryBody` It will no longer be shown to you or to any clinic. If a clinic recorded or confirmed it, it stays in your medical record, marked as removed.; `entryRemoved` The entry is removed.; `entrySaved` Saved to your profile.; `entryChanged` This entry was changed in the meantime. The latest version is shown.; `entryInvalid` Some details are not valid. Check them and try again.; `entryNotAllowed` You cannot change this entry.; `entryNotFound` This entry is no longer available.; `doseHeading` Dose (optional); `doseAmount` Amount; `doseUnit` Unit; `unitMg` milligrams (mg); `unitG` grams (g); `unitUg` micrograms (µg); `unitMl` millilitres (mL); `unitIu` international units (IU); `unitTablet` tablets; `unitCapsule` capsules; `unitDrop` drops; `unitPuff` puffs (inhaler); `unitPatch` patches; `doseBoth` Enter both the amount and the unit, or neither.; `numberAboveZero` Enter a number above 0.; `howOftenHeading` How often (optional); `timesLabel` Times; `everyLabel` Every; `periodUnitLabel` Unit of time; `periodHours` hours; `periodDays` days; `periodWeeks` weeks; `periodMonths` months; `howOftenAll` Fill in all three, or leave them empty.; `asNeeded` As needed; `medicationActive` Taking it now; `medicationStopped` Stopped; `doseMg` {value} mg; `doseG` {value} g; `doseUg` {value} µg; `doseMl` {value} mL; `doseIu` {value} IU; `doseTablet` {count, plural, one{{value} tablet} other{{value} tablets}}; `doseCapsule` {count, plural, one{{value} capsule} other{{value} capsules}}; `doseDrop` {count, plural, one{{value} drop} other{{value} drops}}; `dosePuff` {count, plural, one{{value} puff} other{{value} puffs}}; `dosePatch` {count, plural, one{{value} patch} other{{value} patches}}; `doseLine` Dose: {dose}; `howOftenPerHour` {frequency}× an hour; `howOftenPerDay` {frequency}× a day; `howOftenPerWeek` {frequency}× a week; `howOftenPerMonth` {frequency}× a month; `howOftenEveryHours` {period, plural, one{{frequency}× every {every} hour} other{{frequency}× every {every} hours}}; `howOftenEveryDays` {period, plural, one{{frequency}× every {every} day} other{{frequency}× every {every} days}}; `howOftenEveryWeeks` {period, plural, one{{frequency}× every {every} week} other{{frequency}× every {every} weeks}}; `howOftenEveryMonths` {period, plural, one{{frequency}× every {every} month} other{{frequency}× every {every} months}}.

Web (five staff languages, `web/src/i18n/allergies/<language>.json`, keys under `allergies.`): `warningTitle` Active allergies; `reactionSeverity` {reaction} ({severity, select, mild {mild} moderate {moderate} severe {severe} other {}}); `warningLoading` Loading allergies…; `warningFailed` Allergies could not be loaded. Do not assume the patient has none.; `retry` Try again; `tab` Allergies and medications; `allergies` Allergies; `medications` Medications; `noAllergies` No allergies recorded.; `noMedications` No medications recorded.; `loading` Loading allergies and medications…; `loadFailed` Allergies and medications could not be loaded.; `confirmed` Confirmed; `unconfirmed` Not confirmed; `confirm` Confirm; `confirmLabel` Confirm {text}; `confirmChanged` This entry changed since it was read. The list is updated: check it and confirm again.; `confirmForbidden` You cannot confirm this entry.; `confirmFailed` The entry could not be confirmed. Try again.; `category.food` Food; `category.medication` Medication; `category.environment` Environment; `category.biologic` Biologic; `severity.mild` Mild; `severity.moderate` Moderate; `severity.severe` Severe; `clinicalStatus.active` Active; `clinicalStatus.inactive` Inactive; `clinicalStatus.resolved` Resolved; `status.active` Current; `status.stopped` Stopped; `dose.mg` {value, number} mg; `unit.mg` mg; `dose.g` {value, number} g; `unit.g` g; `dose.ug` {value, number} µg; `unit.ug` µg; `dose.mL` {value, number} mL; `unit.mL` mL; `dose.iU` {value, number} IU; `unit.iU` IU; `dose.tablet` {value, plural, one {# tablet} other {# tablets}}; `unit.tablet` Tablet; `dose.capsule` {value, plural, one {# capsule} other {# capsules}}; `unit.capsule` Capsule; `dose.drop` {value, plural, one {# drop} other {# drops}}; `unit.drop` Drop; `dose.puff` {value, plural, one {# puff} other {# puffs}}; `unit.puff` Puff; `dose.patch` {value, plural, one {# patch} other {# patches}}; `unit.patch` Patch; `timing.h` {frequency, plural, one {once} other {# times}} {period, plural, one {an hour} other {every # hours}}; `timing.d` {frequency, plural, one {once} other {# times}} {period, plural, one {a day} other {every # days}}; `timing.wk` {frequency, plural, one {once} other {# times}} {period, plural, one {a week} other {every # weeks}}; `timing.mo` {frequency, plural, one {once} other {# times}} {period, plural, one {a month} other {every # months}}; `periodUnit.h` Hours; `periodUnit.d` Days; `periodUnit.wk` Weeks; `periodUnit.mo` Months; `asNeeded` As needed; `addAllergy` Add an allergy; `addMedication` Add a medication; `substance` Substance; `substanceHint` Choose a term from the list, or type the substance.; `medication` Medication; `medicationHint` Choose a term from the list, or type the medication.; `fromTerms` From the medical terms list; `terms` Matching terms; `categoryLabel` Category; `notSpecified` Not specified; `reaction` Reaction; `severityLabel` Severity; `severityHint` Enter a reaction to choose its severity.; `statusLabel` Status; `doseLabel` Dose; `unitLabel` Unit; `noUnit` No unit; `frequency` Times; `period` Every; `periodUnitLabel` Unit of time; `save` Save; `saving` Saving…; `cancel` Cancel; `noVisit` Join the consultation before you record an entry.; `invalid` Check the entry: a value is missing or not allowed.; `doseBoth` Enter both the dose and its unit, or neither.; `timingAll` Fill in Times, Every and Unit of time, or leave all three empty.; `saveFailed` The entry could not be saved. Try again.; `saveForbidden` You cannot record entries in this consultation..
