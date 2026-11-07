# Languages (i18n)

Status: owner's decisions taken 2026-11-07 (§2); the rest is the plan's (§3). Start after the attachments
feature is merged into `main`: every user-facing string on `main` at that point is in scope.

## 1. Goal

Salus speaks the languages of the people who use it. Staff (web) use Bosnian, Croatian, Serbian (Latin and
Cyrillic) or English. Patients (mobile) use those and the languages of the tourists who visit Bosnia and
Herzegovina. Every screen, e-mail and error message is translated; questionnaires can be translated by the
clinic, and a patient whose language a questionnaire doesn't have chooses one it does have.

## 2. Settled decisions

Given by the owner, 2026-11-07. Do not reopen them.

| Topic | Decision |
| --- | --- |
| Staff languages (web) | Bosnian, Croatian, Serbian in Latin script, Serbian in Cyrillic script, English. All five are musts |
| Patient languages (mobile) | The five staff languages, plus German, Arabic and Turkish (musts), plus Italian and French (added by the plan; "anything beyond is great to have") |
| Choosing the language | The browser's language (web) or the device's language (mobile) by default. A language the user picks is saved on their account and wins from then on, on every device |
| Questionnaire content | Written by the clinic in one language. The clinic may translate it into any patient language. A patient whose language the questionnaire doesn't have is told which languages it has and chooses one |
| Language list | A database enum. The builder offers its values when the doctor adds a translation |

## 3. Decisions taken in this plan

Reopen only with the owner.

### Languages

- **Tags.** BCP 47, exactly these values of the Postgres enum `language`: `bs`, `hr`, `sr-Latn`, `sr-Cyrl`,
  `en`, `de`, `ar`, `tr`, `it`, `fr`. Staff languages are the first five; the service refuses a staff account's
  choice outside them (422 `language_unavailable`). Adding a language later is a forward migration
  (`ALTER TYPE language ADD VALUE`) plus its catalogs.
- **Serbian is ijekavian** in both scripts, as Serbian is written in Bosnia and Herzegovina. `sr-Latn` and
  `sr-Cyrl` carry the same text in two scripts; the Cyrillic catalog is written as Cyrillic, not machine
  transliterated at runtime.
- **Matching the browser or device** (first match wins, in the order the browser or device lists its
  languages): `bs*` → `bs`; `hr*` → `hr`; `sr-Latn*`, `sr-ME`, `sh`, `cnr` → `sr-Latn`; `sr`, `sr-Cyrl*`,
  `sr-RS`, `sr-BA` → `sr-Cyrl` (CLDR's default script for `sr` is Cyrillic); `en*`, `de*`, `ar*`, `tr*`, `it*`,
  `fr*` → that language where the app offers it. No match → `bs` on the web, `en` on mobile (a tourist's
  phone that matches nothing is more likely to read English than Bosnian).
- **Picking and saving.** The picker shows each language by its own name (COPY.md L1). Signed in: the choice
  is saved at once (`PATCH /auth/me`, `language`). Before sign-in (sign-in, registration, invite screens): the
  choice is kept on the device and sent with the next registration or sign-in, which saves it on the account.
  "Use the browser's language" / "Use the device's language" clears the saved choice (`language: null`).
- **The account's language fields.** `account.language` (nullable: the user's explicit choice) and
  `account.detected_language` (nullable: the language the client matched at the last registration or
  sign-in, sent as `detectedLanguage`; updated on every sign-in). The UI uses `language`, else what it
  matches itself. E-mails use `language`, else `detected_language`, else the clinic's default language (staff)
  or `bs` (patients).
- **The clinic's default language.** `organization.default_language`, one of the staff languages, default
  `bs`, set by an admin in clinic settings. Used for e-mails to staff with neither field set (an invite before
  the account exists) and as the default original language of a new questionnaire.
- **Formatting.** Dates, times and numbers through `Intl` (web) and `intl` (Flutter) with the UI language.
  Times are 24-hour in every language. Digits are Latin in every language, Arabic included (`ar-u-nu-latn`),
  so times, phone numbers and codes look the same everywhere.
- **Plurals and placeholders.** ICU MessageFormat in every catalog: Bosnian, Croatian and Serbian have three
  plural forms, Arabic six. No string is built by joining pieces.
- **Right to left.** Arabic is right to left on mobile. Flutter's `Directionality` does the layout;
  directional icons (back, forward, chevrons) are mirrored, non-directional ones (camera, microphone,
  calendar) are not; phone numbers, codes and times stay left to right inside right-to-left text.
- **Fonts.** The current fonts are checked for Cyrillic and Turkish letters; mobile bundles Noto Sans Arabic
  (SIL Open Font License) for Arabic. No font is loaded from the network (CLAUDE.md).
- **Server text.** The API keeps returning codes, never sentences (as now). E-mails are the only text the
  server writes; they are templates per language in `internal/notification`. Specialty names are shown from a
  client catalog keyed by the SNOMED code, never from the code's English display.

### Catalogs

- **Web:** `web/src/i18n/<area>/<language>.json`, one file per screen area and language (areas in
  IMPLEMENTATION_PLAN.md), FormatJS (`react-intl`) with ICU messages. English is the source.
