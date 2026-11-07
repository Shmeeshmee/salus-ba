# Languages progress

| Package | Status | Commit |
| --- | --- | --- |
| 0.1 Baseline: database, contract, checks | done (integrated 2026-11-07) | 2929f27 |
| 0.2 Web: i18n foundation | done (integrated 2026-11-07) | 7e306fa |
| 0.3 Mobile: i18n foundation | done (integrated 2026-11-07) | 506f3cb |
| 1.1 internal/auth: the account's language | not started | |
| 1.2 internal/organization: the clinic's default language | not started | |
| 1.3 internal/notification: e-mails in every language | not started | |
| 1.4 internal/questionnaire: original language and translations | not started | |
| 1.5 Web area A: shell, sign-in, calendar, admin | not started | |
| 1.6 Web area B: questionnaires, consultation, attachments | not started | |
| 1.7 Mobile: every screen in ten languages | not started | |
| 2.1 internal/questionnaireresponse: the language answered in | not started | |
| 2.2 Web: the translation editor | not started | |
| 2.3 Mobile: choosing the questionnaire's language | not started | |
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

### 1.5 Web area A: shell, sign-in, calendar, admin

### 1.6 Web area B: questionnaires, consultation, attachments

### 1.7 Mobile: every screen in ten languages

### 2.1 internal/questionnaireresponse: the language answered in

### 2.2 Web: the translation editor

### 2.3 Mobile: choosing the questionnaire's language

### 3.1 End-to-end, walk and privacy
