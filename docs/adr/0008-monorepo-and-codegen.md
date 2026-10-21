# 8. Monorepo and generated API clients

Status: accepted, 2026-10-19

## Context

The Go backend, the React web app (practitioners and admins) and the Flutter app (patients) change together:
an API change must reach both clients in the same review. Hand-written clients drift from the server.

## Decision

One GitHub repository holds `cmd/`, `internal/`, `pkg/`, `web/` and `mobile/`. `api/openapi.yaml` is the contract
for non-FHIR endpoints; the TypeScript client (openapi-typescript) and the Dart client (openapi-generator) are
generated from it with `make gen-clients` and committed. GitHub Actions runs one workflow per app with path
filters. Dependabot updates Go modules, npm, pub, Docker images and Actions.

## Consequences

- One PR can change the contract, the server and both clients.
- CI fails when generated clients are stale against `openapi.yaml`.
- Path filters keep CI fast: a web-only change does not run the Go suite.
- FHIR endpoints are not in `openapi.yaml`; they are described by `api/fhir/CapabilityStatement.json`.
