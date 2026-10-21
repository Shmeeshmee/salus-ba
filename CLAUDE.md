# Salus

Telemedicine platform: patients book and attend remote consultations with practitioners, and the
clinical record of those consultations is kept as FHIR resources. Go binaries + Postgres + Redis + object
store, LiveKit for video, a React web app for practitioners and admins, a Flutter app for patients. One
monorepo. Everything runs through `docker compose` (`deployments/`): the host needs only Docker and make.
No `go`, `npm`, `node` or `flutter` on the host; `make` wraps every command in `docker compose run --rm`.

Module path: `github.com/Shmeeshmee/salus`. Toolchain versions are pinned as image tags in `build/package/`
and `deployments/`; Go tools (sqlc, goose, golangci-lint) as `tool` lines in a separate `tools/go.mod`, run
with `go tool -modfile=tools/go.mod <tool>`, so the app's `go.mod` holds only what the app needs; Go itself
also by the `go` and `toolchain` lines in `go.mod`.

## Top rules

These four beat everything else in this file. When two rules pull against each other, the earlier one wins
unless it would break the law: GDPR is never traded for performance, simplicity or elegance.

1. **Fast, then simple, then elegant.** In that order. Performance is measured, never assumed: a benchmark,
   a profile or a load test justifies any trade of simplicity for speed, and the numbers go in the package's
   `PROGRESS.md`. Without a measurement, the simplest thing that passes the tests is the right thing.
   - No abstraction without a second real use or a test seam that needs it. No interface with one
     implementation unless a test fakes it.
   - No DI framework, no ORM, no event bus, no CQRS. The only code generators are what Go ships, sqlc for
     queries, and the OpenAPI client generators (see "Layout").
   - No microservices. Several binaries in `cmd/` share `internal/` and one database; they are one
     application deployed as separate processes, not separately owned services. Split a binary out only
     when its load or lifecycle differs, and never give it its own copy of another domain's data.
   - Do not add a dependency for what the standard library does in under ~50 lines.
   - Prefer boring, explicit code over clever code. A reader new to the repo should follow a request
     from handler to SQL without a debugger.
   - Delete code that is not used. Do not build for a requirement that is not in a SPEC.
2. **Test first (TDD).** No production code without a failing test that asks for it. See "TDD".
3. **FHIR is the data model.** Clinical and administrative data are FHIR R4 resources. See "FHIR".
4. **GDPR by design.** Health data is special-category data (Art. 9). See "GDPR and health data".

## Layout

Follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout). Only create a
directory when something goes in it.

| Path | What it is |
| --- | --- |
| `cmd/<app>/main.go` | One executable each, composition only: load config, connect, build, run, shut down. `api` (HTTP and FHIR), `migrate` (goose), `worker` (River jobs), `scheduler` (periodic jobs), `notifier` (e-mail, later SMS and push), `agent` (LiveKit agent for live transcription) |
| `internal/<domain>/` | Application code, one package per domain — see "Go architecture" |
| `internal/config` | The only package that reads the environment |
| `internal/api` | `app.go` builds every domain (the shared composition root); `router/` mounts middleware and routes |
| `internal/worker` | The River client for `cmd/worker` and the registration of every job's worker |
| `pkg/` | Infrastructure with no knowledge of this product: `postgres`, `httpx`, `clock`, `middleware`, `redis`, `fhir` (generic FHIR primitives: `Reference`, `CodeableConcept`, `OperationOutcome`, bundles). Never imports `internal/` |
| `api/` | Contracts: `openapi.yaml` for non-FHIR endpoints, `fhir/` for the CapabilityStatement, profiles (StructureDefinitions) and ValueSets we use. The TypeScript client (openapi-typescript) and the Dart client (openapi-generator) are generated from `openapi.yaml` into `web/` and `mobile/` with `make gen-clients`; never edit them by hand |
| `configs/` | Config templates and defaults (`.env.example`) |
| `deployments/` | `docker-compose.yml` (the stack, plus the `tools`, `node`, `landing-dev` and `flutter` toolchain services behind profiles), and later the orchestration manifests |
| `build/` | Dockerfiles (`build/package/`): `dev.Dockerfile` (the Go toolchain behind every `make` target), `flutter.Dockerfile`, `landing.Dockerfile` (builds `landing/` and serves it with nginx), and the runtime image |
| `.github/` | GitHub Actions workflows, one per app (Go, web, landing, mobile) with path filters, and Dependabot |
| `migrations/` | goose SQL migrations, numbered, forward-only (no `-- +goose Down`), embedded and applied by `cmd/migrate`. River's schema is migrated here too. A file on `main` is never modified or deleted (`scripts/check-migrations.sh`) |
| `scripts/` | Scripts the `Makefile` calls. Keep the `Makefile` small |
| `test/` | End-to-end tests and their fixtures (`test/e2e/`, `test/testdata/`) |
| `tools/` | Repo tooling, its own module (`tools/go.mod`): the pinned Go tools and `tools/archcheck` |
| `docs/` | `architecture/`, ADRs (`docs/adr/NNNN-title.md`), and one folder per feature (see "Working on a feature") |
| `web/` | React + TypeScript (Vite) app for practitioners and admins |
| `mobile/` | Flutter app (iOS and Android) for patients only |
| `landing/` | Marketing landing page: React + TypeScript (Vite), its own app beside `web/`, prerendered to static HTML at build time (`scripts/prerender.ts`) and served with its `nginx.conf` by the `landing` compose service. Copy and data in `src/content/`, one folder per page section in `src/sections/`. No third-party requests (fonts are bundled from npm). Not deployed yet: fill in the `[PRICE]`, `[X]`, `[YOUR EMAIL]`, `[Hosting]` and `[Certification]` placeholders in `src/content/` before it is |

