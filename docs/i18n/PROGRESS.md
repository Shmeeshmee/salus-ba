# Languages progress

| Package | Status | Commit |
| --- | --- | --- |
| 0.1 Baseline: database, contract, checks | done (integrated 2026-11-07) | 2929f27 |
| 0.2 Web: i18n foundation | done (integrated 2026-11-07) | 7e306fa |
| 0.3 Mobile: i18n foundation | done (integrated 2026-11-07) | 506f3cb |
| 1.1 internal/auth: the account's language | done (integrated 2026-11-08) | bc7c1d8 |
| 1.2 internal/organization: the clinic's default language | done (integrated 2026-11-08) | e454d94, 49522e9 |
| 1.3 internal/notification: e-mails in every language | done (integrated 2026-11-08) | 1b8d833 |
| 1.4 internal/questionnaire: original language and translations | done (integrated 2026-11-08) | 0d23a80 |
| 1.5 Web area A: shell, sign-in, calendar, admin | done (integrated 2026-11-08) | 27675e3 |
| 1.6 Web area B: questionnaires, consultation, attachments | done (integrated 2026-11-08) | 85a34de |
| 1.7 Mobile: every screen in ten languages | done (integrated 2026-11-08) | d93bd73 |
| 2.1 internal/questionnaireresponse: the language answered in | done (integrated 2026-11-08) | 9fcb474 |
| 2.2 Web: the translation editor | done (integrated 2026-11-08) | 2c19147 |
| 2.3 Mobile: choosing the questionnaire's language | done (integrated 2026-11-08) | 7d04ee5 |
| 3.1 End-to-end, walk and privacy | not started | |

## Facts

- Plan written 2026-11-07 from the owner's answers in the desktop chat; SPEC §2 holds them.
- COPY.md: draft, approved when the owner starts the run.
- COPY.md approved by the owner when starting the run.

## Decisions

(none yet)

## Decided while the owner was away

#### The orchestrator

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

#### 0.1 Baseline: database, contract, checks

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

#### 0.2 Web: i18n foundation

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

#### 0.3 Mobile: i18n foundation

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

#### 1.1 internal/auth: the account's language

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

#### 1.2 internal/organization: the clinic's default language

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

#### 1.3 internal/notification: e-mails in every language

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

#### 1.4 internal/questionnaire: original language and translations

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

#### 1.5 Web area A: shell, sign-in, calendar, admin

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

#### 1.6 Web area B: questionnaires, consultation, attachments

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

#### 1.7 Mobile: every screen in ten languages

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

#### 2.1 internal/questionnaireresponse: the language answered in

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

#### 2.2 Web: the translation editor

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

#### 2.3 Mobile: choosing the questionnaire's language

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

## Waiting on people

- Native-speaker review of every language (SPEC §7, §8).
- A lawyer's review of the `legal` strings in every language (SPEC §8).

## Log

### 0.1 Baseline: database, contract, checks

Done 2026-11-07, on `i18n/0.1` (unattended run, wave 0).

Built:
- `migrations/00029_language.sql`: the enum `language` (the ten tags, staff languages first);
  `account.language` and `account.detected_language` (nullable); `organization.default_language` (not null,
  default `bs`, check: a staff language); `questionnaire.language` (not null, default `bs`, backfilled from the
  clinic's default); `questionnaire_translation` (primary key `questionnaire_id, language`, `texts jsonb`
  object, on delete cascade), row-level secured through `questionnaire` (each scope reads the translations of
  the versions it sees; only the clinic scope writes, only a draft's); `questionnaire_response.language`
  (nullable). Grants to `salus_app`; `questionnaire_translation` in `ownership.yaml` (owner `questionnaire`,
  `personal_data: false`). Tests: `migrations/language_test.go`.
- Go models and sqlc queries for the new columns, with repository tests: `auth.Account.Language`,
  `.DetectedLanguage`, `Repository.SetLanguage`, `.SetDetectedLanguage` (Create stores both);
  `organization.Organization.DefaultLanguage`, `Changes.DefaultLanguage` (read everywhere, written by Update);
  `questionnaire.Questionnaire.Language`, `Content.Language`, `.Translations`, `Translation`,
  `ItemTranslation`, `Repository.Translations`, `.PutTranslations` (Create, UpdateDraft and CreateVersion
  carry the language; CreateVersion copies the translations); `questionnaireresponse.Response.Language`,
  `Repository.SetLanguage`. No service or handler behaviour (wave 1). The test fakes implement the new port
  methods. Every domain's generated `models.go` gains the `Language` enum type (sqlc emits the schema's enums
  in every package).
- `api/openapi.yaml`: `Language` schema; `language` and `detectedLanguage` on `RegisterRequest`,
  `LoginRequest` and `AcceptInviteRequest`; `language` on `Account`; `PATCH /auth/me` (`updateMe`,
  `UpdateMeRequest {language: Language | null}`, 422 `language_unavailable`); `language` and `translations`
  (`QuestionnaireTranslation`, `QuestionnaireItemTranslation`) on `SaveQuestionnaireRequest` and
  `Questionnaire`; publish 422 `translation_incomplete` (`language`, `linkIds`);
  `POST /questionnaire-responses/{id}/start` (`startQuestionnaire`, `StartQuestionnaireRequest`, 409
  `language_locked`, 422 `language_unavailable`); `language` on `QuestionnaireResponse`; the error codes
  `language_unavailable`, `language_locked`, `translation_incomplete` and the error's `language`. Clients
  regenerated (`web/src/api/schema.d.ts`, `mobile/packages/salus_api`).
