# Medical terminology implementation plan

Read `SPEC.md` first. Each package is one commit `terminology(<n>): <summary>`, made with `/run-package
docs/terminology <n>`. This plan runs on `feat/terminology`, cut from `feat/i18n` once the languages plan's
wave 0 is integrated, so it has the language catalogs and its migration number comes after the languages
migration. Its PR targets `feat/i18n` until that is merged, then `main`.

## Rules for every package

The languages plan's "Rules for every package", "Shared stack" and "Check sets"
(`docs/i18n/IMPLEMENTATION_PLAN.md`) apply here word for word, with `terminology` for `i18n`, plus:

- After 0.1 nobody but 0.1 changes `api/openapi.yaml`, a migration, `ownership.yaml`, the archcheck config,
  `go.mod`, `Makefile` or CLAUDE.md.
- The seed is data: a package never edits `assets/terminology/seed.jsonl` by hand; only the tool writes it.
- Never log a search text; a test asserts it.
- No new Go dependency (the SPARQL fetch is plain `net/http` with an injected client and recorded fixtures).

## Waves

| Wave | Packages | Why their files don't overlap |
| --- | --- | --- |
| 0 | 0.1 | Alone: migration, ownership, archcheck config, `api/openapi.yaml` and the generated clients, the `terminology` model and sqlc queries, `Makefile`, CLAUDE.md, `SOURCES.md`, the `PROGRESS.md` headings |
| 1 | 1.1, 1.2 | `internal/terminology` (service, handler, wiring in `internal/api/app.go` and the router); `tools/terminology` and `assets/terminology/` |
| 2 | 2.1, 2.2 | `cmd/salusctl` terminology commands; the web cheatsheet (`web/src/terminology/` and its `web/src/i18n/terminology/` catalogs, plus one mount line in the shell, see 2.2) |
| 3 | 3.1 | Alone: end-to-end tests, the walk, the export for the partner |

These waves fit between the languages plan's: terminology wave 1 runs with languages wave 1 (five sessions at
most), terminology wave 2 after languages wave 1 is integrated and merged into `feat/terminology` (2.2 needs
the web shell's catalogs and touches the shell once).

## Packages

### 0.1 Baseline

- **Files**: `migrations/<languages migration + 1>_terminology.sql`, `docs/architecture/archcheck/ownership.yaml`,
  `docs/architecture/archcheck/config.yaml` (context `reference: [terminology]`), `sqlc.yaml`,
  `internal/terminology/{model.go,errors.go,repository.go,query.sql,terminologydb/,repository_postgres.go}`
  with repository tests, `api/openapi.yaml` and the generated clients, `Makefile` (`terminology-fetch`,
  `terminology-import`, `terminology-export`, `bench-terminology`), CLAUDE.md (`reference` in the domain list), `docs/terminology/SOURCES.md`,
  `docs/terminology/PROGRESS.md`.
- **Do**: SPEC §3 "Database" tables, enums, extensions, the immutable `unaccent` wrapper, trigram indexes on
  the unaccented lower-cased label, `SELECT` grants only for `salus_app`; the search query with its ranking and a
  repository test per ranking rule (exact, prefix, trigram, accent-insensitive, cross-language through the
  term); the OpenAPI route and error codes; `SOURCES.md` with the licence checks of SPEC §3 "What is not used",
  each with a link (stop and report if Wikidata's licence or the ICD point is not as the SPEC says).
- **Checks**: Go (alone), Clients.

### 1.1 internal/terminology: service and API

- **Files**: `internal/terminology/{service.go,handler_http.go}` and tests, `internal/api/app.go`, the router.
- **Do**: staff-only search (403 for patients), parameter limits, default languages from the caller's UI
  language, the response shape of SPEC §3; a test that the access log and the error paths hold no search text;
  a benchmark (`make bench-terminology`, the target 0.1 adds) with the p95 recorded.
- **Depends on**: 0.1. **Checks**: Go (in a wave).

### 1.2 tools/terminology: the seed

- **Files**: `tools/terminology/` (its own package in the tools module), `assets/terminology/seed.jsonl`,
  `assets/terminology/README.md`.
- **Do**: the SPARQL queries per category (SPEC §3 "Data"), paginated, polite (a descriptive `User-Agent`
  naming Salus without any e-mail, one request at a time, backoff on 429); ranking by sitelinks; labels and
  synonyms in `la` and the ten languages; the Serbian rule; normalisation (trim, collapse spaces, no
  duplicates); deterministic output sorted by category and rank. Tests against recorded responses in
  `testdata/`. Then run it once (`make terminology-fetch`, network) and commit the seed; record the counts per category
  and the fetch date in `assets/terminology/README.md` and `PROGRESS.md`.
  If Wikidata can't be reached, commit the tool, record that, and let 2.1 and 3.1 use a small fixture seed.
- **Depends on**: 0.1. **Checks**: Go (in a wave) for the tools module.

### 2.1 salusctl: import, export and curation

- **Files**: `cmd/salusctl/` terminology commands and tests, `internal/terminology/` import and export methods
  (owner-role writes) and tests.
- **Do**: `salusctl terminology import <file>` (seed JSONL or curated CSV) and `export [--status draft]`
  (CSV, SPEC §3 "Curation"), idempotent upsert, per-row errors reported, `reviewed_at` set on the transition;
  load the seed into the dev database.
- **Depends on**: 1.1, 1.2. **Checks**: Go (in a wave).

### 2.2 Web: the cheatsheet

- **Files**: `web/src/terminology/`, `web/src/i18n/terminology/<language>.json`, and one line in the shell
  component to mount the palette and the header button (the only shell edit; record it).
- **Do**: SPEC §3 "The cheatsheet", COPY.md strings in the five staff languages, keyboard behaviour, copy to
  clipboard, results debounced (150 ms) and cancelled when the text changes, an empty and an error state;
  tests for the keyboard paths and the language columns.
- **Depends on**: 0.1 (client), languages wave 1 merged into `feat/terminology`. **Checks**: Web, Catalogs.

### 3.1 End-to-end, walk and the partner's file

- **Files**: `test/e2e/` (terminology), `docs/terminology/WALK.md`, `docs/terminology/walk/`.
- **Do**: e2e for search as a practitioner, 403 as a patient, and the import–export round trip; the acceptance
  walk (SPEC §5) with screenshots; `salusctl terminology export` written to `~/salus-terminology/terms-<date>.csv`
  on the host for the owner to share with the partner (not committed).
- **Depends on**: every earlier package. **Checks**: every set, E2E.

## Definition of done

Every package committed on `feat/terminology` and integrated; every check set passes locally and in CI; the
seed loaded and its counts recorded; the walk recorded; `PROGRESS.md` complete; the CSV for the partner
written.
