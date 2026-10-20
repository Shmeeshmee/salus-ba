---
name: database-migrations
description: 'Use when adding or changing a Postgres migration in migrations/: the next numbered goose NNNNN_name.sql file, conventions (UUID keys, timestamps, snake_case, column order), table ownership and retention for personal data, idempotent and forward-only DDL, and applying it with make migrate.'
argument-hint: 'Add a numbered SQL migration and apply it'
---

# Database migrations

## How migrations run

- goose SQL files: `migrations/NNNNN_short_name.sql` (five digits), numbered, forward-only. Each starts with
  `-- +goose Up`; there is no `-- +goose Down`. Each file runs once, in its own transaction, with its row in
  goose's `goose_db_version` ledger. Statements that cannot run in a transaction (`CREATE INDEX
  CONCURRENTLY`) go in their own file with `-- +goose NO TRANSACTION`.
- The files are embedded (`migrations.FS`) and applied by `cmd/migrate up`; `make migrate-create name=...`
  creates the next file.
- A file that exists on `main` is immutable: add a new file instead of editing, renaming or deleting an old
  one. goose keeps no checksums, so `scripts/check-migrations.sh` (in `make lint` and CI) fails when a
  migration that exists on `main` was modified or deleted.
- River's schema is migrated through goose too: upgrading River adds the migration it ships as a new file.
- The Go tests build their database from `migrations/` (`pgtest.NewDatabase` migrates a template database
  once and clones it per test), so a new file reaches every repository test automatically.

## Conventions

- Next free number, short name, and a header comment saying what the file does. Check both the folder and the
  ledger (`make migrate-status`) for the next number.
- A table that a domain queries gets its queries in `internal/<domain>/query.sql`; run `make sqlc` after the
  migration so the generated `<domain>db` package matches the schema.
- Primary keys: `id uuid DEFAULT gen_random_uuid() NOT NULL`. FHIR ids are these UUIDs; never derive a key or
  id from personal data.
- Every table has `created_at timestamptz DEFAULT now() NOT NULL`; mutable tables also have `updated_at` and
  `version_id integer NOT NULL DEFAULT 1` for optimistic locking (`meta.versionId`).
- `snake_case`, `jsonb` only for genuinely open structures (FHIR extensions), enums or check constraints for
  closed sets.
- **Column order:** alignment order — `id`, then 8-byte types (`timestamptz`, `bigint`), other `uuid`
  columns, 4-byte types (`integer`, `date`, enums), `smallint`, `boolean`, then variable length (`text`,
  `jsonb`, arrays).
- Index what you filter and join on.
- Coded values are stored as `system` + `code` (+ `display`), never a bare code.

## Personal data

- Every new table goes in `docs/architecture/archcheck/ownership.yaml` (one owning domain) in the same commit.
- Every table holding personal data gets a retention rule in the `retention` domain in the same feature, and
  the column list in the feature's `SPEC.md` says why each personal field is needed.
- Audit tables (`audit_event`) are append-only: no `UPDATE` or `DELETE` grants for the application role.
- Erasure and restriction are modelled up front: a table that can be erased says how (delete, or null the
  personal columns and keep the clinical record the retention law requires).
- Seeds and backfills use synthetic data only.

## Safety

- Prefer additive changes. Destructive changes (drops, type narrowing) need a SPEC decision.
- A backfill is part of the same file, idempotent, and does not invent data.
- Preview anything that rewrites stored values with a `SELECT` first and record counts in the plan's
  `PROGRESS.md`.
- Ship the schema before code that reads a new column.

## Default checks

- `make migrate` applies the file; `make migrate-status` shows nothing pending.
- `scripts/check-migrations.sh` passes (no existing migration changed).
- `make sqlc` leaves no diff.
- `make test` passes with `TEST_DATABASE_URL` set.
- `make archcheck` passes (ownership covers the new table).
