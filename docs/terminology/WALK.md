# Medical terminology: acceptance walk

The walk of SPEC.md §5, done by package 3.1 on 2026-11-08 during the unattended run, against the dev stack built
from `feat/terminology` after wave 2 and the languages plan, with the seed loaded at wave 2's integration.
Synthetic accounts only (a clinic, its admin, a practitioner "Bo Syntheticson" and a patient "Ena Syntheticova",
made through the API as the end-to-end journeys make them). Screenshots and texts are in `docs/terminology/walk/`.

## How it was walked

- No browser pane was available, so the web app was walked by a Playwright script in Chromium (the `playwright`
  service, the Vite dev server of `web/` proxying the running API, as the end-to-end web journeys run). A Go driver
  made the clinic, the accounts, a booked appointment (moved to start two minutes later, so its consultation
  screen opens) and a draft questionnaire through the API, and prepared step 5's CSV rows with `salusctl`. Desktop
  is 1280 × 800; the `-390` files are the same screen at 390 px. The two driver files were run from the worktree
  and not committed (a one-off; `test/e2e/terminology_test.go` and `test/e2e/web/terminology.spec.ts` check the
  same behaviour on every `make e2e`).
- The browser's language is English, so the UI is English unless a step picks another language.

## Steps

| Step | What | Result | Evidence |
| --- | --- | --- | --- |
| 1 | `make terminology-import` loads the seed; counts per category match `PROGRESS.md` | Pass. The seed was loaded at wave 2's integration; its command run again (`salusctl terminology import assets/terminology/seed.jsonl` in the tools container): `created 0, updated 0, unchanged 4943, refused 0`; `terms: anatomy 1500, diagnosis 2000, symptom 500, procedure 143, drug 800`, as recorded. SQL: the same counts, 73,529 labels | this file |
| 2 | Signed in as a practitioner, Ctrl+K from the calendar, the builder and a consultation screen opens the palette | Pass, each time; at 390 px the palette is full screen | `walk/step2-calendar.png`, `walk/step2-calendar-390.png`, `walk/step2-builder.png`, `walk/step2-builder-390.png`, `walk/step2-consultation.png`, `walk/step2-consultation-390.png` |
| 3a | "jetra" finds *hepar* (Latin) and *liver* | Partly. "jetra" finds the liver (Q9368) first, with *liver* in English, *jetra* in Bosnian, Croatian and Serbian (Latin), *јетра* in Cyrillic. Its Latin label in the seed is **iecur** (Wikidata's classical Latin), not *hepar*; *hepar* is only an English synonym. TA98 names the liver *hepar*: a data point for the partner's review, not a search fault | `walk/step3-jetra.png`, `walk/step3-jetra-390.png`, `walk/step3-api.txt` |
| 3b | "hepar" finds *jetra* | Pass: the liver first (through its English synonym *hepar*), with *jetra* in the local languages | `walk/step3-hepar.png`, `walk/step3-api.txt` |
| 3c | "cir" finds *čir* | **Fail on the real seed.** The folding works (the end-to-end journey finds "Čirzynt …" by "cirzynt …", and "cir" finds "Cirkadijalna disritmija"), but the seed has no term labelled *čir* on its own in Bosnian or Croatian, and the only label with the word, *Venski čir* (sr-Latn, Q1573613), is ranked below 24 terms with a label that *starts* with "cir" in some language (English *circle of Willis*, Turkish *cırcır* for diarrhoea, Italian, French, German ones): the prefix tier beats word similarity, in every language, also those the doctor does not see. Typing "čir" gives the same list | `walk/step3-cir.png`, `walk/step3-cir-ui.txt`, `walk/step3-cir-with-hacek.png`, `walk/step3-api.txt` |
| 4 | In the `sr-Cyrl` UI the result shows the Cyrillic label; in Croatian the Croatian one | Pass: in Serbian Cyrillic "jetra" shows Латински *iecur*, Српски (ћирилица) *јетра*, Енглески *liver*, the palette in Cyrillic ("Још није провјерено", "Копирај"); in Croatian it shows Hrvatski *jetra*. The account went back to the browser's language afterwards | `walk/step4-sr-Cyrl.png`, `walk/step4-sr-Cyrl-390.png`, `walk/step4-hr.png`, `walk/step4-hr-390.png` |
| 5 | A draft term shows "Not reviewed yet"; after an export, a `reviewed` edit and an import, it doesn't | Pass, on the liver (`c320dee9-9bff-403a-9a6b-866a3b9ae471`, Q9368): "Not reviewed yet" before; the drafts' export (`salusctl terminology export -status draft`), its row with `review_status` set to `reviewed`, imported (`updated 1`); the same search then shows no mark. Then the exported row as it was imported back (`updated 1`): the liver is a draft again, as the partner has not reviewed it. Its labels are unchanged; its `reviewed_at` keeps the walk's time (2026-11-08 01:19:50 UTC), as 0.1 decided for a term turned back to draft, and the CSV has no date column | `walk/step5-draft.png`, `walk/step5-draft-390.png`, `walk/step5-import-reviewed.txt`, `walk/step5-reviewed.png`, `walk/step5-reviewed-390.png`, `walk/step5-import-back-to-draft.txt` |
| 6 | A patient token gets 403 on the search route | Pass: `HTTP 403 {"error":"forbidden"}` | `walk/step6-patient-403.txt` |
| 7 | The api log after the walk contains none of the search texts used | Pass. `docker compose logs api` after the walk and the two end-to-end runs (1,759 lines, 55 of them `GET /terminology/search` access lines with the route, status, duration and request id only) searched, case-insensitive, for every text the walk and the end-to-end runs typed: `jetra`, `hepar`, `cir`, `čir`, `%C4%8Dir`, `liver`, `iecur`, `Zyntheticum`, `correctum`, `Čirzynt`, `cirzynt`, `%C4%8Cirzynt`, and `q=`: no match, except `cir` twice, both inside random request ids (`…CIRPWB…`, `…GCIRQ7…`) | this file |

## Noticed on the way

- **The ranking across all eleven languages** (step 3c): a short text's prefix match in a language the cheatsheet
  does not show (Turkish, Italian, French, German, Arabic) outranks a whole-word match in Bosnian, Croatian or
  Serbian. For the owner: rank the labels in the requested languages first, or search only Latin, the staff
  languages and English, or give word similarity a whole-word tier. All change `query.sql` (1.1's) and are not
  3.1's to make.
- **Latin labels**: the seed's Latin is Wikidata's, often classical (*iecur*), where doctors use the anatomical Latin
  of TA98 (*hepar*). The partner's review covers it.
- **The footer says "Terms from Wikidata, reviewed by Salus."** (COPY.md) while every term is still a draft; the
  owner may want it to say so until the partner's review is in. (Owner's answer, 2026-11-08: T12 is now "Terms from
  Wikidata. Not every term has been reviewed yet.")
- English synonyms from Wikidata are long for some terms (hepatocellular carcinoma: 15 synonyms), which makes a
  result tall, most visibly at 390 px. A curation point.

## Waits on the owner

- A decision on the ranking point above (step 3c).
- Walking the cheatsheet once by hand in a real browser (this walk was scripted), including the clipboard on macOS
  (⌘K).

## Step 3 again, 2026-11-08

After the owner's answers (the ranking by language within each match type; step 3 with a term that has a Bosnian
label), step 3 was run again on the dev database with the seed, through the search query itself (psql, languages
`la`, `bs`, `en`, UI language Bosnian): `walk/step3-rerun.txt`.

| Step | Result |
| --- | --- |
| 3a "jetra" | The liver first (*iecur*, *jetra*, *liver*), then liver failure (*Zatajenje jetre*) and hepatocellular carcinoma. The Latin is still Wikidata's *iecur*: the curator's (TA98 *hepar*) |
| 3b "hepar" | Pass: the liver first, then *Heparin* |
| 3c "kasalj" | Pass: *kašalj* (tussis, cough) first, then *Veliki kašalj* and *Hronični kašalj*: accents aside, a Bosnian label found and ranked first |
| 3c "cir" | No *čir*: the seed has no such label. The curator adds it (the CSV's note says so). With the new ranking the Latin *Circulus arteriosus cerebri* now comes first and Turkish *cırcır* (diarrhoea) is gone from the top five |

Before and after on the same data ("kost"): *Dengue*, matched only through a prefix in a language the cheatsheet does
not show, was second; it is now below the Bosnian prefix matches *Koštana srž*, *Kosti korijena šake* and others.

