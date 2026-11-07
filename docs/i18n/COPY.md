# Languages: new text (English source)

Status: draft, 2026-11-07. The owner approves it by starting the run (the run prompt records the approval in PROGRESS.md). These are the English source strings the feature adds; each is
translated into every language of its app by the package that adds it. Existing strings keep the wording of
their feature's COPY.md. [Brackets] are filled in by the app; [language] is always the language's own name
(L1).

## Language names (L1), shown in the picker as they are written in their own language

| Tag | Name |
| --- | --- |
| `bs` | Bosanski |
| `hr` | Hrvatski |
| `sr-Latn` | Srpski (latinica) |
| `sr-Cyrl` | Српски (ћирилица) |
| `en` | English |
| `de` | Deutsch |
| `ar` | العربية |
| `tr` | Türkçe |
| `it` | Italiano |
| `fr` | Français |

Inside a sentence ([language] in B and M strings) the name is the one of L1, unchanged.

## Picker (web and mobile)

- L2: "Language"
- L3 (web): "Use the browser's language"; (mobile): "Use the device's language"
- L4 (after a change, signed in): "Language saved."

## Clinic settings (web, admin)

- S1: "Default language"
- S2 (hint): "Used for e-mails to staff who have not chosen a language, and as the original language of new questionnaires."

## Builder (web)

- B1: "Languages"
- B2: "Original language"
- B3: "Add translation"
- B4 (per translation): "[language]: [n] of [total] texts translated"; when complete: "[language]: complete"; the button "Remove translation", and "Remove the [language] translation?" / "Its texts are deleted from this draft." with the buttons "Remove", "Cancel"
- B5 (publishing with no translation): "Add a translation before publishing?" / "Patients who use Salus in another language will be asked to choose one of this questionnaire's languages." with the buttons "Add translation", "Publish without"
- B6 (`translation_incomplete`): "The [language] translation is not complete. Finish it or remove it to publish."
- B7 (on an item missing a translation): "Missing in [language]"
- B8 (translation editor, the two columns): "Original ([language])", "[language]"
- B9 (submitted answers): "Filled in in [language]", with the switch "Show the original questions"

## Mobile

- M1 (the questionnaire isn't in the app's language): "This questionnaire is not available in [language]."
- M2: "It is available in: [list of languages]. Choose the one you understand best."
- M3: "Continue in [language]"
- M4 (`language_unavailable`): "This language is no longer available for this questionnaire. Choose another one."

## Added by a package, not yet reviewed

(empty)
