# Salus

Telemedicine platform: patients book and attend remote consultations with practitioners, and the
clinical record of those consultations is kept as FHIR resources. Go API + Postgres + object store.
Everything runs through `docker compose` (`deployments/`).

Module path: `github.com/Shmeeshmee/salus`.

## Top rules

These four beat everything else in this file. When two rules pull against each other, the earlier one wins
unless it would break the law: GDPR is never traded for simplicity.

1. **Keep the code simple.** The simplest thing that passes the tests is the right thing.
   - No abstraction without a second real use or a test seam that needs it. No interface with one
     implementation unless a test fakes it.
   - No DI framework, no code generators beyond what Go ships, no ORM, no event bus, no CQRS, no
     microservices until one process is demonstrably not enough.
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
| `cmd/<app>/main.go` | One executable each (`api`, `worker`, ...). Composition only: load config, connect, build, run, shut down |
| `internal/<domain>/` | Application code, one package per domain — see "Go architecture" |
| `internal/config` | The only package that reads the environment |
| `internal/api` | `app.go` builds every domain (the shared composition root); `router/` mounts middleware and routes |
| `pkg/` | Infrastructure with no knowledge of this product: `postgres`, `httpx`, `clock`, `middleware`, `fhir` (generic FHIR primitives: `Reference`, `CodeableConcept`, `OperationOutcome`, bundles). Never imports `internal/` |
| `api/` | Contracts: `openapi.yaml` for non-FHIR endpoints, `fhir/` for the CapabilityStatement, profiles (StructureDefinitions) and ValueSets we use |
| `configs/` | Config templates and defaults (`.env.example`) |
| `deployments/` | `docker-compose.yml`, and later the orchestration manifests |
| `build/` | Dockerfiles (`build/package/`) and CI config (`build/ci/`) |
| `migrations/` | SQL migrations, numbered, forward-only |
| `scripts/` | Scripts the `Makefile` calls. Keep the `Makefile` small |
| `test/` | End-to-end tests and their fixtures (`test/e2e/`, `test/testdata/`) |
| `tools/` | Repo tooling, e.g. `tools/archcheck` |
| `docs/` | `architecture/`, ADRs (`docs/adr/NNNN-title.md`), and one folder per feature (see "Working on a feature") |
| `web/` | Frontend, if and when there is one |

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
| `repository_postgres.go`, `repository_<impl>.go` | Implementations: Postgres, an HTTP client, the video provider | import chi or middleware; export anything but its constructor; hold business policy |
| `handler_http.go`, `handler_fhir.go` | Decode, call the service, encode; map domain errors to status codes / `OperationOutcome` | import pgx; hold SQL; make business decisions |

Dependency direction: handler → service → repository port ← repository implementation. Across domains:
service → service, never service → another domain's repository.

**Rules:**
- A port is declared by the package that consumes it. When a sibling service already has a method of that
  shape, wire the concrete service in rather than writing an adapter.
- Objects are built only in `cmd/*/main.go` and `internal/api/app.go`. No DI framework, no package-level
  mutable state, no `init()` side effects.
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
- Repository tests run against a throwaway database (`pgtest.NewDatabase`), never mocked SQL.
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
| `.claude/settings.json` | Stop hook: runs `make arch-gate` (lint, `go vet`, tests, archcheck) once the Makefile has that target |
| `.claude/agents/package-runner.md` | Implements and commits exactly one work package |
| `.claude/commands/run-package.md` | `/run-package <plan> <package>` |
| `.claude/skills/go-domain/` | The test-first loop inside a domain: layers, FHIR handlers, consent and audit |
| `.claude/skills/database-migrations/` | Migration conventions, ownership and retention for personal data |
| `.claude/skills/merge-stack/` | Merge a stacked PR series bottom-up |
| `scripts/run-feature.sh` | Runs a plan end to end |

## Checks

- `make test` — `go test ./...`. Database tests need `TEST_DATABASE_URL`, e.g.
  `postgres://salus:salus@localhost:5432/salus?sslmode=disable`.
- `make cover` — tests with coverage, failing below 80%.
- `make e2e` — the end-to-end suite against `docker compose`.
- `make lint` — gofmt, `go vet`, golangci-lint (depguard, forbidigo, funlen, gocognit, govet fieldalignment,
  errcheck, staticcheck, bodyclose, noctx, unused) over `cmd/ internal/ pkg/`.
- `make archcheck` — `tools/archcheck`: `pkg/` never imports `internal/`; no domain imports another's
  repository; table ownership is respected; forbidden package names are rejected. Its
  `allowlist.yaml` **only shrinks**: never add an entry to make a new violation pass.
- `make fhir-validate` — validates the golden FHIR responses against `api/fhir/` profiles.
- `make arch-gate` — archcheck, `go vet` and the tests; the Claude Code Stop hook runs it at the end of every
  session. Export `TEST_DATABASE_URL` so database tests run rather than skip.
- Before saying a change is done: `gofmt -l` is empty, `go vet ./...`, `go build ./...`, `make test`,
  `make lint`, `make archcheck` all pass.
- Never run `git stash`; use `git worktree add` to compare with and without a change.

## Copy

Plain and professional: no hype, no exclamation marks, no emoji. Sentence case for headings and buttons.
Medical wording in demo or placeholder content stays labelled as a placeholder, and nothing in the product
presents itself as a diagnosis unless a practitioner made it.