Never: `/src`, `internal/utils`, `internal/common`, `internal/shared`, `internal/helpers`, or top-level
technical-layer packages (`internal/services`, `internal/repositories`, `internal/handlers`).

## Go architecture

Clean architecture by domain, not by layer. A domain is one package at `internal/<domain>`, with one file per
layer. The bounded contexts are **lists of packages, not directories**; the tree stays flat
(`docs/architecture/archcheck/config.yaml` is the authority).

Name a domain after the FHIR resource it owns, lowercase: `patient`, `appointment`, `encounter`.

```
identity      auth (accounts, sessions, MFA)
directory     practitioner, organization, practitionerrole
patient       patient, relatedperson
scheduling    schedule, slot, appointment
consultation  encounter, videosession, communication
clinical      observation, condition, medicationrequest, documentreference
privacy       consent, auditevent, datarequest (GDPR subject requests), retention
              and the pure rule package access (who may see what)
notification  notification
```

Start with the fewest domains a feature needs. Add a domain when a SPEC asks for it, and add it to the lists
in the same commit.

**A domain is one package, a file per layer:**

| File | Holds | Must not |
| --- | --- | --- |
| `model.go` | Entities and value types; may carry `json`/`db` tags | hold behaviour that needs I/O |
| `errors.go` | Stable domain errors: `Err…` values and `…Error` types | leak vendor errors |
| `service.go`, `service_*.go` | `Service`, the ports it consumes, every business rule, authorization, transaction boundaries | import pgx or `net/http`; call `time.Now` (take a `clock.Clock`) |
| `repository.go` | The `Repository` interface and other outbound ports | declare a struct |
| `repository_postgres.go`, `repository_<impl>.go` | Implementations: Postgres (through the domain's sqlc package), Redis, an HTTP client, the video provider (LiveKit), the e-mail sender (SES, Mailpit locally) | import `net/http` handlers or middleware; export anything but its constructor; hold business policy |
| `query.sql` → `<domain>db/` | sqlc input and its generated package (`make sqlc`, one entry per domain in the root `sqlc.yaml`), e.g. `internal/patient/patientdb`. `repository_postgres.go` maps its rows to the domain model | be imported by anything but this domain's `repository_postgres.go` (archcheck); be edited by hand |
| `handler_http.go`, `handler_fhir.go` | Decode, call the service, encode; map domain errors to status codes / `OperationOutcome` | import pgx; hold SQL; make business decisions |

Dependency direction: handler → service → repository port ← repository implementation. Across domains:
service → service, never service → another domain's repository.

**Rules:**
- A port is declared by the package that consumes it. When a sibling service already has a method of that
  shape, wire the concrete service in rather than writing an adapter.
- Objects are built only in `cmd/*/main.go`, `internal/api/app.go` and `internal/worker` (the River client
  and its job registrations). No DI framework, no package-level mutable state, no `init()` side effects.
- The environment is read only in `internal/config`. Clocks and HTTP clients are injected;
  `http.DefaultClient` and `http.Get` are not used.
- Middleware is mounted only in `internal/api/router`; routes are declared only there and in each domain's
  `handler_*.go` registration function.
- Every table has one owning domain (`docs/architecture/archcheck/ownership.yaml`). Only that domain's
  repository reads or writes it; others call its service. A new table goes in the ownership file in the same
  commit as its migration.
- Transactions: a service calls `WithTx` on its `postgres.Transactor`. The transaction travels in the
  context; a repository reaches it through `postgres.Conn(ctx, r.db)`. A nested `WithTx` joins the outer one.
  Collect rows before the next query. Calls to the video provider, email and blob writes stay outside the
  transaction where they can.
- Background work is a River job, enqueued in the same transaction as the write that causes it (River's
  `InsertTx`). Job arguments carry opaque ids only, never personal data. The worker loads what it needs.
- Redis holds temporary state only: slot holds, rate limits, idempotency keys, session revocation, caches,
  pub/sub. Every key has a TTL. Nothing in Redis is the only copy of anything, and nothing durable (jobs,
  audit, clinical data) goes there.
- Video goes through a provider port in `consultation`, implemented for self-hosted LiveKit. Speech-to-text
  for live transcription sits behind its own port in `cmd/agent`, so the engine can be swapped.
- E-mail goes through a port in `notification`: SES in production (EU region), Mailpit locally.
- A 500 goes through `httpx.InternalError`, which logs the error and sends a fixed message; internal error
  text never reaches a client.
- One model per domain. Introduce a separate transport type only when representations really diverge (the
  FHIR JSON form does — map it in `handler_fhir.go`) or a field must not cross a boundary.
- Functions under 80 lines and cognitive complexity 30 (enforced). Files under about 400 lines.
- Structs laid out without padding (`govet` fieldalignment).

## TDD

Red, green, refactor — in that order, every time.

1. Write the smallest failing test that states the next behaviour. Run it and see it fail for the right reason.
2. Write the least code that makes it pass.
3. Refactor with the tests green. Simplify; do not add features here.

- A bug fix starts with a test that reproduces the bug.
- Tests follow the file they cover: `service_test.go` next to `service.go`, `repository_postgres_test.go`
  next to `repository_postgres.go`.
- **Service tests are mandatory** for every business rule: hand-written fakes for the ports (prefer fakes to
  mocks), `clock.Fixed` for time, table-driven where cases share a shape. Test behaviour through the exported
  `Service`; do not export anything to make a test easier.
- Repository tests run against a throwaway database (`pgtest.NewDatabase`, built from `migrations/`), never
  mocked SQL. Redis code is tested against a real Redis (`redistest`), never a fake.
- Handler tests use `httptest` and a stub service. FHIR handlers assert the exact JSON against golden files in
  `testdata/`; change them only on purpose with `-update` and review the diff.
- **End-to-end tests** in `test/e2e/` cover the critical journeys through the real stack: register → consent
  → book an appointment → attend the consultation → practitioner records an observation → patient exports
  their record → erasure request. Keep the set small and make it reliable; a flaky e2e test is fixed or
  deleted the same day.
- Test data is synthetic. Never real patient data in a test, fixture, seed or screenshot.
- Never weaken, skip or delete a test to make the build pass.

## FHIR

- **FHIR R4** (4.0.1). Resource names, field names and value sets come from the spec; do not invent a name
  for something FHIR already names.
- The domain model is FHIR-shaped, not a full FHIR implementation. Model only the elements a SPEC needs;
  everything else is absent, not stubbed.
- The FHIR REST API is served under `/fhir/R4/`, `Content-Type: application/fhir+json`, and supports only
  the interactions listed in `api/fhir/CapabilityStatement.json`. Adding an interaction updates that file in
  the same commit.
- Errors on FHIR endpoints are `OperationOutcome` resources with the right HTTP status.
- Resource ids are opaque UUIDs, never derived from personal data (no national ID, MRN or e-mail in an id or
  URL).
- References are `ResourceType/id`. Cross-domain references are resolved through the owning service.
- Coded values use standard terminologies — SNOMED CT, LOINC, ICD-10, UCUM for units — with a `system` URI on
  every `Coding`. No free-text codes.
- Every resource carries `meta.lastUpdated` and `meta.versionId`; updates use optimistic locking
  (`If-Match`).
- Who did what is recorded as `Provenance` for clinical writes and `AuditEvent` for every access (see GDPR).
- Profiles we constrain live in `api/fhir/`. CI validates the golden FHIR responses against them with the
  HL7 validator.
- Non-FHIR concerns (login, video tokens, notification settings) use the plain JSON API in
  `api/openapi.yaml`. Do not force them into FHIR.

## GDPR and health data

Health data is special-category data (GDPR Art. 9). Treat every patient field as sensitive.

- **Lawful basis and consent.** Processing rests on a recorded basis; where that is consent, it is an
  explicit, granular FHIR `Consent`, checked by the service before the data is used or shared, and revocable.
  Revocation takes effect immediately.
- **Data minimisation.** Collect and return only what the use case needs. A new personal-data field needs a
  line in its feature SPEC saying why.
- **Purpose limitation.** Data collected for care is not used for analytics, training or marketing without a
  separate basis.
- **Access control in the service.** Authorization decisions live in the service (`access` holds the pure
  rules): a practitioner sees a patient's data only through a care relationship or an active appointment.
  Break-glass access is allowed, flagged, and audited.
- **Audit every access.** Every read or write of a patient's data writes an `AuditEvent` (who, which resource,
  when, purpose) in the same transaction. Audit records are append-only.
