# Terminology seed

`seed.jsonl` is the medical term list the app starts from (docs/terminology/SPEC.md §3 "Data"): the most common
anatomy, diagnosis, symptom, procedure and drug items on Wikidata (CC0), with their codes and their labels in
Latin and the ten app languages. It is data written by a tool: never edit it by hand. The curator's changes go
into the database through `salusctl terminology import` (a CSV), not into this file.

## Where it comes from

`tools/terminology` (`make terminology-fetch`, network) asks Wikidata's SPARQL endpoint, one query at a time:

1. Per category, the items that carry its identifier, ranked by their sitelinks (the number of Wikipedia
   language editions with an article about the item), the most written about first:
   - symptom: a direct instance (P31) or subclass (P279) of *symptom* (Q169872) or *medical sign* (Q1441305);
   - anatomy: a Terminologia Anatomica 98 ID (P1323);
   - procedure: an ICD-10-PCS code (P1690), or an ICD-9-CM code (P1692) in the procedure format (two digits,
     an optional decimal part);
   - drug: an ATC code (P267) at the substance level (seven characters, such as N02BE01);
   - diagnosis: an ICD-10 code (P494).

   The categories claim items in that order: an item that fits several is kept in the first (a symptom with an
   ICD-10 R code stays a symptom; diagnosis takes what is left). Limits: anatomy 1,500, diagnosis 2,000,
   symptom 500, procedure 500, drug 800.
2. For the kept items, in batches of 100: labels and aliases in `la`, `bs`, `hr`, `sr`, `sr-ec`, `sr-el`,
   `en`, `de`, `ar`, `tr`, `it`, `fr`, and the codes (ICD-10, TA98, ATC, ICD-10-PCS, ICD-9-CM procedure codes).

The exact queries are in `tools/terminology/fetch.go` (`rankQuery`, `detailsQuery`) and in
docs/terminology/SOURCES.md.

## Format

JSON Lines, one term per line, sorted by category (`anatomy`, `diagnosis`, `symptom`, `procedure`, `drug`), then
rank (higher first), then Wikidata id (numerically):

```json
{"labels":{"bs":["Oko"],"en":["eye","eyeball"],"la":["oculus"],"sr-Cyrl":["око"],"sr-Latn":["oko"]},"wikidata":"Q7364","category":"anatomy","ta98":"A01.1.00.007, A15.2.00.001","rank":261}
```

- `rank` is the item's sitelinks count.
- Codes (`icd10`, `ta98`, `atc`, `pcs`) are absent when the item has none; several of one kind are sorted and
  joined with `", "`. `pcs` holds ICD-10-PCS and ICD-9-CM procedure codes. Codes are copied as Wikidata has
  them, odd ones included: the curator fixes them.
- `labels` has one array per language that has a label, the preferred label (Wikidata's label) first, then the
  synonyms (its aliases) in byte order. Labels are trimmed, runs of white space collapsed, and a label without
  any letter (an emoji, a bare number), one over 300 characters, or one that repeats another of the same
  language in a different case is left out. A missing language stays missing: nothing is machine translated.
- Serbian: Cyrillic `sr` and `sr-ec` labels fill `sr-Cyrl`. `sr-Latn` is Wikidata's `sr-el` when it has a label,
  else the `sr` labels transliterated (a Latin `sr` label is taken as it is).
- An item with no label in any of these languages is left out.

## This seed

Fetched on 2026-11-07 at 23:14 UTC from https://query.wikidata.org/sparql (76 seconds, 5 ranking queries and 50
batches). 4,943 terms, 73,529 labels:

| Category | Terms | Limit |
| --- | ---: | ---: |
| anatomy | 1,500 | 1,500 |
| diagnosis | 2,000 | 2,000 |
| symptom | 500 | 500 |
| procedure | 143 | 500 |
| drug | 800 | 800 |
| total | 4,943 | 5,300 |

Procedure has fewer than its limit: only 143 items on Wikidata carry an ICD-10-PCS code or an ICD-9-CM
procedure code. Every kept item had at least one label, so none was left out for that.

Terms with at least one label per language: en 4,943, fr 4,782, ar 4,729, de 4,123, it 3,995, sr-Latn 2,523,
tr 2,507, sr-Cyrl 1,963, la 1,601, hr 1,226, bs 1,169. The gaps (Latin, Bosnian and Croatian above all) are for the
curator to fill.

