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

### 1.2 tools/terminology: the seed

### 2.1 salusctl: import, export and curation

### 2.2 Web: the cheatsheet

### 3.1 End-to-end, walk and the partner's file