- **Nothing sensitive in logs, metrics or traces.** A log line carries a request id, resource type, opaque
  ids, status and timings only: never names, contact details, birth dates, national IDs, clinical values,
  notes, message text or video metadata. The same holds for error messages and panics.
- **Encryption.** TLS everywhere, including inside the cluster. Postgres and the object store are encrypted
  at rest. Attachments and consultation recordings (if any are kept) are encrypted with keys from the KMS,
  never from config files.
- **Subject rights are features, built and tested like any other:** access and portability (export as a FHIR
  `Bundle`, `Patient/$everything`), rectification, restriction, objection, and erasure. Erasure is limited by
  medical-record retention law: the `retention` domain decides per resource what is deleted, what is
  restricted, and when. Requests are tracked in `datarequest` with their legal deadline.
- **Retention.** Every table with personal data has a retention rule in `retention`; a scheduled job applies
  it. No table without one.
- **Residency and processors.** Data stays in the EU/EEA (or the jurisdiction the SPEC names). Every third
  party that touches personal data (video, e-mail, SMS, hosting, any AI service) needs a DPA and an entry in
  `docs/privacy/processors.md` before it is wired in. No patient data is sent to a third-party AI model
  unless the SPEC and the DPA explicitly allow it.
- **Pseudonymise** data used outside direct care (reports, debugging copies, staging). Production data never
  reaches a developer machine or staging.
