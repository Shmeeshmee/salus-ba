# Salus

Telemedicine platform: patients book and attend remote consultations with practitioners; the clinical
record is kept as FHIR R4 resources. Go API and workers, Postgres, Redis, LiveKit, a React web app for
practitioners and admins, and a Flutter app for patients. Rules for working in this repository: `CLAUDE.md`.

## Requirements

Docker (with the compose plugin) and make. Nothing else runs on the host: every target below runs its
toolchain in a container, with versions pinned in `build/package/`, `deployments/docker-compose.yml` and
`go.mod`. Module and build caches live in named Docker volumes.

## Common commands

| Command | What it does |
| --- | --- |
| `make test` | Go tests against the compose Postgres and Redis |
| `make down` | Stop the containers (volumes and caches are kept) |

More targets arrive with the bootstrap plan (`docs/bootstrap/IMPLEMENTATION_PLAN.md`): lint, archcheck,
migrations, the local stack, the web and mobile checks.

To run a command no target covers: `docker compose -f deployments/docker-compose.yml run --rm tools <command>`.

## Mobile on a device or emulator

Docker covers `flutter analyze`, `flutter test` and code generation only. To run the app on a device or an
emulator, install the Flutter SDK at the version in `build/package/flutter.Dockerfile` on your machine and
run `flutter run` in `mobile/`. This is a manual step and not part of any check.
