# Every target runs in Docker through compose; the host needs only Docker and make. Longer commands live in
# scripts/ and run inside the containers.

export HOST_UID := $(shell id -u)
export HOST_GID := $(shell id -g)

COMPOSE := docker compose -f deployments/docker-compose.yml
GO      := $(COMPOSE) run --rm tools
TOOL    := go tool -modfile=tools/go.mod
NODE    := $(COMPOSE) run --rm -T node
FLUTTER := $(COMPOSE) run --rm -T flutter
LANDING := $(COMPOSE) run --rm -T -w /src/landing node
E2E     := $(COMPOSE) run --rm -T playwright

# The services the Go tests reach (the tools service's depends_on), and the tests themselves.
TEST_SERVICES := postgres redis mailpit livekit objectstore clamd
GO_TEST        = go test $(TESTFLAGS) ./... && go -C tools test $(TESTFLAGS) ./...
GO_NODEPS     := $(COMPOSE) run --rm --no-deps tools

.PHONY: test cover bench-hash bench-auth bench-slots bench-stt bench-join bench-join-http bench-removal bench-questionnaire-http bench-binary-http bench-terminology bench-consult-stream turn-check livekit-turn-cert lint i18n-check archcheck arch-gate e2e e2e-install e2e-image fhir-validate migrate migrate-status migrate-create salusctl sqlc \
	terminology-fetch terminology-import terminology-export \
	gen-clients gen-client-ts gen-client-dart mobile-analyze mobile-test web-install web-lint web-typecheck web-test web-build web-dev \
	landing-install landing-lint landing-typecheck landing-test landing-build landing-dev flutter-image tools-image node-image up landing psql down

## test: Go tests of both modules against the compose Postgres and Redis (TESTFLAGS="-run X -count=1")
test: tools-image
	$(GO) sh -c '$(GO_TEST)'

## cover: tests with coverage, failing when any package is below 80% (cmd/*/main.go excluded)
cover: tools-image
	$(GO) bash scripts/cover.sh

## bench-hash: the argon2id parameter grid and the parallel throughput that size auth's password hashing
## (docs/accounts/PROGRESS.md, package 2)
bench-hash: tools-image
	$(GO) go test -run '^$$' -bench 'BenchmarkHash' -benchtime=20x -cpu 1,2,4 ./internal/auth

## bench-auth: the per-request session check, and a login burst against a pool of 4 and 16 connections
## (docs/accounts/PROGRESS.md, package 3)
bench-auth: tools-image
	$(GO) sh -c "go test -run '^$$' -bench 'BenchmarkAuthenticate' -benchtime=3s ./internal/auth && \
		go test -run '^$$' -bench 'BenchmarkLoginBurst' -benchtime=5x ./internal/auth"

## bench-slots: slot generation for 50 schedules, 28 days ahead, first run and re-run
## (docs/booking/PROGRESS.md, package 5)
bench-slots: tools-image
	$(GO) go test -run '^$$' -bench 'BenchmarkGenerate' -benchtime=5x -count=3 ./internal/worker

## bench-stt: the partial caption's latency, frame in to partial event read back from the compose Redis, with the
## fake engine and the Azure adapter against its test server, and the LiveKit hop to the agent's decoded frame,
## as p50/p95/p99 (docs/plans/consult-intelligence.PROGRESS.md, SAL-4)
bench-stt: tools-image
	$(GO) sh -c "go test -run '^$$' -bench 'PartialLatency' -benchtime=1000x -count=3 ./internal/transcript && \
		go test -run '^$$' -bench 'FrameLatency' -benchtime=100x -count=3 ./internal/videosession"

## bench-join: the consultation's token request through the service on the real database, with the compose
## LiveKit and with a fake provider (docs/consultation/PROGRESS.md, package 1)
bench-join: tools-image
	$(GO) go test -run '^$$' -bench 'BenchmarkJoin' -benchtime=2s -count=3 ./internal/api

