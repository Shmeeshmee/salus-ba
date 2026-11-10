# Medical terminology: text (English source)

Status: approved by the owner, 2026-11-07 (by starting the run); T12 changed and T13 to T15 approved 2026-11-08. Translated into the five staff languages by 2.2.

- T1 (header button, tooltip): "Search medical terms (Ctrl+K)"; on macOS "(⌘K)"
- T2 (palette placeholder): "Search in Latin, Bosnian, Croatian, Serbian or English"
- T3 (column headings): "Latin", "[UI language name]", "English"
- T4: "More languages", "Fewer languages"
- T5 (categories): "Anatomy", "Diagnosis", "Symptom or sign", "Procedure", "Drug"
- T6: "Copy", "Copied"
- T7 (tooltip on codes): "ICD-10 [code]", "TA98 [code]", "ATC [code]", "ICD-10-PCS [code]"
- T8 (draft term): "Not reviewed yet"
- T9 (no result): "No terms found. Try fewer letters or another language."
- T10 (error): "The search is not available. Try again."
- T11 (no label in a language): "No term in [language] yet"
- T12 (footer of the palette): "Terms from Wikidata. Not every term has been reviewed yet." (owner's answer,
  2026-11-08; was "Terms from Wikidata, reviewed by Salus.")

## Added by a package, approved

Written by a package during the unattended run of 2026-11-07; approved by the owner as written, 2026-11-08.

### 2.2 Web: the cheatsheet

- T13 `terminology.title` (the palette's and the result list's accessible name): "Medical terms".
- T14 `terminology.close` (the palette's close button, needed where there is no Escape key, e.g. on a phone):
  "Close".
- T15 `terminology.language.bs`, `.hr`, `.srLatn`, `.srCyrl` (cell captions and T11's `{language}`, beside T3's
  "Latin" and "English"): "Bosnian", "Croatian", "Serbian (Latin)", "Serbian (Cyrillic)".

## Added by SAL-13, waiting for the owner's approval

- T16 `terminology.explainNote` (web, under a term's explanation in its tooltip; `legal`): "Informational, not a
  diagnosis". Beside it, until a reviewer approved the explanation, T8 "Not reviewed yet" (reused).
- Mobile (`mobile/lib/l10n`, all ten languages): `termExplanationNote` ("Informational, not a diagnosis", `[legal]`),
  `termNotReviewed` ("Not reviewed yet"), `termClose` ("Close", the explanation dialog's button).
- The 2,160 explanations themselves (`assets/terminology/explanations.draft.jsonl`) are the doctor partner's to
  review, in the review panel.
