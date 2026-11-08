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

### The orchestrator

- **The stack is started with `docker compose up -d --build --wait`, not `make up`.** `make up` passes
  `--remove-orphans`, which can remove the owner's `salus-demo-web` and `salus-demo-emu` containers (they carry
  the compose project's labels). It is stopped with `docker compose down`, never `make down`.
- **Host ports moved for this run.** Another local project (NOMOS) holds 8025, 8080 and 8090, so the git-ignored
  `deployments/.env` of every worktree sets `SALUS_MAILPIT_PORT=8026`, `SALUS_API_PORT=8180` and
  `SALUS_LANDING_PORT=8190`. Nothing committed changes; the checks reach the services on the compose network.
- **Integration runs in its own worktree** (`../salus-int`), which starts and stops the shared stack, because the
  main checkout holds the owner's uncommitted CLAUDE.md edit and cannot switch branches.
- **A package records its decisions and new copy in its own PROGRESS.md section**; the integration step copies
  them here, into the SPEC's open points and into COPY.md, so packages running at the same time never edit the
  same lines.
- **Wave 0 integration: two mobile tests follow the new contract.** The regenerated Dart client writes the
  optional `language` and `detectedLanguage` of sign-in and registration as `null`, so the exact request bodies
  in `register_test.dart` and `sign_in_test.dart` now include them as `null` (still asserted exactly). 1.7 makes
  the app send the matched language; 1.1 makes the server accept both fields, `null` included (the auth handler
  refuses unknown fields, so until 1.1 lands the app's sign-in on this branch is refused).
- **The flaky CallScreen silence test is fixed at its cause** (commit "web: the silence tests wait for the
  level's interval…"): the meter can render before React runs the effect that sets the polling interval, so the
  test advanced a fake clock nothing was listening to. The tests now wait for that interval (a condition), with
  no longer timeout and no retry. Probe: missing once in 120 renders under CPU load before; 120/120 after.
- **Wave 1 integration: the `common` web catalog is the union of 1.5's and 1.6's keys.** They agreed on every
  shared key but "Something went wrong. Try again.": 1.5's "Nešto nije u redu. Pokušajte ponovo." (and its hr and sr
  forms) is kept, the wording two of 1.5's tests assert (one 1.6 test that asserted the other wording follows the catalog); 1.6's "Došlo je do greške." is the alternative for the reviewer.
- **`web/src/messages.ts` is deleted**, and `timeOf` and `longDate` in `web/src/time.ts` with their one test:
  after wave 1 nothing imports them (CLAUDE.md: delete unused code), and they were English-only formatters.
- **The questionnaire service gets the organization service as its `Clinics` port** in `internal/api/app.go` and
  the test composition in `questionnaire_cancel_test.go`; the three `internal/api` tests 1.4 left red pass.
- **The CapabilityStatement's Organization sentence names the default-language extension** (and its golden
  `metadata.json`); no interaction changed.
- **Area A and area B keep separate keys for a few shared English strings** (`calendar.join` and
  `consultation.join`, the consultation states, the default questionnaire texts): folding them is a later
  clean-up, not needed for correctness.
- **Wave 1 integration: the worker's booking fixture gives its patient the language `en`.** Since 1.3, a patient
  with no language gets Bosnian e-mails (SPEC §3), and the worker tests assert the English texts; the fixture now
  states the choice instead of the tests changing what they assert. Two test files had `renderInLanguage`
  imported twice after the merge of areas A and B; one import each was removed.
- **The dev database stayed at migration 00028 until wave 2's integration**: `docker compose run migrate` reuses
  the image built when the stack started, and migrations are embedded at build time. Wave 2's integration
  rebuilt the stack from `feat/i18n` (`docker compose up -d --build --wait`), which applied 00029.
- **`make e2e` is run as its steps** (web and e2e installs, the rate-limit reset, the Go and web journeys, the
  LiveKit log check) without its `up` prerequisite, for the reason the first point gives.
