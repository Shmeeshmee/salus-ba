# 3. pgx v5 and sqlc for data access

Status: accepted, 2026-10-19

## Context

CLAUDE.md forbids an ORM and wants a reader to follow a request from handler to SQL. Hand-written row scanning
is repetitive and error prone; `database/sql` adds a layer pgx does not need and loses Postgres types.

## Decision

pgx v5 (`pgxpool`) is the driver. sqlc is the one allowed code generator for queries: each domain keeps its SQL
in `internal/<domain>/query.sql`, and sqlc generates `internal/<domain>/<domain>db/`, imported only by that
domain's `repository_postgres.go`. `pkg/postgres` provides `Transactor.WithTx` (nested calls join) and
`Conn(ctx, db)` so generated queries run inside the context's transaction.

## Consequences

- The SQL is plain and reviewable; the generated Go is type checked against the migrations.
- The generated package is the one exception to "a domain is one package"; archcheck enforces that nothing
  else imports it.
- `make sqlc` must be run after changing a query or a migration; CI fails if the output is stale.
- Generated files are never edited by hand.
