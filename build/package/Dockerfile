# Runtime image for every Go binary under cmd/ (api, migrate, worker, scheduler, notifier, agent, salusctl).
# Compose picks the binary with `command`. Static binaries on distroless, running as a non-root user.
FROM golang:1.27.3-trixie@sha256:af1bfc2614ebb1dbd31af514b78740fd678850e76498762b425ecee4c5c6ea80 AS build
ENV CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS=-buildvcs=false
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod go mod download
COPY api ./api
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
COPY pkg ./pkg
RUN --mount=type=cache,target=/root/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

FROM gcr.io/distroless/static-debian13:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5
COPY --from=build /out/ /usr/local/bin/
USER nonroot:nonroot
