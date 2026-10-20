---
name: go-domain
description: 'Use when adding or changing a Salus domain, a FHIR or JSON endpoint, a business rule or a query in internal/<domain>: the test-first order, the file per layer, FHIR handlers and OperationOutcome, consent and AuditEvent checks, WithTx transactions, clock injection, error mapping, and the tests per layer.'
argument-hint: 'Add or change a domain, an endpoint, a rule or a query'
---

# Go domain

How to build inside one domain package. CLAUDE.md ("Top rules", "Go architecture", "TDD", "FHIR", "GDPR and
health data") is the authority; this is the working loop.

## Before you start

1. Find the owning domain. A behaviour belongs to the domain that owns its FHIR resource and its table
   (`docs/architecture/archcheck/ownership.yaml`). Prefer an existing domain over a new one.
2. Read the feature's `SPEC.md` for the contract and the personal data it touches.
3. Read the package you are changing and reuse its patterns (`writeError`, filter structs, fakes) before
   inventing new ones.

## The loop, test first

Each step starts with a failing test.

1. **Migration**, if the schema changes (see the `database-migrations` skill). Add the table to
   `ownership.yaml` and give it a retention rule.
2. **Model** in `model.go`: only the FHIR elements the SPEC needs.
3. **Service rule**: a `service_test.go` case with a hand-written fake of the port and `clock.Fixed`, then the
   code in `service.go`. The test names the behaviour: `TestBook_RejectsSlotAlreadyTaken`.
4. **Repository**: a `repository_postgres_test.go` case against `pgtest.NewDatabase`, then the query in
   `repository_postgres.go`.
5. **Handler**: a `handler_*_test.go` case with `httptest` and a stub service, covering the success path and
   400, 403, 404, 409. For a FHIR endpoint compare against a golden file in `testdata/`.
6. **Wiring** in `internal/api/app.go` and the route in the domain's registration function.
7. Refactor with everything green. Then run the checks.

## Rules

Layers
- `service.go` never imports pgx or `net/http` and never calls `time.Now`; it takes a `clock.Clock`.
- Handlers decode, call one service method, encode. No SQL, no business decisions.
- A port is declared by the package that consumes it. Another domain's data comes through its service, never
  its repository or its tables.
- Objects are built only in `cmd/*/main.go` and `internal/api/app.go`. No package-level state.

Patient data
- Every service method that reads or writes a patient's data checks the access rule (`access`) and, where the
  basis is consent, the active `Consent`, before doing anything else.
- It writes an `AuditEvent` (who, which resource, purpose, outcome) through the `auditevent` service in the
  same transaction as the read or write. A denied access is audited too.
- No patient data in a log line, metric, trace, error message, panic or test fixture. Log opaque ids only.
- Return only the elements the caller needs and is allowed to see.

FHIR
- Endpoints under `/fhir/R4/<ResourceType>`, `application/fhir+json`. Every new interaction is listed in
  `api/fhir/CapabilityStatement.json` in the same commit.
- The FHIR JSON form is mapped in `handler_fhir.go`; the domain model stays plain Go.
- Errors are an `OperationOutcome` with the right status: 400 invalid, 403 forbidden, 404 not found,
  409/412 version conflict (`If-Match` against `meta.versionId`), 422 business rule.
- Ids are opaque UUIDs. Every `Coding` has a `system`.

Data
- Multi-statement changes run in `WithTx` on the service's `postgres.Transactor`; repositories reach the
  transaction with `postgres.Conn(ctx, r.db)`. Collect rows before the next query. Calls to the video
  provider, e-mail and blob writes stay outside the transaction where they can.
- Optimistic concurrency: `UPDATE ... WHERE id = $1 AND version_id = $2`; zero rows means 412 with the current
  version.
- Instants are `timestamptz` and travel with their zone; a zone-less date-time is a 400.

Errors
- Domain errors live in `errors.go`. A 500 goes through `httpx.InternalError`; internal error text never
  reaches a client.

Size
- Functions under 80 lines and cognitive complexity 30. Files under about 400 lines: split by concern
  (`service_reschedule.go`), not by size alone.

## What good looks like

- The rule reads in the service without knowing SQL or HTTP.
- The handler is three steps: decode, call, encode.
- Every patient-data path has a test proving the access check and the `AuditEvent`.
- The golden files change only where the change meant them to.

## Default checks

- `make lint`
- `make test` (in Docker, against the compose Postgres and Redis)
- `make archcheck`
- `make fhir-validate` when a FHIR response changed
- `-update` for golden files only on purpose (`docker compose -f deployments/docker-compose.yml run --rm tools
  go test ./<pkg> -update`), then read the diff
