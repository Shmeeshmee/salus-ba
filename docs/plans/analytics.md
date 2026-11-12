# Analytics: clinic statistics and platform-wide trends

Epic SAL-18 (SAL-19 to SAL-22). Status: planned, 2026-11-12, **waiting for the owner's approval**; no code is
written before it. Decision record: docs/adr/0012-icd10-diagnosis-groups-and-anonymized-analytics.md.

## 1. Goal

- **My clinic** (SAL-20): a clinic's own appointments and diagnoses over time, its busiest months, weekdays and
  hours, its top diagnosis groups, broken down by practitioner and specialty.
- **All of Salus** (SAL-21): combined figures across all clinics by diagnosis group, region, city and time, with
  no clinic and no patient identifiable, and a flag on unusual rises of a diagnosis group in a region.
- Both rest on **coded diagnoses** (SAL-19): every signed document's ICD-10 codes, confirmed by the signing
  practitioner, kept as FHIR `Condition`, and grouped into our own diagnosis groups based on ICD-10 blocks.
- Both are shown as **interactive charts** in `web/` (SAL-22).

Data sources: `Condition` (from signed documents) and `Appointment`. Never transcripts, suggestions, free-text
fields or questionnaire answers.

## 2. Decisions (proposed, for the owner to approve)

| Topic | Decision | Why |
| --- | --- | --- |
| ICD-10 source | Codes as identifiers only (from the doctor and from the terminology's Wikidata items, CC0); no WHO or national-edition text in any language | The only option legally usable without a licence; same as the terminology's settled decision (ADR 0012) |
| Groups | Our own 27 groups and 7 subgroups defined by ICD-10 code ranges on block boundaries, names written by us in all ten languages (appendix A), reviewed in the term review panel | Ranges are facts; names are ours, so no WHO titles are adapted or translated |
| Where a diagnosis comes from | A template's `coded` field is a diagnosis. Signing writes one `Condition` per filled coded field, in the signing transaction | The field type already exists; no second way to record a diagnosis |
| Confirmation | A coded value suggested by the drafting model must be confirmed by the practitioner (a per-field confirm in the document editor) before the document can be signed; a value the practitioner typed is confirmed by typing it | SAL-19 "the drafting service suggests a code, the doctor confirms" |
| Backfill | In the migration, in SQL: every signed (`final`) document's well-formed coded values become `Condition`s. A signed document whose coded field holds a malformed value is "unmapped": not a table, a query (signed, coded value, no `Condition`), counted in the clinic view and listed by `salusctl conditions unmapped` (ids only) | Nothing is deployed (ADR 0011), so this only touches dev data; a derived flag needs no table |
| Aggregates | Pre-computed per clinic, local day and hour; incremental River job per clinic-day on every change, nightly recompute of the last 35 days plus the platform build | Charts read small tables; the nightly pass repairs a lost job |
| Cross-clinic reads | A `salus_stats` database role (BYPASSRLS, column-level SELECT on source columns only, no patient id) used only by the stats jobs in `cmd/worker` | Mirrors `salus_retention`; makes "no patient id in aggregates" structural |
| Geography | The clinic's location: region (ISO 3166-2 subdivision) and city (a Wikidata municipality item), set by the clinic admin in Clinic settings | No new patient data; see question O1 |
| Platform anonymization | Rules A1 to A13 below, one pure function, tested exhaustively | SAL-21 acceptance criteria |
| Unusual rises | Weekly EARS C2 adaptation per region and group (section 6) | Explainable, needs only 10 weeks of history |
| Charts | Hand-written SVG components in `web/src/analytics/` (line, bars, heatmap); regions as bars, no map in v1 | A chart library is a large dependency for four chart types; a map needs boundary data with its own licence and weight |
| Domains | `condition` (bounded context `clinical`); `clinicstats` and `trends` (new context `analytics`); groups and places in `terminology` (`reference`) | CLAUDE.md "Go architecture" |
| Who sees what | My clinic: admins everything of their clinic; practitioners the clinic's totals and their own breakdown, not other practitioners'. All of Salus: every staff account | SAL-20 "admins and doctors per permission" |

CLAUDE.md lists the bounded contexts too. The owner has an uncommitted edit to CLAUDE.md, so no PR of this epic
touches that file: `docs/architecture/archcheck/config.yaml` (the authority) gets the new context, and the
owner adds `analytics     clinicstats, trends` and `condition` under `clinical` to CLAUDE.md when convenient.

## 3. Coded diagnoses and groups (SAL-19)

**Condition** (`internal/condition`, table `condition`, owned by `condition`):

| Column | Why |
| --- | --- |
| `id`, `created_at`, `updated_at`, `version_id` | FHIR id and meta |
| `organization_id` | Clinic scope (RLS), aggregates |
| `patient_id` | `Condition.subject`; the patient's own record and export |
| `encounter_id` | `Condition.encounter` |
| `document_draft_id`, `field_key` | Which signed document and field it came from (`Condition.evidence.detail` → `Composition`); unique together |
| `practitioner_role_id` | `Condition.recorder`; breakdown by practitioner and specialty |
| `recorded_at` | `Condition.recordedDate`, the signing time |
| `code` | `Condition.code.coding[0]`, system `http://hl7.org/fhir/sid/icd-10` |
| `verification_status` | `confirmed`; `entered-in-error` when the document is |

`clinicalStatus` is not modelled (a signed document says what was diagnosed at that visit, not whether it is
still active). Category `encounter-diagnosis`. RLS: the clinic's scope, and the patient's own scope for their
record. Reads: `GET /fhir/R4/Condition/{id}` (CapabilityStatement updated), audited. Retention: the clinical
record's rule (as `document_draft`).

**Groups** (owned by `terminology`): `diagnosis_group` (`code` a stable slug such as `respiratory-infections`,
`parent_code` for a subgroup, `position`), `diagnosis_group_range` (`group_code`, `from_code`, `to_code`,
inclusive, compared on the code's first `length(from_code)` characters), `diagnosis_group_name` (`group_code`,
`language`, `text`, review columns as `term_translation`). `terminology.Group(code) (group, subgroup)` is a pure
function over the ranges loaded once; a test proves every range is disjoint from the others at its level, every
subgroup lies inside its group, and the examples of appendix A map as listed. `GET /terminology/diagnosis-groups`
returns codes, names in the requested language and the parent, for the filters. The review panel gets a
"Diagnosis groups" list beside terms.

Sources for the aggregates, reached only through the services (contract for wave 2, built in ANA-1):

```go
// condition: one row per Condition of a clinic recorded in [from, to); no patient id.
func (s *Service) Recorded(ctx context.Context, orgID string, from, to time.Time) ([]Recorded, error)
type Recorded struct { RecordedAt time.Time; PractitionerRoleID, Code, Status string }

// appointment: one row per appointment of a clinic starting in [from, to); no patient id.
func (s *Service) Starts(ctx context.Context, orgID string, from, to time.Time) ([]Start, error)
type Start struct { Start time.Time; PractitionerRoleID, Status string }
```

Both run on the `salus_stats` pool (the services take it as their repository's pool in `internal/worker`).
`organization.Service.Locations` returns every clinic's id, time zone, region and city for the jobs (ANA-3 adds
region and city; ANA-1 adds the method with id and time zone).

## 4. Clinic statistics (SAL-20)

Tables (owned by `clinicstats`, RLS by `organization_id`, written only by `salus_stats`):

- `clinic_appointment_stat`: `organization_id`, `day` (local date), `hour` (0–23 local), `practitioner_role_id`,
  `specialty_code`, `status`, `count`.
- `clinic_diagnosis_stat`: `organization_id`, `day`, `hour`, `practitioner_role_id`, `specialty_code`,
  `group_code`, `subgroup_code`, `code`, `count`.

Jobs: `clinicstats_refresh{organization_id, day}` (opaque ids; recompute that clinic-day in one transaction),
enqueued with `InsertTx` by signing, an appointment status change or cancellation, marking a document
entered-in-error, erasure and retention deletes; `clinicstats_nightly` (scheduler, 02:30 UTC) recomputes the last
35 days of every clinic. The specialty is taken at refresh time (a role's specialty change shows from then on).

Rules in the service: C1 a caller sees only their clinic (RLS and the service check); C2 a practitioner who is not
an admin gets the clinic totals and their own role's breakdown only; C3 a diagnosis cell from 1 to 4 is returned
as `"<5"` to every caller (question O3), appointment counts exactly; C4 every read writes an `AuditEvent`
(entity `Organization/<id>`, purpose analytics), since the figures are derived from patient data.

Tests: aggregates equal the raw counts for a generated synthetic clinic (several months, every status, time-zone
boundaries, a DST change); RLS tests that a second clinic's scope reads none of the first's rows; the job is
idempotent. Benchmark: refresh of one clinic-day and a yearly overview query on 3 years × 20 practitioners of
synthetic data; numbers in the PROGRESS file.

## 5. Platform anonymization rules (SAL-21)

`trends_cell` (owned by `trends`; no clinic, practitioner or patient column): `week` (ISO week start), `region`,
`city`, `specialty_code`, `group_code`, `subgroup_code`, `code`, `count`, `clinics` (distinct contributing
clinics), `top_clinic_count` (the largest single clinic's count). `trends_hour_cell`: `year`, `quarter`,
`region`, `weekday`, `hour`, `count`, `clinics`, `top_clinic_count`, from appointments (fulfilled). Built nightly
after `clinicstats_nightly` for the last 10 weeks; `salusctl trends rebuild` rebuilds all.

| Rule | |
| --- | --- |
| A1 | No identifiers: no clinic, practitioner, patient, appointment, encounter or document id or name in the tables, the service's types or the JSON. A test walks the response JSON and fails on any key outside an allow-list |
| A2 | Geography is the clinic's region and city only; nothing finer, never the patient's address |
| A3 | Time buckets: ISO week, month, quarter, year. A range is whole buckets, at most 3 years back. No day or hour series; the weekday-by-hour heatmap is per quarter or year |
| A4 | Fixed dimensions: period, region, city, specialty, group, subgroup, code. A query filters on any of them and breaks down by one. No age, sex, clinic or practitioner dimension |
| A5 | Minimum count: a cell is published only if its raw count is at least `k`. `k` is `ANALYTICS_MIN_COUNT`, default 10; config refuses a value below 10 |
| A6 | Clinics: a cell is published only if at least 3 clinics contributed and the largest contributed at most 80% |
| A7 | Complement: a published cell's one-step relaxation (drop one filter, or go one level up: code → subgroup → group → all; city → region → all; week → month) minus the cell is 0 or at least `k`, else the cell is suppressed |
| A8 | Siblings: in a breakdown, the suppressed siblings together are 0 or at least `k`; if not, the next smallest sibling is suppressed too, until they are or the whole breakdown is suppressed |
| A9 | Totals are sums of published cells only, marked `partial` when anything was suppressed; never a raw total |
| A10 | Published counts are rounded to the nearest 5 (question O4) |
| A11 | Excluded at the source: `entered-in-error` documents and appointments, restricted records (Art. 18), erased records (gone) |
| A12 | Rises are shown only on a published cell, with its rounded count and no baseline numbers below `k` |
| A13 | Logs of the build carry counts of cells and of suppressed cells only |

`trends.Suppress(query, cells, k) []Published` implements A5 to A10 as one pure function, applied at query time.
Tests: table-driven cases for each rule; a property test that, for random small synthetic data and every pair
of allowed queries, no difference of two published numbers is between 1 and `k-1`; the JSON allow-list test.

## 6. Unusual rises

Per ISO week `w`, region and diagnosis group: baseline = the counts of weeks `w-9` to `w-2` (8 weeks, skipping
`w-1` so a rise that started last week does not raise its own baseline); mean `μ`, standard deviation `σ`,
`σ' = max(σ, √μ, 1)`. Flag when `count ≥ k`, `count ≥ 1.5·μ` and `count > μ + 3σ'`. Computed nightly on the raw
internal cells (`trends_rise`: week, region, group), shown only per A12. Tests on synthetic series: a flat series
is never flagged, a step to three times the level is flagged in its first week, a slow seasonal ramp of 10% a
week is not, a low-count spike below `k` is not. Copy: "Unusual rise", with a tooltip that it compares with the
previous weeks and is not a public-health alert.

## 7. Personal data and DPIA note

Added to `docs/privacy/dpia.md` by each package (section "Analytics (`docs/plans/analytics.md`)"):

- **Processing added.** `condition`: health data (Art. 9) of identified patients, part of the clinical record.
  `clinic_*_stat`: counts per clinic, day, hour, practitioner and code: no patient data, but personal data of
  practitioners (their workload and diagnoses), and small cells can single out a patient to someone who also
  sees the calendar (hence C3). `trends_*`: counts with no identifier, published only under A1 to A13.
- **Purpose and lawful basis.** The clinic's statistics serve the management of its health-care services
  (Art. 9(2)(h), on the clinic's behalf as controller, Salus as processor). Platform-wide figures are a new
  purpose for which Salus would act as controller of the anonymisation; the basis is open (L1, L2). Not used for
  marketing, profiling or model training.
- **Minimisation.** No age, sex or residence. Region and city are the clinic's. `salus_stats` cannot read a
  patient id. No free text.
- **Retention.** `condition`: the clinical record's rule. `clinic_*_stat`: as long as their source rows, refreshed
  on erasure and retention deletes; rows older than the clinical record's period are deleted. `trends_*`: 5 years
  rolling (anonymous output, kept bounded anyway).
- **Risks and measures.** Re-identification through small counts, differencing or a single clinic in a city:
  A5 to A10. A practitioner learning about patients outside their care through clinic figures: C2, C3.
  Outbreak flags taken as official: copy and question L6.

## 8. API contract (for SAL-22's mocks)

Plain JSON in `api/openapi.yaml` (not FHIR). ANA-2 and ANA-3 add their paths; ANA-4 types its fixtures from
these shapes and switches to the generated client once both are merged.

```
GET /analytics/clinic/appointments?from=2026-01&to=2026-12&practitioner_role=&specialty=
  -> { "series":  [{ "period": "2026-01", "booked": 120, "fulfilled": 98, "noshow": 6, "cancelled": 16 }],
       "heatmap": [{ "weekday": 1, "hour": 9, "count": 14 }],             // ISO weekday 1 = Monday
       "practitioners": [{ "practitioner_role": "<uuid>", "fulfilled": 40 }] }  // admins only
GET /analytics/clinic/diagnoses?from=&to=&group=&subgroup=&specialty=&practitioner_role=&by=group|subgroup|code|practitioner_role|specialty|period
  -> { "rows": [{ "key": "respiratory-infections", "count": 31 }, { "key": "J00", "count": "<5" }],
       "unmapped_documents": 2 }
GET /analytics/platform/diagnoses?from=2026-W01&to=2026-W45&granularity=week|month|quarter|year
                                 &region=&city=&specialty=&group=&subgroup=&by=period|region|city|specialty|group|subgroup|code
  -> { "rows": [{ "key": "BA-09", "count": 45 }, { "key": "BA-05", "count": null }],  // null: suppressed
       "partial": true, "min_count": 10 }
GET /analytics/platform/heatmap?year=2026&quarter=&region=
  -> { "cells": [{ "weekday": 1, "hour": 9, "count": 35 }], "partial": false, "min_count": 10 }
GET /analytics/platform/rises?region=&weeks=8
  -> { "rises": [{ "week": "2026-W44", "region": "BA-09", "group": "respiratory-infections", "count": 60 }] }
GET /terminology/diagnosis-groups            -> [{ "code": "common-cold", "parent": "respiratory-infections", "name": "…" }]
GET /places?country=BA                       -> [{ "id": "Q…", "region": "BA-09", "name": "…" }]
```

Codes go out, names come from the groups and places endpoints and the web catalog; the API returns no sentences.

## 9. Work packages

```
ANA-0 (this PR): ADR 0012 and this plan. Stops for the owner's approval.
ANA-1 SAL-19, alone, by the lead session: condition, groups, confirmation, the source methods, salus_stats.
Wave 2, in parallel after ANA-1 is merged: ANA-2 SAL-20   ANA-3 SAL-21   ANA-4 SAL-22 (mocked data first)
Merge order: ANA-2, ANA-3, ANA-4. Before each merge: main merged into the branch (no force-push), migrations
renumbered if main has taken their numbers (another session is building chat), checks rerun.
```

Each wave-2 package runs as a subagent in its own git worktree and branch, with its own compose project
(`-p salus-ana2`, `-p salus-ana3`, `-p salus-ana4`) and no published host ports. Every package: strict TDD, Docker
only, `make lint`, `make test`, `make archcheck`, `make e2e`, then a review by a fresh subagent (Copilot's quota is
out until 2026-12-01), valid findings fixed, then merged. PROGRESS: `docs/plans/analytics.PROGRESS.md`.

Migrations take the next free number when the branch is merged; main is at `00040` today. Shared files the
wave-2 packages all touch (`api/openapi.yaml`, the generated clients, `ownership.yaml`, `config.yaml`,
`internal/retention/rules.go`, `internal/api/app.go`, the router, `internal/worker`) are merged by hand in merge
order; generated clients are regenerated after the merge, never hand-merged.

### ANA-1 SAL-19 diagnoses (branch `analytics-diagnoses`)

- Migration: `diagnosis_group*` with appendix A seeded (names with review status `pending`), `condition` with RLS
  and the backfill, the `salus_stats` role (as 00014, `STATS_DATABASE_PASSWORD` set by `cmd/migrate`) and its
  column grants on `condition`, `appointment`, `practitioner_role` and `organization`.
- `internal/condition`: model, repository, service (`Record` in the caller's transaction, `Recorded`, `Get`),
  FHIR read, golden JSON, audit, retention rule; ownership and archcheck entries.
- `terminology`: `Group(code)`, the groups endpoint, the review panel's group list (web, five staff languages).
- `composition`: sign refuses an unconfirmed suggested coded value (`409`, code `diagnosis_unconfirmed`); writes
  the `Condition`s through `condition.Service` in the signing transaction; entered-in-error marks them too. Web
  document editor: a confirm control on a suggested code.
- `appointment.Service.Starts`, `organization.Service.Locations` (id and time zone), the `salus_stats` pool in
  `internal/config` and `internal/worker`.
- `salusctl conditions unmapped`. E2E: the record journey's signed document has a `Condition`.

### ANA-2 SAL-20 clinic statistics (branch `analytics-clinic`)

Section 4: tables, the two jobs and their enqueues, `internal/clinicstats` (service rules C1 to C4, JSON
handlers, openapi paths), tests including the RLS and equality tests, the benchmark, DPIA and retention entries.

### ANA-3 SAL-21 platform trends (branch `analytics-platform`)

Clinic location: `organization` region and city (migration, FHIR `Organization.address.state` and `.city`,
the admin's Clinic settings field), places seeded from Wikidata (CC0, fetched with `tools/terminology`'s fetcher
into `assets/places/`, the 13 ISO 3166-2:BA subdivisions with our names in ten languages). Then section 5 and 6:
`internal/trends`, the nightly build, `Suppress`, the rises, the handlers and openapi paths,
`ANALYTICS_MIN_COUNT` in `internal/config`, the DPIA section with the legal basis left open as L1 and L2 say.

### ANA-4 SAL-22 charts (branch `analytics-charts`)

`web/src/analytics/`: an Analytics page with "My clinic" and "All of Salus", SVG line, bar and heatmap components
with tooltips, keyboard focus and `prefers-reduced-motion`; filters (group with drill-down, range, region,
specialty, practitioner on My clinic only); dates and numbers through `Intl` in the UI language, five staff
languages; suppressed cells shown as "Fewer than 10" (platform) and "Fewer than 5" (clinic). Built against typed
fixtures of section 8 first; after ANA-2 and ANA-3 are merged, the generated client replaces the fixtures and an
e2e test switches views and filters. Component tests in Vitest. A render benchmark on 3 years of weekly data.

## 10. Open questions

### For the owner

- **O1 Geography.** Clinic location (proposed) or the patient's municipality of residence (better for spotting
  regional rises in telemedicine, but a new personal-data field in the mobile app and the profile)?
- **O2 Clinics outside BiH.** Places are seeded for BiH only; another country is a later migration. Fine?
- **O3 Small cells in My clinic.** Show diagnosis counts 1 to 4 as "fewer than 5" for everyone (proposed), only
  for practitioners, or exactly?
- **O4 Rounding.** Round platform counts to the nearest 5 (proposed), or show exact counts above 10?
- **O5 Who sees All of Salus.** Every staff account (proposed), or admins only?
- **O6 Clinic opt-out.** May a clinic decline to contribute to the platform figures (a contract question too)?
- **O7 Group list.** Appendix A's groups and names; a doctor should review the names before they leave `pending`.

### For a lawyer (before real patient data reaches the platform view)

- **L1 Role.** Salus processes clinics' data as their processor. Building platform-wide figures is Salus's own
  purpose: does the anonymisation step need the clinics' instruction in the DPA (Art. 28(3)(a)), or does Salus
  become a controller for it, and with which contract clauses?
- **L2 Lawful basis** for that step, for health data: Art. 9(2)(h), (i) or (j) with Art. 89 and the purpose
  compatibility test of Art. 6(4), or the patient's consent; and the same under the BiH personal data protection
  law in force. If consent is needed, patients without it must be left out of the figures (a new `Consent`
  category).
- **L3 Anonymous or not.** Are the published figures anonymous in the sense of Recital 26 under rules A1 to A13
  (k = 10, 3 clinics, complement and sibling suppression, rounding)? Are the internal `trends_cell` rows
  (before suppression) personal data, and which retention do they need?
- **L4 Transparency and objection.** What the patients' privacy notice must say about statistics (Art. 13,
  14), and whether an objection (Art. 21) must remove a patient from the figures.
- **L5 Practitioners.** Per-practitioner figures are employee data: may the clinic admin see each
  practitioner's diagnoses and workload, and what must practitioners be told?
- **L6 Rises.** Does a rise flag create or touch any duty under the entity laws on protection against
  infectious diseases (notifiable diseases), and is the disclaimer enough?
- **L7 ICD-10.** Showing ICD-10 codes and grouping them by ICD-10 block ranges without WHO text: permitted under
  WHO's terms without a licence (the terminology's open question, extended to block ranges as a structure)?
  Does any official use of our documents in BiH (sick leave, referrals) require the national MKB-10 edition?
- **L8 DPIA.** Whether this feature needs consultation of the supervisory authority (Art. 36) after the DPIA.

## Appendix A. Diagnosis groups (draft names, `pending` review)

Ranges are inclusive and compared on the code's first characters (`J00` matches `J00` and `J00.0`). A subgroup
(indented key) lies inside its group. U07 is WHO's emergency code for COVID-19. V01–Y98 (external causes) and
any code outside every range map to `other`.

| Key | Ranges | en | bs | hr | sr-Latn | sr-Cyrl | de | ar | tr | it | fr |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `intestinal-infections` | A00–A09 | Intestinal infections | Crijevne infekcije | Crijevne infekcije | Crijevne infekcije | Цријевне инфекције | Darminfektionen | عدوى الأمعاء | Bağırsak enfeksiyonları | Infezioni intestinali | Infections intestinales |
| ↳ `gastroenteritis` | A08–A09 | Gastroenteritis | Gastroenteritis | Gastroenteritis | Gastroenteritis | Гастроентеритис | Gastroenteritis | التهاب المعدة والأمعاء | Gastroenterit | Gastroenterite | Gastro-entérite |
| `other-infections` | A15–B99 | Other infectious diseases | Druge zarazne bolesti | Druge zarazne bolesti | Druge zarazne bolesti | Друге заразне болести | Andere Infektionskrankheiten | أمراض معدية أخرى | Diğer bulaşıcı hastalıklar | Altre malattie infettive | Autres maladies infectieuses |
| `covid-19` | U07.1–U07.2 | COVID-19 | COVID-19 | COVID-19 | COVID-19 | COVID-19 | COVID-19 | كوفيد-19 | COVID-19 | COVID-19 | COVID-19 |
| `tumours` | C00–D48 | Tumours | Tumori | Tumori | Tumori | Тумори | Tumoren | الأورام | Tümörler | Tumori | Tumeurs |
| `blood-immune` | D50–D89 | Blood and immune disorders | Bolesti krvi i imuniteta | Bolesti krvi i imunosti | Bolesti krvi i imuniteta | Болести крви и имунитета | Blut- und Immunerkrankungen | اضطرابات الدم والمناعة | Kan ve bağışıklık hastalıkları | Malattie del sangue e del sistema immunitario | Maladies du sang et du système immunitaire |
| `diabetes` | E10–E14 | Diabetes | Dijabetes | Šećerna bolest | Dijabetes | Дијабетес | Diabetes | داء السكري | Diyabet | Diabete | Diabète |
| `endocrine-metabolic` | E00–E07, E15–E90 | Thyroid, hormone and metabolic disorders | Bolesti štitne žlijezde, hormona i metabolizma | Bolesti štitnjače, hormona i metabolizma | Bolesti štitne žlijezde, hormona i metabolizma | Болести штитне жлијезде, хормона и метаболизма | Schilddrüsen-, Hormon- und Stoffwechselerkrankungen | اضطرابات الغدة الدرقية والهرمونات والاستقلاب | Tiroid, hormon ve metabolizma hastalıkları | Malattie della tiroide, ormonali e metaboliche | Maladies thyroïdiennes, hormonales et métaboliques |
| `mental-health` | F00–F99 | Mental health | Mentalno zdravlje | Mentalno zdravlje | Mentalno zdravlje | Ментално здравље | Psychische Gesundheit | الصحة النفسية | Ruh sağlığı | Salute mentale | Santé mentale |
| `nervous-system` | G00–G99 | Nervous system disorders | Bolesti nervnog sistema | Bolesti živčanog sustava | Bolesti nervnog sistema | Болести нервног система | Erkrankungen des Nervensystems | أمراض الجهاز العصبي | Sinir sistemi hastalıkları | Malattie del sistema nervoso | Maladies du système nerveux |
| `eye` | H00–H59 | Eye conditions | Bolesti oka | Bolesti oka | Bolesti oka | Болести ока | Augenerkrankungen | أمراض العين | Göz hastalıkları | Malattie dell'occhio | Maladies de l'œil |
| `ear` | H60–H95 | Ear conditions | Bolesti uha | Bolesti uha | Bolesti uha | Болести уха | Ohrerkrankungen | أمراض الأذن | Kulak hastalıkları | Malattie dell'orecchio | Maladies de l'oreille |
| `high-blood-pressure` | I10–I15 | High blood pressure | Povišen krvni pritisak | Povišeni krvni tlak | Povišen krvni pritisak | Повишен крвни притисак | Bluthochdruck | ارتفاع ضغط الدم | Yüksek tansiyon | Ipertensione | Hypertension artérielle |
| `heart-circulation` | I00–I09, I20–I99 | Heart and circulation disorders | Bolesti srca i krvnih žila | Bolesti srca i krvnih žila | Bolesti srca i krvnih sudova | Болести срца и крвних судова | Herz- und Kreislauferkrankungen | أمراض القلب والدورة الدموية | Kalp ve dolaşım hastalıkları | Malattie del cuore e della circolazione | Maladies du cœur et de la circulation |
| `respiratory-infections` | J00–J22 | Respiratory infections | Respiratorne infekcije | Infekcije dišnog sustava | Respiratorne infekcije | Респираторне инфекције | Atemwegsinfektionen | التهابات الجهاز التنفسي | Solunum yolu enfeksiyonları | Infezioni respiratorie | Infections respiratoires |
| ↳ `common-cold` | J00 | Common cold | Prehlada | Prehlada | Prehlada | Прехлада | Erkältung | نزلة البرد | Soğuk algınlığı | Raffreddore | Rhume |
| ↳ `upper-airway-infections` | J01–J06 | Sinus, throat and other upper airway infections | Upale sinusa, grla i gornjih disajnih puteva | Upale sinusa, grla i gornjih dišnih putova | Upale sinusa, grla i gornjih disajnih puteva | Упале синуса, грла и горњих дисајних путева | Infektionen der Nasennebenhöhlen, des Halses und der oberen Atemwege | التهابات الجيوب الأنفية والحلق والمجاري التنفسية العليا | Sinüs, boğaz ve diğer üst solunum yolu enfeksiyonları | Infezioni dei seni paranasali, della gola e delle vie aeree superiori | Infections des sinus, de la gorge et des voies respiratoires supérieures |
| ↳ `flu` | J09–J11 | Flu | Gripa | Gripa | Grip | Грип | Grippe | الإنفلونزا | Grip | Influenza | Grippe |
| ↳ `pneumonia` | J12–J18 | Pneumonia | Upala pluća | Upala pluća | Upala pluća | Упала плућа | Lungenentzündung | الالتهاب الرئوي | Zatürre | Polmonite | Pneumonie |
| ↳ `chest-infections` | J20–J22 | Bronchitis and other chest infections | Bronhitis i druge infekcije donjih disajnih puteva | Bronhitis i druge infekcije donjih dišnih putova | Bronhitis i druge infekcije donjih disajnih puteva | Бронхитис и друге инфекције доњих дисајних путева | Bronchitis und andere Infektionen der unteren Atemwege | التهاب القصبات والتهابات الصدر الأخرى | Bronşit ve diğer alt solunum yolu enfeksiyonları | Bronchite e altre infezioni delle basse vie respiratorie | Bronchite et autres infections des voies respiratoires inférieures |
| `asthma-chronic-lung` | J40–J47 | Asthma and chronic lung disease | Astma i hronične bolesti pluća | Astma i kronične bolesti pluća | Astma i hronične bolesti pluća | Астма и хроничне болести плућа | Asthma und chronische Lungenerkrankungen | الربو وأمراض الرئة المزمنة | Astım ve kronik akciğer hastalıkları | Asma e malattie polmonari croniche | Asthme et maladies pulmonaires chroniques |
| `other-respiratory` | J30–J39, J60–J99 | Other respiratory conditions | Druge bolesti disajnih organa | Druge bolesti dišnog sustava | Druge bolesti disajnih organa | Друге болести дисајних органа | Andere Atemwegserkrankungen | أمراض تنفسية أخرى | Diğer solunum yolu hastalıkları | Altre malattie respiratorie | Autres maladies respiratoires |
| `digestive` | K00–K93 | Digestive disorders | Bolesti probavnog sistema | Bolesti probavnog sustava | Bolesti sistema za varenje | Болести система за варење | Erkrankungen des Verdauungssystems | أمراض الجهاز الهضمي | Sindirim sistemi hastalıkları | Malattie dell'apparato digerente | Maladies de l'appareil digestif |
| `skin` | L00–L99 | Skin conditions | Bolesti kože | Bolesti kože | Bolesti kože | Болести коже | Hauterkrankungen | الأمراض الجلدية | Cilt hastalıkları | Malattie della pelle | Maladies de la peau |
| `musculoskeletal` | M00–M99 | Muscles, bones and joints | Bolesti mišića, kostiju i zglobova | Bolesti mišića, kostiju i zglobova | Bolesti mišića, kostiju i zglobova | Болести мишића, костију и зглобова | Muskel-, Knochen- und Gelenkerkrankungen | أمراض العضلات والعظام والمفاصل | Kas, kemik ve eklem hastalıkları | Malattie di muscoli, ossa e articolazioni | Maladies des muscles, des os et des articulations |
| ↳ `back-neck` | M40–M54 | Back and neck problems | Problemi s leđima i vratom | Problemi s leđima i vratom | Problemi sa leđima i vratom | Проблеми са леђима и вратом | Rücken- und Nackenbeschwerden | مشكلات الظهر والرقبة | Sırt ve boyun sorunları | Disturbi della schiena e del collo | Troubles du dos et du cou |
| `kidney-urinary` | N00–N39 | Kidney and urinary tract | Bolesti bubrega i mokraćnih puteva | Bolesti bubrega i mokraćnog sustava | Bolesti bubrega i mokraćnih puteva | Болести бубрега и мокраћних путева | Nieren- und Harnwegserkrankungen | أمراض الكلى والمسالك البولية | Böbrek ve idrar yolu hastalıkları | Malattie dei reni e delle vie urinarie | Maladies des reins et des voies urinaires |
| `reproductive-breast` | N40–N99 | Reproductive and breast conditions | Bolesti spolnih organa i dojke | Bolesti spolnih organa i dojke | Bolesti polnih organa i dojke | Болести полних органа и дојке | Erkrankungen der Geschlechtsorgane und der Brust | أمراض الأعضاء التناسلية والثدي | Üreme organları ve meme hastalıkları | Malattie degli organi genitali e della mammella | Maladies des organes génitaux et du sein |
| `pregnancy` | O00–O99 | Pregnancy and childbirth | Trudnoća i porod | Trudnoća i porod | Trudnoća i porođaj | Трудноћа и порођај | Schwangerschaft und Geburt | الحمل والولادة | Gebelik ve doğum | Gravidanza e parto | Grossesse et accouchement |
| `newborn-congenital` | P00–Q99 | Newborn and congenital conditions | Stanja novorođenčadi i urođene anomalije | Stanja novorođenčadi i prirođene anomalije | Stanja novorođenčadi i urođene anomalije | Стања новорођенчади и урођене аномалије | Erkrankungen bei Neugeborenen und angeborene Fehlbildungen | حالات حديثي الولادة والتشوهات الخلقية | Yenidoğan hastalıkları ve doğuştan anomaliler | Condizioni neonatali e malformazioni congenite | Affections néonatales et malformations congénitales |
| `symptoms` | R00–R99 | Symptoms without a diagnosis | Simptomi bez postavljene dijagnoze | Simptomi bez postavljene dijagnoze | Simptomi bez postavljene dijagnoze | Симптоми без постављене дијагнозе | Symptome ohne Diagnose | أعراض دون تشخيص | Tanı konmamış belirtiler | Sintomi senza diagnosi | Symptômes sans diagnostic |
| `injuries` | S00–T98 | Injuries and poisoning | Povrede i trovanja | Ozljede i trovanja | Povrede i trovanja | Повреде и тровања | Verletzungen und Vergiftungen | الإصابات والتسممات | Yaralanmalar ve zehirlenmeler | Lesioni e avvelenamenti | Blessures et intoxications |
| `checkups` | Z00–Z99 | Check-ups and other visits | Pregledi i druge posjete | Pregledi i drugi posjeti | Pregledi i druge posjete | Прегледи и друге посјете | Vorsorge und andere Anlässe | الفحوصات والزيارات الأخرى | Kontroller ve diğer başvurular | Controlli e altre visite | Bilans de santé et autres consultations |
| `other` | everything else | Other | Ostalo | Ostalo | Ostalo | Остало | Sonstige | أخرى | Diğer | Altro | Autres |

Mapping examples the ANA-1 tests pin: `J00` → respiratory-infections / common-cold; `J06.9` → respiratory-
infections / upper-airway-infections; `J45.0` → asthma-chronic-lung; `J30.1` → other-respiratory; `I10` →
high-blood-pressure; `E11.9` → diabetes; `E03.9` → endocrine-metabolic; `M54.5` → musculoskeletal / back-neck;
`A09` → intestinal-infections / gastroenteritis; `U07.1` → covid-19; `W19` → other; `Z00.0` → checkups.