- **Mobile:** `mobile/lib/l10n/app_<language>.arb` (`gen-l10n`), English the template.
- **E-mail:** `internal/notification/templates/<language>/<kind>.txt` (and `.html` if the e-mails have one).
- **Every key in every language.** A package that adds or changes a string writes it in every language of
  that app in the same commit. `scripts/i18n-check.sh` fails on a missing or extra key, a placeholder or
  plural mismatch, or invalid ICU syntax; it runs in `make web-lint`, `make mobile-analyze` and `make lint`
  (e-mails), and therefore in CI.
- **Who translates.** Claude Code drafts every language. `docs/i18n/TRANSLATIONS.md` records each language's
  status: `drafted` (by Claude), `reviewed` (by a native speaker: who and when). Strings with legal weight
  (consent, privacy notice link text, permission prompts, cancellation and revocation effects) are tagged
  `legal` in the catalogs (`@` metadata in ARB, a `description` in the web JSON) and need a lawyer's review
  per language before production (§8).

### Questionnaires

- **Original language.** `questionnaire.language` (enum), set in the builder, default the clinic's default
  language. Existing questionnaires get their clinic's default language.
- **Translations.** `questionnaire_translation` (`questionnaire_id`, `language`, `texts` `jsonb`): the
  translated title, description, and for every item its text, help text and option texts, keyed by `linkId`
  (options by position). A translation belongs to one version and is edited only while that version is a
  draft; publishing freezes it with the version. A new version copies the previous version's translations;
  texts of items that changed or are new are missing until translated.
- **Complete or nothing.** Publishing refuses a version with an incomplete translation (422
  `translation_incomplete`, with the language and the `linkId`s); the builder shows what is missing and lets
  the doctor finish or remove that translation. A patient never sees a half-translated questionnaire.
- **The prompt.** Publishing a version with no translation asks first whether to add one (COPY.md B5); the
  doctor may publish without.
- **The patient's language.** The questionnaire opens in the patient's UI language if it has it. If not, the
  patient sees which languages it has and chooses one (COPY.md M2); English is preselected when it is one of
  them, else the original. The chosen language is stored on the response (`questionnaire_response.language`,
  set when the patient starts) and cannot change after the first saved answer.
- **The practitioner** reads the questions in the language the patient answered in, with the original
  available, and sees "Filled in in [language]" (COPY.md B9). Free-text answers are shown as written.
- **FHIR.** `Questionnaire.language` is the original language; translations use the R4 core extension
  `http://hl7.org/fhir/StructureDefinition/translation` (`lang`, `content`) on each translated string's
  primitive element (`_title`, `item._text`, `answerOption._valueString`, …).
  `QuestionnaireResponse.language` is the language answered in.

### Out of this feature

Machine translation of questionnaires (an AI processor: needs a DPA and its own SPEC), translating clinic
names or practitioners' names, a language per clinic for the patient app, server-rendered pages, localised
FHIR `display` values, languages beyond the ten above.

## 4. Lawful basis

Language preferences are processed to provide the service the user signed up for (Art. 6(1)(b)). They are not
special-category data. The language a questionnaire was answered in is part of the response, under the
questionnaire's basis (questionnaire SPEC §4).

## 5. Personal data

| Where | Field | Why it is needed |
| --- | --- | --- |
| `account` | `language` | The user's chosen language, so every device and every e-mail uses it |
| `account` | `detected_language` | The language for e-mails when the user hasn't chosen one |
| `questionnaire_response` | `language` | Which translation the patient answered, so the practitioner reads the questions the patient read |

`organization.default_language` and `questionnaire_translation` hold no personal data. Retention: the
account and response rules already cover their rows. No language appears in a log line next to an
account or patient id.

## 6. Acceptance walk

1. A browser set to `sr-Cyrl` opens the web app signed out: the sign-in screen is in Serbian Cyrillic.
2. The practitioner picks Hrvatski in the picker; signs out and in on another browser set to English: the app
   is in Croatian.
3. "Use the browser's language" clears it: the next sign-in follows the browser again.
4. An admin sets the clinic's default language to English and invites a practitioner: the invite e-mail is in
   English.
5. A phone set to German registers: the app and the booking confirmation e-mail are in German.
6. The same phone set to Arabic: every screen is right to left, back arrows point right, times and phone
   numbers read left to right, nothing is cut off.
7. The doctor builds a questionnaire in Bosnian, starts an English translation, tries to publish it half
   done (refused, missing items shown), finishes it and publishes.
8. The German patient opens it: sees that it is in Bosnian and English, English preselected, answers in
   English. The practitioner sees "Filled in in English" and the English questions.
9. An Italian patient with a Bosnian-only questionnaire is told it is only in Bosnian and fills it in.
10. Every catalog passes `scripts/i18n-check.sh`; `TRANSLATIONS.md` lists every language as `drafted`.

## 7. Up next

- Native-speaker review of every language, starting with bs, hr, sr-Latn and sr-Cyrl (the owner and the
  partner), then de, ar, tr, it, fr.
- Machine-translation suggestions in the builder (needs an AI processor, a DPA and its own SPEC).

## 8. Before production

- Every language reviewed by a native speaker (`TRANSLATIONS.md` all `reviewed`).
- Every string tagged `legal` reviewed by a lawyer in each language.
- The privacy notice published in every patient language (`PRIVACY_NOTICE_URL` per language).

## 9. Open points

Points decided while the owner was away, each with its reason; the same list is in `PROGRESS.md` "Decided while
the owner was away".

(none yet)