## bench-join-http: the join endpoint's latency through the running stack, 1,000 requests 10 at a time on the
## compose Postgres and LiveKit, as p50/p95/p99 (docs/consultation/PROGRESS.md, package 4). Not part of make e2e.
bench-join-http: up tools-image
	$(COMPOSE) run --rm -e E2E_BASE_URL=http://api:8080 -e E2E_MAILPIT_URL=http://mailpit:8025 -e E2E_SALUSCTL=/tmp/salusctl \
		tools sh -c 'go build -o /tmp/salusctl ./cmd/salusctl && go test -tags e2e,loadtest -count=1 -v -run TestJoinLatency ./test/e2e/'

## bench-removal: the time from a revocation's commit to the patient's removal from the call, through the running
## stack (api, worker, LiveKit), 3 runs each for a started and a not-started consultation
## (docs/consultation/PROGRESS.md, package 5). Not part of make e2e.
bench-removal: up tools-image
	$(COMPOSE) run --rm -e E2E_BASE_URL=http://api:8080 -e E2E_MAILPIT_URL=http://mailpit:8025 -e E2E_SALUSCTL=/tmp/salusctl \
		tools sh -c 'go build -o /tmp/salusctl ./cmd/salusctl && go test -tags e2e,loadtest -count=1 -v -run TestRemovalLatency ./test/e2e/'

## bench-questionnaire-http: the questionnaire endpoints' latency through the running stack: 1,000 saves of a 200-item
## questionnaire's answers and 1,000 QuestionnaireResponse searches with the version included, 10 at a time, as
## p50/p95/p99 (docs/questionnaire/PROGRESS.md, package 6). Not part of make e2e.
bench-questionnaire-http: up tools-image
	$(COMPOSE) run --rm -e E2E_BASE_URL=http://api:8080 -e E2E_MAILPIT_URL=http://mailpit:8025 -e E2E_SALUSCTL=/tmp/salusctl \
		tools sh -c 'go build -o /tmp/salusctl ./cmd/salusctl && go test -tags e2e,loadtest -count=1 -v -run TestQuestionnaireLatency ./test/e2e/'

## bench-binary-http: a 200 MB generated MP4 uploaded through the api's HTTP server on the compose Postgres and
## object store (MB/s and the peak heap), then 1,000 random 1 MiB ranges of it as p50/p95/p99
## (docs/attachments/PROGRESS.md, package 2.1). Not part of make test.
bench-binary-http: tools-image
	$(GO) go test -tags loadtest -count=1 -v -run TestBinaryHTTPReadings ./internal/api/

## bench-consult-stream: the WebSocket gateway's XADD-to-frame latency and a reconnect's replay through api.New, and
## one stream-discovery sweep over 100,000 keys (docs/plans/consult-intelligence.PROGRESS.md, SAL-6). Not part of make test.
bench-consult-stream: tools-image
	$(GO) sh -c "go test -tags loadtest -count=1 -v -run TestTranscriptStreamLatency ./internal/api/ && \
		go test -tags loadtest -count=1 -v -run TestGroupDiscoveryCost ./pkg/redis/"

## bench-terminology: the terminology search through the service on a database loaded with the seed, 50 searches at
## a time, with its p95 (docs/terminology/SPEC.md §3 "Search": under 15 ms; PROGRESS.md, package 1.1)
bench-terminology: tools-image
	$(GO) go test -run '^$$' -bench 'BenchmarkSearch' -benchtime=2000x -count=3 ./internal/terminology

## turn-check: a call through the compose LiveKit with UDP and its ICE/TCP port blocked: it does not connect
## without TURN, and does through TURN over TLS on 443 (docs/consultation/PROGRESS.md, package 10). Not part of make test.
turn-check: tools-image
	$(COMPOSE) run --rm -T turn-check

## livekit-turn-cert: write a new local TURN/TLS certificate for the compose LiveKit into deployments/livekit-turn/
livekit-turn-cert: tools-image
	$(GO) bash scripts/livekit-turn-cert.sh

## lint: gofmt, go vet, golangci-lint (both modules), the migration immutability check, and the catalog check
## (the e-mail templates)
lint: tools-image i18n-check
	$(GO) bash scripts/lint.sh

