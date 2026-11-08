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

Written by a package during the unattended run of 2026-11-07; each waits for the owner's review.

### 1.5 Web area A: shell, sign-in, calendar, admin

- A1 (web, Availability, a malformed working-hours time): "Enter each time as hours and minutes, from 00:00 to
  23:59." (`calendar.invalidTime`)

### 1.7 Mobile: every screen in ten languages

- 1.7-1 `timeRange`: "[start] to [end] ([zone])" (appointment and booking times; was built from pieces).
- 1.7-2 `dateAndTime`: "[date], [time]" (inside questionnaire M14 "Submitted [date and time]").
- 1.7-3 `specialty<SNOMED code>`: the twelve specialty names of `internal/practitionerrole` ("General practice",
  "Internal medicine", "Cardiology", "Dermatology", "Gynaecology", "Paediatrics", "Psychiatry", "Neurology", "ENT",
  "Ophthalmology", "Orthopaedics", "Endocrinology"), on the practitioner list and the booking screens.
- 1.7-4 `tooManyAttempts`: one plural for "Too many attempts. Try again in [n] minute(s)." (was built in code).
- The other English strings are those of the screens before this package, word for word.

### The integration steps

- `specialtyAtClinic` (mobile, the free times screen): "[specialty], [clinic]", one message so each language
  writes its own separator (Arabic "،"); replaces a line joined in code (found by 3.1's walk).