- `scripts/i18n-check.sh` (in the node service) running `scripts/i18n-check.mjs` and its tests
  (`scripts/i18n-check.test.mjs`, `node --test`): web areas against `en.json` (both value shapes, the legal
  tag compared), ARB files against `app_en.arb` (eleven files, `app_sr.arb` equal to `app_sr_Cyrl.arb`),
  e-mail template folders against `en/` (the same files and `{{…}}` fields); keys, ICU validity, placeholders
  and CLDR plural categories (`Intl.PluralRules`). Folders that don't exist yet are skipped. `make
  i18n-check`, a prerequisite of `make web-lint`, `make mobile-analyze` and `make lint`; the web and mobile CI
  path filters include the script.
- CLAUDE.md "Languages" section and `make i18n-check` under "Checks"; `docs/i18n/TRANSLATIONS.md`.

Test first (the failures seen):
- migrations: `language_test.go:22: ERROR: type "language" does not exist (SQLSTATE 42704)`, and the like for
  every column and table.
- auth: `a.Language undefined (type auth.Account has no field or method Language)`, `repo.SetLanguage
  undefined`. organization: `unknown field DefaultLanguage in struct literal of type
  organization.Organization`. questionnaire: `unknown field Language in struct literal of type
  questionnaire.Questionnaire`, `undefined: questionnaire.Translation`.
- questionnaireresponse: written together with its implementation; seen failing afterwards by removing the
  query's guard (`AND answers = '[]' AND attachments = '[]'`): `TestPostgres_SetLanguageWhileNothingIsAnswered`
  FAIL (the language changed after an answer was saved); guard restored, green.
- i18n-check: the test file run before the checker existed (module not found), then three failures on the
  first draft (a missing file crashed the comparison, placeholders compared wrongly, `it`/`fr` "many").

Checks:
- `go test -count=1 -cover` on `./migrations/`, `./internal/auth/`, `./internal/organization/`,
  `./internal/questionnaire/`, `./internal/questionnaireresponse/`: pass; coverage auth 95.7%, organization
  95.9%, questionnaire 94.3%, questionnaireresponse 93.6%. `go build ./...` and `go vet ./...`: pass.
- lint (`scripts/lint.sh`, `--no-deps`, the main checkout's `.git` mounted): pass, 0 issues; check-migrations
  ok. archcheck: pass.
- Clients: `gen:api` and the Dart generator with `dart format`, run twice: the second run leaves no diff.
- Catalogs: `bash ../scripts/i18n-check.sh` in the node service: 8 tests pass, "catalogs pass" (no catalogs on
  this branch). The same check over 0.2's and 0.3's committed catalogs (exported read-only with
  `git archive`): no problem.
- Not in the package's set, run because the clients changed: `npm run typecheck` and `npm test` in web (434
  tests pass), `flutter analyze` in mobile (no issues).
- `make test`, `make cover`, `make fhir-validate` and `make e2e` are left to the integration step (RUN-RULES).

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (no user-facing text in this package).

### 0.2 Web: i18n foundation

Done 2026-11-07 on `i18n/0.2`.

Built, all in `web/src/i18n/`:
- `languages.ts`: the five staff languages and `matchLanguage` (SPEC §3 "Matching", narrowed to what the web
  offers; `de`, `ar`, `tr`, `it`, `fr` are skipped; no match gives `bs`).
- `saved.ts`: the choice kept on the device (`localStorage` key `salus.language`, every access in try/catch,
  a value that is not a staff language is ignored), `startLanguage` (saved, else the browser's).
- `format.ts`: `formatTime` (24-hour), `formatDate` (`long`, `short`), `formatNumber`, all with Latin digits
  (`-u-nu-latn`); `intlLocale` maps `en` to `en-GB` and Serbian to CLDR's `sr-Latn-BA` / `sr-Cyrl-BA`.
- `catalog.ts`: every `<area>/<language>.json` loaded with `import.meta.glob`; `messagesFor` prefixes each key
  with its area (`shell.signOut`) and falls back to English for a missing key.
- `I18nProvider.tsx` (react-intl `IntlProvider` plus the language context; sets `<html lang>`; `onError`
  logs nothing), `useT.ts` (`useT`, `useLanguage` with `choose(language | null)`, `useFormat`),
  `renderInLanguage.tsx` (test helper with the real catalogs).
- An empty `{}` catalog for each area (`shell`, `auth`, `calendar`, `admin`, `questionnaire`, `consultation`,
  `attachments`, `common`) and each staff language: 40 files.
- `web/src/main.tsx` wraps `App` in the provider, starting in `startLanguage(navigator.languages)`.

Dependencies: `react-intl` 12.1.5, exact (12.1.6 was 11 days old; 12.1.5 is the newest older than two weeks).
It brings `@formatjs/intl` 6.1.4, `intl-messageformat` 12.1.4 and `@formatjs/icu-messageformat-parser` 3.5.22
(the parser 0.1's `scripts/i18n-check.sh` needs) at the exact versions react-intl pins. No other dependency.

Test first (failures seen before the code):
- module not found for the five new test files, then with stubs: `AssertionError: expected 'bs' to be 'hr'`
  (matching), `expected 'bs' to be null` (saved choice), `expected '' to be '15:05'` (time), `expected [] to
  deeply equal [ Array(8) ]` (areas), `expected undefined to be 'Settings'` (English fallback);
  `TypeError: startLanguage is not a function`. The Serbian ijekavian date was seen ekavian
  (`понедељак`) with `sr-Cyrl` before the `sr-Cyrl-BA` mapping.

Checks (each through `docker compose … run --rm --no-deps -T node`):
- `npm run lint` (oxlint): 0 warnings, 0 errors.
- `npm run typecheck`: clean.
- `npm test`: 29 files, 481 tests passed (47 new in `src/i18n/`).
- `npm run build`: built; the single chunk is 986.45 kB (278.12 kB gzip), with Vite's existing over-500 kB
  warning.
- Catalogs: `scripts/i18n-check.sh` does not exist on this branch (0.1 adds it in the same wave); not run.
  The files keep SPEC §3's layout, `web/src/i18n/<area>/<language>.json`, each `{}`, so the check passes on
  them once 0.1 is merged.

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (0.2 adds no user-facing text; the catalogs are empty).

### 0.3 Mobile: i18n foundation

2026-11-07, branch `i18n/0.3` (wave 0). Done.

Built:
- `mobile/l10n.yaml` and `generate: true`: gen-l10n from `mobile/lib/l10n/app_<language>.arb`, English the template,
  `AppLocalizations.of(context)` non-null. One key so far, `tryAgain` (the start-failure button in `app.dart`,
  M20's existing "Try again"), in all ten languages.
- `mobile/lib/l10n/language.dart`: the ten tags, `matchLanguage` (SPEC §3 "Matching", first match in the device's
  order, `en` when nothing matches), `localeOf` (Serbian by script code), and `LanguageStore` /
  `SecureLanguageStore` (key `language` in the platform's secure storage, beside the session token).
- `mobile/lib/app.dart`: the app's locale is the saved choice when it is one of the ten, else the device's matched
  language; Material, Cupertino and Widgets localizations (right to left for Arabic comes from them); the theme's
  text styles fall back to Noto Sans Arabic; dates in `ar` with Latin digits. `SalusApp` takes `languages`;
  `main.dart` passes the secure store.
- `mobile/assets/fonts/NotoSansArabic-{Regular,Medium,Bold}.ttf` and `NotoSansArabic-OFL.txt`, declared in
  `pubspec.yaml` as family `NotoSansArabic` (400, 500, 700).

Dependencies (pinned exact, recorded here as the plan asks):
- `flutter_localizations` (SDK, Flutter 3.47.7).
- `intl: 0.20.3`, the version `flutter_localizations` resolves (was transitive, now direct: the generated catalogs and
  `app.dart` import it).
- Noto Sans Arabic v2.013, the unhinted static TTFs from the notofonts/arabic GitHub release
  (`NotoSansArabic-v2.013.zip`, sha256 `1301aceaea84c501cf2e6dcfb3182e2328c8eae5725817fcb239672bda7154f1`), and
  `OFL.txt` from the same tag. sha256: Regular `bd86ca02f087d7f3c3788ba458fb6b73744c7639ed276b8d870dba6def6c40d0`,
  Medium `a66dfdfd2fd345d500a8f169d07297eb29fc683938c61d6bdd5717cafca86d51`, Bold
  `f4cb79f842a07d1bcf597879574358fb7010f73761396d1d0f2445dfedb4d2ca`.

Test first (`mobile/test/l10n_test.dart`; `test/support.dart`'s `Harness` gains a `FakeLanguageStore`):
- Red 1: `Error when reading 'lib/l10n/language.dart': No such file or directory` (matching, locales, store).
- Red 2, with `language.dart` and the catalogs in place but the app not wired: every "of the device" case
  `Expected: exactly one matching candidate … Found 0 widgets with type "FilledButton"` (the button still read
  "Try again"); `Expected: TextDirection.rtl Actual: TextDirection.ltr`; `Expected: contains 'NotoSansArabic'`.
- Red 3, the app wired: `Expected: '09:05' Actual: '٠٩:٠٥'` (intl's `ar` dates use Arabic-Indic digits); green
  after `DateFormat.useNativeDigitsByDefaultFor('ar', false)`. Numbers needed nothing: intl's `ar` number symbols
  already have Latin digits (the test guards it).

Checks (from the worktree, `docker compose -f deployments/docker-compose.yml run --rm --no-deps -T flutter …`):
- `flutter analyze`: no issues.
- `flutter test`: 358 passed (the 40 new ones included).
- `dart format -l 120` on the hand-written files: clean.
- Debug APK: not run. `flutter build apk --debug` in the `flutter` service stops at `No Android SDK found` (the image
  is built with `precache --no-android`); CI's `apk` job builds it with the runner's SDK.
- Catalogs: not run. `scripts/i18n-check.sh` is 0.1's and is not on this branch; the catalogs follow SPEC §3's layout
  (`app_<language>.arb`, English the template, `@@locale` in each).
- Go lint, archcheck, Go tests: not applicable (no Go file changed).

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (`tryAgain` is consultation COPY.md M20's "Try again").

Left for later (not this package's files):
- 1.7: `ios/Runner/Info.plist` needs `CFBundleLocalizations` listing the ten languages, or iOS offers Flutter only
  the app's development language; Android 13's per-app language list (`android:localeConfig`) is optional.
- 1.7: the account's `language` (0.1's client field) wins over the device's saved choice once signed in; the picker
  writes `SecureLanguageStore` before sign-in (SPEC §3 "Picking and saving").
- 1.7: golden tests in `ar` load `assets/fonts/NotoSansArabic-*.ttf` with a `FontLoader` (widget tests render with
  the test font otherwise).
- intl has date symbols for `sr` and `sr_Latn`, not `sr_Cyrl`; `Intl.verifiedLocale` should fall back from `sr_Cyrl` to
  `sr` (Cyrillic), which is right; formatting helpers in 1.7 pass the locale's `toString()` through it.

### 1.1 internal/auth: the account's language

Done 2026-11-08 on `i18n/1.1`.

Built, all in `internal/auth/` (no file 0.1 owns, and not `service_notice.go`):
- `service_language.go`: the ten tags and the five staff ones; `SetLanguage` (`PATCH /auth/me`: sets or, for
  `""`, clears the choice; staff take only a staff language, else `ErrLanguageUnavailable`; audited as an
  update, refusals and an unknown account as minor failures); `saveLanguages`, used by sign-in.
- `Register`, `Login` and `AcceptInvite` take the chosen and the detected language (`""` for none). A tag
  outside the enum is `ErrInvalidLanguage` before anything is hashed, counted, read or audited. Registration
  stores both on the new patient account; invite acceptance on the new staff account; sign-in, once the
  password is right and before the TOTP step, writes each that changed and audits one update event after the
  login event, in the same transaction.
- `errors.go`: `ErrInvalidLanguage` (400 `invalid_request`, as the contract says for a tag outside the list)
  and `ErrLanguageUnavailable` (422 `language_unavailable`).
- Handlers: register, login and invite acceptance accept `language` and `detectedLanguage`, each a tag, JSON
  `null` or absent (the Dart client's explicit nulls included); an empty string or another type is 400
  `invalid_request`. `PATCH /auth/me` (route added in `Routes`) requires `language`, a tag or `null`.
  The `Account` JSON gains `language` (the choice only; the detected language never leaves the server).

Test first (failures seen before the code):
- `vet: internal/auth/login_bench_test.go:117:92: too many arguments in call to svc.Login` (the service tests
  in `service_language_test.go` against the new signatures and `SetLanguage`);
- with the handlers compiling but not yet reading the fields: `handler_http_language_test.go:52: register
  with ,"language":"de","detectedLanguage":"ar" = 400 {"error":"invalid_request"}` (unknown field),
  `PATCH /auth/me {} = 405 Method Not Allowed`, `GET /auth/me = 200 {…"mfaEnrolled":false}` without the
  language.
- `TestLanguages_AreSavedByRegistrationSignInAndTheChoiceInTheDatabase` (service on Postgres) was written after
  the code, as a regression check of the whole path; it passed at once.

Checks (each through `docker compose … run --rm --no-deps -T tools`):
- `go test -count=1 -cover ./internal/auth/...`: ok, 96.0% (`authdb` is generated and excluded).
- `go build ./...`, `go vet ./internal/... ./cmd/...`: clean.
- `bash scripts/lint.sh` (with the main checkout's `.git` mounted read-only): 0 issues; check-migrations ok.
- `go -C tools run ./archcheck -root /src`: passes.

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (the server returns codes, not text).

### 1.2 internal/organization: the clinic's default language

Done 2026-11-08, on `i18n/1.2` (unattended run, wave 1). Two commits: the code (e454d94) and this log, which
the first commit missed (no amend allowed).

Built:
- `organization.Service.Update` sets `Changes.DefaultLanguage` when given; one that is not a staff language
  (`bs`, `hr`, `sr-Latn`, `sr-Cyrl`, `en`, exact tags) gives `ErrInvalidDefaultLanguage` before anything is
  accessed, like an invalid time zone. Admins only, `If-Match`, audited: the existing update path, unchanged.
- `organization.Service.DefaultLanguage(ctx, organizationID) (string, error)`, the port shape 1.4 declares (and
  1.3 may use): sets the clinic's scope on the transaction in ctx (joining an outer one), reads the clinic,
  `ErrNotFound` for an empty or unknown id; not audited (no personal data in a clinic).
- `handler_fhir.go`: the extension `https://salus.invalid/fhir/StructureDefinition/organization-default-language`
  (`valueCode`) on every Organization written (read, update, the listed search), read on update like the time
  zone (left out keeps the value; without a code, with another value type, or twice: 400 `invalid`).
  `ErrInvalidDefaultLanguage` is 400 `value`, "The default language must be bs, hr, sr-Latn, sr-Cyrl or en."
