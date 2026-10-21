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
| `make up` | Build and start the local stack: migrations, then the API, worker, scheduler, notifier, agent, LiveKit, Mailpit and the landing page |
| `make landing` | Build and serve only the landing page image (`landing/`) on 127.0.0.1:8090 |
| `make test` | Go tests (both modules) against the compose Postgres and Redis; `TESTFLAGS="-run X -count=1"` narrows |
| `make lint` | gofmt, go vet, golangci-lint, and the check that no migration on main changed |
| `make archcheck` | Architecture rules (`tools/archcheck`, config in `docs/architecture/archcheck/`) |
| `make arch-gate` | archcheck, go vet and the tests (the Claude Code Stop hook runs it) |
| `make cover` | Tests with coverage; fails below 80% |
| `make e2e` | Starts the stack and runs `test/e2e` against it |
| `make fhir-validate` | HL7 validator over the golden FHIR responses |
| `make migrate`, `make migrate-status`, `make migrate-create name=add_x` | goose migrations on the compose Postgres |
| `make sqlc` | Regenerate the domains' query packages |
| `make gen-clients` | Regenerate the API clients from `api/openapi.yaml` |
| `make web-lint`, `web-typecheck`, `web-test`, `web-build` | The web app's checks (`web/`, `node` service) |
| `make web-dev` | Vite dev server on 127.0.0.1:5173, proxying the API (run `make up` first) |
| `make landing-lint`, `landing-typecheck`, `landing-test`, `landing-build` | The landing page's checks and its prerendering build (`landing/`, `node` service) |
| `make landing-dev` | The landing page's Vite dev server on 127.0.0.1:5174 |
| `make mobile-analyze`, `make mobile-test` | The Flutter app's checks (`mobile/`, `flutter` service) |
| `make psql` | psql on the compose Postgres (no port is published to the host) |
| `make down` | Stop the containers (volumes and caches are kept) |

Local stack ports, all on 127.0.0.1: API 8080 (`/healthz`, `/readyz`, `/fhir/R4/metadata`), Mailpit web UI
8025, LiveKit 7880 (dev mode, key `devkey`, secret `secret`), Vite 5173, landing page 8090, landing Vite 5174. If another project holds one, set
the matching `SALUS_*_PORT` from `configs/.env.example` in `deployments/.env` (git-ignored). Postgres and
Redis publish no port; use `make psql`.

To run a command no target covers: `docker compose -f deployments/docker-compose.yml run --rm tools <command>`.

## Mobile on a device or emulator

Docker covers `flutter analyze`, `flutter test` and code generation only. To run the app on a device or an
emulator, install the Flutter SDK at the version in `build/package/flutter.Dockerfile` on your machine, start
the stack with `make up`, and in `mobile/`:

    flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080   # Android emulator (10.0.2.2 is the host)
    flutter run --dart-define=API_BASE_URL=http://localhost:8080  # iOS simulator

Use your `SALUS_API_PORT` if you moved it. Android debug builds allow plain HTTP to the local API; on iOS,
add an App Transport Security exception for local networking in your local `Info.plist` and do not commit it.
This is a manual step and not part of any check.
