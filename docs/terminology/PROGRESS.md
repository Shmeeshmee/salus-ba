# Medical terminology progress

| Package | Status | Commit |
| --- | --- | --- |
| 0.1 Baseline | done (integrated 2026-11-07) | 9b906c3 |
| 1.1 internal/terminology: service and API | not started | |
| 1.2 tools/terminology: the seed | not started | |
| 2.1 salusctl: import, export and curation | not started | |
| 2.2 Web: the cheatsheet | not started | |
| 3.1 End-to-end, walk and the partner's file | not started | |

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

### 2.1 salusctl: import, export and curation

### 2.2 Web: the cheatsheet

### 3.1 End-to-end, walk and the partner's file