## i18n-check: every catalog (web, mobile, e-mail) has the source language's keys and no other, valid ICU
## messages with the source's placeholders, and the plural categories each language needs (scripts/i18n-check.sh,
## docs/i18n/SPEC.md §3 "Catalogs"). In the node service, with web/ installed for the ICU parser.
i18n-check: web-install
	$(NODE) bash ../scripts/i18n-check.sh

## archcheck: the architecture rules (tools/archcheck)
archcheck: tools-image
	$(GO) go -C tools run ./archcheck -root /src

## arch-gate: archcheck, go vet and the tests; the Claude Code Stop hook runs it. It starts only the test services
## that are not running and recreates none: `run` alone recreates a running service whose config differs, and
## LiveKit's and Mailpit's do from another worktree (their bind-mount paths are in the config hash). No orphan is
## removed.
arch-gate: tools-image
	$(COMPOSE) up -d --no-recreate --wait $(TEST_SERVICES)
	$(GO_NODEPS) go -C tools run ./archcheck -root /src
	$(GO_NODEPS) go vet ./...
	$(GO_NODEPS) sh -c '$(GO_TEST)'

## e2e: start the stack and run the end-to-end tests against it in the playwright service, on the compose network:
## the Go journeys (the API, Mailpit's API, salusctl built here on the owner's DATABASE_URL, LiveKit), the web
## journey in Chromium that the consultation journey starts (test/e2e/web), then the check that LiveKit's log
## holds none of the run's synthetic names, e-mails and ids. Each run first clears the login rate-limit counters
## (ratelimit:*) in the compose Redis: a run signs in about 9 times from one IP, so back-to-back runs would reach
## the production limit of 30 per 15 minutes. The limit itself is unchanged.
e2e: up web-install e2e-install
	$(COMPOSE) exec -T redis sh -c "redis-cli --scan --pattern 'ratelimit:*' | xargs -r redis-cli del >/dev/null"
	$(E2E) sh -c 'rm -f /e2e/livekit-check.tsv && go build -o /tmp/salusctl ./cmd/salusctl && go test -tags e2e -count=1 ./test/e2e/...'
	$(COMPOSE) logs --no-color --no-log-prefix livekit | $(E2E) bash scripts/e2e-livekit-log.sh /e2e/livekit-check.tsv

## e2e-install: the e2e image, and npm ci in test/e2e/web, only when package.json or the lockfile changed
e2e-install: e2e-image test/e2e/web/node_modules/.package-lock.json
test/e2e/web/node_modules/.package-lock.json: test/e2e/web/package.json test/e2e/web/package-lock.json
	$(COMPOSE) run --rm -T -w /src/test/e2e/web playwright npm ci

## e2e-image: build the end-to-end runner image, Playwright with the Go toolchain (cached)
e2e-image:
	$(COMPOSE) build -q playwright

## fhir-validate: validate the golden FHIR responses with the HL7 validator
fhir-validate:
	$(COMPOSE) build -q fhir-validator
	$(COMPOSE) run --rm fhir-validator bash scripts/fhir-validate.sh

## migrate, migrate-status, migrate-create name=add_thing: goose migrations on the compose Postgres
migrate: tools-image
	$(GO) go run ./cmd/migrate up
migrate-status: tools-image
	$(GO) go run ./cmd/migrate status
migrate-create: tools-image
	$(GO) go run ./cmd/migrate create $(name)

## salusctl args="create-clinic -name <name> -admin-email <e-mail>" | args="reset-mfa <account-id>": the operator
## command against the compose Postgres, connected as the owner (docs/accounts/SPEC.md "Operator commands")
salusctl: tools-image
	$(GO) go run ./cmd/salusctl $(args)

## terminology-fetch: fetch the seed from Wikidata's SPARQL endpoint (network) and write
## assets/terminology/seed.jsonl (tools/terminology; docs/terminology/SPEC.md §3 "Data"). Never part of a build.
terminology-fetch: tools-image
	$(GO) go -C tools run ./terminology -out ../assets/terminology/seed.jsonl

## terminology-import [file=<path>]: load the seed (the default) or a curated CSV into the compose Postgres as the
## owner (salusctl terminology import; docs/terminology/SPEC.md §3 "Curation")
terminology-import: tools-image
	$(GO) go run ./cmd/salusctl terminology import $(or $(file),assets/terminology/seed.jsonl)