- **Wave 2 integration: the end-to-end journeys act as the apps now do.** Their patients register with
  `detectedLanguage` `en` (a patient with no language gets Bosnian e-mails, and the journeys read the English
  ones), and the builder journey answers the B5 prompt ("Publish without") before the publish confirmation.
  Both were seen failing first (9 Go journeys waiting 30 s for an English e-mail; the web journey missing the
  confirmation).
- **pgtest caps its pools at 8 connections** (`pgtest.PoolMaxConns`, test first): on this 28-core host the
  test packages' CPU-sized pools together passed Postgres's 400 connection slots ("too many clients", three
  packages in one `make test`). The terminology benchmark opens its own pool as wide as its concurrency.
- **A second CallScreen race fixed at its cause**: the join test asserted the room was shown the moment Leave
  rendered, but the stage shows it from an effect. It now waits for that condition (no timeout, no retry);
  under CPU load the probe failed 1 in 123 before, 0 in 164 after.
- **3.1 integration: the practitioner's line on the mobile free times screen is one catalog message**
  (`specialtyAtClinic`, all ten languages, Arabic with "،") instead of a `join(', ')` in code, which SPEC §3
  forbids and which put a Latin comma in Arabic. Test first; the Arabic booking golden was updated on purpose
  for the comma and looked at.
- **`postgres.SetPassword` serialises behind an advisory lock instead of retrying** (test first): every test
  package sets the app roles' passwords as it starts, and on this host ten jittered retries of "tuple
  concurrently updated" ran out (one package failed to start under `make cover` in the terminology 3.1
  integration). 48 callers on their own connections failed every time before the change and pass after.

### 0.1 Baseline: database, contract, checks

- **Clinic settings are FHIR, so `defaultLanguage` is not in `openapi.yaml`.** The clinic settings screen
  reads and updates `Organization` through FHIR (`PUT /fhir/R4/Organization/{id}` with `If-Match`), like the
  listing and the time zone, which are the extensions `organization-listed` and `organization-timezone`. The
  default language follows them as the extension
  `https://salus.invalid/fhir/StructureDefinition/organization-default-language` (`valueCode`, a staff tag),
  for package 1.2 to map in its `handler_fhir.go`. Not `Organization.language`: that is the language the
  resource is written in, not the clinic's choice.
- **The patient's questionnaire and the response are read through FHIR**, so `languages` on the patient's
  read is not a JSON field: the app derives them from `Questionnaire.language` and the `translation`
  extensions (package 1.4), and the response read carries `QuestionnaireResponse.language` (2.1). The JSON
  `Questionnaire` (what the builder's writes return) and `QuestionnaireResponse` (what the answer writes
  return) gain `language`, and the questionnaire `translations`.
- **Starting a response is a new endpoint, `POST /questionnaire-responses/{id}/start`** `{language}`: no
  endpoint started a response before. It may be repeated with another language until the first answer or
  file is saved; after that it answers 409 `language_locked`, a third new error code the plan did not list,
  added so 2.1 needs no contract change. The query enforces it too (no change once `answers` or
  `attachments` is not empty), and it moves the response to its next version.
- **Invite acceptance takes `language` and `detectedLanguage` too**, as registration and sign-in do: it
  creates the staff account (SPEC §3 "Picking and saving" names the invite screens).
- **`questionnaire.language` has the column default `bs`**, and the create query falls back to it when no
  language is given, so the existing service and every seeded test keep working until 1.4 passes the
  clinic's default; `UpdateDraft` with no language keeps the draft's. The backfill from the clinic's default
  runs in the migration (every clinic is `bs` at that point).
- **No database check on `account.language`**: the SPEC puts the staff-language rule in the service (422
  `language_unavailable`); the clinic's default language has the check the plan asks for.
