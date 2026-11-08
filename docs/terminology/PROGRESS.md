# Medical terminology progress

| Package | Status | Commit |
| --- | --- | --- |
| 0.1 Baseline | done (integrated 2026-11-07) | 9b906c3 |
| 1.1 internal/terminology: service and API | done (integrated 2026-11-08) | 8f9f6ff |
| 1.2 tools/terminology: the seed | done (integrated 2026-11-08) | cd5ded5 |
| 2.1 salusctl: import, export and curation | done (integrated 2026-11-08) | 7a25651 |
| 2.2 Web: the cheatsheet | done (integrated 2026-11-08) | 60a129d |
| 3.1 End-to-end, walk and the partner's file | done (integrated 2026-11-08) | a52e53d |

## Facts

- Plan written 2026-11-07 from the owner's answers in the desktop chat; SPEC §2 holds them.
- COPY.md: draft, approved when the owner starts the run.
- COPY.md approved by the owner when starting the run.

## Decisions

(none yet)

## Decided while the owner was away

#### The orchestrator

- **The shared stack and the integration steps run as in the languages plan** (`docs/i18n/PROGRESS.md`, "Decided
  while the owner was away", the orchestrator): started with `docker compose up -d --build --wait` from the
  integration worktree, host ports moved off those another local project holds, and each package's decisions
  and new copy copied here from its own log section at integration.
- **Wave 2 integration: the web dev server proxies `/terminology` to the API** (`web/vite.config.ts`, one line),
  as 2.2 found the cheatsheet otherwise gets `index.html` on `make web-dev`.
- **The seed is in the dev database**, loaded at wave 2's integration with `make terminology-import` after the
  stack was rebuilt from `feat/terminology` (which applied 00030): created 4,943, refused 0; anatomy 1,500,
  diagnosis 2,000, symptom 500, procedure 143, drug 800.
- **pgtest caps its pools at 8 connections** (`pgtest.PoolMaxConns`, test first): on this 28-core host the
  test packages' CPU-sized pools together passed Postgres's 400 connection slots ("too many clients", three
  packages in one `make test`). The terminology benchmark opens its own pool as wide as its concurrency.
- **A second CallScreen race fixed at its cause**: the join test asserted the room was shown the moment Leave
  rendered, but the stage shows it from an effect. It now waits for that condition (no timeout, no retry);
  under CPU load the probe failed 1 in 123 before, 0 in 164 after.
- **Walk step 3c ("cir" finds "čir") is left failing for the owner, not fixed at integration.** Accent folding
  works (the e2e journey finds "Čirzynt" by "cirzynt"), but the seed has no plain "čir" term in bs or hr, and the
  ranking puts 24 labels starting with "cir" in other languages first. A fix changes the ranking in `query.sql`
  (languages requested first, fewer search languages, or a whole-word tier) while the p95 is already at its
  limit, or fills the data, which is the partner's. Safest: record it and leave the choice to the owner.
- **The p95 at 50 concurrent is at or over the 15 ms target**: 13.5–15.2 ms (1.1), 14.9–15.8 ms (wave 2
  integration), 16.7–20.1 ms (3.1, host loaded by another project). The levers need a migration (a stored
  folded label or a GiST index); left to the owner.

#### 0.1 Baseline

- **`rank` holds the sitelinks count**, not a position: comparable across categories in a search without a
  category, and the export shows something a curator can read. Higher ranks first.
- **`review_status` is `text` with a check**, not a third enum, as `binary.status`; `term_reviewed_has_date`
  makes a reviewed term carry its date. Turning a term back to draft keeps its last `reviewed_at`.
- **No `version_id` on `term`**: it is not a FHIR resource and the app never writes it; the import is the
  only writer.
- **Word similarity (`<%`), not plain similarity (`%`)**, for the trigram tier: a multi-word label such as
  "ulcus ventriculi" is found by "ventricul", which whole-label similarity misses.
- **ATC codes**: the WHOCC's ATC/DDD index forbids commercial copying. Not one of the SPEC's stop points
  (Wikidata's licence and the ICD point both came out as the SPEC says), so the plan continues: the ATC code is
  taken only from Wikidata (CC0) as an identifier, nothing from the index, and the point is listed in
  `SOURCES.md` "For the lawyer" before production (SPEC §7). Safest without blocking: nothing reaches
  production before that check.
- **ICD**: WHO's classifications are CC BY-ND 3.0 IGO (commercial use allowed without adaptation; translations
  and mappings need WHO's agreement), not plainly "copyrighted": what we take is unchanged (no WHO text, the
  code only as an identifier from Wikidata). Recorded in `SOURCES.md`.
- **`term_label.label` at most 300 characters**: a bound for the index and the CSV round trip; Wikidata labels
  are limited to 250.

#### 1.1 internal/terminology: service and API

- **Staff is the session's clinic admin or practitioner** as `auth.Service.ClinicMember` reads it (not audited,
  like the other callers); a patient's session (no clinic) is refused without a read.
