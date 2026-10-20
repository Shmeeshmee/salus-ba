# Every target runs in Docker through compose; the host needs only Docker and make.
# Targets are added as the bootstrap plan's packages land (docs/bootstrap/IMPLEMENTATION_PLAN.md).

export HOST_UID := $(shell id -u)
export HOST_GID := $(shell id -g)

COMPOSE := docker compose -f deployments/docker-compose.yml
GO      := $(COMPOSE) run --rm tools

.PHONY: test tools-image up psql down

## test: run the Go tests against the compose Postgres and Redis (TESTFLAGS="-run X -count=1" to narrow)
test: tools-image
	$(GO) go test $(TESTFLAGS) ./...

## tools-image: build the Go toolchain image (cached; rebuilt when its Dockerfile changes)
tools-image:
	$(COMPOSE) build -q tools

## up: build and start the local stack (migrations first); see README.md for the ports
up:
	$(COMPOSE) up -d --build --wait --remove-orphans

## psql: open psql on the compose Postgres (no host port is published)
psql:
	$(COMPOSE) up -d --wait postgres
	$(COMPOSE) exec postgres psql -U salus -d salus

## down: stop the stack (volumes and caches are kept)
down:
	$(COMPOSE) --profile tools --profile web --profile mobile down
