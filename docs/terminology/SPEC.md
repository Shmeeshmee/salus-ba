# Medical terminology

Status: owner's decisions taken 2026-11-07 (§2); the rest is the plan's (§3). Runs alongside the languages plan
(`docs/i18n/`), on `feat/terminology` cut from `feat/i18n` after its wave 0.

## 1. Goal

A doctor who can't recall a term finds it in seconds, in Latin, Bosnian, Croatian, Serbian or English, from
anywhere in the web app. The same term list later feeds questionnaire answer options, transcription and
clinical notes, so it is built once, as data, with codes.

## 2. Settled decisions

Given by the owner, 2026-11-07. Do not reopen them.

| Topic | Decision |
| --- | --- |
| What | A terminology database with Latin and the local languages, usable everywhere in the app: a cheatsheet now; questionnaires, transcripts and document filling later |
| Source | Free, downloadable data; the most common terms, not everything |
| Curation | The owner's doctor partner curates and reviews every term |

## 3. Decisions taken in this plan

Reopen only with the owner.

### Data

- **Source: Wikidata** (structured data under CC0, so free to copy, change and use commercially). The fetch
  takes, per category, the items that carry a recognised medical identifier, with their labels and synonyms in
  Latin and the ten app languages:
  - **Anatomy**: items with a *Terminologia Anatomica 98* ID (P1323).
  - **Diagnosis**: items with an ICD-10 code (P494).
  - **Symptom or sign**: items that are an instance or subclass of *symptom* or *medical sign*.
  - **Procedure**: items with an ICD-10-PCS (P1690) or ICD-9-CM procedure code (P1692).
  - **Drug**: items with an ATC code (P267) that are a medication or active ingredient.
- **"Most common".** Ranked by the number of Wikipedia language editions that have an article about the item
  (its sitelinks), the most widely written about first. Limits per category: anatomy 1,500, diagnosis 2,000,
  symptom 500, procedure 500, drug 800: about 5,300 terms, enough to be useful and few enough to review.
- **What is not used.** No SNOMED CT (needs a national licence BiH doesn't have, as far as the plan knows), no
  WHO ICD titles (WHO's text is copyrighted; the code itself, as an identifier from Wikidata, is a fact). 0.1
  checks these points and records them with links in `docs/terminology/SOURCES.md`; if a check comes out
  differently, it stops and reports.
- **Serbian.** Wikidata's `sr` labels are mostly Cyrillic: they fill `sr-Cyrl`, and `sr-Latn` is their
  transliteration (Serbian Cyrillic to Latin is one to one), unless Wikidata has an `sr-el` label, which wins.
- **Gaps stay gaps.** A missing label is empty, never machine translated. Latin is missing for many diagnoses
  and drugs: the partner fills what matters.
- **The seed file** is generated once by a tool (`tools/terminology`), reviewed, and committed as
  `assets/terminology/seed.jsonl`, with the fetch date and query in `SOURCES.md`, so a build never calls
  Wikidata.

### Database

- **Domain `terminology`**, in a new context `reference` (archcheck config and CLAUDE.md list).
- **Tables** (owner `terminology`, `personal_data: false`, no retention rule needed):
  - `term`: `id` (uuid), `category` (enum `term_category`: `anatomy`, `diagnosis`, `symptom`, `procedure`,
    `drug`), `wikidata_id`, `icd10`, `ta98`, `atc`, `pcs` (each nullable), `rank` (from the sitelinks),
    `review_status` (`draft` | `reviewed`), `reviewed_at`, `note` (the curator's, nullable), timestamps.
  - `term_label`: `term_id`, `language` (enum `term_language`: `la` plus the ten app languages), `label`,
    `preferred` (one preferred label per term and language), primary key (`term_id`, `language`, `label`).
- **Search**: Postgres `pg_trgm` and `unaccent` (through an immutable wrapper so it can be indexed): typing
  "cir" finds "čir", "hepar" finds "jetra" through the shared term, misspellings within trigram distance still
  match. Ranking: exact label, then prefix, then trigram similarity, then the term's `rank`. Measured: p95 under
  15 ms for one search at 50 concurrent, recorded in `PROGRESS.md`.
- **Read-only for the app.** Grants to `salus_app`: `SELECT` only. Writes go through `salusctl` as the owner.

### Curation

- **Round trip through a spreadsheet**, no curator role in the app (yet):
  `salusctl terminology export` writes a CSV (one row per term: id, category, codes, rank, the preferred label
  per language, synonyms per language, review status, note); the partner edits it in Google Sheets or Excel
  (fixes labels, fills Latin, sets `reviewed`, writes notes); `salusctl terminology import` applies it:
  upsert by id, labels replaced, `reviewed_at` set when a row turns `reviewed`. Idempotent; an unknown column
  or an id that doesn't exist is refused for that row and reported, the rest applied.
- **No personal data** in the file or the tables: no reviewer name, only the date.

### API and UI

- `GET /terminology/search?q=&languages=&category=&limit=` (JSON, staff only: practitioners and admins; 403 for
  patients): matches with their preferred labels in the requested languages (default `la`, the UI language,
  `en`), synonyms, category, codes and review status. `q` 1 to 100 characters; `limit` up to 25.
- **Never logged**: the search text could hold anything a doctor types, a patient's name included. The access
  log already omits query strings; a test asserts it for this route.
- **Not audited**: no patient data is read.
- **The cheatsheet**: a search palette opened with Ctrl+K (⌘K on macOS) or the search button in the header,
  on every staff screen. Results show the Latin term, the UI-language term and English; "More languages"
  opens the others; codes behind a tooltip; a copy button on each label; a "Not reviewed yet" mark on draft
  terms. Keyboard first: arrows, Enter copies the preferred label in the UI language, Escape closes.

### Out of this feature (Up next, §6)

Picking questionnaire answer options from the terminology (coded answers), patient-facing explanations,
transcription vocabulary, clinical-note autocomplete, a curator screen in the app, clinic-added terms, FHIR
`CodeSystem`/`ValueSet` `$expand`.

## 4. Personal data

None. The search text is not stored or logged.

## 5. Acceptance walk

1. `make terminology-import` loads the seed; counts per category match `PROGRESS.md`.
2. Signed in as a practitioner, Ctrl+K from the calendar, the builder and a consultation screen: the palette
   opens each time.
3. "jetra" finds *hepar* (Latin) and *liver*; "hepar" finds *jetra*; "cir" finds *čir*.
4. In the `sr-Cyrl` UI the result shows the Cyrillic label; in Croatian the Croatian one.
5. A draft term shows "Not reviewed yet"; after an export, a `reviewed` edit and an import, it doesn't.
6. A patient token gets 403 on the search route.
7. The api log after the walk contains none of the search texts used.

## 6. Up next

- Coded questionnaire answers from the terminology (`answerOption.valueCoding`, ICD-10 or Wikidata system).
- Explanations for patients in their language.
- A medical vocabulary for the speech-to-text engine of the transcription feature.
- A curator screen in the app, replacing the spreadsheet round trip.

## 7. Before production

- The partner's review of at least the anatomy and diagnosis terms.
- `SOURCES.md` checked by a lawyer together with the other licences.

## 8. Open points

Points decided while the owner was away, each with its reason; the same list is in `PROGRESS.md` "Decided while
the owner was away".

### The orchestrator

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

### 0.1 Baseline

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

### 1.1 internal/terminology: service and API

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

### 1.2 tools/terminology: the seed

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

### 2.1 salusctl: import, export and curation

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

### 2.2 Web: the cheatsheet

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

### 3.1 End-to-end, walk and the partner's file

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
