# Analytics: progress

The running record of docs/plans/analytics.md. GitHub Copilot's review quota is used up until 2026-12-01: every PR is
reviewed by a fresh subagent that has not seen the work, and its valid findings are fixed before the merge. The
platform view stays behind `ANALYTICS_PLATFORM_ENABLED`, off, until the owner confirms the legal answers (L1 to L8).

## ANA-0 plan (PR #71, merged 2026-11-12)

ADR 0012 and the plan. Two subagent reviews: the first (14 findings) moved the platform figures from per-query to
per-month suppression over distinct patients, froze closed months, took out features that do not exist yet; the
second, of sections 5 and 6 only (12 findings), made the relations inequalities, measured secondary suppression on
the distinct union, ran the audit in the build failing closed, and kept moving or opting-out clinics out of a
period and of the rises. The owner's answers O1 to O8 and the feature flag are in the plan (§2, §2a, §10).

## ANA-1 SAL-19 diagnoses (branch `analytics-diagnoses`)

Built: migration 00041 (diagnosis groups with appendix A's names, unreviewed; places; `condition` with RLS and the
backfill as `unconfirmed`; the `salus_stats` role and its column grants), `internal/condition`, the catalog and the
group-name review in `terminology`, the confirmation in `composition`'s signing, the stats readers, `pkg/pseudonym`,
`salusctl conditions unconfirmed`, the web confirm control and the review panel's "Diagnosis groups" tab, the e2e
browser journey confirming a code and its Condition checked in the database.

As built, where it differs from the plan (agent decisions, for the owner to review):

- **No FHIR read of Condition yet.** Nothing reads it and no SPEC asks for one (CLAUDE.md: no code for a requirement
  not in a SPEC); the model is FHIR-shaped and the read is added with its first consumer (the patient's export).
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
- **JSON field names** of the new endpoints are camelCase, as the rest of `terminology`'s; section 8's shapes are
  indicative and the generated client is the contract.
