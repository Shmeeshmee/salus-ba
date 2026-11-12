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

## ANA-4 SAL-22 charts (branch `analytics-charts`), mocked data

Built: `web/src/analytics/`, an Analytics page (nav entry, `/analytics`) with My clinic and All of Salus tabs.
Hand-written SVG: `LineChart` (months, one line per series, hidden months break the line), `BarChart` (largest first,
hidden last, drill-down on click or Enter), `Heatmap` (ISO weekday × hour, one-hue ramp); one tab stop per chart, arrow
keys between marks, a tooltip with the exact value on focus and hover, a legend for two or more series, a "Show as
table" view, transitions only under `prefers-reduced-motion: no-preference`. Filters: month range, specialty,
practitioner (admins only), diagnosis group → subgroup → code (also by clicking bars, with a breadcrumb back), and on
All of Salus the table (gd, gs, sd) whose two dimensions are the only filters offered, region (entity or region, then
city). Hidden cells read "Hidden" with the §5 tooltip; sums over months are "Patient-months" with a "Partial" tag;
rises read "Unusual rise" with the §6 tooltip. Strings in the new `analytics` catalog area, five staff languages;
months and weekdays through `Intl` (`formatMonth`, `formatWeekday` in `i18n/format.ts`).

Data: `data.ts` holds the §8 shapes in camelCase and one interface (`Reference` + `Stats`). `analyticsData(api, fhir,
account)` already reads the groups, places, specialties and staff from the API; the statistics come from
`fixtures.ts` (synthetic, labelled placeholder). The switch to the generated client replaces `fixtureStats` in
`analyticsData` only; the Vite proxy (and Caddy) will need `^/analytics/(clinic|platform)/` then, since `/analytics`
itself is a screen.

Measured (`render.bench.test.tsx`, jsdom in the node service, 30 renders after 3 warm-ups, 3 years of monthly data,
every group: 27 groups + 7 subgroups):

| Chart | Median | p95 |
| --- | --- | --- |
| Line, 34 series × 36 months (1,224 marks) | 65 ms | 79 ms |
| Bars, 27 groups summed over 36 months | 5 ms | 9 ms |
| Heatmap, 7 × 24 | 8 ms | 11 ms |

`formatNumber` now keeps one `Intl.NumberFormat` per language and options: building one per call made the line chart
131 ms (median). The separate hit circle of a line point (a target bigger than the mark) costs about 12 ms of the 65;
kept. The UI never draws more than 4 series; the 34-series case is the stress case. The test asserts the marks drawn, not
the time (inside the whole suite the workers share the machine: a run there took 240 ms a render); measure with the
file alone.

Agent decisions, for the owner to review:

- **"The clinic contributes"** is learnt from `GET /analytics/platform/months`: `analytics_not_contributing` hides the
  tab; with the feature off the call is never made. `features` is typed locally (`AnalyticsAccount`) until the client
  has it.
- **Vitest 5 has no bench mode**, so the benchmark is a test with a generous budget that logs its numbers.
- **The clinic's range** is the 36 months up to the current month (UTC), default the last 12; the platform's is the
  published months, default the last 12. The platform heatmap offers the quarters whose three months are published.
- **Heatmap and rises** on All of Salus take the region (or the city's region); an entity or "All of BiH" asks for all.
- **Labels of a bar** are cut to the label column; the full name is in the mark's accessible name, tooltip and table.
- `vite.config.ts` reads `analytics.css` as text in tests too (the reduced-motion test).