- `api/fhir/StructureDefinition-organization-default-language.json` (an Organization extension, `valueCode`),
  and its name in `api/fhir/embed_test.go`'s loop over the Organization extensions.
- Golden files: `organization_default_language.json`, `organization_default_language_updated.json`,
  `invalid_default_language.json`; `organization.json`, `organization_updated.json`, `organization_listed.json`
  and `search_listed.json` gain the extension (`bs`), reviewed.

Test first (the failures seen):
- Build: `undefined: organization.ErrInvalidDefaultLanguage`, `….DefaultLanguage undefined (type
  *organization.Service has no field or method DefaultLanguage)`.
- With the error declared and `DefaultLanguage` a stub: `Update(hr) = {… DefaultLanguage:bs …}; want the
  language stored`; `Update("de") = <nil>, calls [SetOrganization ByID Update], events 1; want
  ErrInvalidDefaultLanguage before any access`; `extension [{"url":"…organization-default-language"}]: status =
  200 after 1 service calls, want 400 and none`; `status = 500, want 400` for the error's mapping; the golden
  files missing.
- `api/fhir`: `StructureDefinition-organization-default-language.json: open …: no such file or directory`.
- After the fakes' clinics got `DefaultLanguage: "bs"` (as the database gives them), two existing tests'
  expected clinics gained the same field (`TestUpdate_RenamesTheClinicAndAuditsIt`,
  `TestUpdate_ListsTheClinicAndSetsItsTimeZone`); nothing else in them changed.

Checks:
- `go test -count=1 -cover ./internal/organization/... ./api/fhir/...`: pass; organization 95.9%, api/fhir
  100%.
