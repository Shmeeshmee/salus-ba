# 4. goose for migrations

Status: accepted, 2026-10-19

## Context

Every binary and every repository test needs the same schema. A hand-rolled migration runner is under 100
lines but still has to handle locking, ordering and status; goose does this and supports embedding.

## Decision

Migrations are goose SQL files in `migrations/`, numbered, forward-only (no `-- +goose Down`). They are
embedded with `embed.FS` and run by `cmd/migrate` (`up`, `status`, `create`). River's own schema is added as
goose migrations too, so one tool owns the whole schema. `pgtest.NewDatabase` applies the same files.

## Consequences

- One ledger (`goose_db_version`) for the whole schema, including River's tables.
- An applied file is immutable; a fix is a new file.
- Upgrading River means adding the migration it ships for that version.
- `cmd/migrate` runs before the other binaries in compose and in deployment.