- **The UI language for the default** is the account's `language`, else its `detectedLanguage`, else none (Latin
  and English only), read through `auth.Service.Me` (audited as a read of the caller's own account) and only
  when the request has no `languages`; the cheatsheet (2.2) should send `languages`, so the read is rare. auth
  has no unaudited language read and is not this package's to change.
- **The search text is trimmed** before the checks and the search; blank, invalid UTF-8 or a NUL is 400.
- **`Cache-Control: no-store`** on the response: the search text is in the URL.
- **query.sql tuned** (0.1's contract allows it): the prefix tier stays in the ranking, but its LIKE is in the
  filter only for a one-character text (a label that starts with a text of n ≥ 2 characters shares n of its
  n+1 trigrams, ≥ 0.67 against pg_trgm's 0.6, so `<%` already finds it): one index scan instead of a BitmapOr
  that evaluated both conditions twice per candidate; each candidate label folded once (materialized CTE);
  the labels come back in the same query as three parallel arrays (one round trip). `TermLabels` removed
  (unused). Every 0.1 ranking test unchanged and green; two cases added (one letter, two letters).
- **The p95 target is met, without margin.** Further levers need a migration, which only 0.1 may change: a
  stored folded-label column (no `term_fold` in the recheck), or a GiST trigram index for a `<<->`
  nearest-neighbour order that stops at the limit. Not done; for the owner. The real seed will have fewer
  labels than the synthetic 88,330 (Wikidata leaves many languages empty); 2.1 or 3.1 can rerun the benchmark
  against the loaded seed to confirm.

#### 1.2 tools/terminology: the seed

- **Symptoms are a direct instance or subclass** (`wdt:P31|wdt:P279` of *symptom* Q169872 or *medical sign*
  Q1441305), not the transitive closure: `(wdt:P31|wdt:P279)/wdt:P279*` timed out on the query service both
  times we tried (60 s). The SPEC says "an instance or subclass of"; the direct form gives 1,317 candidates, more than the
  500 kept.
- **One category per item, claimed in the order symptom, anatomy, procedure, drug, diagnosis** (`wikidata_id` is
  unique, so an item can't sit in two). A symptom with an ICD-10 R code (headache, fever, pain) stays a symptom;
  diagnosis, the widest, takes what is left. Each ranking query asks for its limit plus the items already kept,
  so a category still fills.
- **"Medication or active ingredient" is an ATC code at the substance level** (seven characters, e.g. N02BE01):
  that level names one substance; shorter codes are groups. It is the simplest test Wikidata supports reliably
  (drug items are typed inconsistently); it lets in medical gases (oxygen, helium), which ATC classifies as
  medicines. The curator can drop what doesn't belong.
- **ICD-9-CM (P1692) codes count as procedure codes only in the procedure format** (two digits and an optional
  decimal part, e.g. 47.0); ICD-9-CM diagnosis codes (three digits) are not taken. They go into `pcs` with the
  ICD-10-PCS codes, as `term.pcs` is documented ("ICD-10-PCS or ICD-9-CM procedure code").
- **Procedure has 143 terms, not 500**: that is every item with either code. Not widened beyond the SPEC's
  selection; the curator or a later plan can add procedures.
- **Labels without a letter are dropped** (emoji such as "👀", bare numbers), and duplicates that differ only in
  case: noise for a search and for the curator.
- **A Latin-script `sr` label goes to `sr-Latn` as it is** and never into `sr-Cyrl` (Latin to Cyrillic is not one
  to one); `sr-ec` labels add to `sr-Cyrl`; Љ, Њ, Џ become LJ, NJ, DŽ before a capital, else Lj, Nj, Dž.
- **Codes are kept as Wikidata has them**, odd ones included (e.g. an ICD-10 `B2424.`, an ICD-10-PCS `B?2`): the
  tool copies, the curator corrects; filtering them would hide data errors from the review.
- **JSON key order** is `labels` first, then `wikidata`, `category`, codes, `rank` (struct order, which `govet`
  fieldalignment sets); the 0.1 contract's example order is not significant to a JSON reader.

#### 2.1 salusctl: import, export and curation

- **The seed import adds only terms that are not there** (by Wikidata id) and never changes an existing one, its
  rank and codes included: a later fetch must not overwrite the partner's labels or review. New seed data for an
  existing term reaches it through the CSV.
- **A CSV row without an id is refused** ("new terms come from the seed"): the SPEC's import is an upsert by id
  with unknown ids refused, and inserting id-less rows would add a duplicate on every re-import (not idempotent).
  Clinic- or curator-added terms are out of this feature (SPEC §3).
- **A CSV header with an unknown, missing or repeated column refuses the whole file** before any write: a column
  renamed in a spreadsheet would otherwise empty that language's labels on every row. A row with another number
  of cells, or a rank that is not a whole number, is refused for that row. The SPEC's "an unknown column … is
  refused for that row" read this way: a header column is not a row's.
- **A row equal to its stored term is not written** (labels compared in any order), so re-importing a file
  leaves `updated_at` alone: idempotent in effect, and the report's `unchanged` tells the curator what they
  touched.
- **A repeated id in one file is refused after its first row**, so the result never depends on row order.
- **Synonyms are joined with ` | `** in one cell: labels in the seed contain `;` and `,` but never `|`; a label
  with `|` is refused at import so the round trip stays exact.
- **A cell starting with `=`, `+`, `-`, `@`, a tab or a carriage return gets a leading `'`** on export and loses it
  on import: Wikidata text is open to anyone's edits and must not run as a spreadsheet formula on the partner's
  machine (e.g. `-(+)-biotin`).
- **A byte order mark starts the export** (Excel reads UTF-8 CSV without it as the local code page, breaking
  Cyrillic and Arabic); the import skips it when present.
- **Not audited**: the terms are reference data without personal data (SPEC §4), as the search is not.
- **The `reviewed_at` set on the transition is salusctl's clock**, not the file's: the CSV has no date column, so
  a curator can't backdate a review.
- **The dev database was not migrated or loaded.** It is at migration 00028 (the wave notes said 00030):
  `term` does not exist there. Migrating it would apply the languages plan's 00029, which rewrites questionnaire
  rows, beyond the one write wave 2 is allowed; that belongs to the integration step. The load was verified on a
  throwaway database instead (above). The integration step runs `make migrate`, then `make terminology-import`,
  and should see the same counts.

#### 2.2 Web: the cheatsheet

- **"More languages" adds the other staff languages** (Bosnian, Croatian, Serbian in both scripts, whichever are
  not shown already), not German, Arabic, Turkish, Italian or French: SPEC §1 and T2 name Latin and the local
  languages and English, and the web app has names only for its own languages.
- **One request carries all six languages** (Latin, UI, English, then the others, `languages` comma separated as
  the contract's `explode: false`), so "More languages" shows at once without a second search and the API
  never reads the account's language (1.1's audited fallback).
- **Language names in the cells are translated per UI language** (e.g. "Engleski" in the Croatian UI), from the
  terminology catalog, not the endonyms of `web/src/language.ts`: T3 has "Latin" and "English" translated, and
  mixing endonyms with translated names in one row reads oddly.
- **Each language cell carries its own small language caption** instead of a table header row, so the same
  markup reads at 390 px with the cells stacked.
- **A Copy button per shown preferred label**; synonyms are shown but not copied by a button (SPEC: "a copy
  button on each label"; one per cell keeps the row calm). "Copied" stays on that button until the next copy or
  answer; no timer.
- **A refused clipboard write shows nothing** (no "Copied"); no new error text.
- **Enter on a term without a UI-language label copies nothing.** The cell shows "No term in … yet".
- **Ctrl+K and ⌘K both open it on every platform**; only the name in the tooltip follows the platform
  (`navigator.platform` matching Mac, iPhone, iPad).
- **ICD-10 is "MKB-10" in Bosnian, Croatian and Serbian** (МКБ-10 in Cyrillic), as doctors in BiH write it; TA98,
  ATC and ICD-10-PCS stay as they are.
- **Bosnian and Serbian "Procedura", Croatian "Postupak"** for Procedure; "izraz" rather than "pojam" in T11
  (`{language}: još nema izraza`), because "nema pojma" reads as "no idea".
- **The palette's styles are in `web/src/terminology/cheatsheet.css`**, imported by the component, so `app.css`
  (not this package's) is untouched.

#### 3.1 End-to-end, walk and the partner's file

- **The e2e journeys add and delete their own synthetic term** instead of using a seed term: the dev database keeps
  its seed exactly (checked after the runs: 4,943 terms, all draft) and CI, whose stack has no seed, runs the same
  journeys. The cleanup deletes the term as the owner (`DATABASE_URL` of the playwright service, as `startAt`
  writes). No delete command was added to salusctl (out of 3.1's files and not in the SPEC).
- **Step 5 set the liver back to draft** after the walk: the SPEC's curation is the partner's ("reviews every term"),
  so a term marked reviewed by a walk would tell the partner something untrue. The import of a `draft` row is
  allowed (0.1: "Turning a term back to draft keeps its last `reviewed_at`"); the liver's labels are unchanged, its
  `reviewed_at` keeps the walk's time.
- **Step 3c is recorded as failed, not fixed**: the cause is the ranking in `query.sql` (1.1's file) and the seed's
  data; 3.1 owns the tests and the walk only.

## Waiting on people

- The partner's review of the terms (SPEC §7), from the CSV 3.1 writes.

## Log

### 0.1 Baseline

2026-11-07, done. Worktree `salus-term-0.1`, branch `terminology/0.1` from `feat/terminology`.

Built:
- `migrations/00030_terminology.sql`: extensions `pg_trgm` and `unaccent`; `term_fold(text)` (lower case,
  unaccented; an immutable `BEGIN ATOMIC` SQL function over `public.unaccent('public.unaccent', …)`); enums
  `term_category` and `term_language` (`la` plus the ten app tags, its own enum beside `language`); tables
  `term` and `term_label` (SPEC §3 "Database"), one preferred label per term and language (partial unique
  index), labels trimmed, not blank, at most 300 characters; a reviewed term has its `reviewed_at`; a GIN
  trigram index on `term_fold(label)`; `GRANT SELECT` only to `salus_app`. Tests in `migrations/terminology_test.go`.
- `ownership.yaml` (`term`, `term_label`: owner `terminology`, `personal_data: false`), archcheck context
  `reference: [terminology]`, CLAUDE.md's domain list, `sqlc.yaml` entry `terminologydb` (every other
  `<domain>db/models.go` regenerates with the new enums and tables, as with every migration).
- `internal/terminology`: `model.go` (`Term`, `Label`, `Search`), `errors.go` (`ErrForbidden`,
  `ErrInvalidSearch` for 1.1), `repository.go` (`Search`), `query.sql` (`SearchTerms`, `TermLabels`),
  `repository_postgres.go` and its tests (exact, prefix, trigram, accents and case, word inside a label,
  misspelling, rank tie-break, category filter, LIKE wildcards literal, limit, labels per requested language).
- `api/openapi.yaml`: `GET /terminology/search` (`searchTerminology`, tag `terminology`) with `q`,
  `languages` (comma separated), `category`, `limit` (1–25, default 10); schemas `TermLanguage`, `TermCategory`,
  `TermCodes`, `TermLabels`, `Term`, `TermList`; errors `invalid_request` (400), `unauthenticated` (401),
  `forbidden` (403), 500. No new error code. Clients regenerated (`web/src/api/schema.d.ts`,
  `mobile/packages/salus_api`).
- `Makefile`: `terminology-fetch`, `terminology-import`, `terminology-export`, `bench-terminology`.
- `docs/terminology/SOURCES.md`: the licence checks, each with a link.

Test first (failures seen):
- `migrations`: `ERROR: relation "term" does not exist (SQLSTATE 42P01)`, `function term_fold(unknown) does not exist`.
- `internal/terminology` with `Search` returning nothing: `Search("hepar", "") = [], want [… …]` for every ranking
  case, `Search(jetra) = [], <nil>; want one term`. Then `unable to encode []terminologydb.TermLanguage … unknown
  type`: the languages travel as `text[]` (`language::text = ANY(…)`). Then "hepar" also found *hepatitis* by
  word similarity (0.67): correct behaviour, the case now asserts exact, then prefix, then trigram.

Checks (all through `docker compose … run --rm --no-deps -T`):
- `go test -count=1 -cover ./internal/terminology/... ./migrations/`: ok; `internal/terminology` 95.5% (the
  uncovered lines are the two query error returns; `terminologydb` is generated and excluded).
- `scripts/lint.sh` (gofmt, vet, golangci-lint both modules, check-migrations): 0 issues, ok.
- archcheck: exit 0.
- sqlc: regenerated; a second run leaves no diff.
- Clients: `npm run -s gen:api`, openapi-generator dart, `dart format`: a second run leaves no diff.
- `scripts/i18n-check.sh` not run: 0.1 adds no catalog string.

Contracts for the next packages:
- **Search ranking** (`SearchTerms`): a term counts by its best label in any language; tier 3 exact
  (`term_fold(label) = term_fold(q)`), 2 prefix (`LIKE term_fold(escaped q) || '%'`, `\ % _` escaped by the
  repository), 1 word similarity (`term_fold(q) <% term_fold(label)`, pg_trgm's default threshold 0.6); then
  `word_similarity` descending, `rank` descending, id. Both conditions can use the GIN trigram index; 1.1
  measures it (`make bench-terminology`) and may tune `query.sql`.
- **`rank`** is the Wikidata item's sitelinks count (higher = more common), so it compares across categories.
- **Codes**: one column per kind; several codes of one kind joined with `", "`; `wikidata_id` matches
  `^Q[1-9][0-9]*$` and is unique; empty strings are refused (use NULL).
- **1.1**: `Repository.Search(ctx, Search{Text, Category, Languages, Limit})` returns `[]Term` with `Labels`
  only in `Languages`, ordered by language (enum order), preferred first, then synonyms alphabetically; the
  service validates (q 1–100 characters, limit 1–25, known languages and category → `ErrInvalidSearch`; a
  patient → `ErrForbidden`) and builds the response: `labels` one entry per requested language that has a
  label, in request order, `preferred` the marked label or the first alphabetically. Benchmark:
  `BenchmarkSearch` in `internal/terminology`, run by `make bench-terminology` (`-run '^$' -bench
  BenchmarkSearch -benchtime=2000x -count=3`); it reports the p95 at 50 concurrent with `b.ReportMetric(…,
  "p95-ms")`.
- **1.2**: `tools/terminology` is `package main` in the tools module, run as `go -C tools run ./terminology -out
  ../assets/terminology/seed.jsonl` (`make terminology-fetch`). The seed is JSON Lines, one term per line,
  sorted by category (enum order) then rank descending then Wikidata id: `{"wikidata":"Q9368",
  "category":"anatomy","rank":123,"icd10":"…","ta98":"…","atc":"…","pcs":"…","labels":{"la":["hepar","iecur"],
  "bs":["jetra"],…}}`, codes absent when none, each language's array with the preferred label first, a language
  with no label absent. 1.2 records the fetch date and the queries in `SOURCES.md` "Used".
- **2.1**: `salusctl terminology import <file>` (`.jsonl` = seed: upsert by `wikidata_id`, new terms `draft`;
  `.csv` = curated: upsert by `id`) and `salusctl terminology export -o <path> [-status draft|reviewed]`
  (`make terminology-import [file=…]`, `make terminology-export out=<path> [status=draft]`; the path is inside
  the container, under `/src`). Write queries go in `query.sql` with `make sqlc`; salusctl connects as the
  owner, which has every privilege on the tables.

Decided while the owner was away:
- **`rank` holds the sitelinks count**, not a position: comparable across categories in a search without a
  category, and the export shows something a curator can read. Higher ranks first.
- **`review_status` is `text` with a check**, not a third enum, as `binary.status`; `term_reviewed_has_date`
  makes a reviewed term carry its date. Turning a term back to draft keeps its last `reviewed_at`.
- **No `version_id` on `term`**: it is not a FHIR resource and the app never writes it; the import is the
  only writer.
- **Word similarity (`<%`), not plain similarity (`%`)**, for the trigram tier: a multi-word label such as
  "ulcus ventriculi" is found by "ventricul", which whole-label similarity misses.
- **ATC codes**: the WHOCC's ATC/DDD index forbids commercial copying. Not one of the SPEC's stop points
  (Wikidata's licence and the ICD point both came out as the SPEC says), so the plan continues: the ATC code is
  taken only from Wikidata (CC0) as an identifier, nothing from the index, and the point is listed in
  `SOURCES.md` "For the lawyer" before production (SPEC §7). Safest without blocking: nothing reaches
  production before that check.
- **ICD**: WHO's classifications are CC BY-ND 3.0 IGO (commercial use allowed without adaptation; translations
  and mappings need WHO's agreement), not plainly "copyrighted": what we take is unchanged (no WHO text, the
  code only as an identifier from Wikidata). Recorded in `SOURCES.md`.
- **`term_label.label` at most 300 characters**: a bound for the index and the CSV round trip; Wikidata labels
  are limited to 250.

COPY.md additions (not yet reviewed): none (0.1 adds no user-facing text).

Dev stack: nothing started, stopped or written; the migration reaches the dev database when the integration
step runs `make migrate`.

### 1.1 internal/terminology: service and API

2026-11-08, done. Worktree `salus-term-1.1`, branch `terminology/1.1` from `feat/terminology` after 0.1.

Built:
- `internal/terminology/service.go`: `Service.Search(ctx, Caller, Search) ([]Match, error)`: staff only
  (`ErrForbidden` for a patient), the text trimmed and checked (1–100 characters, valid UTF-8, no NUL), limit
  1–25, known category, known and unrepeated languages (`ErrInvalidSearch`); without languages, Latin, the
  caller's UI language and English (deduplicated). Labels grouped per requested language in request order
  (`Names`: preferred, synonyms); a language without a label is left out. Errors never hold the search text.
- `internal/terminology/model.go`: `Caller`, `Match`, `Names`.
- `internal/terminology/handler_http.go`: `GET /terminology/search` (`q`, `languages` comma separated,
  `category`, `limit` default 10), the `Term`/`TermList` JSON of `api/openapi.yaml`, codes absent when empty,
  `Cache-Control: no-store` (the URL holds the search text); 400 `invalid_request`, 401 `unauthenticated`,
  403 `forbidden`, 500 through `httpx.InternalError`.
- `internal/api/app.go`: the route mounted; `terminologyCaller` (staff = the session's clinic admin or
  practitioner, through `auth.Service.ClinicMember`; a patient's session needs no read); `uiLanguage` (the
  account's `language`, else `detectedLanguage`, through `auth.Service.Me`); `readiness` split out of `New` to
  keep it under 80 lines. No change to `internal/api/router` was needed: the access log writes the route
  pattern, never the URL; `internal/api/app_terminology_test.go` asserts it for this route, with the 400 path.
- `internal/terminology/query.sql` tuned (below); `repository_postgres.go` maps the one query.
- `internal/terminology/search_bench_test.go`: `BenchmarkSearch` (`make bench-terminology`).

Test first (failures seen):
- service: `undefined: terminology.Caller`, then with an empty `Search`: `Search as a patient = <nil> …; want
  ErrForbidden`, `Search(limit 26) = <nil> …; want ErrInvalidSearch`, `UI language "hr": searched in <nil>; want
  [la hr en]`, `Search = []` for the grouping case.
- handler: `undefined: terminology.NewHTTPHandler`, then `nobody signed in: GET = 200 ; want 401`, `a patient:
  GET = 200 ; want 403`, `headers map[]; want JSON, no-store`.
- app: `route=unmatched status=404` (route not mounted); the detected-language fallback with `uiLanguage`
  returning "": `GET as the admin (no choice, hr detected) = 200 … want 200 with Latin, Croatian and English`.
- repository, one letter (after the tuning dropped the LIKE from the filter): `one letter: … Search("u", "") =
  [], want [hepatitis ulcer]`; green with the one-character branch.

Benchmark (`go test -run '^$' -bench BenchmarkSearch -benchtime=2000x -count=3 ./internal/terminology`, as
`make bench-terminology`, through `docker compose … run --rm --no-deps tools`; i7-14700HX, 28 threads, so the
app's default pool has 28 connections; the shared compose Postgres; other wave-1 packages running tests at the
same time, load average 7 to 37). Data: synthetic, generated in a pgtest database: 5,300 terms (anatomy 1,500,
diagnosis 2,000, symptom 500, procedure 500, drug 800), 88,330 labels (Latin for 70% of terms, each app
language for 85%, one to three labels of one to three words, words from per-script syllables: South Slavic
Latin, Cyrillic, Arabic, Latin), then `VACUUM ANALYZE` (the state after an import once autovacuum has run).
Searches: a third each of 3–6-letter prefixes, whole labels, labels with one letter missing; languages la, hr,
en; limit 10. `concurrent=1` is one search at a time, `concurrent=50` the SPEC's measure.

| Query version | alone p50 / p95 | 50 concurrent p50 / p95 |
| --- | --- | --- |
| 0.1's query, words from 60 two-letter syllables, no vacuum | – | 29–44 / 61–71 ms |
| 0.1's query, realistic syllables, no vacuum (GIN pending list) | – | 32 / 53 ms |
| 0.1's query, vacuumed | 1.6 / 5.8–6.5 ms | 6.4–6.6 / 17.4–18.7 ms |
| label folded once (materialized) | 1.4–1.9 / 4.6–6.4 ms | 5.8–6.1 / 17.3–19.3 ms |
| + LIKE in the filter only for a one-character text | 1.3–1.5 / 4.5–5.2 ms | 4.2–5.5 / 14.0–16.3 ms |
| + labels in the same query (committed) | 1.2–1.5 / 3.5–4.8 ms | 4.4–5.0 / **13.5–15.2 ms** |

Back-to-back A/B of the last two rows (same load, two rounds of three): two queries p95 14.1–16.3 ms, one query
13.5–15.2 ms. The committed version meets the SPEC's p95 under 15 ms in 5 of 6 runs, with no margin. The tail
is short texts (3–4 letters) matching 600–1,600 labels by word similarity, each checked with `word_similarity`
(about 2.3 µs a label; `term_fold` about 0.5 µs).

Checks (all through `docker compose … run --rm --no-deps -T`):
- `go test -count=1 -cover ./internal/terminology/... ./internal/api/... ./migrations/`: ok; terminology 99.0%,
  api 91.6%, router 100%.
- `scripts/lint.sh`: 0 issues (both modules), check-migrations ok.
- archcheck: exit 0.
- sqlc: regenerated; a second run leaves no diff.
- Clients, catalogs, web, mobile: not run; 1.1 changes no contract, catalog or client code.

Decided while the owner was away:
- **Staff is the session's clinic admin or practitioner** as `auth.Service.ClinicMember` reads it (not audited,
  like the other callers); a patient's session (no clinic) is refused without a read.
- **The UI language for the default** is the account's `language`, else its `detectedLanguage`, else none (Latin
  and English only), read through `auth.Service.Me` (audited as a read of the caller's own account) and only
  when the request has no `languages`; the cheatsheet (2.2) should send `languages`, so the read is rare. auth
  has no unaudited language read and is not this package's to change.
- **The search text is trimmed** before the checks and the search; blank, invalid UTF-8 or a NUL is 400.
- **`Cache-Control: no-store`** on the response: the search text is in the URL.
- **query.sql tuned** (0.1's contract allows it): the prefix tier stays in the ranking, but its LIKE is in the
  filter only for a one-character text (a label that starts with a text of n ≥ 2 characters shares n of its
  n+1 trigrams, ≥ 0.67 against pg_trgm's 0.6, so `<%` already finds it): one index scan instead of a BitmapOr
  that evaluated both conditions twice per candidate; each candidate label folded once (materialized CTE);
  the labels come back in the same query as three parallel arrays (one round trip). `TermLabels` removed
  (unused). Every 0.1 ranking test unchanged and green; two cases added (one letter, two letters).
- **The p95 target is met, without margin.** Further levers need a migration, which only 0.1 may change: a
  stored folded-label column (no `term_fold` in the recheck), or a GiST trigram index for a `<<->`
  nearest-neighbour order that stops at the limit. Not done; for the owner. The real seed will have fewer
  labels than the synthetic 88,330 (Wikidata leaves many languages empty); 2.1 or 3.1 can rerun the benchmark
  against the loaded seed to confirm.

COPY.md additions (not yet reviewed): none (1.1 adds no user-facing text).

Dev stack: nothing started, stopped or written; the tests and the benchmark use throwaway pgtest databases.

### 1.2 tools/terminology: the seed

2026-11-08, done. Worktree `salus-term-1.2`, branch `terminology/1.2` from `feat/terminology`.

Built:
- `tools/terminology` (`package main` in the tools module, standard library only): `sparql.go` (POST to the
  endpoint, `User-Agent: Salus-terminology-seed/1.0 (https://github.com/Shmeeshmee/salus; medical term list for
  a telemedicine app)`, no e-mail; one request at a time; on 429, 502, 503, 504 it waits `Retry-After` or a
  doubling backoff from 5 s, five tries), `fetch.go` (one ranking query per category by sitelinks, then labels,
  aliases and codes in batches of 100 items), `seed.go` (labels: trim, collapse white space, drop labels without
  a letter, over 300 characters or repeated in another case; preferred first, synonyms in byte order; the
  Serbian rule; JSON Lines sorted by category, rank descending, Wikidata id numerically), `main.go` (`-out`,
  `-endpoint`; writes a temporary file and renames it, so a failed fetch keeps the old seed; logs the counts).
- Tests against responses recorded from Wikidata on 2026-11-07 (`testdata/rank-<category>.json`,
  `testdata/details.json`, small limits) with a golden seed (`testdata/seed.golden.jsonl`, `-update`).
- `assets/terminology/seed.jsonl` (fetched 2026-11-07 23:14 UTC, 76 s) and `assets/terminology/README.md`;
  the fetch date and the queries in `SOURCES.md` "Used".

Seed counts: anatomy 1,500; diagnosis 2,000; symptom 500; procedure 143 (limit 500); drug 800; total 4,943
terms, 73,529 labels. Terms with a label per language: en 4,943, fr 4,782, ar 4,729, de 4,123, it 3,995,
sr-Latn 2,523, tr 2,507, sr-Cyrl 1,963, la 1,601, hr 1,226, bs 1,169.

Test first (failures seen):
- `sparql_test.go` before `sparql.go`: `undefined: sparql`, `undefined: statusError`, `undefined: sleepCtx`.
- `seed_test.go` against stubs: `buildLabels = map[]  want map["en":["liver"] "la":["hepar" "a b" "iecur"]]`,
  `transliterate("Љубљана") = "", want "Ljubljana"`, the seed empty. Then the recorded data had emoji aliases
  (`"👀"`): new case, `want map["en":["liver"] …]` failed until labels without a letter were dropped.
- `fetch_test.go` against stubs: `rank queries = [], want [symptom=3 anatomy=6 procedure=8 drug=10
  diagnosis=13]`, `Q1 codes: icd10 "", pcs ""`, `0 categories, want 5`.
- `main_test.go`: `undefined: run`.

Checks (all through `docker compose … run --rm --no-deps -T`):
- `go -C tools test -count=1 -cover ./terminology/`: ok, 86.2% (`main()` itself is the uncovered part).
- `scripts/lint.sh` (gofmt, vet, golangci-lint both modules, check-migrations): 0 issues, ok.
- archcheck: exit 0.

Decided while the owner was away:
- **Symptoms are a direct instance or subclass** (`wdt:P31|wdt:P279` of *symptom* Q169872 or *medical sign*
  Q1441305), not the transitive closure: `(wdt:P31|wdt:P279)/wdt:P279*` timed out on the query service both
  times we tried (60 s). The SPEC says "an instance or subclass of"; the direct form gives 1,317 candidates, more than the
  500 kept.
- **One category per item, claimed in the order symptom, anatomy, procedure, drug, diagnosis** (`wikidata_id` is
  unique, so an item can't sit in two). A symptom with an ICD-10 R code (headache, fever, pain) stays a symptom;
  diagnosis, the widest, takes what is left. Each ranking query asks for its limit plus the items already kept,
  so a category still fills.
- **"Medication or active ingredient" is an ATC code at the substance level** (seven characters, e.g. N02BE01):
  that level names one substance; shorter codes are groups. It is the simplest test Wikidata supports reliably
  (drug items are typed inconsistently); it lets in medical gases (oxygen, helium), which ATC classifies as
  medicines. The curator can drop what doesn't belong.
- **ICD-9-CM (P1692) codes count as procedure codes only in the procedure format** (two digits and an optional
  decimal part, e.g. 47.0); ICD-9-CM diagnosis codes (three digits) are not taken. They go into `pcs` with the
  ICD-10-PCS codes, as `term.pcs` is documented ("ICD-10-PCS or ICD-9-CM procedure code").
- **Procedure has 143 terms, not 500**: that is every item with either code. Not widened beyond the SPEC's
  selection; the curator or a later plan can add procedures.
- **Labels without a letter are dropped** (emoji such as "👀", bare numbers), and duplicates that differ only in
  case: noise for a search and for the curator.
- **A Latin-script `sr` label goes to `sr-Latn` as it is** and never into `sr-Cyrl` (Latin to Cyrillic is not one
  to one); `sr-ec` labels add to `sr-Cyrl`; Љ, Њ, Џ become LJ, NJ, DŽ before a capital, else Lj, Nj, Dž.
- **Codes are kept as Wikidata has them**, odd ones included (e.g. an ICD-10 `B2424.`, an ICD-10-PCS `B?2`): the
  tool copies, the curator corrects; filtering them would hide data errors from the review.
- **JSON key order** is `labels` first, then `wikidata`, `category`, codes, `rank` (struct order, which `govet`
  fieldalignment sets); the 0.1 contract's example order is not significant to a JSON reader.

COPY.md additions (not yet reviewed): none (no user-facing text).

Dev stack: nothing started, stopped or written; the seed reaches the dev database in 2.1.

### 2.1 salusctl: import, export and curation

2026-11-08, done. Worktree `salus-term-2.1`, branch `terminology/2.1` from `feat/terminology` after wave 1 and the
languages plan's wave 1.

Built:
- `internal/terminology/service_curation.go`: `Curator` (salusctl's, on the owner's connection; the app never
  builds one), `ImportSeed`, `ImportCurated`, `Export`, `Counts`. Every row is checked (category, Wikidata id
  form, rank not negative, review status, known languages, labels with white space collapsed, empty ones and
  repeats dropped, at most 300 characters, no `|`, one preferred label per language) and runs in its own
  transaction: an invalid row is refused with a reason naming the column (never a value) and the rest applied; a
  database error stops the import (the rows before it stay applied; rerunning is safe).
- `internal/terminology/handler_file.go`: `DecodeSeed` (the 0.1 JSON Lines contract, unknown keys refused per
  line), `EncodeCSV` and `DecodeCSV` (the curator's file, below).
- `internal/terminology/repository_postgres.go` and `query.sql`: `NewPostgresCurationRepository` with
  `CreateFromSeed` (`ON CONFLICT (wikidata_id) DO NOTHING`), `Term`, `UpdateTerm` (columns, then labels deleted
  and inserted from three parallel arrays), `Terms` (one query with every label), `CountByCategory`; the
  `CurationRepository` port in `repository.go`; `Note` on `Term`; `ErrTermNotFound`, `ErrDuplicateWikidata`,
  `ErrInvalidStatus`.
- `cmd/salusctl/terminology.go`: `salusctl terminology import <seed.jsonl | terms.csv>` (by extension) prints
  `created, updated, unchanged, refused`, a line per refused row (`line N (id …): reason`) and the terms per
  category, and exits non-zero when a row was refused (after applying the rest); `salusctl terminology export -o
  <path> [-status draft|reviewed]`. Both as the 0.1 Makefile targets call them (`make terminology-import
  [file=…]`, `make terminology-export out=… [status=…]`).

The curator's CSV: UTF-8 with a byte order mark; columns `id, category, wikidata, icd10, ta98, atc, pcs, rank`,
the preferred label per language (`la, bs, hr, sr-Latn, sr-Cyrl, en, de, ar, tr, it, fr`), the synonyms per
language (`la synonyms` … `fr synonyms`, joined with ` | `), `review_status, note`. One row per term, by category,
rank descending, id. An import takes the columns in any order.

Loaded (the real seed, on a throwaway database migrated to 00030 on the compose Postgres, then dropped; see "Dev
stack"): `created 4943, updated 0, unchanged 0, refused 0`; terms: anatomy 1,500, diagnosis 2,000, symptom 500,
procedure 143, drug 800 (the seed's counts in 1.2); 73,529 labels; all draft. A second seed import: `unchanged
4943` in 0.7 s. The full export imported back: `unchanged 4943` in 1.1 s (the round trip is exact on the real
data). First load 35 s (4,943 transactions, one per row as the SPEC's per-row refusal needs; a one-off operator
command, so not optimised).

Test first (failures seen):
- repository: `undefined: terminology.NewPostgresCurationRepository`; then sqlc: `function unnest(unknown,
  unknown, unknown) does not exist` (multi-argument unnest in FROM; the select-list form zips the arrays).
- service, against a stub `Curator`: `ImportSeed = {Refused:[] Created:0 …}; want one created, one unchanged`,
  `ImportSeed with the database gone: error = nil`, `Export(done) = <nil>, want ErrInvalidStatus`, `ImportCurated
  = {…}; want 3 updated, 1 unchanged, refused map[…]`. Then `want … 9:en: a label with |` before the `|` rule.
- files, against stubs: `EncodeCSV = (empty) want …`, `DecodeCSV = 0 rows …; want 2 rows`, `DecodeCSV with a header
  with a column unknown: error = nil`; then a refused row's id taken from cell 0 instead of the id column.
- salusctl: `import = unknown command "terminology"; …`, `0 draft terms, want 2`.

Checks (all through `docker compose … run --rm --no-deps -T`):
- `go test -count=1 -cover ./internal/terminology/... ./cmd/salusctl/ ./internal/api/`: ok; terminology 94.5%,
  salusctl 85.7%, api 91.6% (unchanged; run because it builds the terminology service).
- `scripts/lint.sh` (gofmt, vet, golangci-lint both modules, check-migrations): 0 issues, ok.
- archcheck: exit 0.
- sqlc: regenerated (`terminologydb` only); a second run leaves no diff.
- Clients, catalogs, web, mobile: not run; 2.1 changes no contract, catalog or client code.

Decided while the owner was away:
- **The seed import adds only terms that are not there** (by Wikidata id) and never changes an existing one, its
  rank and codes included: a later fetch must not overwrite the partner's labels or review. New seed data for an
  existing term reaches it through the CSV.
- **A CSV row without an id is refused** ("new terms come from the seed"): the SPEC's import is an upsert by id
  with unknown ids refused, and inserting id-less rows would add a duplicate on every re-import (not idempotent).
  Clinic- or curator-added terms are out of this feature (SPEC §3).
- **A CSV header with an unknown, missing or repeated column refuses the whole file** before any write: a column
  renamed in a spreadsheet would otherwise empty that language's labels on every row. A row with another number
  of cells, or a rank that is not a whole number, is refused for that row. The SPEC's "an unknown column … is
  refused for that row" read this way: a header column is not a row's.
- **A row equal to its stored term is not written** (labels compared in any order), so re-importing a file
  leaves `updated_at` alone: idempotent in effect, and the report's `unchanged` tells the curator what they
  touched.
- **A repeated id in one file is refused after its first row**, so the result never depends on row order.
- **Synonyms are joined with ` | `** in one cell: labels in the seed contain `;` and `,` but never `|`; a label
  with `|` is refused at import so the round trip stays exact.
- **A cell starting with `=`, `+`, `-`, `@`, a tab or a carriage return gets a leading `'`** on export and loses it
  on import: Wikidata text is open to anyone's edits and must not run as a spreadsheet formula on the partner's
  machine (e.g. `-(+)-biotin`).
- **A byte order mark starts the export** (Excel reads UTF-8 CSV without it as the local code page, breaking
  Cyrillic and Arabic); the import skips it when present.
- **Not audited**: the terms are reference data without personal data (SPEC §4), as the search is not.
- **The `reviewed_at` set on the transition is salusctl's clock**, not the file's: the CSV has no date column, so
  a curator can't backdate a review.
- **The dev database was not migrated or loaded.** It is at migration 00028 (the wave notes said 00030):
  `term` does not exist there. Migrating it would apply the languages plan's 00029, which rewrites questionnaire
  rows, beyond the one write wave 2 is allowed; that belongs to the integration step. The load was verified on a
  throwaway database instead (above). The integration step runs `make migrate`, then `make terminology-import`,
  and should see the same counts.

COPY.md additions (not yet reviewed): none (operator output only, no product text).

Dev stack: nothing started, stopped or written in the dev database `salus`. A throwaway database
`salus_term21_load` was created on the compose Postgres for the load, then dropped.

### 2.2 Web: the cheatsheet

2026-11-08, done. Worktree `salus-term-2.2`, branch `terminology/2.2` from `feat/terminology` after terminology
wave 1 and languages wave 1.

Built:
- `web/src/terminology/Cheatsheet.tsx`: the header button (a search icon, named and titled "Search medical terms
  (Ctrl+K)", "(⌘K)" on macOS) and the palette it opens, also on Ctrl+K or ⌘K anywhere. A modal dialog: a
  combobox (at most 100 characters) over a listbox of terms; the search waits 150 ms after the last change and
  aborts the one before (`AbortController`), so only the last text's answer is shown; blank text asks nothing.
  Each term shows its category (codes behind its tooltip, `TipTag`), "Not reviewed yet" on a draft, and a cell
  per language (Latin, the UI language, English; "More languages" adds the other staff languages): the
  preferred label with a Copy button ("Copied" after), its synonyms, or "No term in … yet". Keyboard: arrows
  move the chosen term (`aria-activedescendant`), Enter copies its UI-language label (announced in a status
  region), Escape closes and gives the focus back, Tab stays inside. Empty and error states; the source line
  in the footer.
- `web/src/terminology/cheatsheet.css`: the palette near the top of the page, full screen at 390 px with the
  language cells stacked; a 120 ms fade only under `prefers-reduced-motion: no-preference`. One custom icon
  (the search glass, the app's 1.6 stroke).
- `web/src/i18n/terminology/{en,bs,hr,sr-Latn,sr-Cyrl}.json`: COPY.md T1 to T12 and the additions below.
- The shell: `web/src/App.tsx` gets one mount line, `{staff && <Cheatsheet api={api} />}` after the
  navigation in the header, plus its import line. Nothing else in the shell changed.
- `web/src/i18n/catalog.test.ts`: `terminology` added to the list of areas it expects (the test fails on any
  new area; this package's plan adds one).

Test first (failures seen), `web/src/terminology/Cheatsheet.test.tsx` against a stub returning nothing:
- `Unable to find an accessible element with the role "button" and name "Search medical terms (Ctrl+K)"`,
  `Unable to find role="dialog"`, and every search case `expected 0 to be greater than 0`; 18 of 19 failed (the
  plain-K case passes by its nature).
- Then with the component and the catalogs: `expected 'terminology.failed' to be 'Pretraga trenutno nije
  dostupna. Pokušajte ponovo.'`: Vitest's persistent module cache (`web/node_modules/.vitest-cache`) had kept
  `import.meta.glob`'s result from before the new catalog folder existed. Cleared; green. A fresh install (CI)
  has no such cache; anyone adding a catalog area locally may need to clear it.
- `catalog.test.ts` `has the areas of the plan` failed once the area existed, as it should; updated.

Debounce and cancellation tests use `vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })`, advance
exactly 149 ms and 1 ms, and wait for conditions with `vi.waitFor`; no sleep, no retry. The late-answer test
holds the first request open, changes the text, then resolves it: the first request's signal is aborted and
its answer never shown. Eight back-to-back runs of the file: 19/19 each.

Checks (all through `docker compose -f deployments/docker-compose.yml run --rm --no-deps -T node`):
- `bash ../scripts/i18n-check.sh`: catalogs pass.
- `npm run lint` (oxlint): 0 warnings, 0 errors.
- `npm run typecheck`: ok.
- `npm test`: 31 files, 542 tests passed.
- `npm run build`: built (the existing over-500 kB chunk warning, unchanged in kind).
- Go, archcheck, sqlc, clients: not run; 2.2 changes no Go code, contract or client.
- Browser walk: not done; this session has no browser tool, and 3.1 walks (RUN-RULES).

For the integration step:
- **The Vite dev server does not proxy `/terminology`** (`web/vite.config.ts` lists each API prefix; 0.1 did not
  add this one). Without `'/terminology': apiUrl` in its `proxy`, the cheatsheet on `make web-dev` gets the
  app's `index.html` and shows the error state. `vite.config.ts` is not this package's file, so it is not
  changed here; the integration step or 3.1 adds the line before the walk.

Decided while the owner was away:
- **"More languages" adds the other staff languages** (Bosnian, Croatian, Serbian in both scripts, whichever are
  not shown already), not German, Arabic, Turkish, Italian or French: SPEC §1 and T2 name Latin and the local
  languages and English, and the web app has names only for its own languages.
- **One request carries all six languages** (Latin, UI, English, then the others, `languages` comma separated as
  the contract's `explode: false`), so "More languages" shows at once without a second search and the API
  never reads the account's language (1.1's audited fallback).
- **Language names in the cells are translated per UI language** (e.g. "Engleski" in the Croatian UI), from the
  terminology catalog, not the endonyms of `web/src/language.ts`: T3 has "Latin" and "English" translated, and
  mixing endonyms with translated names in one row reads oddly.
- **Each language cell carries its own small language caption** instead of a table header row, so the same
  markup reads at 390 px with the cells stacked.
- **A Copy button per shown preferred label**; synonyms are shown but not copied by a button (SPEC: "a copy
  button on each label"; one per cell keeps the row calm). "Copied" stays on that button until the next copy or
  answer; no timer.
- **A refused clipboard write shows nothing** (no "Copied"); no new error text.
- **Enter on a term without a UI-language label copies nothing.** The cell shows "No term in … yet".
- **Ctrl+K and ⌘K both open it on every platform**; only the name in the tooltip follows the platform
  (`navigator.platform` matching Mac, iPhone, iPad).
- **ICD-10 is "MKB-10" in Bosnian, Croatian and Serbian** (МКБ-10 in Cyrillic), as doctors in BiH write it; TA98,
  ATC and ICD-10-PCS stay as they are.
- **Bosnian and Serbian "Procedura", Croatian "Postupak"** for Procedure; "izraz" rather than "pojam" in T11
  (`{language}: još nema izraza`), because "nema pojma" reads as "no idea".
- **The palette's styles are in `web/src/terminology/cheatsheet.css`**, imported by the component, so `app.css`
  (not this package's) is untouched.

COPY.md additions (not yet reviewed):
- T13 `terminology.title` (the palette's and the result list's accessible name): "Medical terms".
- T14 `terminology.close` (the palette's close button, needed where there is no Escape key, e.g. on a phone):
  "Close".
- T15 `terminology.language.bs`, `.hr`, `.srLatn`, `.srCyrl` (cell captions and T11's `{language}`, beside T3's
  "Latin" and "English"): "Bosnian", "Croatian", "Serbian (Latin)", "Serbian (Cyrillic)".

Dev stack: nothing started, stopped or written. `npm ci` filled this worktree's `web/node_modules` (git-ignored).

### 3.1 End-to-end, walk and the partner's file

2026-11-08, done. Worktree `salus-term-3.1`, branch `terminology/3.1` from `feat/terminology` after wave 2 and the
whole languages plan.

Built:
- `test/e2e/terminology_test.go`: `TestTerminology_APractitionerFindsATermAndAPatientIsRefused` (the practitioner
  finds a term by its Latin label with its Croatian and English labels, `Cache-Control: no-store`, and by its
  Croatian label typed without the háček; a patient gets 403 `forbidden`, nobody signed in 401; then the web
  journey) and `TestTerminology_TheCuratorsRoundTrip` (the drafts' export holds the term; its row edited, Latin
  corrected, a note, `reviewed`, imported with a row of an unknown id: `updated 1, refused 1` naming the id and a
  non-zero exit; the search shows the new label as reviewed; the reviewed export has the note; the same row again:
  `unchanged 1`). Each journey adds its own synthetic symptom with `salusctl terminology import` of a one-line seed
  (a Wikidata id `Q9` + 15 digits, beyond Wikidata's own; made-up labels "Zyntheticum …", "Čirzynt …", "E2E
  synthetic term …") and deletes it in its cleanup as the owner, so the dev database's seed terms are never
  changed and a stack without the seed (CI) runs them the same.
- `test/e2e/web/terminology.spec.ts`: Ctrl+K on the calendar opens the palette; the Latin label finds the term with
  its English label and "Not reviewed yet"; Enter copies the English (UI) label to the clipboard ("Copied" in the
  status region); Escape closes; the header button opens it on the questionnaires screen.
- `docs/terminology/WALK.md` and `docs/terminology/walk/` (24 files, 1.4 MB): the acceptance walk, below.

Test first (failures seen):
- The search journey before the web spec existed: `the web journey failed: exit status 1 … Error: No tests found.`
- The web spec's first run: `expect(locator).toBeVisible() failed … getByRole('dialog', { name: 'Medical terms' })
  … <element(s) not found>`: Ctrl+K was pressed while the shell was still reading the account (the header, and its
  key listener, render only after it); the spec now waits for the header button first, as a user would see it.
- The round trip passed on its first run: it checks 2.1's behaviour, already built; it was not weakened to pass.

The walk (SPEC §5; details, evidence and the notes in `WALK.md`):

| Step | Result | Screenshots and texts (`docs/terminology/walk/`) |
| --- | --- | --- |
| 1 Seed and counts | Pass: `unchanged 4943`; anatomy 1,500, diagnosis 2,000, symptom 500, procedure 143, drug 800 | – |
| 2 Ctrl+K on calendar, builder, consultation | Pass | `step2-calendar(-390).png`, `step2-builder(-390).png`, `step2-consultation(-390).png` |
| 3a "jetra" | Partly: finds the liver with *liver*; its Latin is *iecur* in the seed, *hepar* only an English synonym | `step3-jetra(-390).png`, `step3-api.txt` |
| 3b "hepar" | Pass: finds *jetra* | `step3-hepar.png` |
| 3c "cir" | **Fail on the real seed**: no plain *čir* term; *Venski čir* ranks below 24 prefix matches in other languages | `step3-cir.png`, `step3-cir-ui.txt`, `step3-cir-with-hacek.png` |
| 4 sr-Cyrl and Croatian UI | Pass: *јетра*; *jetra* | `step4-sr-Cyrl(-390).png`, `step4-hr(-390).png` |
| 5 Draft, export–edit–import | Pass, on the liver (`c320dee9-9bff-403a-9a6b-866a3b9ae471`, Q9368); set back to draft after | `step5-draft(-390).png`, `step5-import-reviewed.txt`, `step5-reviewed(-390).png`, `step5-import-back-to-draft.txt` |
| 6 Patient 403 | Pass | `step6-patient-403.txt` |
| 7 No search text in the api log | Pass (1,759 lines, 55 search requests) | – |

The partner's file: `salusctl terminology export` of every term (in the tools container, into the worktree, then
copied and the worktree copy deleted) at `/home/mesha/salus-terminology/terms-2026-11-08.csv` on the host: 4,943
rows plus the header (32 columns), UTF-8 with a byte order mark, all `draft` (the liver included), no synthetic
term. Not committed.

Benchmark (`go test -run '^$' -bench BenchmarkSearch -benchtime=2000x -count=3 ./internal/terminology`, as `make
bench-terminology`, through `run --rm --no-deps tools`), beside 1.1's committed row (4.4–5.0 / 13.5–15.2 ms):

| Run | 50 concurrent p50 / p95 |
| --- | --- |
| 1.1, committed query (its log above) | 4.4–5.0 / 13.5–15.2 ms |
| 3.1, same query and synthetic data, 2026-11-08 | 4.0–4.3 / **16.7–20.1 ms** |

The 3.1 run misses the SPEC's p95 under 15 ms in all three rounds. The host was not idle: a local language-model
server of another project used about four cores and the swap was full (load average 7 to 11 during the run);
nothing of this stack was running. The benchmark still measures the synthetic 88,330 labels, not the loaded seed
(73,529). 1.1 already found the target met without margin; the levers it named (a stored folded-label column, a
GiST nearest-neighbour order) need a migration and are the owner's call.

Checks (all through `docker compose … run --rm --no-deps -T`, nothing else running against the stack):
- Go (alone): `scripts/lint.sh`: 0 issues (both modules), check-migrations ok; archcheck: exit 0; `scripts/cover.sh`
  (the main module's `go test ./...` with coverage): pass, total 94.2%, every package at 80% or more, 43 packages
  ok; tools module `go test ./...`: ok.
- FHIR: `scripts/fhir-validate.sh` in `fhir-validator`: every golden file `Success: 0 errors`.
- Clients: `npm run -s gen:api`, the Dart generator and `dart format`: no diff. sqlc: no diff.
- Catalogs: `scripts/i18n-check.sh`: pass.
- Web: lint 0 warnings, typecheck ok, 33 files / 575 tests passed, build ok (the existing chunk-size warning).
- Mobile: `flutter analyze` no issues; `flutter test` 421 passed. Debug APK: not built (the flutter image has no
  Android SDK, as in the languages plan; CI's `apk` job builds it).
- E2E: the `make e2e` steps with `--no-deps`, twice in a row: both runs `ok` (64 s, 58 s), 19 journeys each, the
  terminology ones included; the LiveKit log check passed both times.

Decided while the owner was away:
- **The e2e journeys add and delete their own synthetic term** instead of using a seed term: the dev database keeps
  its seed exactly (checked after the runs: 4,943 terms, all draft) and CI, whose stack has no seed, runs the same
  journeys. The cleanup deletes the term as the owner (`DATABASE_URL` of the playwright service, as `startAt`
  writes). No delete command was added to salusctl (out of 3.1's files and not in the SPEC).
- **Step 5 set the liver back to draft** after the walk: the SPEC's curation is the partner's ("reviews every term"),
  so a term marked reviewed by a walk would tell the partner something untrue. The import of a `draft` row is
  allowed (0.1: "Turning a term back to draft keeps its last `reviewed_at`"); the liver's labels are unchanged, its
  `reviewed_at` keeps the walk's time.
- **Step 3c is recorded as failed, not fixed**: the cause is the ranking in `query.sql` (1.1's file) and the seed's
  data; 3.1 owns the tests and the walk only.

COPY.md additions (not yet reviewed): none (3.1 adds no product text).

Dev stack: nothing started, stopped or rebuilt. Written in the dev database and left: the synthetic clinics,
accounts, appointments and the draft questionnaire of the walk and of the two e2e runs (as every e2e run leaves
them); the walk's account language set and cleared again. The liver term's `review_status` went to `reviewed` and
back to `draft` (`updated_at` and `reviewed_at` 2026-11-08 01:19 UTC). The e2e journeys' synthetic terms are
deleted. Redis: the login rate-limit counters cleared before each run, as `make e2e` does.
