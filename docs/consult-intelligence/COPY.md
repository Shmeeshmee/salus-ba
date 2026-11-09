# Consultation intelligence: approved text

The epic's plan is `docs/plans/consult-intelligence.md` (epic SAL-1); this file holds its user-facing strings.

Status: approved by the owner, 2026-11-09 (by starting the run). A change to any string goes back to the owner
first. W is the web app (practitioners); the patient app (mobile) has no string here yet. [count] and [text] are
filled in by the page. The English text below is the source of the web catalog `web/src/i18n/consultation/en.json`
(keys `consultation.live.*`, given after each string); the Bosnian, Croatian and Serbian (Latin and Cyrillic)
catalogs carry Claude's drafts of the same keys, `drafted` in `docs/i18n/TRANSLATIONS.md`, as for every other
string since the languages feature.

## The call page (SAL-9)

### Beside the video
- W1: "Transcript, suggestions and documents" (`panels`): the name of the panel beside the video, read by screen
  readers.
- W2: "Transcript", "Suggestions", "Documents" (`tabTranscript`, `tabSuggestions`, `tabDocuments`): its three tabs.
  The Suggestions tab shows the number of suggestions not dismissed beside its name, as a figure.

### Captions (on the patient's video)
- W3: "Captions" (`captions`): the name of the caption band, read by screen readers.
- W4: "Original:" (`original`), followed by [text], what the patient said in their language: the tooltip of a
  caption, shown on hover and on keyboard focus, and read as the caption's description.

### Transcript
- W5: "Practitioner", "Patient" (`speaker.practitioner`, `speaker.patient`): who said a line.
- W6: "The transcript appears here as the conversation goes on." (`transcriptEmpty`)
- W7: "In progress" (`inProgress`): read by screen readers after a line still being recognised (shown greyed).
- W8: "Show the latest" (`latest`): offered when new lines arrive while the practitioner has scrolled up.

### Suggestions
- W9 (`suggestionsNote`, tagged `legal`): "Suggestions are prompts drawn from the transcript, not a diagnosis.
  Every decision is yours."
- W10, the label above each suggestion, one per kind (`kind.diagnosis`, `kind.question`, `kind.red_flag`,
  `kind.term`): "Suggestion: a diagnosis to consider", "Suggestion: a question to ask", "Suggestion: a red flag to
  check", "Suggestion: a term to explain".
- W11: "Pin", "Unpin", "Dismiss" (`pin`, `unpin`, `dismiss`)
- W12 (`sources`): "Show the source line" / "Show the [count] source lines": opens the transcript at the lines a
  suggestion or a drafted field came from, highlighted.
- W13: "No suggestions yet." (`noSuggestions`)
- W14: "The suggestion could not be updated. Try again." (`statusFailed`)

### Documents
- W15: "Template" (`template`), "Choose a template" (`chooseTemplate`): the template picker and its empty choice.
- W16: "Loading the templates…" (`loadingTemplates`), "The templates could not be loaded. Try again later."
  (`templatesFailed`), "This clinic has no document templates yet." (`noTemplates`)
- W17: "Generate", "Regenerate" (`generate`, `regenerate`)
- W18: "Generating the draft from the transcript…" (`generating`), "The draft could not be generated. Try again."
  (`generateFailed`)
- W19: "Drafted from the transcript. Check every field before you sign." (`draftNote`)
- W20: "Not covered in the conversation yet" (`missing`): the flag on a field the draft left empty.
- W21: "Sign" (`sign`); the confirmation: "Sign this document?" (`signQuestion`), "Once signed, it is final and can
  no longer be edited." (`signFinal`), and when fields are empty "[count] field is empty and stays empty." /
  "[count] fields are empty and stay empty." (`signEmpty`); "Keep editing" (`keepEditing`) cancels.
- W22: the confirmation of a regenerate that would replace the practitioner's edits: "Regenerate the draft?"
  (`regenerateQuestion`), "Your changes to its fields are replaced by a new draft from the transcript."
  (`regenerateReplaces`), with "Regenerate" and "Keep editing".
- W23: "Signed. This document is final." (`signed`), "The document could not be signed. Try again."
  (`signFailed`)

### Pending the owner's approval (2026-11-09, the wiring follow-up to #34)
- W24 (`requiredEmpty`): "[count] required field is empty. Fill it in before you sign." / "[count] required fields
  are empty. Fill them in before you sign.": shown instead of the sign confirmation while a field the template
  marks required is empty (SAL-8 refuses such a sign, `422 required_missing`), and when the server refuses a sign
  for that reason; the empty required fields are marked (`aria-invalid`, a red border). New, because no approved
  string says why a sign cannot go ahead; drafted in bs, hr, sr-Latn and sr-Cyrl.

### Reused on the call page, pending the owner's review (2026-11-09, the wiring follow-up to #34)
No new string. When the live stream ends for good, the panel beside the video shows, instead of its tabs, an
existing approved string of the consultation screens: `consultation.peerWithdrewConsent` (tagged `legal`) when the
patient withdrew their consent, and `consultation.notAvailable` when the practitioner may no longer follow the
consultation. A line whose translation failed (SAL-5) shows its original, as one still awaiting its translation:
marking it ("translation unavailable") would need a new string, left for the owner to decide.

Open for the owner: the contract (SAL-7) says whatever shows a suggestion labels it as AI assistance. The page
shows only suggestions whose `origin` is `ai`, each under its W10 label and below W9, but no approved string names
AI. A wording that does (for example W9 as "AI suggestions are prompts drawn from the transcript, not a diagnosis.
Every decision is yours.") is the owner's to approve; nothing was changed.

Template titles and field labels are the clinic's own text, from its templates (`document_template`), not copy.
Every medical line in the tests and fakes (`web/src/test/fakeConsult.ts`) is a labelled placeholder
("Placeholder reason", "Placeholder suggestion text"), never real clinical wording.
