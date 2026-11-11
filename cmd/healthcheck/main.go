// Command healthcheck is the Docker healthcheck of the services on the distroless image, which has no shell,
// curl or wget: `healthcheck http://127.0.0.1:8080/healthz` exits 0 when the URL answers 2xx within the timeout
// and 1 otherwise (deployments/production/docker-compose.prod.yml).
package main

import (
	"net/http"
	"os"
	"time"
)

// timeout ends the probe before Docker's own healthcheck timeout does, so the reason is in the health log.
const timeout = 2 * time.Second

func main() {
	os.Exit(run(os.Args[1:], &http.Client{Timeout: timeout}, os.Stderr))
}
