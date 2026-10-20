# Every target runs in Docker through compose; the host needs only Docker and make. Longer commands live in
# scripts/ and run inside the containers.

export HOST_UID := $(shell id -u)
export HOST_GID := $(shell id -g)

COMPOSE := docker compose -f deployments/docker-compose.yml
GO      := $(COMPOSE) run --rm tools
TOOL    := go tool -modfile=tools/go.mod

.PHONY: test cover lint archcheck arch-gate e2e fhir-validate migrate migrate-status migrate-create sqlc \
	tools-image up psql down

## test: Go tests of both modules against the compose Postgres and Redis (TESTFLAGS="-run X -count=1")
test: tools-image
	$(GO) sh -c 'go test $(TESTFLAGS) ./... && go -C tools test $(TESTFLAGS) ./...'

## cover: tests with coverage, failing below 80%
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

## tools-image: build the Go toolchain image (cached)
tools-image:
	$(COMPOSE) build -q tools

## up: build and start the local stack (migrations first); see README.md for the ports
up:
	$(COMPOSE) up -d --build --wait --remove-orphans

## psql: open psql on the compose Postgres (no host port is published)
psql:
	$(COMPOSE) up -d --wait postgres
	$(COMPOSE) exec postgres psql -U salus -d salus

## down: stop everything (volumes and caches are kept)
down:
	$(COMPOSE) --profile tools --profile web --profile mobile down
