# Development and CI toolchain: Go plus the Go tools pinned as `tool` lines in go.mod (sqlc, goose,
# golangci-lint), run with `go tool <name>` and compiled into the cached build directory on first use.
# Used by the `tools` service in deployments/docker-compose.yml. Nothing runs on the host but Docker and make.
FROM golang:1.27.2-trixie

# Caches live in named volumes mounted at /cache; world-writable so the container can run as the host user
# (files it writes into the repo stay owned by that user).
RUN mkdir -p /cache/gomod /cache/gobuild /cache/golangci /cache/home && chmod -R 0777 /cache
ENV GOMODCACHE=/cache/gomod \
    GOCACHE=/cache/gobuild \
    GOLANGCI_LINT_CACHE=/cache/golangci \
    HOME=/cache/home \
    GOTOOLCHAIN=local \
    GOFLAGS=-buildvcs=false

WORKDIR /src