- A new feature touching personal data adds or updates the DPIA notes in `docs/privacy/`.
- Breach handling (72-hour notification) is documented in `docs/privacy/breach.md`.

## Working on a feature

Each feature gets a folder in `docs/<feature>/`:

- `SPEC.md` — the single source of truth, including its settled decisions and the personal data it touches.
- `IMPLEMENTATION_PLAN.md` — numbered work packages, each small enough for one session.
- `PROGRESS.md` — the running record, updated at the end of every package.

- `RUN.tsv` — the run manifest: `# prefix:`, `# title:`, optional `# epic:`, `# base:`, `# stack: yes`, then one
  row per package: `package<TAB>tickets<TAB>title[<TAB>hold]`.

Work one package at a time, in order. Build to the SPEC's decisions; do not reopen a settled decision. If the
plan conflicts with the code, stop and report rather than improvise. Leave the repository building and the
tests green at the end of every package. Measure before claiming a performance gain.

- Interactive: `/run-package docs/<feature> 1.2` runs one package with the `package-runner` agent.
- Unattended: `bash scripts/run-feature.sh docs/<feature> [--stacked]` runs every package, one headless session
  and one commit each, then opens the PR(s). Jira tickets are fetched when `JIRA_EMAIL`, `JIRA_API_TOKEN` and
  `JIRA_BASE_URL` are set.
