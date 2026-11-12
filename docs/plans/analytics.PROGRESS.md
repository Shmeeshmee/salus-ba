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

Built: migration 00043 (`clinic_appointment_stat`, `clinic_diagnosis_stat`: RLS by clinic for `salus_app`'s reads,
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
- **C2 for appointments:** a practitioner who is not an admin may narrow the clinic's figures to their own role or to
  a specialty, never to another role (403). For diagnoses they always get their own role's, and a practitioner without
  a role is refused. An admin who is also a practitioner gets the clinic's.
- **C3's purpose is `HOPERAT`** (v3-ActReason has no analytics code), subtype `search-type`, action `E`, entity
  `Organization/<clinic>`; refusals are audited with outcome 4. An invalid query is refused before any read and is not
  audited (like appointment's `ErrInvalidStatus`).
- **Empty keys:** a diagnosis without a subgroup counts under the subgroup key `""`, a role without a specialty under
  the specialty key `""`, so a breakdown sums to the total.
- **Concurrency:** a refresh takes a transaction-level advisory lock per clinic, so two refreshes of one clinic never
  interleave their delete and insert.
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
