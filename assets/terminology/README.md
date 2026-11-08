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
