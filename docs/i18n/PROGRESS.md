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

### 1.2 internal/organization: the clinic's default language

### 1.3 internal/notification: e-mails in every language

### 1.4 internal/questionnaire: original language and translations

### 1.5 Web area A: shell, sign-in, calendar, admin

### 1.6 Web area B: questionnaires, consultation, attachments

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

### 2.2 Web: the translation editor

### 2.3 Mobile: choosing the questionnaire's language

### 3.1 End-to-end, walk and privacy
