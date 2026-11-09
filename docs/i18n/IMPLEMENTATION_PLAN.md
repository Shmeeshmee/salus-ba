# Languages (i18n) implementation plan

Read `SPEC.md` first. Each package is one commit `i18n(<n>): <summary>`, made with `/run-package docs/i18n <n>`.
Packages run in waves (below), each package of a wave in its own worktree and branch, merged back into
`feat/i18n` in the wave's integration step. One PR into `main` at the end. The terminology plan
(`docs/terminology/`) runs alongside on `feat/terminology`, cut from `feat/i18n` after wave 0; the two
plans' waves share the one compose stack (below).

## Rules for every package

- Test first: write the failing test, run it and record the failure in `PROGRESS.md`, then the least code to
  pass, then refactor. Service tests with hand-written fakes and `clock.Fixed`; repository tests on
  `pgtest.NewDatabase` (`NewAppDatabase` for RLS); handlers with `httptest`; FHIR handlers against golden
  files; e-mails against golden files per language; web with Vitest and Testing Library rendering in a
  language; mobile with widget tests and, for right to left, golden screenshots.
- Build to `SPEC.md` §2 and §3. Do not reopen a settled decision; if the code and the plan disagree, stop and
  report.
- **No hard-coded user-facing string** in a package's files after it is done: every one goes through the
  catalog. A string the package adds or changes is written in **every language of that app** in the same
  commit (web: the five staff languages; mobile: all ten; e-mail: all ten), as a Claude draft, and
  `docs/i18n/TRANSLATIONS.md` is not touched outside 0.1 and 3.1. Strings with legal weight carry the
  `legal` tag (SPEC §3 "Catalogs").
- English source text is COPY.md word for word for strings that already exist (earlier features' COPY.md
  files) and `docs/i18n/COPY.md` for new ones. A new string that neither has is written in the same style,
  added to `docs/i18n/COPY.md` under "Added by a package, not yet reviewed", and the package continues.
- Translations: natural, plain, professional, as a native speaker writes for a clinic, never word for word.
  Serbian ijekavian. Bosnian and Croatian follow their own standard (e.g. Bosnian "historija", Croatian
  "povijest"; Bosnian "sedmica", Croatian "tjedan"). Medical terms as a doctor in BiH writes them.
- Every migration is a new file, with its grants and RLS. Policies read settings as
  `NULLIF(current_setting('…', true), '')`.
- After 0.1 nobody but 0.1 changes `api/openapi.yaml`, a migration, `ownership.yaml`, the archcheck config,
  `go.mod`, `package.json`, `pubspec.yaml`, `Makefile` or CLAUDE.md; a package that needs a change stops and
  reports.
- Dependencies: web `react-intl` (with its `@formatjs/*` peers), mobile `flutter_localizations` and `intl`
  (SDK) and the Noto Sans Arabic font asset; nothing else. Pin exact; record each in `PROGRESS.md`.
- In a wave, a package edits only the files it owns (below) and its own section of `PROGRESS.md`.
- No language next to an account or patient id in a log line. Never log a search or free-text value.
- Timing tests wait for a condition, never on a longer sleep.

## Shared stack

`deployments/docker-compose.yml` has a fixed project name (`salus`), so every worktree, of this plan and of
the terminology plan, shares one stack. The orchestrating session starts it once from the main checkout with
`make up` before a wave and stops it after the integration steps. Inside a wave:

- the stack runs from one checkout only: the main checkout, or the integration worktree if integration moves
  there (then that worktree starts and stops it, and stays until the run ends, because services bind-mount
  files from the checkout that started them). Nothing else starts, recreates or stops a shared service;
- every check from any other worktree uses `docker compose -f deployments/docker-compose.yml run --rm --no-deps
  …`, never a `make` target that runs `docker compose run` without `--no-deps`: from a second checkout such a
  run converges the shared services and recreates the ones whose bind-mounted path differs (LiveKit's
  `livekit.yaml`), failing the other runs (found by the attachments run, `docs/attachments/PROGRESS.md`, wave 1
  integration). That includes lint and archcheck: `… run --rm --no-deps -v <main checkout>/.git:<main
  checkout>/.git:ro tools bash scripts/lint.sh` (the migration check reads git, which a worktree's `.git` file
  points to outside the container) and `… run --rm --no-deps tools go -C tools run ./archcheck -root /src`;
