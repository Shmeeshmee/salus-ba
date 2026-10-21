// Package router mounts the middleware and declares the routes of the API binary.
package router

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	fhirapi "github.com/Shmeeshmee/salus/api/fhir"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

// readyTimeout bounds all readiness checks together.
const readyTimeout = 2 * time.Second

// Check is one dependency /readyz pings.
type Check struct {
	Ping func(ctx context.Context) error
	Name string
}

// Config is what the router needs.
type Config struct {
	Logger  *slog.Logger
	Checks  []Check
	Timeout time.Duration // per request
}

// New returns the API's handler: request id, access log, recover and timeout around the routes.
func New(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", readyz(cfg.Logger, cfg.Checks))
	mux.HandleFunc("GET /fhir/R4/metadata", metadata)
	mux.HandleFunc("/fhir/R4/", fhirNotFound)

	var h http.Handler = mux
	h = http.TimeoutHandler(h, cfg.Timeout, `{"error":"timeout"}`)
	h = middleware.Recover(cfg.Logger)(h)
	h = middleware.AccessLog(cfg.Logger, mux)(h)
	return middleware.RequestID(h)
}

// healthz is liveness: the process serves requests. It touches no dependency.
func healthz(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz is readiness: every dependency answers. The response names failing checks; their errors go to
// the log only.
func readyz(log *slog.Logger, checks []Check) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()
		failed := []string{}
		for _, c := range checks {
			if err := c.Ping(ctx); err != nil {
				log.WarnContext(ctx, "readiness check failed", "check", c.Name, "error", err)
				failed = append(failed, c.Name)
			}
		}
		if len(failed) > 0 {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unavailable", "failed": failed})
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func metadata(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", fhir.ContentType)
	_, _ = io.WriteString(w, fhirapi.CapabilityStatement())
}

// fhirNotFound answers every FHIR path the CapabilityStatement does not list.
func fhirNotFound(w http.ResponseWriter, _ *http.Request) {
	fhir.WriteError(w, http.StatusNotFound, "not-found", "This resource or interaction is not supported.")
}