## terminology-export out=<path> [status=draft]: write the terms as the curator's CSV (salusctl terminology export),
## to a path under the repository
terminology-export: tools-image
	$(if $(out),,$(error terminology-export: out=<path> is required, e.g. make terminology-export out=terms.csv))
	$(GO) go run ./cmd/salusctl terminology export -o $(out) $(if $(status),-status $(status))

## sqlc: generate the domains' query packages (sqlc.yaml arrives with the first domain)
sqlc: tools-image
	@if [ -f sqlc.yaml ]; then $(GO) $(TOOL) sqlc generate; else echo "sqlc: no sqlc.yaml yet; it arrives with the first domain's queries"; fi

## gen-clients: regenerate the API clients from api/openapi.yaml: TypeScript into web/src/api, Dart into
## mobile/packages/salus_api (generator options in mobile/openapi-generator.yaml)
gen-clients: gen-client-ts gen-client-dart
gen-client-ts: web-install
	$(NODE) npm run -s gen:api
gen-client-dart: flutter-image
	$(COMPOSE) run --rm -T openapi-generator generate -g dart -i api/openapi.yaml -c mobile/openapi-generator.yaml \
		--global-property apiTests=false,modelTests=false,apiDocs=false,modelDocs=false -o mobile/packages/salus_api
	$(FLUTTER) dart format packages/salus_api

## web-install: npm ci in web/, only when package.json or the lockfile changed
web-install: node-image web/node_modules/.package-lock.json
web/node_modules/.package-lock.json: web/package.json web/package-lock.json
	$(NODE) npm ci

## web-lint, web-typecheck, web-test, web-build: the web app's checks
web-lint: web-install i18n-check
	$(NODE) npm run lint
web-typecheck: web-install
	$(NODE) npm run typecheck
web-test: web-install
	$(NODE) npm test
web-build: web-install
	$(NODE) npm run build

## web-dev: the Vite dev server on 127.0.0.1:5173 (SALUS_WEB_PORT), proxying the API; run `make up` first
web-dev: web-install
	$(COMPOSE) --profile web up node

## landing-install: npm ci in landing/, only when package.json or the lockfile changed
landing-install: node-image landing/node_modules/.package-lock.json
landing/node_modules/.package-lock.json: landing/package.json landing/package-lock.json
	$(LANDING) npm ci

## landing-lint, landing-typecheck, landing-test, landing-build: the landing page's checks; the build prerenders
## the page into landing/dist
landing-lint: landing-install
	$(LANDING) npm run lint
landing-typecheck: landing-install
	$(LANDING) npm run typecheck
landing-test: landing-install
	$(LANDING) npm test
landing-build: landing-install
	$(LANDING) npm run build

## landing-dev: the landing page's Vite dev server on 127.0.0.1:5174 (SALUS_LANDING_DEV_PORT)
landing-dev: landing-install
	$(COMPOSE) --profile landing up landing-dev

## mobile-analyze, mobile-test: the Flutter app's checks (mobile/, flutter service). Device runs are manual.
mobile-analyze: flutter-image i18n-check
	$(FLUTTER) flutter analyze
mobile-test: flutter-image
	$(FLUTTER) flutter test

## flutter-image: build the Flutter toolchain image (cached)
flutter-image:
	$(COMPOSE) build -q flutter

## node-image: build the Node toolchain image (cached)
node-image:
	$(COMPOSE) build -q node

## tools-image: build the Go toolchain image (cached)
tools-image:
	$(COMPOSE) build -q tools

## up: build and start the local stack (migrations first); see README.md for the ports
up:
	$(COMPOSE) up -d --build --wait --remove-orphans

## landing: build and serve only the landing page (landing/) on 127.0.0.1:8090 (SALUS_LANDING_PORT)
landing:
	$(COMPOSE) up -d --build --wait landing

## psql: open psql on the compose Postgres (no host port is published)
psql:
	$(COMPOSE) up -d --wait postgres
	$(COMPOSE) exec postgres psql -U salus -d salus

## down: stop everything (volumes and caches are kept)
down:
	$(COMPOSE) --profile tools --profile web --profile landing --profile mobile --profile e2e down
