# Analytics: clinic statistics and platform-wide trends

Epic SAL-18 (SAL-19 to SAL-22). Status: approved by the owner, 2026-11-12, with the answers in section 10; the
platform view is built behind a feature flag that stays off until the legal questions are answered (section 2a). Decision record: docs/adr/0013-icd10-diagnosis-groups-and-anonymized-analytics.md. Revised the
same day after two subagent reviews of PR #71 (suppression fixed over the whole month instead of per query,
distinct patients, frozen months, the clinic rules, features that do not exist yet taken out).

As built so far: docs/plans/analytics.PROGRESS.md records where a package departs from this plan.

## 1. Goal

- **My clinic** (SAL-20): a clinic's own appointments and diagnoses over time, its busiest months, weekdays and
  hours, its top diagnosis groups, broken down by practitioner and specialty.
- **All of Salus** (SAL-21): combined figures across all clinics by diagnosis group, region, city and month,
  with no clinic and no patient identifiable, and a flag on unusual rises of a diagnosis group in a region.
- Both rest on **coded diagnoses** (SAL-19): the ICD-10 codes of signed documents, confirmed by the signing
  practitioner, kept as FHIR `Condition`, grouped into our own diagnosis groups based on ICD-10 blocks.
- Both are shown as **interactive charts** in `web/` (SAL-22).

Data sources: `Condition` (from signed documents) and `Appointment`. Never transcripts, suggestions, free-text
fields or questionnaire answers.

## 2. Decisions (approved 2026-11-12)