- **`questionnaire_translation` writes go through `questionnaire`'s own policies** (an EXISTS on a draft), so
  a published version's translations never change and another clinic's are invisible; a deleted draft's go
  by the cascade.
- **Translations are read with their own repository call**, not with every questionnaire read, so searches
  stay one query.
- **The catalog check accepts French and Italian plurals without `many`**: CLDR gives them `many` only for
  millions in compact or exponent form, which the apps never format; a message may still have it. Every other
  category CLDR lists for a language is required, and one the language doesn't use is refused.
- **The legal tag is compared across languages only in the web catalogs**; ARB metadata lives in the template
  (`app_en.arb`) alone.
- **The checker loads the ICU parser from `web/node_modules`** (react-intl brings it, 0.2), only when there is
  a message to check; `make i18n-check` installs web first. This branch alone has no react-intl, so the
  parser was installed with `npm install --no-save` to run the checker's tests (git-ignored
  `web/node_modules`; nothing committed).

### 0.2 Web: i18n foundation

- Message ids are `<area>.<key>`: the provider prefixes each key with its area folder, so two areas can use
  the same key and a catalog file holds plain keys. Simplest way to merge eight areas into one react-intl map.
- A catalog value is an ICU string, or `{ "defaultMessage": "…", "description": "legal" }` when it carries the
  `legal` tag (SPEC §3 "Who translates": "a `description` in the web JSON"). Both shapes are read; 0.1's check
  should accept both (it runs on empty files in wave 0).
- Formatting locales: `en` is formatted as `en-GB` (day before month, as the app wrote dates before), Serbian as
  `sr-Latn-BA` / `sr-Cyrl-BA` so dates are ijekavian (SPEC §3 "Serbian is ijekavian"; plain `sr` is ekavian
  in CLDR). Intl's own punctuation is kept ("Monday, 9 November 2026"), not post-processed.
- Signed-in saving (`PATCH /auth/me`) and applying the account's `language` after `/auth/me` belong to 1.5
  (the picker); `useLanguage().choose` switches the UI and keeps the choice on the device, and 1.5 calls the
  API beside it. No generated API client is used in 0.2.
- `<html lang>` is set from the provider at runtime; `web/index.html` (not this package's file) keeps
  `lang="en"` until the script runs.

### 0.3 Mobile: i18n foundation

- **`app_sr.arb` beside `app_sr_Cyrl.arb` and `app_sr_Latn.arb`.** gen-l10n refuses script-coded catalogs without a
  base one ("Arb file for a fallback, sr, does not exist … Please create a {fileName}_sr.arb file"). The base is
  Serbian Cyrillic, CLDR's default script for `sr`, word for word the Cyrillic catalog (a test fails if the two
  differ, `@@locale` aside), so SPEC §3's ten files stay as they are and the fallback is never reached (the app always
  asks for a script). For 0.1's `i18n-check.sh` and every later package: `app_sr.arb` is an eleventh catalog, checked
  like `sr-Cyrl` (same plural categories), and every Cyrillic string is written into both files.
- **The generated `lib/l10n/app_localizations*.dart` are committed**, as gen-l10n writes them (not reformatted): the
  mobile CI job fails on any changed or untracked file after the tests, and `flutter pub get`, `analyze` and `test`
  regenerate them. A package that changes an `.arb` commits the regenerated files with it.
- **Noto Sans Arabic is a fallback family in every language**, not only in `ar`: only Arabic letters ever reach it,
  so the result is the same and the theme stays one. Three weights (400, 500, 700): the ones Material's text styles
  and bold text use; the unhinted fonts, since hinting does nothing on phone screens and they are half the size.
- **Latin digits**: `app.dart` sets `DateFormat.useNativeDigitsByDefaultFor('ar', false)` once when the app starts
  (process-wide intl setting), since intl writes Arabic-Indic digits in `ar` dates otherwise.
