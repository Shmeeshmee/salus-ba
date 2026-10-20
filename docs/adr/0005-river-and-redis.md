# 5. River for jobs, Redis for temporary state

Status: accepted, 2026-10-19

## Context

Writes often cause later work (send a reminder, apply retention, notify a practitioner). If the job is queued
outside the write's transaction, a crash loses it or runs it for a write that rolled back. Separately, slot
holds, rate limits and idempotency keys need fast shared state that may expire.

## Decision

Background jobs use River, which stores jobs in Postgres; a job is enqueued with `InsertTx` in the same
transaction as the write that caused it. Redis is used only for temporary state: slot holds, rate limiting,
idempotency keys, session revocation, caches and pub/sub. Every Redis key has a TTL. Redis never holds jobs or
the only copy of anything.

## Consequences

- No lost or phantom jobs; job state is visible with plain SQL.
- Job arguments carry opaque ids only, so no personal data sits in the job table; River's completed-job
  cleanup keeps it short-lived.
- Losing Redis loses holds and caches, never records; the system degrades, it does not corrupt.
- `/readyz` checks both Postgres and Redis.