- a package never runs `make up`, `make down`, `make test`, `make cover` or `make e2e`;
- a package runs only its own Go tests, `docker compose -f deployments/docker-compose.yml run --rm --no-deps
  tools go test -count=1 -cover ./<its packages>/...`; web and mobile packages run their npm and flutter
  commands the same way (`… run --rm --no-deps -T node npm …`, `… -T flutter flutter …`);
- a new worktree gets a copy of the main checkout's git-ignored `deployments/.env` (the port settings), or
  compose falls back to the default ports another project may hold;
- at most five package sessions run at the same time, across both plans; the rest wait for a free slot.

The full `make lint`, `make test`, `make archcheck`, `make cover`, `make fhir-validate` and (from wave 2) `make
e2e` run in each integration step, with nothing else running against the stack. Integration steps of the two
plans run one after the other, never together.

## Check sets

Every check runs through `make` or `docker compose … run --rm tools`, in Docker.

- **Go (in a wave)**: `make lint`, `make archcheck`, the package's own tests with `-cover` (each package it owns
  at 80% or more).
- **Go (alone: integration steps, 3.1)**: `make lint`, `make test`, `make archcheck`, `make cover`.
- **FHIR**: `make fhir-validate`.
- **Clients**: `make gen-clients` leaves no diff.
- **Catalogs**: `bash scripts/i18n-check.sh` (also inside `make web-lint`, `make mobile-analyze`, `make lint`).
- **Web**: `make web-lint`, `make web-typecheck`, `make web-test`, `make web-build`.
- **Mobile**: `make mobile-analyze`, `make mobile-test`, and the debug APK builds (`make mobile-apk` or the CI
  job attachments 4.1 added).
- **E2E**: `make e2e` (integration steps from wave 2, and 3.1, twice in a row in 3.1).

## Waves