- **The saved choice** is the key `language` in secure storage (the same storage as the session token; nothing else
  on the device holds it). A value that is not one of the ten, or storage that cannot be read, leaves the device's
  language. `LanguageStore.write` and `delete` exist for 1.7's picker and are tested on the secure store.
- **`sr-Cyrl-ME`** (an explicit Cyrillic script on a Montenegrin device) is Cyrillic: the explicit script wins over
  the "`sr-ME` → `sr-Latn`" rule, which reads as Montenegro without a script.

### 1.1 internal/auth: the account's language

- **Sign-in saves the languages once the password is right, before the TOTP step.** Staff always have a TOTP
  step and the languages arrive only with `POST /auth/login`; waiting for the code would lose a choice made on
  the sign-in screen (SPEC §3 "Picking and saving"). A wrong password saves nothing. Only a value that changed
  is written and audited, so a sign-in that changes nothing adds no update event.
- **A patient language sent for a staff account at sign-in or invite acceptance is dropped, not refused.** The
  contract lists no 422 for those endpoints, the web app only offers staff languages, and a preference must
  not stop a sign-in. `PATCH /auth/me` refuses it with 422 `language_unavailable`, as the plan says.
- **A tag outside the enum is 400 `invalid_request` everywhere** (the contract's wording for `PATCH /auth/me`),
  checked in the service before any work; an empty string is refused by the handler.
- **`SetLanguage` with the same language still writes and audits**: the user asked for it; simpler than a
  compare, and it happens only on a click.
- **The tag list is a slice in `service_language.go`**, not read from `authdb`, which only the repository may
  import; the enum migration is the source and the tests pin every tag.

### 1.2 internal/organization: the clinic's default language

- **An invalid default language is 400 `value`**, as an invalid time zone is (both a coded value of the
  clinic outside what is allowed), not the 422 `language_unavailable` of an account's language: that code is
  the JSON API's, and this is the FHIR update. The web screen offers only the five staff languages, so it is
  not reached from the UI.
- **A StructureDefinition for the extension** (`api/fhir/`, outside the package's listed files): the other two
  Organization extensions have one and `make fhir-validate` loads them; a new file, so no other package's
  lines are touched. Its value has no binding (no R4 value set holds exactly the five tags, and there is no
  terminology server); the service enforces the five.
- **The CapabilityStatement's Organization documentation is not changed** (it names the listed and time-zone
  extensions): no interaction changes, and its golden copy `internal/api/router/testdata/metadata.json` is in
  `internal/api/router`, which terminology 1.1 owns in this wave. The integration step may add
  "organization-default-language" to that sentence and regenerate the golden.
- **The default language is in every Organization the API writes, the public directory search included**: it
  is no personal data, and one mapping keeps the resource the same everywhere.
- **`DefaultLanguage` sets the clinic's scope on the caller's transaction** (as `Read` does): callers in
  another scope on the same transaction (a worker) should call it before setting their own, or outside it.

### 1.3 internal/notification: e-mails in every language

- **The account's e-mail language is read by widening `auth.Service.AccountEmail`** to return `(email,
  language, err)`: the read the job already makes, audited with the job as agent, so no second audited read
  and no read of `account` outside auth. Auth gives the account's chosen language, else the detected one; the
  fallback (`bs`, or the clinic's default) stays in notification. The edit is in `internal/auth/
  service_notice.go` and its test only, files 1.1 does not change.
- **No staff-account branch of the fallback.** Every e-mail with an account goes to a patient (booking,
  cancellation, reminder); the only staff e-mail is the invite, which has no account and uses the clinic's
  default language. A staff e-mail to an existing account would add the clinic's default as its fallback.
- **The date is written by a per-language template and a names file**, not by a date library (none in the
  standard library, and the SPEC adds no dependency). Bosnian and Serbian write the month in the nominative
  ("7. januar 2030."), Croatian in the genitive ("7. siječnja 2030."), as each standard writes a date alone.
- **In every language but English the start is a line of its own** ("Datum i vrijeme: …", "Datum und
  Uhrzeit: …"), and the clinic follows a noun ("u ustanovi X", "Klinik: X" in Turkish), so neither the
  weekday nor the clinic's name ever has to be declined or take a suffix. English keeps its approved wording.
- **Patients and invitees are addressed formally** (vi/Vi, Sie, vous, Lei in Italian); Arabic uses the usual
  masculine imperatives of UI text and a passive for "you are invited".
- **"Salus" stays in Latin letters in the Cyrillic and Arabic e-mails**: it is the product's name.
- **The e-mail subject is RFC 2047 encoded** (`mime.QEncoding`, standard library) when it is not ASCII; an
  ASCII subject is sent as before.
- **No `legal` tag in the e-mail templates**: the catalog check has no metadata for them. The sentence with
  legal weight is the cancellation window ("You can cancel it up to 24 hours before it starts.", in `booked`):
  listed here for the lawyer's review per language (SPEC §8).

### 1.4 internal/questionnaire: original language and translations

- **`Questionnaire` carries its `Translations`** (`model.go`): the reads, the writes' answers and `ToFHIR` (also
  used by the QuestionnaireResponse `_include`) need them; filled by the service with one repository call per
  version, as 0.1 decided, so a staff search costs one query per version returned (no new sqlc query: the
  queries are 0.1's).
- **An absent `translations` keeps the draft's, an empty list removes them** (the contract's "Absent keeps the
  draft's").
- **A save drops a translated text whose original wording changed, unless the same save changes that
  translation too**, and the entries of items no longer in the draft, instead of refusing them: the builder
  sends back what it read, and a patient must never read the translation of an older wording (SPEC §3 "texts
  of items that changed … are missing until translated"). Text, help and options are judged separately.
- **An unknown language tag is 422 `invalid_questionnaire` without linkIds**, as the other content rules, not
  400; a translation in the original language (also after changing the original) is refused the same way.
- **The original language may be any of the ten tags**, like the translations (a clinic may write a
  questionnaire for German patients); only accounts and the clinic's default are limited to staff languages.
- **Complete means**: a title; a description when the original has one; for every item its text, its help text
  when the original has one, and every option. While a draft, an option may be blank (not yet translated)
  and a translation may hold no options yet.
- **Publish names the first incomplete translation in the enum's order** (the error carries one language).
- **A patient reads only complete translations** (`Referenced` and the patient's search filter them), staff
  every one: the builder reads drafts through the FHIR search, and a published version's are complete anyway.
- **No CapabilityStatement or profile change**: the translation extension is R4 core.

### 1.5 Web area A: shell, sign-in, calendar, admin

- **The picker is in the header and on the sign-in and invite cards**; the web app has no personal settings page,
  so "header and settings" is the header. Patients (who are told to use the app) get no picker.
- **Language names (L1) are a constant (`language.ts`), not catalog entries**: each name is the same in every UI
  language. The one exception to "no literal strings" in these files, besides the logo's "Salus".
- **The header switches the UI at once and saves after**; a failed save leaves the UI and the device in the chosen
  language and shows the fixed error. After `GET /auth/me` the account's language (none: the browser's) is
  applied and kept on the device, so the account wins on every device (SPEC §2).
- **Sign-in sends `language` only when one was picked on the device**, `detectedLanguage` always (the contract
  makes both optional).
- **Working-hours times are text fields (HH:MM, numeric keyboard)**: `<input type="time">` follows the browser's
  locale, which is the AM/PM issue the plan wants gone. A new message for a malformed time (below).
- **Area A uses its own catalog keys for strings area B also has** (`calendar.join`, `calendar.joinFrom`, the
  consultation states, the default questionnaire's W30/W31 texts), so this package needs none of 1.6's catalog
  files; English is word for word the same. The integration step may fold them into 1.6's ids.
- **"Practitioner" is translated as the doctor ("Ljekar", "Liječnik", "Љекар")**, as clinics in BiH say it; the
  twelve specialties are all doctors' specialties. ENT is "ORL"/"ОРЛ", as doctors write it.
- **"Times in [zone]" is "Vremenska zona: [zone]"** in bs, hr and sr: plainer than a word-for-word "Vremena u".
- **Small edits outside the area's components**: `web/src/api/fhir.ts` and its test (the default-language
  extension), `web/src/app.css` (the picker, text time fields).

### 1.6 Web area B: questionnaires, consultation, attachments

- **`messages.ts` keeps what 1.5's files still import.** Calendar, AppointmentDetail, App and Availability (1.5's)
  read `consultation` W1, W2, W35 to W38 and `questionnaires` W1, W30, W31 from `messages.ts`; removing them in
  this package would break this branch's build or need edits in 1.5's components. They stay there, marked to be
  deleted once nothing imports them; the catalogs hold the same texts (`consultation.join`, `consultation.states.*`,
  `questionnaire.defaultLabel`, …) for 1.5 to use. The integration step deletes the leftovers with the file.
- **Pure functions take the translator** (`ReturnType<typeof useT>`) and, where they format dates or numbers, the
  language, rather than returning message descriptors: the tests read real English through `test/translate.ts`.
  `problems` returns message ids, since its results are only keys and texts of the item cards.
- **Dates come from `i18n/format.ts`**, so English long dates gain Intl's comma ("Wednesday, 4 November 2026", as
  0.2 decided). W34 is "Submitted {date}, {time}" in English and "Predan: {date} u {time}" (sr "Predat: …") in the
  others, since a weekday after "Predan" reads badly without the colon. Answer dates stay without the weekday, as
  before; answer numbers keep every digit the patient wrote, with the language's decimal sign and no grouping.
- **W19 (`dependents`) keeps COPY.md's English word for word** in both plural branches ("1 items are shown …"), since
  the plan takes existing English verbatim; the other languages have their correct plural forms. The owner may
  want an English singular (question in the report).
- **`legal` tags**: the consultation's consent withdrawn (W18a), cancellation (W18b) and consent not active (W32),
  the request's `consent_required` (W40) and the withdrawal's deletion of saved answers (W36). The camera and
  microphone instructions (W10, W11) are browser settings, not permission prompts, and are not tagged.
- **Words chosen**: "Retire" is *arhivirati* (to keep it apart from "Withdraw", *povući*); the practitioner is
  *doktor* (bs, sr) and *liječnik* (hr); consent is *saglasnost* (bs, sr) and *privola* (hr); consultation is
  *konsultacija* (bs, sr) and *konzultacija* (hr); sentences with the patient's name avoid gendered past participles
  ("{name} još nije u konsultaciji.", "{name} ima isključenu kameru"). ENT is *ORL*.
- **Serbian Cyrillic** was written from the ijekavian Latin text by transliteration when the catalogs were drafted
  (not at runtime), keeping JPEG, PNG, PDF, MP4, KB, MB, Salus and the Escape key's name in Latin.
- **A request's title is optional in `responses.ts`**; the row shows "Questionnaire" (W35's word, as before) when
  the search did not include the version, so `requestsOf` needs no translator.
- **`common` keys** `somethingWrong` and `loading` are shared words; 1.5 may add the same keys, which the
  integration step unions.

### 1.7 Mobile: every screen in ten languages

- **The picker sits at the top of sign-in, registration and the home screen**: the patient app has no settings
  screen, and the home screen is the signed-in menu. A globe icon with the current language's own name, so a
  patient who cannot read the current language still finds it.
- **Signed in, the account's language wins entirely**: an account without one follows the device and clears the
  device's saved choice, so a "Use the device's language" on one phone holds on every phone (SPEC §2). A failed
  `PATCH /auth/me` keeps the choice on this device and says why; the next sign-in sends it again.
- **`app.dart` and `l10n/language.dart` are 0.3's scaffolding**: 1.7 edited `app.dart` (the account's language,
  the picker's state, the snack bar), as the wave notes ask; the endonyms live in `lib/language_picker.dart`.
- **The `legal` tag in ARB is a description starting with `[legal]`** in `app_en.arb` (consent, revocation,
  cancellation effects, permission explanations: 30 keys); ARB metadata lives in the template only.
- **English long dates keep "Thursday 5 November 2026"** (an explicit `EEEE d MMMM y` pattern) rather than intl's
  `en_GB` "Thursday, 5 November 2026", so the existing English copy and tests are unchanged.
- **Serbian dates are ijekavian by replacing intl's three ekavian weekday names** (ponedeljak, sreda, nedelja and
  their Cyrillic forms): Dart's intl has no `sr_BA` data.
- **The phone example became the placeholder `{example}`** in `phoneHint` and `invalidPhone`, isolated with
  U+2066/U+2069 in Arabic only: written into the catalog, the isolates reached the generated Dart literals, which
  the analyzer refuses.
- **Merged messages**: "1 file" and "[n] files" (M12) are one plural `fileCount`; "Submitted [date and time]" takes
  `dateAndTime` ("{date}, {time}") so each language orders them; an appointment's times are `timeRange`.
- **A message already shown stays in its language** after a language change until the next action (sign-in
  notices, errors): only the file problems of an upload, kept by `Uploads`, are written when shown.
- **Arabic "General practice" is طب الأسرة** (the draft had two terms with a slash).

### 2.1 internal/questionnaireresponse: the language answered in

- **Starting shares every check of saving the answers but If-Match**: the contract gives the start no If-Match and
  no 412, and a repeated start is harmless; the language query itself refuses a change once anything is saved.
- **The order of the refusals**: the answers' refusals first (forbidden, submitted, withdrawn, appointment closed,
  past due, consent required), then `language_locked`, then `language_unavailable`. The version is read (and that
  read audited, as for every save) before the last two, since one helper serves all the patient's changes.
- **A tag outside the enum is refused in the handler** with 400 `invalid_request` (the contract's "a tag outside
  the list included"), from a slice of the ten tags in `handler_http.go`, as auth and questionnaire keep theirs; the
  service refuses any other unknown language as `language_unavailable` anyway.
- **Choosing the same language again moves the response to its next version** like any other choice: simpler than
  comparing, and it only happens on a tap.
- **The language is in a Full view only**: the FHIR read without the answers (the clinic's admin, the practitioner
  before the submit) does not carry it, since it says which translation the patient answered and goes with the
  answers (data minimisation). The JSON writes, which only the patient receives, always carry it once set.
- **No test through the real questionnaire service in this package**: that a patient is given only complete
  translations is 1.4's rule, tested there; the end-to-end journey is 3.1's.

### 2.2 Web: the translation editor

- **The translation editor replaces the item list while a translation is open**, instead of a second column on
  every item card: the cards keep their drag handles, types and options, and the translator needs only the
  texts. The panel's buttons switch between the original and each translation, so the builder stays one column
  with nothing new on the cards but B7.
- **B5 comes before the usual publish confirmation**, not instead of it: "Publish without" goes on to "Publish this
  questionnaire?", which keeps its warning that a published version cannot be changed. The six existing publish
  tests answer B5 first (`withoutTranslation()`); nothing they assert changed.
- **The builder checks completeness before it asks**, as it already checks the items: an incomplete translation
  shows B6 and B7 without a request. On a 422 `translation_incomplete` the language comes from the same rule on
  the translations the API holds (the save's answer), because `attempt` (`web/src/api/result.ts`, not this
  package's file) passes on `linkIds` but not `language`; when the rule finds none, the alert is "Some items need
  attention." on the items named.
- **B7 is shown on the item cards** (the original view, where the doctor publishes) for the items the refused
  translation lacks, and goes away as they are translated or when the translation is removed; the empty fields of
  the translation editor say the rest.
- **A translation with no text at all is not kept across a reload**: the builder reads drafts through FHIR, where
  only translated texts carry the extension. Saved and reloaded before its title is written, it is gone, which
  loses nothing.
- **The ten language names live in `questionnaire.ts`** (the staff five from `language.ts`, plus Deutsch,
  العربية, Türkçe, Italiano, Français), not in `language.ts`, which is 1.5's file; the FHIR translation fields are
  typed there too (`FHIRQuestionnaire`), not in `api/fhir.ts`, for the same reason. The integration step may move
  them.
- **The response's language is read from FHIR `QuestionnaireResponse.language`**, which 2.1 adds; before 2.1 is
  merged no response carries one and the answers view is as before.
- **Choice answers are shown in the patient's language** in the translated view (the option at the same position),
  since the patient read them so; the switch shows the original options with the original questions.
- **The styles are in `web/src/translation.css`**, imported by the two components, not in `app.css`, which the
  terminology package edits in the same wave.
- **Words**: translation is *prijevod* (bs, hr) and *prevod* (sr, ijekavian Serbian uses it); "Original
  ([language])" is *Izvornik* (bs, hr) and *Original* (sr); "complete" is "sve prevedeno"; "Filled in in" is
  "Popunjeno na jeziku:" (bs, sr) and "Ispunjeno na jeziku:" (hr). The language name stays in the nominative in
  parentheses or after a colon, so no case ending is built onto an endonym.

### 2.3 Mobile: choosing the questionnaire's language

- **The response is started when the questionnaire opens**, not before its first save: a file upload saves without
  the answers' call, and a response must have its language before anything is saved, or it would be locked with
  none. The app's language starts it at once; a choice starts it with M3.
- **A response already started opens in its language**, even when the app's language has changed since and the
  questionnaire has it: the patient chose it, and switching silently would show other questions than the ones they
  began with. The start endpoint allows choosing again before the first answer, but the app offers no "change
  language" control (no COPY for one); the patient keeps what they started.
- **A questionnaire that names no language opens as written and is not started** (FHIR `language` is optional in
  R4; since 1.4 the API always sends it). Nothing is sent that the app would have to invent.
- **The language list in M2 is joined with ", "** in every language, Arabic included, since the names are each
  language's own name (L1) and mostly Latin script.
- **The choice page's title is the questionnaire's original title**: no language is chosen yet.
- **The test fixtures say the language**: `questionnaire()` is written in `en` and `response()` is started in `en`
  unless a test says otherwise, so the existing questionnaire, attachment and upload tests keep their requests
  exactly (no start call) and new tests pass `language: null` for a response not started.
- **`lib/questionnaire.dart` is about 460 lines** (364 before): the translation reading sits with the FHIR reading
  it extends; splitting one parser over two files would not make it simpler.

### 3.1 End-to-end, walk and privacy

- **The walk's drivers are not committed.** The Go driver and Playwright script that took the web screenshots, and
  the Flutter test that rendered the mobile ones, served one walk; the behaviour they show is checked on every
  `make e2e` by `language_test.go` and `language.spec.ts`, and by 1.7's and 2.3's widget tests. Keeping them would
  add code nothing runs (CLAUDE.md).
- **The staff journey signs in through the API with a fresh client**, as another browser would, rather than in a
  second web journey: the web journey already covers the browser side; the API journey adds the refusal of a
  patient language and the clinic's default language for the invite.
- **The `legal` count of the e-mails is 0** (1.3: the templates carry no metadata); the one e-mail sentence with
  legal weight, the cancellation window, is named in `TRANSLATIONS.md` for the lawyer.
- **The web `legal` column says "–" for the five patient-only languages**: the web app does not have them.
- **The DPIA section names 3.1 as its package** and describes the fields 0.1, 1.1 and 2.1 added, since 3.1 owns
  `docs/privacy/`.
