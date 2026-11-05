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

.PHONY: test cover bench-hash bench-auth bench-slots bench-join bench-join-http lint archcheck arch-gate e2e fhir-validate migrate migrate-status migrate-create salusctl sqlc \
	gen-clients gen-client-ts gen-client-dart mobile-analyze mobile-test web-install web-lint web-typecheck web-test web-build web-dev \
	landing-install landing-lint landing-typecheck landing-test landing-build landing-dev flutter-image tools-image node-image up landing psql down

## test: Go tests of both modules against the compose Postgres and Redis (TESTFLAGS="-run X -count=1")
test: tools-image
	$(GO) sh -c 'go test $(TESTFLAGS) ./... && go -C tools test $(TESTFLAGS) ./...'

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

## bench-join: the consultation's token request through the service on the real database, with the compose
## LiveKit and with a fake provider (docs/consultation/PROGRESS.md, package 1)
bench-join: tools-image
	$(GO) go test -run '^$$' -bench 'BenchmarkJoin' -benchtime=2s -count=3 ./internal/api

## bench-join-http: the join endpoint's latency through the running stack, 1,000 requests 10 at a time on the
## compose Postgres and LiveKit, as p50/p95/p99 (docs/consultation/PROGRESS.md, package 4). Not part of make e2e.
bench-join-http: up tools-image
	$(COMPOSE) run --rm -e E2E_BASE_URL=http://api:8080 -e E2E_MAILPIT_URL=http://mailpit:8025 -e E2E_SALUSCTL=/tmp/salusctl \
		tools sh -c 'go build -o /tmp/salusctl ./cmd/salusctl && go test -tags e2e,loadtest -count=1 -v -run TestJoinLatency ./test/e2e/'

## lint: gofmt, go vet, golangci-lint (both modules) and the migration immutability check
lint: tools-image
	$(GO) bash scripts/lint.sh

## archcheck: the architecture rules (tools/archcheck)
archcheck: tools-image
	$(GO) go -C tools run ./archcheck -root /src

## arch-gate: archcheck, go vet and the tests; the Claude Code Stop hook runs it
arch-gate: archcheck tools-image
	$(GO) go vet ./...
	$(MAKE) test

## e2e: start the stack and run the end-to-end tests against it, on the compose network: the API, Mailpit's API
## and salusctl (built here, run on the owner's DATABASE_URL of the tools service)
e2e: up tools-image
	$(COMPOSE) run --rm -e E2E_BASE_URL=http://api:8080 -e E2E_MAILPIT_URL=http://mailpit:8025 -e E2E_SALUSCTL=/tmp/salusctl \
		tools sh -c 'go build -o /tmp/salusctl ./cmd/salusctl && go test -tags e2e -count=1 ./test/e2e/...'

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
web-lint: web-install
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
mobile-analyze: flutter-image
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
	$(COMPOSE) --profile tools --profile web --profile landing --profile mobile down