| Wave | Packages | Why their files don't overlap |
| --- | --- | --- |
| 0 | 0.1, 0.2, 0.3 | 0.1 owns every shared file (migration, `ownership.yaml`, archcheck config, `api/openapi.yaml` and the generated clients in `web/src/api/` and `mobile/packages/salus_api/`, the Go models and sqlc queries for the new columns, `scripts/i18n-check.sh`, `Makefile`, CI workflows, CLAUDE.md, `docs/i18n/TRANSLATIONS.md`, the `PROGRESS.md` headings). 0.2 owns `web/src/i18n/` (except area catalogs' contents), `web/src/main.tsx` and `web/package.json`. 0.3 owns `mobile/lib/l10n/`, `mobile/l10n.yaml`, `mobile/lib/main.dart`, `mobile/lib/app.dart`, `mobile/pubspec.yaml` and `mobile/assets/fonts/`. The apps code against the clients 0.1 generates; 0.2 and 0.3 need none of them |
| 1 | 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7 | One domain or one app area each: `internal/auth`, `internal/organization`, `internal/notification`, `internal/questionnaire`, web area A, web area B, `mobile/lib` (everything but `l10n/` scaffolding). Web areas A and B own disjoint components and their own catalog files |
| 2 | 2.1, 2.2, 2.3 | `internal/questionnaireresponse`; the web builder's translation editor and the answers view (area B files, after 1.6); the mobile questionnaire screens (after 1.7) |
| 3 | 3.1 | Alone: end-to-end tests, the acceptance walk, `TRANSLATIONS.md`, `docs/privacy/` |

### Running a wave

1. From the main checkout on `feat/i18n`, with the previous wave integrated: `make up`.
2. Per package: `git worktree add ../salus-i18n-<n> -b i18n/<n> feat/i18n`, then in that worktree
   `/run-package docs/i18n <n>` (or the package-runner subagent with the same inputs). The runner commits
   once on `i18n/<n>` and does not push.
3. Integration step, in the main checkout, once every package of the wave is committed: `git merge --no-ff
   i18n/<n>` for each, in package order; the `PROGRESS.md` status table updated; the full check sets; a fix
   the merge needs is one commit `i18n(wave <w>): integration`. Then remove the worktrees and branches, and
   `make down` unless the terminology plan's next wave starts at once.

## Packages

### 0.1 Baseline: database, contract, checks

- **Files**: `migrations/<next>_language.sql`, `docs/architecture/archcheck/ownership.yaml`, `api/openapi.yaml`,
  `web/src/api/` and `mobile/packages/salus_api/` (generated), `internal/auth/{model.go,query.sql,authdb/}`,
  `internal/organization/{model.go,query.sql,organizationdb/}`,
  `internal/questionnaire/{model.go,query.sql,questionnairedb/}`,
  `internal/questionnaireresponse/{model.go,query.sql,questionnaireresponsedb/}`, `scripts/i18n-check.sh`,
  `Makefile`, `.github/workflows/*.yml`, `CLAUDE.md`, `docs/i18n/TRANSLATIONS.md`, `docs/i18n/PROGRESS.md`.
- **Migration**: the enum `language` (SPEC §3); `account.language`, `account.detected_language` (nullable);
  `organization.default_language` (not null, default `bs`, check: a staff language);
  `questionnaire.language` (not null, backfilled from the clinic's default language);
  `questionnaire_translation` (`questionnaire_id` references `questionnaire` on delete cascade, `language`,
  `texts jsonb`, timestamps, primary key (`questionnaire_id`, `language`), check `language <> ` the
  questionnaire's own language enforced in the service), RLS as `questionnaire`'s, owner `questionnaire`,
  `personal_data: false`; `questionnaire_response.language` (nullable until started). Grants to `salus_app`.
  Migration numbers: this plan takes the next free number on `main`; the terminology plan takes the one after.
- **OpenAPI**: `language` and `detectedLanguage` on register and sign-in requests; `language` (nullable) on
  `GET /auth/me` and `PATCH /auth/me`; `defaultLanguage` on the clinic settings read and update; `language`
  and `translations` (`{language, title, description, items: {linkId: {text, help, options[]}}}`) on the
  questionnaire draft read and `PUT`; `languages` (original first) on the patient's questionnaire read;
  `language` on starting a response and on the response read; the error codes `language_unavailable` and
  `translation_incomplete` (with `language` and `linkIds`). Then `make gen-clients`.
- **Go**: only the models' new fields and the sqlc queries that read and write the new columns, with
  repository tests. No service behaviour (wave 1).
- **`scripts/i18n-check.sh`**: compares every web area's language files with its `en.json`, every
  `app_<language>.arb` with `app_en.arb`, every e-mail template folder with `en/`: same keys, same ICU
  placeholders and plural categories each language needs (CLDR), valid ICU syntax. Runs inside the
  `node` service (a small script; no new dependency beyond `@formatjs/icu-messageformat-parser`, which
  `react-intl` already brings). Wired into `make web-lint`, `make mobile-analyze`, `make lint` and CI.
- **CLAUDE.md**: a short "Languages" section: the ten tags, staff versus patient languages, no hard-coded
  strings, every string in every language in the same commit, endonyms in the picker, `legal` tags, Latin
  digits, `docs/i18n/` for the rest.
- **`TRANSLATIONS.md`**: the table of languages, each `drafted`, and the review rules (SPEC §3).
- **Checks**: Go (alone), Clients, Catalogs (passes on the empty skeleton).

### 0.2 Web: i18n foundation

- **Files**: `web/src/i18n/` (provider, `useT`, language matching, saved choice, formatting helpers, test
  helper `renderInLanguage`, and an empty `<area>/<language>.json` pair for every area and staff language),
  `web/src/main.tsx`, `web/package.json` and its lockfile.
- **Areas**: `shell` (navigation, header, settings, picker), `auth` (sign-in, MFA, invite), `calendar`
  (calendar, availability, appointment detail without questionnaires), `admin` (practitioners, clinic
  settings), `questionnaire` (library, builder, answers), `consultation` (call screens), `attachments`
  (viewer), `common` (errors and shared words).
- **Do**: `react-intl` provider fed by the matched or saved language (SPEC §3 "Matching", "Picking and
  saving"); the language kept in `localStorage` before sign-in (try/catch, works without it); `<html lang>`
  set; date, time and number helpers (24-hour, Latin digits); a fallback to English for a missing key that
  logs nothing and fails the catalog check instead.
- **Checks**: Web, Catalogs.

### 0.3 Mobile: i18n foundation

- **Files**: `mobile/l10n.yaml`, `mobile/lib/l10n/` (`app_en.arb` template and the nine other languages,
  with only the keys the scaffolding needs), `mobile/lib/main.dart`, `mobile/lib/app.dart`,
  `mobile/pubspec.yaml` and its lock, `mobile/assets/fonts/NotoSansArabic-*.ttf` with its licence.
- **Do**: `gen-l10n` set up; locale resolution (SPEC §3 "Matching") with the saved choice in secure storage;
  the Arabic font as a fallback for `ar`; `Directionality` from the locale; Latin digits in `ar`.
- **Checks**: Mobile, Catalogs.

### 1.1 internal/auth: the account's language

- **Files**: `internal/auth/` (service, handlers and their tests; not the files 0.1 owns).
- **Do**: register and sign-in save `detectedLanguage` (and `language` when sent); `PATCH /auth/me` sets or
  clears `language`; staff accounts may only take a staff language (422 `language_unavailable`); every change
  audited like any account write; `GET /auth/me` returns it.
- **Depends on**: 0.1. **Checks**: Go (in a wave).

### 1.2 internal/organization: the clinic's default language

- **Files**: `internal/organization/` (service, handlers, tests; not 0.1's files).
- **Do**: clinic settings read and update `defaultLanguage` (admins only, a staff language, `If-Match` as the
  existing update); audited.
- **Depends on**: 0.1. **Checks**: Go (in a wave).

### 1.3 internal/notification: e-mails in every language

- **Files**: `internal/notification/` including `templates/<language>/`.
- **Do**: every e-mail kind that exists on `main` (invite, booking confirmation, cancellation, reminder,
  the questionnaire mention, revocation cancellations, and any the attachments feature added) as a template
  per language; language chosen per SPEC §3 "The account's language fields"; golden files per kind and
  language; the subject line translated too; the existing rules on what an e-mail may hold unchanged.
- **Depends on**: 0.1. **Checks**: Go (in a wave), Catalogs.

### 1.4 internal/questionnaire: original language and translations

- **Files**: `internal/questionnaire/` (service, handlers, `handler_fhir.go`, `testdata/`; not 0.1's files),
  `api/fhir/` profile or CapabilityStatement only if the translation extension needs listing (otherwise none).
- **Do**: a draft's `language` and `translations` saved with the draft (`If-Match`); validation: a translation
  language is not the original and appears once, every text it has respects the existing limits, option
  counts match; publish refuses an incomplete translation (422 `translation_incomplete`, language and
  `linkIds`); a new version copies translations, and items new or changed since are missing; the patient's
  read lists `languages`; FHIR `Questionnaire.language` and the `translation` extension (SPEC §3 "FHIR"),
  golden files, `make fhir-validate`.
- **Depends on**: 0.1. **Checks**: Go (in a wave), FHIR.

### 1.5 Web area A: shell, sign-in, calendar, admin

- **Files**: the components and tests of the `shell`, `auth`, `calendar` and `admin` areas, and those areas'
  catalog files.
- **Do**: every string through the catalog, in all five staff languages; the language picker in the header
  and settings (COPY.md L1–L4), saving per SPEC §3; the clinic's default language in clinic settings (admin);
  dates and times through the helpers; the browser-locale AM/PM issue from the demo is gone because times
  come from the helpers.
- **Depends on**: 0.1, 0.2. **Checks**: Web, Catalogs.

### 1.6 Web area B: questionnaires, consultation, attachments

- **Files**: the components and tests of the `questionnaire`, `consultation` and `attachments` areas, and
  their catalog files.
- **Do**: every string through the catalog in all five staff languages; specialty names from the `common`
  catalog keyed by SNOMED code; no new behaviour (the translation editor is 2.2).
- **Depends on**: 0.1, 0.2. **Checks**: Web, Catalogs.

### 1.7 Mobile: every screen in ten languages

- **Files**: `mobile/lib/` except `l10n/` scaffolding and `main.dart`/`app.dart`; `mobile/lib/l10n/*.arb`
  keys; `mobile/test/`.
- **Do**: every string through `AppLocalizations` in all ten languages (`messages.dart` becomes lookups);
  the language picker in settings and on the sign-in and registration screens (COPY.md L1–L4), saving per
  SPEC §3; the Android permission explanation and the iOS `Info.plist` permission strings localised
  (`InfoPlist.strings` per language); right to left: directional icons mirrored, numbers and phone fields
  left to right; golden screenshot tests in `ar` and `sr-Cyrl` for sign-in, My appointments, booking, the
  consent screen and the call screen.
- **Depends on**: 0.1, 0.3. **Checks**: Mobile, Catalogs.

### 2.1 internal/questionnaireresponse: the language answered in

- **Files**: `internal/questionnaireresponse/` (not 0.1's files) and its `testdata/`.
- **Do**: starting a response takes a `language` that is the original or a complete translation (422
  `language_unavailable` otherwise), stored; it cannot change after the first saved answer; the practitioner's
  read returns it; FHIR `QuestionnaireResponse.language`, golden files.
- **Depends on**: 1.4. **Checks**: Go (in a wave), FHIR.

### 2.2 Web: the translation editor

- **Files**: the builder's components and tests (area `questionnaire`), its catalog files.
- **Do**: the original-language select; "Add translation" offering the enum's languages not yet used;
  side-by-side editing per item (original left, translation right), completeness per language, the publish
  prompt (COPY.md B5) and the `translation_incomplete` errors on the items; the answers view shows the
  language answered in (COPY.md B9) and a switch to the original.
- **Depends on**: 1.4, 1.6. **Checks**: Web, Catalogs.

### 2.3 Mobile: choosing the questionnaire's language

- **Files**: the mobile questionnaire screens and their tests, ARB keys.
- **Do**: open in the UI language when available; otherwise the notice and the choice (COPY.md M1–M3),
  English preselected when available; texts rendered from the chosen translation; the choice sent when the
  response starts.
- **Depends on**: 1.7 (and the API from 0.1). **Checks**: Mobile, Catalogs.

### 3.1 End-to-end, walk and privacy

- **Files**: `test/e2e/`, `docs/i18n/WALK.md`, `docs/i18n/walk/`, `docs/i18n/TRANSLATIONS.md`, `docs/privacy/`.
- **Do**: Go e2e: registration with `detectedLanguage` `de` and the German confirmation e-mail; a staff
  language change that persists; a translated questionnaire answered in English by a German patient and the
  stored language; an Italian patient on a Bosnian-only questionnaire. Playwright: the web app in `sr-Cyrl`
  from the browser locale, then a saved Croatian choice. The acceptance walk (SPEC §6) on the dev stack and the
  emulator, with screenshots, `ar` included. `TRANSLATIONS.md` lists every language and the count of `legal`
  strings per language. The DPIA notes gain the three fields of SPEC §5.
- **Depends on**: every earlier package. **Checks**: every set, E2E twice in a row.

## Definition of done

Every package committed on `feat/i18n` and integrated; every check set passes locally and in CI; no
hard-coded user-facing string left in `web/src` or `mobile/lib` (the catalog check and a grep for literal
strings in widgets and JSX both clean, the exceptions listed in `PROGRESS.md`); the acceptance walk recorded;
`PROGRESS.md` complete.
