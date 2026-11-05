# The end-to-end runner of `make e2e` (the `playwright` service, profile e2e): the official Playwright image,
# with Chromium and Node, plus the Go toolchain of dev.Dockerfile. The Go journeys in test/e2e/ start the web
# journey (test/e2e/web, @playwright/test of the same version) as a subprocess, so the browser, the Go SDK test
# participant and the test that checks the API all run in one container on the compose network.
FROM golang:1.27.3-trixie AS go

FROM mcr.microsoft.com/playwright:v1.55.0-noble
COPY --from=go /usr/local/go /usr/local/go

# Caches live in named volumes mounted under /cache, and /e2e holds the run's LiveKit log check; world-writable
# so the container can run as the host user (files it writes into the repo stay owned by that user).
RUN mkdir -p /cache/gomod /cache/gobuild /cache/npm /cache/home /e2e && chmod -R 0777 /cache /e2e
ENV PATH=/usr/local/go/bin:$PATH \
    GOMODCACHE=/cache/gomod \
    GOCACHE=/cache/gobuild \
    GOTOOLCHAIN=local \
    GOFLAGS=-buildvcs=false \
    CGO_ENABLED=0 \
    HOME=/cache/home \
    npm_config_cache=/cache/npm \
    npm_config_update_notifier=false \
    npm_config_fund=false

WORKDIR /src