- lint (`scripts/lint.sh`, `--no-deps`, the main checkout's `.git` mounted): 0 issues, check-migrations ok.
  archcheck: pass.
- FHIR: `scripts/fhir-validate.sh` in the `fhir-validator` service with `--no-deps`: 82 files, 0 errors (the
  new StructureDefinition and golden files included; warnings are the usual dom-6 narrative ones).

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (no user-facing text; the OperationOutcome diagnostics are
developer-facing English like the existing ones).

### 1.3 internal/notification: e-mails in every language

Done 2026-11-08 on `i18n/1.3` (unattended run, wave 1).

Built:
- `internal/notification/templates/<language>/` for the ten languages: `invite.txt`, `booked.txt`,
  `cancelled.txt`, `reminder.txt` (first line the subject, a blank line, the body; `text/template`), with COPY E1
  inside `booked` and `reminder` as `{{if .Questionnaire}}…{{end}}`; `date.txt` (how the start is written) and
  `names.txt` (weekdays from Sunday, then months, in the form the date uses: genitive in Croatian). Embedded in
  the binary and parsed once in `NewService` (`service_language.go`). Revocation cancellations use the
  cancellation e-mail (the same job), so they are covered; the attachments feature added no e-mail.
- The language: appointment e-mails (to patients) use the account's `language`, else `detected_language`, else
  `bs`; the invite (no account yet) uses the clinic's `default_language`, else `bs`. An unknown tag falls back
  to `bs`. Times are 24-hour with Latin digits in every language; the zone's name stays as before.
- `auth.Service.AccountEmail` now also returns the language to write in (the chosen one, else the detected
  one, else ""): read through the auth service, audited as before.
- `repository_smtp.go`: the subject is RFC 2047 encoded when it is not ASCII (Cyrillic, Arabic, accents).
- Golden files per kind and language: `testdata/<language>/{invite,booked,booked_questionnaire,cancelled,
  reminder,reminder_questionnaire}.txt` (subject line, blank line, body; 60 files). The English ones are the
  earlier goldens moved into `testdata/en/` with their subject line added, unchanged otherwise.

Test first (the failures seen):
- auth: `service_notice_test.go:15:17: assignment mismatch: 3 variables but f.svc.AccountEmail returns 2 values`.
- notification: with hand-written `bs/booked.txt`, `bs/invite.txt` and `hr/invite.txt`: `body of bs/booked.txt:
  Subject: Your appointment is booked …` (English sent to a Bosnian reader), the same for both invites. The
  other goldens were then written with `-update` and read one by one.
- SMTP: `the subject header "Subject: Подсјетник на ваш термин" is not ASCII`.

Checks:
- `go test -count=1 -cover ./internal/notification/ ./internal/auth/`: pass; coverage notification 86.0%, auth
  95.7%. `go build ./...` and `go vet ./...`: pass.
- lint (`scripts/lint.sh`, `--no-deps`, the main checkout's `.git` mounted): 0 issues, check-migrations ok.
  archcheck: pass.
- Catalogs: `bash ../scripts/i18n-check.sh` in the node service after `npm ci`: "catalogs pass"; a field
  renamed on purpose in `de/booked.txt` was reported, then restored.

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (the English text is the earlier features' approved e-mails, word
for word; the date's English form is unchanged).

### 1.4 internal/questionnaire: original language and translations

Done 2026-11-08 on `i18n/1.4` (unattended run, wave 1).

Built:
- Original language: a new questionnaire takes the language sent, else the clinic's default language through a
  new port `Clinics.DefaultLanguage(ctx, organizationID) (string, error)` (organization's, added by 1.2); an edit
  without a language keeps the draft's. Any of the ten tags; another tag is 422 `invalid_questionnaire`.
- Translations saved with the draft (`POST /questionnaires`, `PUT` with If-Match): absent keeps the draft's, an
  empty list removes them all. Validated (`validate_translation.go`): a known language, not the original, once;
  title, description, text, help and option texts within the original's limits; help only where the original
  has it; options none yet or as many as the item has, the non-blank ones different. Incomplete is fine in a
  draft. Each save drops the translated texts whose original wording changed (unless the save translates them
  again) and the entries of removed items, so a new version's changed or new items are missing until translated.
- Publish refuses an incomplete translation: 422 `translation_incomplete` with `language` (the first in the
  enum's order) and the missing items' `linkIds` (none when only the title or description is missing), audited
  as a refusal. Revise copies the translations (0.1's repository); Retire and Revise answer with them.
- Reads: `Questionnaire.Translations` filled on every read and write; staff see every translation (the builder
  reads drafts through FHIR), a patient (`Referenced`, the patient's search) only complete ones. JSON
  `Questionnaire` carries `language` and `translations`.
- FHIR: `Questionnaire.language`, and the R4 core `translation` extension (`lang`, `content`) on `_title`,
  `_description` (only when there is a description), `item._text`, the help item's `_text` and
  `answerOption._valueString`, one per language that has the text. Golden `questionnaire_translated.json`
  (a draft in Bosnian, English complete, German partial); the existing goldens gain `language`. No
  CapabilityStatement or profile change (a core extension, read and search unchanged).

Test first (the failures seen):
- Service: `fakes_test.go:349:103: unknown field Clinics in struct literal of type questionnaire.Ports` (and
  `undefined: questionnaire.TranslationIncompleteError`); after the code, three rules checked red by breaking
  them: the publish check removed → `TestPublish_RefusesAnIncompleteTranslation` FAIL; the reworded-text drop
  removed → `TestUpdateDraft_DropsTheTranslationOfAChangedText` FAIL (`invalid questionnaire`); the patient's
  filter removed → `TestRead_GivesTheTranslations` FAIL. Two first-run failures were the tests' fixtures (the
  fake kept the staff scope for the patient call; the draft had no description), fixed in the tests.
- HTTP: `POST = 400 {"error":"invalid_request"}` (translations refused as an unknown field), the bodies without
  `language` and `translations`, `publish with questionnaire: translation incomplete = 500`.
- FHIR: `open testdata/questionnaire_translated.json: no such file or directory`, then the three goldens
  without `language`; rewritten with `-update`, diff reviewed (only `"language":"bs"`/`"hr"` added).
- `TestService_TranslationsOnTheDatabase` (real database, salus_app, RLS) passed on its first run, written
  after the service.

Checks:
- `go test -count=1 -cover ./internal/questionnaire/`: pass, 94.4%. `go build ./...`, `go vet ./...`: pass.
  `./internal/questionnaireresponse/` (encodes `ToFHIR`): pass.
- lint (`scripts/lint.sh`, `--no-deps`, main `.git` mounted): 0 issues, check-migrations ok. archcheck: pass.
- FHIR: the HL7 validator (`fhir-validator`, `--no-deps`) over `internal/questionnaire/testdata/*.json`: 0
  errors (warnings: dom-6 narrative, and `Questionnaire.language` "unable to validate code without a server").
- `./internal/api/` is red on this branch until the wiring below: `TestNew_ServesTheQuestionnaireLibrary…`,
  `TestNew_ServesQuestionnaireResponses…`, `TestNew_RefusesQuestionnaireChangesOnACancelledAppointment` (the
  `Clinics` port is nil in `internal/api`).
- Benchmarks (`-benchtime=200x -count=3`, HEAD copy and this package side by side, the machine shared with
  other sessions): BenchmarkPublish 11.3–12.9 ms before, 9.7–10.7 ms after; BenchmarkReadVersion 1.0–1.2 ms
  before, 1.3–1.7 ms after (one more query, the translations); BenchmarkFHIRReadVersion 0.45–0.50 ms before,
  0.36–0.45 ms after. Within the noise of the shared machine but for the read's extra query.

Integration step must wire (not this package's files): `internal/api/app.go`
`questionnaire.Ports{…, Clinics: clinics}` (the organization service once 1.2's `DefaultLanguage(ctx,
organizationID) (string, error)` is merged), and the same in `internal/api/questionnaire_cancel_test.go`'s
`newResponses`.

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (the server returns codes only).

### 1.5 Web area A: shell, sign-in, calendar, admin

Done 2026-11-08 on `i18n/1.5` (unattended run, wave 1).

Built:
- Every string of App (header, navigation), SignIn, AcceptInvite, Enrolment, Calendar, Availability,
  AppointmentDetail (all but the questionnaire and file parts), Practitioners, PractitionerPicker, ClinicSettings
  and useSubmit goes through the catalogs `shell`, `auth`, `calendar`, `admin` and `common`, in the five staff
  languages (English word for word from the earlier COPY.md files, as the code had it). Plurals in ICU ("Too many
  attempts. Try again in # minutes": one, few, other in bs, hr, sr). The cancellation question carries the `legal`
  tag. Specialty names come from `common` keyed `specialty.<SNOMED code>`, the API's text only for an unknown code.
- `LanguagePicker.tsx`: the picker (COPY.md L1 to L3); in the header for staff, saving with `PATCH /auth/me` and
  showing L4 "Language saved." (or the fixed error); `DeviceLanguagePicker` at the foot of the sign-in and invite
  cards (`AuthCard`), kept on the device. `language.ts`: the L1 names and `signInLanguages`, which adds
  `detectedLanguage` (matched from the browser) and the device's `language` (when picked) to sign-in and invite
  acceptance. After `GET /auth/me` the account's language is applied (none: the browser's).
- Clinic settings: "Default language" (S1, S2) among the staff languages; `api/fhir.ts` maps the extension
  `organization-default-language` (`valueCode`, `bs` when absent) in `settingsOf` and `withSettings`.
- Dates, times and numbers through `useFormat` / react-intl in the UI language: calendar day headings, times,
  the appointment's date, joined-at times, birth date, invite expiry (24-hour). The working-hours fields are text
  fields typed as HH:MM instead of `<input type="time">`, which showed AM/PM in an en-US browser (the demo issue);
  a malformed time is refused before saving.
- `messages.ts`: `wrongCode`, `tooManyAttempts` and `statusLabels` removed; `somethingWrong` stays (area B files
  import it). `time.ts`: `dayLabel` removed; `timeOf` and `longDate` stay while area B files import them.

Test first (failures seen before the code):
- `LanguagePicker.test.tsx`: module not found.
- App: `Unable to find an accessible element with the role "heading" and name "Kalendar"`, and the like for the
  header picker. SignIn: `expected { email: 'staff@example.test', …(1) } to deeply equal { …(2) }` (no
  `detectedLanguage`), `Unable to find … "Пријава"`. AcceptInvite: the same for the acceptance body and Bosnian.
- Calendar, AppointmentDetail, Practitioners, ClinicSettings: `Unable to find …` for each translated name;
  `api/fhir.test.ts`: `expected { name: 'Synthetic clinic', …(2) } to deeply equal { …(3) }` (no default language).
- Availability: `expected 'time' to be 'text'`; `expected 'Something went wrong. Try again.' to be 'Enter each time
  as hours and minutes, …'`. useSubmit: `expected 'Something went wrong. Try again.' to be 'Nešto nije u redu.
  Pokušajte ponovo.'`.
- Seen passing from the start (a guard, not a new behaviour): Calendar "writes times in 24 hours" in sr-Cyrl.
- Existing tests changed on purpose: they render with `renderInLanguage(…, 'en')`; the sign-in and acceptance
  bodies now carry `detectedLanguage: 'en'`; the appointment date reads "Monday, 9 November 2026, 09:00–09:30"
  (Intl's punctuation, 0.2's decision); the clinic save sends the default-language extension; `time.test.ts` loses
  the `dayLabel` line with the function.

Checks (each through `docker compose … run --rm --no-deps -T node`):
- `npm run lint`: 0 warnings, 0 errors. `npm run typecheck`: clean. `npx vitest run`: 30 files, 511 tests passed.
  `npm run build`: built (the existing over-500 kB chunk warning).
- Catalogs: `bash ../scripts/i18n-check.sh`: "catalogs pass".
- Go lint, archcheck, Go tests: not applicable (no Go file changed). No browser walk (3.1 walks).

Decided while the owner was away:
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

COPY.md additions (not yet reviewed):
- A1 (web, Availability, a malformed working-hours time): "Enter each time as hours and minutes, from 00:00 to
  23:59." (`calendar.invalidTime`)


### 1.6 Web area B: questionnaires, consultation, attachments

Done 2026-11-08, on `i18n/1.6` (unattended run, wave 1).

Built:
- Catalogs `web/src/i18n/{questionnaire,consultation,attachments}/<language>.json` in the five staff languages:
  every text of the questionnaire library, builder and appointment requests (questionnaire COPY.md W1 to W42,
  attachments COPY.md W1 to W6), the call screens (consultation COPY.md W1 to W40) and the file list and viewer
  (attachments COPY.md W7 to W13). English is the old `messages.ts` text word for word. Plurals as ICU
  (`attachments.count`, `questionnaire.maxFilesSummary`, `questionnaire.dependents`), three forms in bs, hr and sr.
- `common` catalog: `somethingWrong`, `loading`, and `specialty.<SNOMED code>` for the owner's twelve specialties
  (`practitionerrole.Specialties`), for 1.5's practitioner screens.
- Every component and module of area B reads its text through `useT` (Questionnaires, Builder, ItemCard,
  builder-dnd, AppointmentQuestionnaires, CallScreen, DeviceCheck, Files); the pure functions take the translator
  and the language: `at`/`cancelled` (builder.ts), `valueText`/`statusOf` (responses.ts), `sizeText`
  (attachment.ts), `joinMessage` (consultation.ts); `problems` (builder.ts) returns message ids. Dates and times
  through `i18n/format.ts` (`useFormat`) in the UI language; answer dates and numbers and file sizes with the
  language's own month names and decimal sign. call.ts, questionnaire.ts had no user-facing text.
- `web/src/messages.ts`: the `files` block removed; of `consultation` and `questionnaires` only the entries 1.5's
  files still import are left (see the decision below).
- Test helper `web/src/test/translate.ts` (`translator(language)`: what `useT` returns, outside React).
- Outside the package's files, the smallest edits: `App.test.tsx` and `AppointmentDetail.test.tsx` (1.5's) render
  with `renderInLanguage(…, 'en')` instead of `render`, since the screens they render now include area B's
  components, which need the provider. No other line of those files changed.

Test first (the failures seen, before the code):
- `AssertionError: expected '1.5 MB' to be '1,5 MB'` (sizeText in hr); `expected [ 'Photo, 2 KB', 'Document, 1.5 MB' ]
  to deeply equal [ 'Fotografija, 2 KB', …(1) ]` (Files in hr).
- `expected [ 'Add text to this item.' ] to deeply equal [ 'questionnaire.noText' ]` and the like for every problem.
- joinMessage, valueText, statusOf and at in bs, hr and sr-Cyrl: the English text returned; the new date form
  (`Submitted Sunday, 8 November 2026, 18:45`) failed against the old one.
- Questionnaires (sr-Cyrl), Builder (hr), CallScreen (bs), AppointmentQuestionnaires (sr-Latn): `Unable to find
  role="heading" and name "Упитници"` / `"Provjerite kameru i mikrofon"` and the like.
- The request title fallback (a response whose version the search did not include is named "Questionnaire"): a
  new test, seen failing with the fallback removed (`Unable to find an element with the text: Questionnaire`),
  passing with it.

Checks (each through `docker compose … run --rm --no-deps -T node`, after `npm ci`):
- Catalogs, `bash ../scripts/i18n-check.sh`: 8 tests pass, "catalogs pass".
- `npm run lint` (oxlint): 0 warnings, 0 errors.
- `npm run typecheck`: clean.
- `npm test`: 29 files, 494 tests passed; run twice, no flake.
- `npm run build`: built (Vite's existing over-500 kB chunk warning).
- No Go, mobile or migration file changed: Go lint, archcheck, Go tests and mobile checks not applicable.
- Literal-string grep over area B's files (JSX text, `aria-label`/`title`/`alt` attributes, capitalised string
  literals): nothing left but non-text values (`'Escape'`, `'Tab'` key names, FHIR resource type names).

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (every English text is an existing COPY.md text or the existing
"Loading"; the specialty names are booking's `Text` values).

### 1.7 Mobile: every screen in ten languages

2026-11-08, branch `i18n/1.7` (wave 1). Done.

Built:
- Every string of `mobile/lib` through `AppLocalizations`: 255 keys in `app_en.arb` (the English of the booking,
  consultation, questionnaire and attachments COPY.md files word for word, `tryAgain` kept), drafted in all ten
  languages (`app_sr.arb` word for word `app_sr_Cyrl.arb`), the generated `app_localizations*.dart` committed.
  `messages.dart` keeps only the error-code mappings and validators, each taking the `AppLocalizations`, and the
  `context.t` extension. Specialties are named from the catalog by SNOMED CT code (`specialty<code>`, the twelve
  of `internal/practitionerrole`); the API's text only for a code the catalog lacks.
- The language picker (`lib/language_picker.dart`, COPY.md L1 to L4): a globe button with the shown language's own
  name on sign-in, registration and the signed-in home screen; the device's language first, then the ten
  endonyms. Before sign-in the choice is kept on the device (`SecureLanguageStore`) and sent with sign-in and
  registration as `language`, with the matched device language as `detectedLanguage`. Signed in it is saved at
  once (`PATCH /auth/me`) and "Language saved." (or the failure) is shown in the new language. The account's
  `language` wins once signed in (null: the device's) and is copied to the device for the next sign-in screen.
- Dates through intl in the UI language (`Zone.date`/`longDate` take gen-l10n's `localeName`), Latin digits;
  Serbian weekday names made ijekavian; MB sizes with the language's decimal mark.
- Right to left: back buttons, the call's badges and the local video placed by start/end, not left/right; e-mail,
  code, date of birth and phone fields typed left to right; the phone example isolated as a left-to-right run in
  Arabic.
- iOS: `CFBundleLocalizations` lists the ten languages (Serbian Cyrillic is iOS's `sr`), `InfoPlist.strings` per
  language with the three permission texts (M15 to M17), in the Xcode project as a variant group.
- Golden screenshots in `ar` and `sr-Cyrl` of sign-in, My appointments, booking (the slots), the consent screen and
  the call (`test/golden_test.dart`, `test/goldens/`), with Roboto and the Material icons from the Flutter SDK and
  Noto Sans Arabic loaded by `FontLoader`.

Test first:
- Red (`language_picker_test.dart`, `golden_test.dart`): `Error: The getter 'signInTitle' isn't defined for the type
  'AppLocalizations'` (and every other new key); `strings_test.dart` and `zones_test.dart` did not compile
  (`languageNames`, `Zone.date(start, 'en')`).
- Red (`sign_in_test.dart`, `register_test.dart`): the body sent `'detectedLanguage': null`, expected `'en'`.
- Red (`copy_test.dart`, now against `app_en.arb`): `Expected: true Actual: <false>` for every COPY string.
- Red on the way: `Expected: 'Thursday 5 November 2026' Actual: 'Thursday, 5 November 2026'` (intl's `en_GB`
  pattern; English keeps the app's own); `dependOnInheritedWidgetOfExactType<_LocalizationsScope>() … was called
  before _QuestionnaireScreenState.initState() completed` (a typed answer's field text needs no catalog).

Checks (from the worktree, `docker compose -f deployments/docker-compose.yml run --rm --no-deps -T flutter …`):
- `flutter analyze`: no issues.
- `flutter test`: 403 passed (358 before).
- `dart format -l 120` on the hand-written files: clean.
- Catalogs (`node` service, `bash ../scripts/i18n-check.sh`): pass.
- Debug APK: not run; the flutter image has no Android SDK (CI's `apk` job builds it).
- Go lint, archcheck, Go tests: not applicable (no Go file changed).

Decided while the owner was away:
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

COPY.md additions (not yet reviewed):
- 1.7-1 `timeRange`: "[start] to [end] ([zone])" (appointment and booking times; was built from pieces).
- 1.7-2 `dateAndTime`: "[date], [time]" (inside questionnaire M14 "Submitted [date and time]").
- 1.7-3 `specialty<SNOMED code>`: the twelve specialty names of `internal/practitionerrole` ("General practice",
  "Internal medicine", "Cardiology", "Dermatology", "Gynaecology", "Paediatrics", "Psychiatry", "Neurology", "ENT",
  "Ophthalmology", "Orthopaedics", "Endocrinology"), on the practitioner list and the booking screens.
- 1.7-4 `tooManyAttempts`: one plural for "Too many attempts. Try again in [n] minute(s)." (was built in code).
- The other English strings are those of the screens before this package, word for word.

Translation notes for the native-speaker review (drafted by Claude):
- "Practitioner": bs/sr "ljekar", hr "liječnik", it "medico", ar "طبيب" (narrower than the English), de "Behandler",
  tr "sağlık uzmanı", fr "praticien".
- Consent: bs "saglasnost", hr "privola", sr "сагласност"/"saglasnost", tr "onay" (a lawyer may prefer "rıza");
  it "titolare del trattamento" and de "verantwortlich" for "responsible" in the consent text (legal review).
- E-mail: bs "e-mail", hr and sr "e-pošta".

Left for later:
- Android 13's per-app language list (`android:localeConfig`) is optional and not added.
- The questionnaire's own language (choose it, COPY.md M1 to M4) is 2.3.

### 2.1 internal/questionnaireresponse: the language answered in

Done 2026-11-08 on `i18n/2.1` (unattended run, wave 2).

Built:
- `Service.Start(ctx, caller, id, language)` (`service_language.go`): the patient chooses the language they answer
  in. It runs through the same helper as saving the answers (`answer`), so the same checks and refusals hold: the
  patient only (403 to staff who see it, 404 to anyone else), in progress, the appointment booked (locked against a
  cancellation) and not started, an active consent; in one transaction in the patient's scope, audited as an update,
  refusals included. Then 409 `language_locked` once an answer or a file is saved, and 422 `language_unavailable`
  for a language that is neither the version's original nor a translation the questionnaire service gives the
  patient (`Referenced`, complete translations only, 1.4): the language comes from that service, never from a
  table of this domain. Stored with 0.1's `SetLanguage` (next version); it may be chosen again until the first
  answer or file.
- `answer` takes the If-Match version as `*int32`: nil (start) skips the version check, as the attachment changes
  already do by passing the response's own version; its `change` callback gets the whole questionnaire version
  instead of its items.
- HTTP: `POST /questionnaire-responses/{id}/start` `{"language"}` (the contract 0.1 wrote): a tag outside the enum,
  an empty or null one, or an unknown field is 400 `invalid_request` without calling the service; the JSON
  `QuestionnaireResponse` carries `language` once started. Routes are declared in the domain's `Routes`, so
  `internal/api` needed no edit.
- FHIR: `QuestionnaireResponse.language` (after `meta`) on a Full view, so the practitioner's read of a submitted
  response and the patient's own read carry it; golden `response_language.json` (a submitted response answered in
  `sr-Cyrl`). The existing goldens are unchanged (their responses have no language).

Test first (the failures seen):
- Service: first `f.svc.Start undefined (type *questionnaireresponse.Service has no field or method Start)`; with a
  stub `Start` and the two errors: `Start(en) = {…Language: …VersionID:0}, <nil>; want English at version 4`,
  `Start("de") = <nil>, want ErrLanguageUnavailable`, `an answer, Start(en) = <nil>, want ErrLanguageLocked`,
  `submitted: = <nil>, want questionnaireresponse: submitted` (and every other refusal unaudited).
- HTTP: `start = 404 404 page not found` for every start case (route, refusals, error codes).
- FHIR: `testdata/response_language.json differs from the response` (no `language` in the body), then written with
  `-update`; diff reviewed (only the new file).

Checks:
- `go test -count=1 -cover ./internal/questionnaireresponse/...`: pass, 93.7% (`questionnaireresponsedb` is
  generated, excluded).
- lint (`scripts/lint.sh`, `--no-deps`, main `.git` mounted): 0 issues, check-migrations ok. archcheck: pass.
- FHIR: the HL7 validator (`fhir-validator`, `--no-deps`) over the package's 11 golden files: 0 errors (warnings:
  dom-6 narrative, the questionnaire canonical not resolvable, attachment URLs not fetchable, as before).

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (the server returns codes only).

### 2.2 Web: the translation editor

Done 2026-11-08, on `i18n/2.2` (unattended run, wave 2).

Built:
- `questionnaire.ts`: a `Version` carries its original `language` and its `translations`, read from the FHIR
  `Questionnaire.language` and the R4 `translation` extensions (`_title`, `_description`, `item._text`, the help
  item's `_text`, `answerOption._valueString`, options by position) and from the JSON answer; `toRequest` sends
  `language` and every translation, held to what the original has (blanks, removed items, help the original lacks
  and a description the original lacks left out; a choice's options as many as the item has). The ten content
  languages and their own names (COPY.md L1) are a constant there.
- `translation.ts`: `progress` (n of total texts, the items that lack one, by 1.4's "Complete means"),
  `firstIncomplete` (the API's rule: first in the enum's order), `unused`, `withItem`.
- `Translations.tsx` and `translation.css`: the languages panel above the builder (B1 "Languages", B2 the
  original-language select, B4 per translation a quiet button with "[language]: n of total texts translated" or
  "[language]: complete" that opens it, and "Remove translation" with its confirmation, B3 "Add translation"
  offering the enum's unused languages by their own names); the translation editor (B8): per text the original
  on the left (its kind and its text) and the translation's field on the right, named by the kind and the
  language, described by the original, `lang` and `dir="auto"` on the field (Arabic is written right to left);
  stacked on a narrow screen. The palette and the item cards give way to it while a translation is open;
  "Original ([language])" goes back.
- `Builder.tsx`: the draft's language and translations are saved with it; a save takes the answer's
  translations (the API drops a text whose original changed). Publishing: problems first as before; then an
  incomplete translation is not offered (B6 alert, B7 "Missing in [language]" and the attention border on its
  items); with no translation, B5 asks first ("Add translation" opens the language choice, "Publish without"
  goes on to the usual confirmation); a 422 `translation_incomplete` shows B6 and B7 the same way.
- `responses.ts` and `AppointmentQuestionnaires.tsx`: a response's `language` (FHIR
  `QuestionnaireResponse.language`, 2.1); answered in a translation, the answers show the questions, section
  titles and choice answers as the patient read them (the original's text where the translation lacks one), with
  "Filled in in [language]" and the switch "Show the original questions" (B9). Answered in the original, only the
  "Filled in in" line. Free-text answers are shown as written.
- Catalog keys in `web/src/i18n/questionnaire/<language>.json`, five languages: `languages`,
  `originalLanguage`, `addTranslation`, `translationProgress` (plural on the total), `translationComplete`,
  `removeTranslation`, `removeTranslationQuestion`, `removeTranslationWarning`, `removeConfirm`,
  `translateQuestion`, `translateWarning`, `publishWithout`, `translationIncomplete`, `missingIn`,
  `originalColumn`, `filledIn`, `showOriginal`. None has legal weight.
- `textLabel` moved from `ItemCard.tsx` to `builder.ts` (the item card and the translation editor share it, and
  a component file exports components only); `all` in `builder.ts` exported.

Test first (the failures seen, before the code):
- `questionnaire.test.ts`: `expected undefined to deeply equal [ { language: 'en', …(3) } ]` (toRequest without
  translations), and `v.translations` undefined on the FHIR and JSON reads.
- `translation.test.ts`: `Failed to resolve import "./translation"`.
- `Translations.test.tsx`: all nine red (`Unable to find a label with the text of: Original language`).
- `Builder.test.tsx` with the publish checks taken out of `Builder.tsx` for the run: the five B5 to B7 tests red
  (`Unable to find role="alertdialog" and name "Add a translation before publishing?"`, `expected 'Some items need
  attention.' to be 'The English translation is not comple…'`), as were the six existing publish tests once they
  answer B5. Two guards (marks gone after removing the translation, a complete translation goes straight to the
  confirmation) pass either way. One first run red after the code was the test's own: a change to the same title
  value is no change in React, so nothing was saved; the test now renames.
- `responses.test.ts`: `expected undefined to be 'en'`; `AppointmentQuestionnaires.test.tsx`: `Unable to find an
  element with the text: Filled in in English` (and the Bosanski and hr cases).

Checks (each through `docker compose -f deployments/docker-compose.yml run --rm --no-deps -T node`, after `npm ci`):
- Catalogs, `bash ../scripts/i18n-check.sh`: "catalogs pass".
- `npm run lint` (oxlint): 0 warnings, 0 errors.
- `npm run typecheck`: clean.
- `npm test`: 32 files, 556 tests passed; run twice in a row, no flake.
- `npm run build`: built (Vite's existing over-500 kB chunk warning).
- No Go, mobile, migration or contract file changed: Go lint, archcheck, Go tests, FHIR validation and mobile
  checks not applicable.
- Literal-string grep over the package's files: no user-facing literal but the ten language names (L1), as in
  `language.ts`.
- No browser walk (3.1 walks; this session has no browser tool).

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none. Every English text is COPY.md B1 to B9 word for word; B4's count is
written with a plural on the total, so a questionnaire of one text reads "1 of 1 text translated".

### 2.3 Mobile: choosing the questionnaire's language

2026-11-08, branch `i18n/2.3` (wave 2). Done.

Built:
- `Version` (`lib/questionnaire.dart`) reads `Questionnaire.language` and its `languages` (the original first, then
  every language a `translation` extension names, in the order met, only the app's ten), and `inLanguage(tag)`
  reads the same resource in a translation: title, item texts, help texts and option labels. Option values stay as
  written (`Item.options`), since answers and `enableWhen` name them; `Item.label` shows them, in the fields and in
  the submitted answers. `Response.language` reads `QuestionnaireResponse.language` (2.1).
- `lib/questionnaire_language.dart` (`QuestionnaireLanguage`), between the appointment card and the questionnaire
  screen: a response already started (or submitted) opens in its language; otherwise, when the questionnaire has
  the app's language, the response is started in it at once (`POST /questionnaire-responses/{id}/start`,
  `Booking.start`); otherwise COPY.md M1 and M2 with the languages by their own names, English preselected when it
  is one of them, else the original, and M3 starts it. The questionnaire screen then gets the version in that
  language and the response at the version the start answered (its next `If-Match`).
- Refusals: 422 `language_unavailable` says M4 and takes that language out of the choice; 409 `language_locked`
  reads the response again and opens in the language it has; anything else says the answers' message and keeps the
  choice to try again (a 404 or another 409 hides the button).
- The appointment card names the questionnaire in the language it opens in.
- ARB: `questionnaireNotInLanguage`, `questionnaireLanguages`, `continueInLanguage`, `languageUnavailable` (COPY.md M1
  to M4, word for word) in all ten languages and `app_sr.arb`; generated `app_localizations*.dart` committed.
  `answersMessage` maps `language_unavailable` to M4. `copy_test.dart` checks M1 to M4 against the catalog.

Test first:
- Red (`copy_test.dart`): `Expected: true Actual: <false>` for "This questionnaire is not available in [language]."
- Red (`questionnaire_test.dart`): did not compile (`languages`, `inLanguage`, `label`, `Response.language`
  undefined); then `Expected: (bs, [bs, en, de]) Actual: (bs, [bs, en, de])` (a record compares lists by identity;
  the assertion was split, the code was right).
- Red (`questionnaire_language_test.dart`): `Error: The method 'questionnaireNotInLanguage' isn't defined for the
  type 'AppLocalizations'` (and M2 to M4); then `Expected: 'W/"3"' Actual: 'W/"4"'` (the test read the second save;
  it now reads the first).

Checks (from the worktree, `docker compose -f deployments/docker-compose.yml run --rm --no-deps -T flutter …`):
- `flutter analyze`: no issues.
- `flutter test`: 420 passed (403 before; 12 widget, 4 unit and 1 copy test added).
- `dart format -l 120` on the changed hand-written files: clean.
- Catalogs (`node` service, `npm ci` then `bash ../scripts/i18n-check.sh`): pass.
- Debug APK: not run; the flutter image has no Android SDK (`flutter doctor`: "Unable to locate Android SDK"); CI's
  `apk` job builds it.
- Go lint, archcheck, Go tests: not applicable (no Go file changed).

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none (M1 to M4 are in COPY.md).

Translation notes for the native-speaker review (drafted by Claude):
- The language's own name is never declined: bs/hr/sr "na jeziku: {language}", tr "şu dilde: {language}", ar
  "باللغة: {language}"; de "auf {language}", it "in {language}", fr "en {language}" read naturally with the names a
  German, Italian or French patient sees (Deutsch, English, Bosanski).
- Imperatives follow each catalog: bs and hr "Odaberite", sr "Izaberite"/"Изаберите", it the formal "Scelga".

Left for later:
- Changing the language after starting and before the first answer (the API allows it; the app has no control).

### 3.1 End-to-end, walk and privacy

2026-11-08, branch `i18n/3.1` (wave 3, alone). Done.

Built:
- `test/e2e/language_test.go`, five journeys: a German phone registers (`detectedLanguage` `de`) and gets the
  booking confirmation in German; the admin sets the clinic's default language to English and the invite e-mail is
  English, then chooses Croatian (a patient language is refused, 422 `language_unavailable`), signs in again from an
  English browser and keeps Croatian, and clears it; a Bosnian questionnaire whose half-done English translation
  publishing refuses (422 `translation_incomplete`, `en`, `t-choice`) is finished, published and made the role's
  default, a German patient sees Bosnian and English (German refused), answers in English, cannot change the
  language after the first answer (409 `language_locked`), and the practitioner reads `language` `en`; an Italian
  patient on a Bosnian-only questionnaire is refused Italian and English and answers in Bosnian; and the web
  journey below. `newPatientIn(t, language)` beside `newPatientWithProfile` (which stays English), and the
  journeys' `response` reads `language`.
- `test/e2e/web/language.spec.ts`: the sign-in screen of a browser in `sr-Cyrl` is in Serbian Cyrillic; the admin
  picks Hrvatski, signs out, signs in from another English browser and the app is Croatian; "Use the browser's
  language" brings English back. `signin.ts` remembers the last TOTP step it used, so one spec can sign in twice.
- The acceptance walk (SPEC §6) in `docs/i18n/WALK.md`, screenshots and e-mail texts in `docs/i18n/walk/`.
- `TRANSLATIONS.md`: every language but the English source `drafted`, and the count of `legal` strings per
  language.
- `docs/privacy/dpia.md`: "Languages", the three fields of SPEC §5 with purpose, basis, access, logs and retention.

Test first:
- The journeys test behaviour packages 1.1 to 2.3 built, so they passed on their first run against the stack. Each
  was then seen failing for the right reason by breaking its input, not its assertion: the German patient
  registered with `en` instead of `de` — `no e-mail "Ihr Termin ist gebucht" to the address in Mailpit within 30s`;
  the second sign-in sending `language` `en` — `GET /auth/me after signing in from an English browser = …, want the
  chosen hr`; the web journey's first browser in `en` instead of `sr-Cyrl` — `expect(locator).toBeVisible() failed
  … Received: <element(s) not found>` (the heading "Пријава"). The inputs were restored.
- Lint found a struct out of field order in the new file (`fieldalignment`); fixed.

Checks (from the worktree, each `docker compose -f deployments/docker-compose.yml run --rm --no-deps -T …`):
- Catalogs (`bash ../scripts/i18n-check.sh` in `node`): pass.
- Lint (`scripts/lint.sh`): 0 issues after the field-order fix; migrations unchanged.
- Archcheck: pass. `go vet -tags e2e ./test/e2e/`: clean.
- `scripts/cover.sh` (`go test ./...` with coverage, both modules' app tests): pass, total 94.2%, every package at
  80% or more (some packages' results came from the build cache: their code is unchanged). `go -C tools test ./...`:
  pass.
- FHIR validation (`scripts/fhir-validate.sh`): 84 resources, no errors.
- Clients: `npm run gen:api` left no diff (no contract change; the Dart client not regenerated for the same reason).
- Web: lint, typecheck, 556 tests, build: pass. Mobile: `flutter analyze` no issues, 420 tests pass. Debug APK not
  built (the flutter image has no Android SDK, as in 2.3; CI's `apk` job builds it).
- E2E twice in a row (the `make e2e` steps with `--no-deps`): run 1 ok in 70.4 s, run 2 ok in 56.6 s; the LiveKit
  log check passed both times (none of the 22 synthetic values, each of the 2 rooms).

The walk: no browser pane, so the web steps ran as a Playwright script in Chromium in the `playwright` service,
driven by a Go test that made the synthetic data through the API (both files run from the worktree, not committed:
a one-off, and the committed journeys cover the same behaviour). The owner's emulator was down, so the mobile
steps are widget-rendered screenshots with the real fonts, as 1.7's goldens (a test file run with
`--update-goldens`, not committed); no mobile step was walked on a device. The e-mails came from Mailpit's API.

The `legal` counts, by this script (Python, run on the host over the repository; not committed):

```python
import glob, json, os
web = {}
for f in glob.glob('web/src/i18n/*/*.json'):
    lang = os.path.basename(f)[:-5]
    web[lang] = web.get(lang, 0) + sum(1 for v in json.load(open(f)).values() if isinstance(v, dict) and v.get('description') == 'legal')
arb = json.load(open('mobile/lib/l10n/app_en.arb'))
mobile = sum(1 for k, v in arb.items() if k.startswith('@') and not k.startswith('@@') and isinstance(v, dict) and str(v.get('description', '')).startswith('[legal]'))
print(web, mobile)  # {bs, hr, sr-Latn, sr-Cyrl, en: 6 each} 30
```

Decided while the owner was away:
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

COPY.md additions (not yet reviewed): none.

Noticed outside this package (in WALK.md, not changed here): `mobile/lib/slots_screen.dart` joins the specialty
and the clinic's name with `', '` in code (a Latin comma in Arabic too), against SPEC §3 "No string is built by
joining pieces"; hr "Nema termina ovaj tjedan." would read better as "ovog tjedna".
