# Development and CI toolchain: Go plus the Go tools pinned as `tool` lines in tools/go.mod (sqlc, goose,
# golangci-lint), run from the repo root with `go tool -modfile=tools/go.mod <name>` and compiled into the
# cached build directory on first use. The app's go.mod stays free of tool dependencies.
# Used by the `tools` service in deployments/docker-compose.yml. Nothing runs on the host but Docker and make.
FROM golang:1.27.3-trixie

# Caches live in named volumes mounted at /cache; world-writable so the container can run as the host user
# (files it writes into the repo stay owned by that user).
RUN mkdir -p /cache/gomod /cache/gobuild /cache/golangci /cache/home && chmod -R 0777 /cache
ENV GOMODCACHE=/cache/gomod \
    GOCACHE=/cache/gobuild \
    GOLANGCI_LINT_CACHE=/cache/golangci \
    HOME=/cache/home \
    GOTOOLCHAIN=local \
    GOFLAGS=-buildvcs=false

# iptables for `make turn-check`, which blocks UDP in its own container to prove TURN (scripts/turn-check.sh).
RUN apt-get update && apt-get install -y --no-install-recommends iptables && rm -rf /var/lib/apt/lists/*

WORKDIR /src