Every term is a `draft` until the curator reviews it (SPEC.md §3 "Curation").

# Plain-language explanations: DRAFTS for the doctor partner

`explanations.draft.jsonl` holds a short plain-language explanation of 216 common terms in each of the ten app
languages (SAL-13, docs/terminology/SPEC.md §3 "Explanations"): 2,160 texts. **Every one is a draft, written by
Claude (an AI model) on 2026-11-10 and not yet checked by a doctor.** `make terminology-import` (or `salusctl
terminology import-explanations`) loads them as `unreviewed`, with no reviewer, so each lands in the review panel's
queue (`/term-review`), where the doctor partner approves it, corrects it or marks it as needing changes. Until it is
approved, the app shows it with "Not reviewed yet". The import only adds what a term does not have yet: it never
overwrites an explanation that is there, so a reviewer's work is safe from a second import.

```json
{"wikidata":"Q9368","explanations":{"bs":"Veliki organ u gornjem desnom dijelu trbuha. …","en":"A large organ in the upper right part of the belly. …"}}
```

- One term per line, by its Wikidata id (the seed's `wikidata`), and its text per app language (never Latin):
  trimmed, 1 to 500 characters (the texts here are at most 178).
- Which terms: the 300 most common of the seed by `rank`, less those a patient needs no explanation for or that
  are no clinical concept met in a consultation: everyday body parts (eye, hand, knee and the like), chemical
  elements and laboratory chemicals the drug category catches (helium, xenon, phenol), a feather, a virus
  subfamily, social events (traffic collision, domestic violence), feelings (sadness, thirst) and a few sensitive
  topics a tooltip should not explain (suicide, abortion, recreational drugs). One more, the tonsil (the 311th):
  the ticket's example, tonsillitis, is no term of the seed, and the tonsil is the nearest that is.
- How they were written: plain language, one or two short sentences, informational only: no advice, no
  reassurance, no dose and nothing that reads as a diagnosis. The English text came first and was translated into
  the other nine languages; Bosnian and Croatian follow their own standards, Serbian is ijekavian in both scripts and
  `sr-Cyrl` is the Cyrillic of `sr-Latn`.
- Points the translators were unsure of, for the partner's review: Bosnian "ugljik-dioksid" (as the seed's label;
  "ugljen-dioksid" is common too), "tečnost" rather than "tekućina", "mokraćni mjehur"; Croatian "ugljikov dioksid"
  (the seed's hr label "ugljikov oksid" reads as carbon monoxide); Serbian "mokraćna bešika" and "plućni mjehurići"
  (for the alveoli); Turkish "Karbonat" for baking soda (the seed's "Kabartma tozu" is baking powder) and the seed's
  wrong second label "Hepatit c" for jaundice; Arabic "العين الوردية" (pink eye) and "أعلى الفخذ" for the groin.

# Translation flags: wrong labels found, for the reviewer

`translation-flags.jsonl` lists labels of the seed found to be wrong, so the reviewer fixes them first. `make
terminology-import` (or `salusctl terminology import-flags`) marks each flagged term's translation in that language
as `needs_changes`, with the flag's note and no reviewer, so it comes up under "Needs changes" in the review panel
(`/term-review`); the term itself needs changes too when the language is Latin or a staff language. Nothing here
changes a label: the reviewer does, in the panel. A flag applies only to a translation never reviewed, so a second
import, or one after the reviewer's fix, changes nothing; a flag whose label the seed does not have is refused.

```json
{"wikidata":"Q133244","term":"jaundice","language":"tr","label":"Hepatit c","note":"\"Hepatit c\" names hepatitis C, …"}
```

- One flag per line: the term's Wikidata id, its English name (`term`, for the reader, not read), the language, the
  wrong label as the seed has it, and the note (1 to 500 characters), in English.
- The flags of 2026-11-11, found while the explanations were drafted (SAL-13): Turkish "Hepatit c" under jaundice,
  Turkish "Kabartma tozu" (baking powder) for sodium bicarbonate, Croatian "ugljikov oksid" (reads as carbon
  monoxide) for carbon dioxide, and Serbian "grudna žlezda" / "грудна жлезда" for the thymus, in both scripts.