| Topic | Decision | Why |
| --- | --- | --- |
| ICD-10 source | Codes as identifiers only (from the doctor and from the terminology's Wikidata items, CC0); no WHO or national-edition text in any language | The only option legally usable without a licence; extends the terminology's settled decision (docs/terminology/SPEC.md "Data" and §7) |
| Groups | Our own 27 groups and 7 subgroups defined by ICD-10 code ranges on block boundaries, names written by us in all ten languages (appendix A), reviewed in the term review panel | Ranges are facts; names are ours, so no WHO titles are adapted or translated |
| Where a diagnosis comes from | A template's `coded` field is a diagnosis. Signing writes one `Condition` per filled coded field, in the signing transaction | The field type exists; no second way to record a diagnosis |
| Confirmation | The sign request's value gets `confirmed` (`{key, value, confirmed}`); signing refuses a filled coded field without `confirmed: true` (`409`, code `diagnosis_unconfirmed`). The document editor shows a suggested code with a confirm control and sends `confirmed` only when the practitioner confirmed it or typed the code. Nothing new is stored in `document_draft.fields` | SAL-19 "the drafting service suggests a code, the doctor confirms"; stateless |
| Backfill | In the migration, in SQL: every coded value of an already signed (`final`) document becomes a `Condition` with `verification_status` `unconfirmed`. Analytics counts only `confirmed`. "Flagged" (SAL-19) = `unconfirmed`; `salusctl conditions unconfirmed` lists their ids | Signing already refuses malformed codes, so every old value is well-formed, but nobody confirmed it. Nothing is deployed, so this only touches dev data |
| Clinic aggregates | Pre-computed per clinic, local day and hour; an incremental River job per clinic-day, enqueued in the transaction of every signing and appointment write; nightly recompute of the last 35 days | Charts read small tables; the nightly pass repairs a lost job |
| Platform figures | Closed calendar months only, built once seven days after the month ends and then frozen; a fixed set of two-dimensional tables per month, suppressed as a whole (section 5) | Per-query suppression can be undone by combining queries; nightly rebuilds of open periods leak daily changes |
| What a platform count counts | Distinct patients, through a per-build pseudonym (HMAC-SHA256 of the patient id under a key that lives only during the build) | A chronic patient seen weekly must not reach `k` alone |
| Cross-clinic reads | A `salus_stats` database role (BYPASSRLS, column-level SELECT on source columns only), used only by the build jobs in `cmd/worker` | Mirrors `salus_retention` (00014) |
| Geography | The clinic's location: a leaf region (the 10 FBiH cantons, Republika Srpska, Brčko District: 12 ISO 3166-2:BA codes, with the entity as a parent level) and a city (a Wikidata municipality item), set by the clinic admin in Clinic settings | No new patient data; see O1 |
| Unusual rises | Weekly EARS C2 adaptation per region and group on closed weeks; a flag has no count (section 6) | Explainable, needs 10 weeks of history, publishes no weekly number |
| Charts | Hand-written SVG components in `web/src/analytics/` (line, bars, heatmap); regions as bars, no map in v1 | A chart library is a large dependency for four chart types; a map needs boundary data with its own licence and weight |
| Domains | `condition` (context `clinical`); `clinicstats` and `trends` (new context `analytics`); groups and places in `terminology` (`reference`) | CLAUDE.md "Go architecture" |
| Who sees what | My clinic: admins see all of their clinic, the clinic's full diagnosis totals included; a practitioner sees the clinic's appointment figures and the diagnoses of their own role only. All of Salus: every staff account of a clinic that contributes (O6) | SAL-20 "admins and doctors per permission"; a practitioner learns nothing about diagnoses of patients outside their care |
| Clinic opt-out | A clinic admin may opt the clinic out of the platform figures (`organization.analytics_contributes`, default true, in Clinic settings). An opted-out clinic's rows are left out of every build from the next one on (frozen months stay as built) and its staff lose the All of Salus view (`403`, code `analytics_not_contributing`) | Owner, O6: no view without contributing |

## 2a. Feature flag: the platform view is off until the legal answers (owner, 2026-11-12)

Questions L1 to L8 are with the owner's lawyer. Everything is built and tested, but `ANALYTICS_PLATFORM_ENABLED`
(`internal/config`, default `false`, read once at start) gates the platform view:

- Off: the scheduler registers none of the `trends` builds (month, quarter, week), `salusctl trends build`
  refuses, every `/analytics/platform/*` path answers `404` (the routes are not mounted), and the web app hides
  the All of Salus tab (it learns the flag from `GET /auth/me`'s `features`, a list of codes). No platform cell
  or rise is ever computed, so no data is processed for that purpose.
- On: only after the owner confirms the legal answers, as a change of the production `.env` recorded in this
  plan's PROGRESS file. Tests run both ways; the e2e suite turns it on for its own stack.
- My clinic (SAL-20) is not behind the flag: it is the clinic's own processing on its behalf.

CLAUDE.md also lists the bounded contexts. The owner has an uncommitted edit to CLAUDE.md, so no PR of this
epic touches it: `docs/architecture/archcheck/config.yaml` (the authority) gets the new context, and the owner
adds `analytics     clinicstats, trends` and `condition` under `clinical` to CLAUDE.md when convenient.

## 3. Coded diagnoses and groups (SAL-19)

**Condition** (`internal/condition`, table `condition`, owned by `condition`):

| Column | Why |
| --- | --- |
| `id`, `created_at`, `updated_at`, `version_id` | FHIR id and meta |
| `organization_id` | Clinic scope (RLS), aggregates |
| `patient_id` | `Condition.subject`; the patient's own record; distinct-patient counts (pseudonymised before it leaves the package) |
| `encounter_id` | `Condition.encounter` |
| `document_draft_id`, `field_key` | The signed document and field it came from (`Condition.evidence.detail` → `Composition`); unique together |
| `practitioner_role_id` | `Condition.recorder`; breakdown by practitioner and specialty |
| `recorded_at` | `Condition.recordedDate`, the signing time |
| `code` | `Condition.code.coding[0]`, system `http://hl7.org/fhir/sid/icd-10` |
| `verification_status` | `confirmed` from signing; `unconfirmed` from the backfill |

`clinicalStatus` is not modelled (a signed document says what was diagnosed at that visit). Category
`encounter-diagnosis`. RLS: the clinic's scope, and the patient's own scope. `GET /fhir/R4/Condition/{id}`
(CapabilityStatement updated), audited. Retention: kept with the clinical record (`ThenKeep`, as
`document_draft`). A signed document never changes, so its Conditions never change.

**Groups** (owned by `terminology`): `diagnosis_group` (`code` a stable slug such as `respiratory-infections`,
`parent_code` for a subgroup, `position`), `diagnosis_group_range` (`group_code`, `from_code`, `to_code`,
inclusive, compared on the code's first `length(from_code)` characters), `diagnosis_group_name` (`group_code`,
`language`, `text`, review columns as `term_translation`). `terminology.Group(code) (group, subgroup)` is a pure
function over ranges loaded once; tests prove the ranges are disjoint at each level, every subgroup lies inside
its group, and appendix A's examples map as listed. `GET /terminology/diagnosis-groups` returns codes, names in
the requested language and the parent. The review panel gets a "Diagnosis groups" list beside terms.

**Places** (owned by `terminology`, ANA-1 so that wave 2 shares it): `place` (`id` a Wikidata QID or an ISO
3166-2 code, `kind` entity, region or city, `parent_id`), `place_name` (`language`, `text`). The 3 entities and
12 leaf regions with our names in ten languages; cities fetched once from Wikidata (CC0) into
`assets/places/ba.jsonl` with `tools/terminology`'s fetcher. `GET /places?country=BA`.

**Contract for wave 2** (built in ANA-1, reached only through the services; no wave-2 package changes these):

```go
// condition: one row per confirmed Condition of a clinic recorded in [from, to).
func (s *Service) Recorded(ctx context.Context, orgID string, from, to time.Time, key []byte) ([]Recorded, error)
type Recorded struct { RecordedAt time.Time; Patient [32]byte; PractitionerRoleID, Code string }

// appointment: one row per appointment of a clinic starting in [from, to).
func (s *Service) Starts(ctx context.Context, orgID string, from, to time.Time, key []byte) ([]Start, error)
type Start struct { Start time.Time; Patient [32]byte; PractitionerRoleID, Status string }

// practitionerrole: every role of a clinic with its specialty (SNOMED CT code, empty if none).
func (s *Service) Specialties(ctx context.Context, orgID string) (map[string]string, error)

// organization: every clinic with its time zone and, once ANA-3 adds them, its region and city.
func (s *Service) Locations(ctx context.Context) ([]Location, error)
type Location struct { LocationSince, ContributesSince time.Time; ID, TimeZone, Region, City string; Contributes bool }

// The enqueue port each producer declares (appointment, composition), so none imports clinicstats:
type StatsQueue interface { EnqueueStatsRefresh(ctx context.Context, orgID string, day time.Time) error }
```

`Patient` is HMAC-SHA256(key, patient id); `clinicstats` passes a nil key and gets a zero value (it does not
count patients); `trends` passes the build's key. Before ANA-1 is merged, `StatsQueue` is implemented by a no-op
in `internal/api/app.go`; ANA-2 replaces it with `clinicstats`' River enqueue. `Region`, `City`, `LocationSince`,
`Contributes` and `ContributesSince` are zero until ANA-3 (no clinic contributes before ANA-3, so no build counts anything).

## 4. Clinic statistics (SAL-20)

Tables (owned by `clinicstats`, RLS by `organization_id`, written only by `salus_stats`):

- `clinic_appointment_stat`: `organization_id`, `day` (local date), `hour` (0–23 local), `practitioner_role_id`,
  `specialty_code`, `status`, `count`.
- `clinic_diagnosis_stat`: `organization_id`, `day`, `hour`, `practitioner_role_id`, `specialty_code`,
  `group_code`, `subgroup_code`, `code`, `count` (confirmed Conditions).

Jobs: `clinicstats_refresh{organization_id, day}` (opaque ids; recomputes that clinic-day in one transaction),
enqueued through `StatsQueue` by signing and by every appointment write (booking, cancellation, status change);
`clinicstats_nightly` (scheduler, 02:30 UTC) recomputes the last 35 days of every clinic. The specialty is taken
at refresh time. Erasure and restriction do not exist yet; when `datarequest` is built it enqueues the refresh
for the affected days (noted in its future SPEC, not built here).

Rules in the service: C1 a caller sees only their own clinic (RLS and the service check); C2 a practitioner who
is not an admin gets the clinic's appointment figures and their own role's diagnosis figures, never another
practitioner's breakdown and never the clinic's diagnosis totals; admins get everything of their clinic, counts
exact; C3 every read writes an `AuditEvent` (entity `Organization/<id>`, purpose analytics).

Tests: aggregates equal the raw counts for a generated synthetic clinic (months of data, every status, local-day
boundaries, a DST change); RLS tests that one clinic's scope reads none of another's rows; the refresh is
idempotent; C2 per role. Benchmark: one clinic-day refresh and a yearly overview on 3 years × 20 practitioners
of synthetic data; numbers in the PROGRESS file.

## 5. Platform anonymization (SAL-21)

**What is published.** Distinct-patient counts, built once per closed period and then frozen:

| Table | Dimensions (each with an "all" level) | Source | Period, built |
| --- | --- | --- | --- |
| G×D | geography (all → entity → region → city) × diagnosis (all → group → subgroup → code) | Confirmed Conditions | Calendar month, on the 8th of the next month |
| G×S | geography × specialty | Confirmed Conditions | Same |
| S×D | specialty × diagnosis, geography = all only | Confirmed Conditions | Same |
| Heatmap | geography (all → entity → region) × weekday (all → day) × hour (all → hour) | Fulfilled appointments | Calendar quarter, on the 8th after the quarter |

Every cell of every table, including totals, is one row of `trends_cell` (`period`, `table`, the dimension
values, `count` rounded, `suppressed`). No clinic, practitioner or patient column exists. A query picks a table,
fixes any of its dimensions and breaks down by one; there is no other combination (no region × specialty ×
group, no age, sex, clinic or practitioner).

**Which clinics count.** A clinic contributes to a period only if it contributed (O6) and kept the same region
and city for the whole period: `organization.Service.Locations` returns `LocationSince` and `ContributesSince`,
and a clinic whose location or contribution changed during the period is left out of it. A late build
(`salusctl trends build`) therefore never places a moved clinic in its new city for an old month.

**What a count is.** The number of distinct patients (per-build pseudonyms) in the cell. A patient can be in two
children of one parent (two subgroups, two cities), so a relation is an inequality:
`max(children) ≤ parent ≤ Σ children`. A clinic's contribution to a cell is its distinct patients in it.

**How suppression is decided** (`trends.Suppress`, pure, in the build). The build holds, in memory only, each
cell's pseudonym set and its contribution per clinic.

| Rule | |
| --- | --- |
| A1 | No identifiers in any table, service type or response. A test walks the response JSON and fails on any key outside an allow-list |
| A2 | Geography is the clinic's region and city only, never the patient's address |
| A3 | Periods as in the table above; nothing is published for an open period; a period is published once and never recomputed |
| A4 | Primary suppression. With `T` the cell's patients and `x1 ≥ x2` its two largest clinic contributions: published only if `T ≥ k`, at least 3 clinics contributed, and `T − x1 − x2 ≥ max(k, 0.1·x1)`, so that the two largest clinics together, knowing their own patients, still face at least `k` patients they cannot attribute. `k` is `ANALYTICS_MIN_COUNT`, default 10; config refuses below 10 |
| A5 | Relations: every parent with its children along each dimension, in every table, and the cells shared between tables (a G×D cell with diagnosis = all is the G×S cell with specialty = all), are one set solved together for the period |
| A6 | Secondary suppression: in every relation, the suppressed cells are none, or at least 2 whose distinct union of patients passes A4 (the union of pseudonym sets and of clinic contributions, not the sum). Otherwise the smallest published non-total cell of the relation is suppressed too, ties broken by key; when only the relation's total is left published, the total is suppressed. Repeated over all relations until nothing changes; it ends because suppression only grows over a finite set |
| A7 | Audit, in the build, fail closed: for every suppressed cell, the interval of values consistent with the published cells (each read as `[v−2, v+2]` for the rounding, and as having passed A4, a public rule: with the two known clinics holding `y1` and `y2` of its patients, `o` of them shared, at least `o + max(k, ⌈max(y1, y2)/10⌉)` patients outside them), the inequality relations, and the two largest clinics' own contributions to every cell treated as known, must be at least `k` wide. Computed as an LP (our own simplex, PROGRESS), interval ends rounded inward; any solver failure or non-finite answer counts as 0 wide. A failing cell's relation gets one more suppression and A6 and A7 run again; if the period still fails, its `code` level is dropped for that period, and if it still fails, the period is not published (logged by count only) |
| A8 | Keys: a breakdown returns every key of its dimension, with `null` for 0 and for any suppressed count alike, so absence never tells 0 from 1. For `code`, the keys are the codes published at geography = all and specialty = all for that period; any other code is not a key anywhere in that period |
| A9 | Totals over several periods are sums of published cells, labelled "patient-months" (a patient seen in two months counts twice), marked `partial` when any is suppressed; never a raw total. Rounding error adds up (±2 per month), said in the tooltip |
| A10 | Published counts are rounded to the nearest 5 (O4); A4 to A7 use the raw counts |
| A11 | Only confirmed Conditions; restricted records are left out once restriction exists |
| A12 | Rise flags only per section 6 |
| A13 | Build logs carry counts of cells, suppressed cells and refused periods only |

Tests: table-driven cases per rule; an exact integer check (enumeration over small synthetic months, two
clinics' contributions known, published values as intervals) that A7's LP audit never passes a month the exact
check fails; named cases for the attacks the reviews found (a city's specialty from the city total and its other
specialties; a region minus a city leaving one or two clinics; two subgroups holding the same patients; two large
clinics recovering a small third; 0 versus suppressed; a clinic moving between cities); the JSON allow-list
test; a frozen period is not changed by a later build; the same input gives the same suppression.

The All of Salus view shows a suppressed cell as "Hidden", with a tooltip that small or dominated groups are
hidden to protect privacy (a cell hidden by A6 or by the clinic rules can hold more than `k` patients).

## 6. Unusual rises

Per closed ISO week `w`, leaf region and diagnosis group, over the clinics that contributed and kept their
location in that region through all of weeks `w-9` to `w` (so a clinic joining, leaving or moving never raises a
flag by itself): `c_i` the week's distinct patients for week `i`; baseline weeks `w-9` to `w-2` (skipping `w-1`);
`μ` their mean, `σ` their sample standard deviation (n−1), `σ' = max(σ, √μ, 1)`. Flag when the week's cell passes
all of A4 (`k`, 3 clinics, the two-largest rule), the baseline's `Σ c_i` is at least `k`, `c_w ≥ 1.5·μ` and
`c_w > μ + 3σ'`. Computed once, on the Tuesday after the week, frozen (`trends_rise`: week, region, group; no count).

Tests, on fixed series and on seeded Poisson series: a flat series is flagged in at most 1% of 1,000 seeds; a
step to three times the level (μ ≥ 3) is flagged in its first week; a ramp of 5% a week is not (ratio about 1.31
after 8 weeks, under 1.5); a spike below `k` is not; a rise from a baseline below `k` in total is not; a clinic
joining the region with many patients is not; a week dominated by one clinic is not. Copy: "Unusual rise", with a
tooltip that it compares with the previous weeks and is not a public-health alert.

## 7. Personal data and DPIA note

Added to `docs/privacy/dpia.md` by each package (section "Analytics (`docs/plans/analytics.md`)"):

- **Processing added.** `condition`: health data (Art. 9) of identified patients, part of the clinical record.
  `clinic_*_stat`: counts per clinic, day, hour, practitioner and code: no patient identifier, but personal data
  of practitioners (workload and diagnoses). `trends_*`: frozen counts with no identifier, published under A1 to
  A12. The per-build patient pseudonym and per-clinic contributions exist only in the build's memory.
- **Purpose and lawful basis.** The clinic's statistics serve the management of its health-care services
  (Art. 9(2)(h), on the clinic's behalf as controller, Salus as processor). Platform-wide figures are a new
  purpose for which Salus would act as controller of the anonymisation; the basis is open (L1, L2). Not used for
  marketing, profiling or model training.
- **Minimisation.** No age, sex or residence. Region and city are the clinic's. `salus_stats` reads no name,
  contact detail or field value; patient ids leave the owning package only as a per-build pseudonym. No free text.
- **Retention.** `condition`: with the clinical record. `clinic_*_stat`: as long as their sources (recomputed
  from them). `trends_*`: 5 years rolling.
- **Risks and measures.** Re-identification through small counts, combinations of queries, dominant clinics or
  repeated builds: A3 to A9. A practitioner learning about patients outside their care: C2. Rise flags taken as
  official alerts: copy, and L6.

## 8. API contract (for SAL-22's mocks)

Plain JSON in `api/openapi.yaml` (not FHIR). ANA-2 and ANA-3 add their paths; ANA-4 types its fixtures from
these shapes and switches to the generated client once both are merged. A count is a number or `null`
(suppressed); the API returns codes, never sentences.

**As built (ANA-2, `api/openapi.yaml` wins):** the clinic endpoints are camelCase like the rest of the plain JSON
API: the query parameter `practitionerRole` (not `practitioner_role`), the breakdown `by=practitionerRole`, and the
field `practitioners[].practitionerRole`; `from` and `to` are required. A non-admin may narrow the appointments to
their own role only, never to another role or a specialty (403 `forbidden`). ANA-4 types its fixtures from the
generated client, not from the lines below.

```
GET /analytics/clinic/appointments?from=2026-01-01&to=2026-12-31&practitioner_role=&specialty=
  -> { "series":  [{ "period": "2026-01", "booked": 120, "fulfilled": 98, "noshow": 6, "cancelled": 16 }],
       "heatmap": [{ "weekday": 1, "hour": 9, "count": 14 }],                     // ISO weekday, 1 = Monday
       "practitioners": [{ "practitioner_role": "<uuid>", "fulfilled": 40 }] }    // admins only
GET /analytics/clinic/diagnoses?from=&to=&group=&subgroup=&specialty=&practitioner_role=&by=group|subgroup|code|practitioner_role|specialty|month
  -> { "rows": [{ "key": "respiratory-infections", "count": 31 }] }
GET /analytics/platform/months                 -> { "months": ["2026-09", "2026-10"], "min_count": 10 }
GET /analytics/platform/cells?table=gd|gs|sd&from=2026-01&to=2026-10&<dimension>=<value>…&by=<dimension>|month
     dimensions: entity, region, city, group, subgroup, code, specialty
  -> { "rows": [{ "key": "BA-09", "count": 45 }, { "key": "BA-05", "count": null }], "partial": true }
GET /analytics/platform/heatmap?quarter=2026-Q3&region=
  -> { "cells": [{ "weekday": 1, "hour": 9, "count": 35 }, { "weekday": 7, "hour": 3, "count": null }] }
GET /analytics/platform/rises?region=&weeks=8
  -> { "rises": [{ "week": "2026-W44", "region": "BA-09", "group": "respiratory-infections" }] }
GET /terminology/diagnosis-groups              -> [{ "code": "common-cold", "parent": "respiratory-infections", "name": "…" }]
GET /places?country=BA                         -> [{ "id": "BA-09", "kind": "region", "parent": "BA-BIH", "name": "…" }]
```

## 9. Work packages

```
ANA-0 (this PR): ADR 0013 and this plan. Stops for the owner's approval.
ANA-1 SAL-19, alone, by the lead session: condition, groups, places, confirmation, the wave-2 contract, salus_stats.
Wave 2, in parallel after ANA-1 is merged: ANA-2 SAL-20   ANA-3 SAL-21   ANA-4 SAL-22 (mocked data first)
Merge order: ANA-2, ANA-3, ANA-4. Before each merge: main merged into the branch (no force-push), migrations
renumbered if main has taken their numbers (another session is building chat), checks rerun.
```

Each wave-2 package runs as a subagent in its own git worktree and branch, with its own compose project
(`-p salus-ana2`, `-p salus-ana3`, `-p salus-ana4`) and no published host ports. Every package: strict TDD, Docker
only, `make lint`, `make test`, `make archcheck`, `make e2e`, then a review by a fresh subagent (Copilot's quota is
out until 2026-12-01), valid findings fixed, then merged. PROGRESS: `docs/plans/analytics.PROGRESS.md`.

Migrations take the next free number when the branch is merged; main is at `00040` today. Shared files the
wave-2 packages touch (`api/openapi.yaml`, the generated clients, `ownership.yaml`, `config.yaml`,
`internal/retention/rules.go`, `internal/api/app.go`, the router, `internal/worker`) are merged by hand in merge
order; generated clients are regenerated after the merge, never hand-merged.

### ANA-1 SAL-19 diagnoses (branch `analytics-diagnoses`)

- Migration: `diagnosis_group*` with appendix A seeded (names `unreviewed`), `place*` seeded, `condition` with RLS
  and the backfill, the `salus_stats` role (as 00014, `STATS_DATABASE_PASSWORD` set by `cmd/migrate`) and its
  column grants on `condition`, `appointment`, `practitioner_role` and `organization`.
- `internal/condition`: model, repository, service (`Record` in the caller's transaction, `Recorded`, `Get`),
  FHIR read, golden JSON, audit, retention rule; ownership and archcheck entries.
- `terminology`: `Group(code)`, places, both endpoints, the review panel's group list (web, five staff languages).
- `composition`: `confirmed` on sign values, `diagnosis_unconfirmed`; the `Condition`s written through
  `condition.Service` in the signing transaction; `StatsQueue` port (no-op until ANA-2). Web document editor:
  the confirm control on a suggested code.
- The section 3 contract: `appointment.Service.Starts` (and its `StatsQueue` port), `practitionerrole.Service
  .Specialties`, `organization.Service.Locations` (id and time zone), the `salus_stats` pool in
  `internal/config` and `internal/worker`.
- `salusctl conditions unconfirmed`. E2E: the record journey's signed document has a confirmed `Condition`.

### ANA-2 SAL-20 clinic statistics (branch `analytics-clinic`)

Section 4: tables, the two jobs, `StatsQueue`'s River implementation replacing the no-op, `internal/clinicstats`
(rules C1 to C3, JSON handlers, openapi paths), the RLS, equality and role tests, the benchmark, DPIA and
retention entries.

### ANA-3 SAL-21 platform trends (branch `analytics-platform`)

Clinic location: `organization` region and city (migration, FHIR `Organization.address.state` and `.city`, the
admin's Clinic settings field, `Locations` filled). Then sections 5 and 6: `internal/trends`, the month,
quarter and week builds, `Suppress`, the rises, handlers and openapi paths, `ANALYTICS_MIN_COUNT` in
`internal/config`, `salusctl trends build <month>` for a missed build, the opt-out (`analytics_contributes`,
Clinic settings, left out of builds, the view refused), `ANALYTICS_PLATFORM_ENABLED` (section 2a) and its tests
both ways, the DPIA section with L1 and L2 open.

### ANA-4 SAL-22 charts (branch `analytics-charts`)

`web/src/analytics/`: an Analytics page with "My clinic" and "All of Salus", SVG line, bar and heatmap components
with tooltips, keyboard focus and `prefers-reduced-motion`; filters (group with drill-down, range, region,
specialty, practitioner on My clinic only; on All of Salus, only the table combinations of section 5, so the UI
never offers a filter the API refuses); dates and numbers through `Intl` in the UI language, five staff
languages; the All of Salus tab shown only when `features` has `analytics_platform` and the clinic contributes;
a suppressed platform cell reads "Hidden" with the tooltip of section 5. Built against
typed fixtures of section 8 first; after ANA-2 and ANA-3 are merged, the generated client replaces the fixtures
and an e2e test switches views and filters. Component tests in Vitest; a render benchmark on 3 years of monthly
data with every group.

## 10. Open questions

### Answered by the owner (2026-11-12)

- **O1 Geography:** the clinic's location.
- **O2 Countries:** BiH only.
- **O3 Practitioners in My clinic:** their own diagnoses plus the clinic's appointment figures; clinic admins also
  see the clinic's full diagnosis totals.
- **O4 Rounding:** to the nearest 5.
- **O5 All of Salus:** all staff of a contributing clinic.
- **O6 Opt-out:** yes; an opted-out clinic loses the All of Salus view (section 2).
- **O7 Group names:** seeded `unreviewed`; the doctor partner reviews appendix A in the review panel.
- **O8 Months:** acceptable.

### For the owner's lawyer (the platform view stays off until they are answered, section 2a)

- **L1 Role.** Salus processes clinics' data as their processor. Building platform-wide figures is Salus's own
  purpose: does the anonymisation step need the clinics' instruction in the DPA (Art. 28(3)(a)), or does Salus
  become a controller for it, and with which contract clauses?
- **L2 Lawful basis** for that step, for health data: Art. 9(2)(h), (i) or (j) with Art. 89 and the compatibility
  test of Art. 6(4), or the patient's consent; and the same under the BiH personal data protection law in force.
  If consent is needed, patients without it must be left out (a new `Consent` category).
- **L3 Anonymous or not.** Are the published figures anonymous in the sense of Recital 26 under rules A1 to A12?
  May frozen months stay after a patient's erasure, as anonymous output?
- **L4 Transparency and objection.** What the patients' privacy notice must say about statistics (Art. 13, 14),
  and whether an objection (Art. 21) must remove a patient from future builds.
- **L5 Practitioners.** Per-practitioner figures are employee data: may the clinic admin see each practitioner's
  diagnoses and workload, and what must practitioners be told?
- **L6 Rises.** Does a rise flag create or touch any duty under the entity laws on protection against
  infectious diseases (notifiable diseases), and is the disclaimer enough?
- **L7 ICD-10.** Showing ICD-10 codes and grouping them by block ranges without WHO text: permitted under WHO's
  terms without a licence (the terminology's open question, extended to block ranges as a structure)? Does any
  official use of our documents in BiH (sick leave, referrals) require the national MKB-10 edition?
- **L8 DPIA.** Whether this feature needs consultation of the supervisory authority (Art. 36) after the DPIA.

## Appendix A. Diagnosis groups (draft names, `unreviewed` until the doctor partner approves them in the review panel)

Ranges are inclusive and compared on the code's first characters (`J00` matches `J00` and `J00.0`). A subgroup
(indented key) lies inside its group. U07.1 and U07.2 are COVID-19, U08 to U10 its history and after-effects (WHO emergency codes); U07.0 (vaping) and a bare U07 fall to `other`. V01–Y98 (external causes) and
any code outside every range map to `other`.

| Key | Ranges | en | bs | hr | sr-Latn | sr-Cyrl | de | ar | tr | it | fr |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `intestinal-infections` | A00–A09 | Intestinal infections | Crijevne infekcije | Crijevne infekcije | Crijevne infekcije | Цријевне инфекције | Darminfektionen | عدوى الأمعاء | Bağırsak enfeksiyonları | Infezioni intestinali | Infections intestinales |
| ↳ `gastroenteritis` | A08–A09 | Gastroenteritis | Gastroenteritis | Gastroenteritis | Gastroenteritis | Гастроентеритис | Gastroenteritis | التهاب المعدة والأمعاء | Gastroenterit | Gastroenterite | Gastro-entérite |
| `other-infections` | A15–B99 | Other infectious diseases | Druge zarazne bolesti | Druge zarazne bolesti | Druge zarazne bolesti | Друге заразне болести | Andere Infektionskrankheiten | أمراض معدية أخرى | Diğer bulaşıcı hastalıklar | Altre malattie infettive | Autres maladies infectieuses |
| `covid-19` | U07.1–U07.2, U08–U10 | COVID-19 and its after-effects | COVID-19 i posljedice | COVID-19 i posljedice | COVID-19 i posljedice | COVID-19 и посљедице | COVID-19 und Folgen | كوفيد-19 وعواقبه | COVID-19 ve sonrası etkileri | COVID-19 e postumi | COVID-19 et séquelles |
| `tumours` | C00–D48 | Tumours | Tumori | Tumori | Tumori | Тумори | Tumoren | الأورام | Tümörler | Tumori | Tumeurs |
| `blood-immune` | D50–D89 | Blood and immune disorders | Bolesti krvi i imuniteta | Bolesti krvi i imunosnog sustava | Bolesti krvi i imuniteta | Болести крви и имунитета | Blut- und Immunerkrankungen | اضطرابات الدم والمناعة | Kan ve bağışıklık hastalıkları | Malattie del sangue e del sistema immunitario | Maladies du sang et du système immunitaire |
| `diabetes` | E10–E14 | Diabetes | Dijabetes | Šećerna bolest | Dijabetes | Дијабетес | Diabetes | داء السكري | Diyabet | Diabete | Diabète |
| `endocrine-metabolic` | E00–E07, E15–E90 | Thyroid, hormone and metabolic disorders | Bolesti štitne žlijezde, hormona i metabolizma | Bolesti štitnjače, hormona i metabolizma | Bolesti štitne žlijezde, hormona i metabolizma | Болести штитне жлијезде, хормона и метаболизма | Schilddrüsen-, Hormon- und Stoffwechselerkrankungen | اضطرابات الغدة الدرقية والهرمونات والاستقلاب | Tiroid, hormon ve metabolizma hastalıkları | Malattie della tiroide, ormonali e metaboliche | Maladies thyroïdiennes, hormonales et métaboliques |
| `mental-health` | F00–F99 | Mental health | Mentalno zdravlje | Mentalno zdravlje | Mentalno zdravlje | Ментално здравље | Psychische Gesundheit | الصحة النفسية | Ruh sağlığı | Salute mentale | Santé mentale |
| `nervous-system` | G00–G99 | Nervous system disorders | Bolesti nervnog sistema | Bolesti živčanog sustava | Bolesti nervnog sistema | Болести нервног система | Erkrankungen des Nervensystems | أمراض الجهاز العصبي | Sinir sistemi hastalıkları | Malattie del sistema nervoso | Maladies du système nerveux |
| `eye` | H00–H59 | Eye conditions | Bolesti oka | Bolesti oka | Bolesti oka | Болести ока | Augenerkrankungen | أمراض العين | Göz hastalıkları | Malattie dell'occhio | Maladies de l'œil |
| `ear` | H60–H95 | Ear conditions | Bolesti uha | Bolesti uha | Bolesti uva | Болести ува | Ohrerkrankungen | أمراض الأذن | Kulak hastalıkları | Malattie dell'orecchio | Maladies de l'oreille |
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
`A09` → intestinal-infections / gastroenteritis; `U07.1` → covid-19; `U09.9` → covid-19; `U07.0` → other; `W19` → other; `Z00.0` → checkups.
