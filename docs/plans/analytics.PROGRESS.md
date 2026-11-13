# Analytics: progress

The running record of docs/plans/analytics.md. GitHub Copilot's review quota is used up until 2026-12-01: every PR is
reviewed by a fresh subagent that has not seen the work, and its valid findings are fixed before the merge. The
platform view stays behind `ANALYTICS_PLATFORM_ENABLED`, off, until the owner confirms the legal answers (L1 to L8).

## ANA-0 plan (PR #71, merged 2026-11-12)

ADR 0013 and the plan. Two subagent reviews: the first (14 findings) moved the platform figures from per-query to
per-month suppression over distinct patients, froze closed months, took out features that do not exist yet; the
second, of sections 5 and 6 only (12 findings), made the relations inequalities, measured secondary suppression on
the distinct union, ran the audit in the build failing closed, and kept moving or opting-out clinics out of a
period and of the rises. The owner's answers O1 to O8 and the feature flag are in the plan (§2, §2a, §10).

## ANA-1 SAL-19 diagnoses (branch `analytics-diagnoses`)

Built: migration 00042 (diagnosis groups with appendix A's names, unreviewed; places; `condition` with RLS and the
backfill as `unconfirmed`; the `salus_stats` role and its column grants), `internal/condition`, the catalog and the
group-name review in `terminology`, the confirmation in `composition`'s signing, the stats readers, `pkg/pseudonym`,
`salusctl conditions unconfirmed`, `GET /fhir/R4/Condition/{id}` (the signed document's rule, audited), the web
confirm control and the review panel's "Diagnosis groups" tab, the e2e
browser journey confirming a code and its Condition checked in the database.

As built, where it differs from the plan (agent decisions, for the owner to review):

- **An unconfirmed code is a 422** `diagnosis_unconfirmed` with the field keys, like `required_missing`, not a 409.
- **The stats-refresh port moves to ANA-2**, its only user: ANA-2 declares `StatsQueue` in `composition` and
  `appointment` and implements it; ANA-1 leaves no no-op behind.
- **Places have one name in two scripts** (`name_latn`, `name_cyrl`): city, region and entity names are proper
  nouns, shown only to staff; no per-language table. 143 municipalities and cities from Wikidata (CC0, fetched
  2026-11-12): the composite cities Sarajevo and Istočno Sarajevo are left out for their municipalities, one
  duplicate Banja Luka item dropped, Doboj, Istočno Sarajevo's municipalities and Vareš placed by hand where
  Wikidata's P131 path had no ISO code. The Federation is the one entity; its cantons are regions under it;
  Republika Srpska and Brčko District are regions with no parent (`GET /places`).
- **The sources, as built** (the wave-2 contract):
  `condition.Service.Recorded(ctx, org, from, to, key) ([]condition.Recorded, error)`,
  `appointment.Stats.Starts(ctx, org, from, to, key) ([]appointment.Start, error)`,
  `practitionerrole.Stats.Specialties(ctx, org) (map[string]string, error)`,
  `organization.Stats.Locations(ctx) ([]organization.Location, error)` with `ID` and `TimeZone` only (ANA-3 adds
  region, city and the since-times), each on its `NewPostgresStatsRepository(statsPool)`;
  `terminology.Catalog.Groups(ctx)` for the grouping; `pkg/pseudonym.Of(key, id)` for the per-build pseudonym.
- **The `salus_stats` pool is not opened yet.** `config.LoadStats` is read by `cmd/migrate` (the role's password);
  ANA-2 opens the pool in `cmd/worker` and adds it to `TestLoadStats_IsCalledOnlyByMigrate`'s list.
  `STATS_DATABASE_PASSWORD` is now required in the production `.env`: **the owner adds the secret before the next
  deploy**.
- **Audit of the builds:** `salus_stats` cannot write `audit_event`; a build audits through the app's pool.
- **Review of the branch** (fresh subagent): nine findings, all fixed: the builds group codes without reading the
  names (`Catalog.Groups` on `Grouping`, which salus_stats may read); a confirmation holds only for the value
  confirmed, so a code that changes under the tick is unconfirmed again; a refusal is kept per draft and shows the
  checkbox even when the template no longer marks the field coded; a pseudonym key under 32 bytes is refused;
  composite keys tie a Condition to its encounter's clinic and patient and to a document of that encounter; the
  group names drop an answer that comes after a language switch; the FHIR read the plan settled, which a first cut
  had left out, is built.
- **JSON field names** of the new endpoints are camelCase, as the rest of `terminology`'s; section 8's shapes are
  indicative and the generated client is the contract.

## ANA-2 SAL-20 clinic statistics (branch `analytics-clinic`)

Built: migration 00048 (00043 to 00047 were taken by patient history and chat) (`clinic_appointment_stat`, `clinic_diagnosis_stat`: RLS by clinic for `salus_app`'s reads,
written only by `salus_stats` with SELECT, INSERT and DELETE), the domain `internal/clinicstats` (context
`analytics`): the `Builder` (the refresh of a clinic-day and the nightly pass, on the stats pool), the `Service`
(rules C1 to C3), the River `Queue`, the JSON handlers, `GET /analytics/clinic/appointments` and
`GET /analytics/clinic/diagnoses` in `api/openapi.yaml` (clients regenerated); `StatsQueue` declared in `appointment`
(booking, cancellation, the practitioner's status, the consultation's outcome, the revocation's cancellations) and in
`composition` (a signature with at least one diagnosis), implemented by `clinicstats.NewRiverQueue` in
`internal/api/app.go` and in the worker's revocation and session services; `cmd/worker` opens the `salus_stats` pool
(`STATS_DATABASE_URL`, required; compose and production compose set it on the worker only, tested) and registers
`clinicstats_refresh` and `clinicstats_nightly`; the scheduler enqueues the nightly pass at 02:30 UTC
(`dailyAt` takes a minute now). Retention rule (kept with their sources), DPIA section, the e2e check (the browser
journey's signed J06.9 is counted in the admin's diagnosis figures).

As built, where it differs from the plan or settles what it left open (agent decisions, for the owner to review):

- **The job's `day` is an instant**, the appointment's start or the signing time, not a date: the producers do not
  know the clinic's time zone, so the worker takes the local date in it (`organization.Stats.Locations`).
- **Query parameters and breakdown values are camelCase** like the JSON (`practitionerRole`, `by=practitionerRole`),
  not §8's `practitioner_role`. `from` and `to` are required local dates, inclusive, at most ten years apart.
- **`booked` counts every appointment** of the month, whatever its status now (§8's example adds up so); `fulfilled`,
  `noshow` and `cancelled` are those with that status. The heatmap counts the appointments that were not cancelled,
  busy cells only. `practitioners` gives the fulfilled appointments per role and is absent for non-admins.
- **C2 for appointments:** a practitioner who is not an admin may narrow the clinic's figures to their own role only,
  never to another role or to a specialty (403, audited; review finding 1: a specialty with one practitioner is
  that colleague's figures, and a specialty minus one's own role the others'). For diagnoses they always get their own role's, and a practitioner without
  a role is refused. An admin who is also a practitioner gets the clinic's.
- **C3's purpose is `HOPERAT`** (v3-ActReason has no analytics code), subtype `search-type`, action `E`, entity
  `Organization/<clinic>`; refusals are audited with outcome 4. An invalid query is refused before any read and is not
  audited (like appointment's `ErrInvalidStatus`).
- **Empty keys:** a diagnosis without a subgroup counts under the subgroup key `""`, a role without a specialty under
  the specialty key `""`, so a breakdown sums to the total.
- **Concurrency:** a refresh takes a transaction-level advisory lock per clinic, so two refreshes of one clinic never
  interleave their delete and insert. Both jobs run on their own River queue `clinicstats`, two at a time (review
  finding 6), so the builds hold at most two `salus_stats` connections and never take the default queue's workers.
  No `UniqueOpts`: River's unique states always include `running`, so a write during a running refresh would lose
  its own refresh, and the job's instant differs per appointment anyway.
- **The refresh does not enqueue for future appointments beyond their day**, and the nightly pass covers the last 35
  days only, as the plan says: a lost refresh of a day further ahead is repaired by the next write on that day.

Measured (`make bench-clinicstats`, `internal/worker`, the compose Postgres on the development machine, 2026-11-12:
3 years × 20 practitioners, each with 8 Conditions and up to 8 appointments a day (none starting at 02:xx), 175,360
Conditions and about 160,000 appointments, nearly every one in its own hour cell, so some 55,000 rows of each
aggregate per year; 50 iterations, 3 runs):

| What | Per operation |
| --- | --- |
| Refresh of one clinic-day (about 150 appointments and 160 Conditions read, counted, replaced in one transaction) | 15.1 to 19.4 ms |
| Yearly overview as the admin opens it (series, heatmap, per practitioner, diagnoses by group; each audited) | 121 to 135 ms |

No index beyond the primary keys (clinic, day, hour, role, status or code), which serve the range scans; not tuned
further without a slower measurement in use.

Re-measured after merging main (chat 00044 to 00047, the shared history lookup 00046; the migration is now 00048),
2026-11-12, same benchmark and machine but under a load average of about 60 on 28 threads from other sessions:
refresh 11.2 to 23.2 ms, yearly overview 215 to 249 ms. The merge changed none of what the benchmark reads or
writes (the two aggregates and their policies, the stats login's sources, `audit_event`): salus_stats bypasses the
new organization and practitioner_role policies, and the aggregates' policies read only settings, so no policy
recursion. The slower overview is attributed to the load, not the merge; re-measure on an idle machine before
tuning.

Re-measured with the real auditevent service in the overview (review finding 5; the first measurement used an
auditor that wrote nothing), 2026-11-13, load average 6.5 to 7.6 on 28 threads (not idle; other sessions on the
machine): refresh 11.4 to 15.0 ms, yearly overview with its two AuditEvents 130 to 144 ms. Close to the first
numbers, so the earlier 215 to 249 ms was the load; an idle re-run is still owed before any tuning decision.

Review (fresh subagent, 2026-11-13): 8 findings; fixed 1 (no specialty filter for non-admins), 3 to 8 (DPIA wording
on the shared source queries, the build-audit comments, the audited benchmark, the `clinicstats` queue at two
workers, §8's as-built names, the stats pool's one-place test and the autumn fall-back test); 2 (one patient's
diagnosis visible to an admin on a one-day, one-role code breakdown) is a DPIA risk open for the owner.

## ANA-3 SAL-21 platform trends (branch `analytics-platform`)

Built: migration 00052 (the clinic's location and opt-out on `organization` with the times they changed, a city
under a leaf region enforced by foreign keys; `trends_period`, `trends_cell`, `trends_rise`, written once by
`salus_stats`, read by the app, never changed, cells and rises deleted by `salus_retention` after five years);
`organization` FHIR `address` (state = region code, city = Latin name, the city's Wikidata id in the
`organization-city` extension) and the `organization-analytics-contributes` extension, both StructureDefinitions,
golden files, `make fhir-validate`; `internal/trends` (context `analytics`): `Suppress` (A4 to A10 with the A7 LP
audit, failing closed), the month (G×D, G×S, S×D), quarter (heatmap) and week (rises) builds, the view with its
four endpoints; `ANALYTICS_MIN_COUNT` and `ANALYTICS_PLATFORM_ENABLED` in `internal/config`; the scheduler's three
builds and the `trends_build` job; `salusctl trends build <period>`; `GET /auth/me` `features`; Clinic settings in
web/ (region and city from `GET /places`, the opt-out with its legal-weight copy, five staff languages); the e2e
journey (location, the view, the opt-out) with the flag on for the e2e stack only; the DPIA section with L1 and L2
open.

**The contract as built** (ANA-4 types against the generated client):

- `organization.Stats.Locations(ctx) ([]organization.Location, error)`, `Location{LocationSince, ContributesSince
  time.Time; ID, TimeZone, Region, City string; Contributes bool}`: `Region` an ISO 3166-2:BA code, `City` a Wikidata
  id, both empty without a location; a contributing clinic with a location counts in a period that starts at `t` in
  its zone only when `LocationSince <= t` and `ContributesSince <= t`.
- `organization.Service.Contributes(ctx, orgID) (bool, error)` for the view's O6 check.
- `GET /auth/me` has `features: ["analytics_platform"]` while the flag is on, no `features` while it is off.
- `GET /analytics/platform/months` → `{months, quarters, minCount}`; `GET /analytics/platform/cells?table=gd|gs|sd&
  from=&to=&by=<dimension>|month&<dimension>=…` → `{rows: [{key, count}], partial}`, dimensions entity, region,
  city, group, subgroup, code, specialty; `GET /analytics/platform/heatmap?quarter=&region=` → `{cells: [{weekday,
  hour, count}]}` (168 cells); `GET /analytics/platform/rises?region=&weeks=` → `{rises: [{week, region, group}]}`.
  Errors `401 unauthenticated`, `403 forbidden` | `analytics_not_contributing`, `400 invalid_request`. A count is an
  integer or `null`.

**Measured.** `BenchmarkSuppress_ASyntheticMonth` (40 clinics in 3 regions and 12 cities, 4,000 patients, 27 groups,
12 specialties: 4,088 cells, 3,600 suppressed, almost all by A4 since most cells have fewer than three clinics):
35.4 s per month on the i7-14700HX, shared with two other agents' test runs; the A7 audit is nearly all of it
(about 10 ms per LP). The `trends_build` job's timeout is one hour. The exact integer check over 60 random synthetic
months runs in about 16 s.

As built, where it differs from the plan (agent decisions, for the owner to review):

- **The A7 LP is our own simplex, not gonum's `lp`.** gonum v0.17.0's `lp.Simplex` was measured first: its first
  phase (QR condition checks per candidate basis) did not finish the 60 synthetic months of the exact check in 200 s,
  nor with a feasible first basis given (dense solves per pivot). A 70-line tableau simplex with Bland's rule, started
  from the true point's slack basis (so no first phase), runs them in 16 s. No new dependency.
- **The audit's LP is over a neighbourhood:** the suppressed cell, every suppressed cell reachable through
  suppressed cells and published cells within two relations, at most 150 cells; every other cell is held at its true
  value. That only adds constraints, so the audit can over-suppress but never pass a cell the whole-period LP fails;
  the exact integer check confirms it on the random months.
- **The relations.** A relation is a parent cell and its children along one dimension at one level; it is a cover
  (`max ≤ parent ≤ Σ`, used by A6 and A7) when the children's patients are exactly the parent's, and otherwise each
  child is only within the parent (A7 alone): a subgroup within its group (codes outside a subgroup), a specialty
  within "all" when some role has no specialty. In the A7 LP the parent's residual (patients the two known clinics did
  not see) is within the sum of its children's residuals, and each child's count within the parent's.
- **A cell with 0 patients is not stored and reads null** (A8); in the audit it is a known 0, which is conservative.
- **Republika Srpska and Brčko District are their own entity** in the geography (`entity` = `region` = `BA-SRP`),
  so `by=entity` gives the Federation, Republika Srpska and Brčko District.
- **A code not a key of a month is suppressed in every cell of that month** inside `Suppress` (A8 applied before
  A6 and A7), so it neither shows nor helps differencing.
- **The build audits after reading its sources and before writing anything,** on the app's pool (salus_stats cannot
  write `audit_event`), in its own transaction; purpose `HRESCH`. A failing audit writes nothing.
- **salus_stats writes the figures** (INSERT only); the app only reads them. `salusctl trends build` runs as the owner.
- **The heatmap's geography stops at the region** (plan §5), and `region=` also takes an entity.
- **`features` is omitted while empty** rather than an empty list, so the existing `/auth/me` shapes are unchanged.
- **Shared files touched** (merge by hand): `api/openapi.yaml` (four paths, five schemas, `Account.features`),
  `internal/api/app.go`, `internal/worker/{worker,scheduler,retention}.go` (`worker.New` takes `worker.Trends`,
  `NewScheduler` takes the flag), `cmd/worker` (after the merge with ANA-2 the trends builds run on its one salus_stats pool,
  `worker.Trends{Enabled, MinCount}`), `internal/config` (`Config.Analytics`), `ownership.yaml`, `config.yaml` (`analytics:
  [trends]`, ANA-2 adds `clinicstats`), `internal/retention/rules.go`, `sqlc.yaml`, the Caddyfile and the Vite proxy
  (`/analytics/platform`), the compose files (`STATS_DATABASE_URL` for the worker; the e2e overlay turns the flag on
  for api, worker and scheduler). Every `*db/models.go` is regenerated by sqlc for the new tables.
- **Merged with main** (chat group consults, chat attachments, patient history, 00044 to 00046): the migration is
  `00049_platform_trends.sql` (00047 chat notifications and 00048 SAL-20 come before it; renamed `00052` after
  SAL-39's 00050 and SAL-40's 00051, in merge order); the archcheck contexts, the
  retention rules and the DPIA keep both sides; generated files are main's, regenerated. Checks after the merge:
  `make lint`, `make archcheck`, `make test` (`cmd/migrate` and `cmd/notifier` flaked under load, the known salus_stats
  password race and a 15 s start timeout, and passed alone), `make cover` (92.4% in all, `internal/trends` 88.1%),
  `make fhir-validate`, `make e2e`, `make web-lint web-typecheck web-test`, `make gen-clients` with no diff.
- **Review (fresh subagent)**: eight findings. Fixed: the A7 LP and the exact check now know a published cell passed
  A4 (`r >= max(v-2-a, k)`; a k = 12 subgroup audited as [8, 22] was really [12, 22],
  `TestSuppress_A7KnowsThatAPublishedCellPassedA4`); an unbounded program or a solver point outside a row counts as 0
  wide (`TestWidth_AnUnboundedProgramFailsClosed`); the heatmap reads only fulfilled appointments in SQL
  (`appointment.Stats.Fulfilled`, `TestBuild_AQuarterHeatmap`, `TestStatsFulfilled_OnlyTheFulfilledPseudonymised`);
  Clinic settings refuses a region without a city, cannot clear a saved location, says when `/places` fails, and hides
  the opt-out and the location hint while the view is off (`ClinicSettings.test.tsx`; the "anonymized" legal copy
  unchanged, pending L3); a `trends_build` on a worker with the flag off is cancelled
  (`TestTrendsWorker_CancelsABuildWhileThePlatformViewIsOff`); a purged period is no longer listed
  (`TestRepository_WritesOnceReadsAndPurges`). Left for the owner: a clinic may opt in, read every frozen month and
  opt out again (strict O6 would refuse the view until `contributes_since` is before the current month). The month
  benchmark after the fix: 34.3 s and 36.1 s (was 35.4 s); the exact check over 60 months 15.5 s.
- **Review of the A7 fix (fresh subagent)**: four findings, all fixed. The lower bound of a published cell is what A4
  reveals, `o + max(k, ⌈ymax/10⌉)` outside the known clinics (`o` their shared patients), not `k`
  (`TestSuppress_A7KnowsTheOverlapOfTheKnownClinics`; the random months now share patients across clinics and may
  have a clinic over 10·k in a cell); a NaN or infinite solver answer counts as 0 wide (`TestWidth_ANaNFailsClosed`);
  the first attack case asserts its repair; §5 A7 describes the attacker's A4 knowledge. Re-measured: the month
  benchmark 37.2 s and 38.0 s, the exact check over 60 months 12.3 s.
