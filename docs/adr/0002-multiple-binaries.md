# 2. Several binaries, one application

Status: accepted, 2026-10-19

## Context

The HTTP API, background jobs, periodic jobs, e-mail delivery and the LiveKit transcription agent have
different load and lifecycles. Running them in one process couples their scaling and failure; splitting them
into microservices would add network contracts, duplicated data and operational cost we do not need.

## Decision

One Go module, several executables in `cmd/`: `api`, `migrate`, `worker`, `scheduler`, `notifier`, `agent`.
Each `main.go` is composition only. They share `internal/` and one Postgres database; table ownership per
domain still applies. This is not a microservice architecture.

## Consequences

- Each binary scales and restarts on its own; one Docker image builds all of them.
- No inter-service API to version: binaries communicate through Postgres (River jobs) and Redis pub/sub.
- A new binary needs a reason (different load or lifecycle) and an entry in CLAUDE.md "Layout".
- Deployments must run `migrate` before the other binaries start.
