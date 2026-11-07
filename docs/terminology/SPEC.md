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
