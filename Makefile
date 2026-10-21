# Every target runs in Docker through compose; the host needs only Docker and make. Longer commands live in
# scripts/ and run inside the containers.

export HOST_UID := $(shell id -u)
export HOST_GID := $(shell id -g)

COMPOSE := docker compose -f deployments/docker-compose.yml
GO      := $(COMPOSE) run --rm tools
TOOL    := go tool -modfile=tools/go.mod
NODE    := $(COMPOSE) run --rm -T node
FLUTTER := $(COMPOSE) run --rm -T flutter

.PHONY: test cover lint archcheck arch-gate e2e fhir-validate migrate migrate-status migrate-create sqlc \
	gen-clients gen-client-ts gen-client-dart mobile-analyze mobile-test web-install web-lint web-typecheck web-test web-build web-dev flutter-image tools-image node-image up landing psql down

## test: Go tests of both modules against the compose Postgres and Redis (TESTFLAGS="-run X -count=1")
test: tools-image
	$(GO) sh -c 'go test $(TESTFLAGS) ./... && go -C tools test $(TESTFLAGS) ./...'

## cover: tests with coverage, failing when any package is below 80% (cmd/*/main.go excluded)
cover: tools-image
	$(GO) bash scripts/cover.sh

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

## e2e: start the stack and run the end-to-end tests against it
e2e: up
	$(COMPOSE) run --rm -e E2E_BASE_URL=http://api:8080 tools go test -tags e2e -count=1 ./test/e2e/...

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
	$(COMPOSE) --profile tools --profile web --profile mobile down