- A stacked series is merged with the `merge-stack` skill.

## Claude Code setup

| Path | What it is |
| --- | --- |
| `.claude/settings.json` | Stop hook: runs `make arch-gate` (`go vet`, tests, archcheck, all in Docker) once the Makefile has that target |
| `.claude/agents/package-runner.md` | Implements and commits exactly one work package |
| `.claude/commands/run-package.md` | `/run-package <plan> <package>` |
| `.claude/skills/go-domain/` | The test-first loop inside a domain: layers, FHIR handlers, consent and audit |
| `.claude/skills/database-migrations/` | Migration conventions, ownership and retention for personal data |
| `.claude/skills/merge-stack/` | Merge a stacked PR series bottom-up |
| `scripts/run-feature.sh` | Runs a plan end to end |

## Checks

Every check runs in Docker through `make`, locally, in CI and in Claude Code sessions alike: never run `go`,
`npm`, `npx` or `flutter` on the host. To run something there is no target for, use
`docker compose -f deployments/docker-compose.yml run --rm tools <command>` (or the `node` / `flutter`
service).

- `make test` — `go test ./...` in the `tools` container. Its `TEST_DATABASE_URL` and `TEST_REDIS_URL` point
  at the compose Postgres and Redis; `pgtest` migrates a template database once per test package and gives
  each test its own copy. Without the URLs the tests fail with a message; they never skip.
- `make cover` — tests with coverage, each package measured by its own tests, failing when any package is
  below 80%. Only `cmd/*/main.go` is excluded.
- `make e2e` — the end-to-end suite against `docker compose`.
- `make lint` — gofmt, `go vet`, golangci-lint (depguard, forbidigo, funlen, gocognit, govet fieldalignment,
  errcheck, staticcheck, bodyclose, noctx, unused) over `cmd/ internal/ pkg/`, and
  `scripts/check-migrations.sh` (no migration that exists on `main` was modified or deleted).
- `make archcheck` — `tools/archcheck`: `pkg/` never imports `internal/`; no domain imports another's
  repository; table ownership is respected; forbidden package names are rejected. Its
  `allowlist.yaml` **only shrinks**: never add an entry to make a new violation pass.
- `make fhir-validate` — the HL7 validator (pinned jar, `fhir-validator` service) over every golden FHIR
  response in `testdata/`, against R4 and the `api/fhir/` profiles once there are any. No terminology server.
- `make arch-gate` — archcheck, `go vet` and the tests; the Claude Code Stop hook runs it at the end of every
  session.
- `make migrate`, `make sqlc`, `make gen-clients`, `make up`, `make down` — apply migrations, regenerate
  queries, regenerate the API clients, start and stop the local stack.
- Web (`web/`, `node` service): `make web-lint` (oxlint), `make web-typecheck`, `make web-test` (Vitest),
  `make web-build`; `make web-dev` runs the Vite dev server on port 5173. `make gen-clients` regenerates the
  API clients; CI fails when they are stale. Mobile (`mobile/`, `flutter` service): `make mobile-analyze`,
  `make mobile-test`; the Dart client in `mobile/packages/salus_api` is generated (`make gen-clients`) and
  excluded from analysis. Running the app on a device or emulator is a manual step on the host (README.md), not
  a check. Landing (`landing/`, `node` service): `make landing-lint`, `make landing-typecheck`,
  `make landing-test`, `make landing-build` (prerenders into `landing/dist`); `make landing-dev` runs its Vite
  dev server on port 5174, and `make landing` serves the built image on port 8090.
- Before saying a change is done: `make lint` (gofmt, `go vet`, golangci-lint), `make test` and
  `make archcheck` all pass.
- Never run `git stash`; use `git worktree add` to compare with and without a change.

## Copy

Plain and professional: no hype, no exclamation marks, no emoji. Sentence case for headings and buttons.
Medical wording in demo or placeholder content stays labelled as a placeholder, and nothing in the product
presents itself as a diagnosis unless a practitioner made it.
