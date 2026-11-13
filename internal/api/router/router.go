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
	Logger *slog.Logger
	// Session authenticates the request and puts its session in the context (auth.Handler.Session); nil for
	// none. It runs inside the timeout, so its read of the session is bounded too.
	Session func(http.Handler) http.Handler
	Checks  []Check
	// Routes are the domains' registration functions (such as auth.Handler.Routes), each declaring its
	// routes on the mux.
	Routes []func(mux *http.ServeMux)
	// Streams are the registration functions of the routes that stream a request or a response body (an
	// upload, a download, such as binary.HTTPHandler.StreamRoutes): they are served without the request
	// timeout, whose buffered writer would hold the whole body and cut a long transfer, and set their own
	// deadlines through http.ResponseController.
	Streams []func(mux *http.ServeMux)
	// Provenances are the sources of GET /fhir/R4/Provenance/{id}, asked in turn (such as
	// allergyintolerance.FHIRHandler.Provenance): each answers and reports true when the id is its own, or writes
	// nothing and reports false. One that audits a read of any id it is given (composition's) goes last.
	Provenances []func(w http.ResponseWriter, r *http.Request) bool
	Timeout     time.Duration // per request
}

// New returns the API's handler: request id, access log, recover, timeout (but for the streams) and the
// session check around the routes.
func New(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", readyz(cfg.Logger, cfg.Checks))
	mux.HandleFunc("GET /fhir/R4/metadata", metadata)
	mux.HandleFunc("/fhir/R4/", fhirNotFound)
	mux.HandleFunc("GET /fhir/R4/Provenance/{id}", firstOf(cfg.Provenances))
	for _, routes := range cfg.Routes {
		routes(mux)
	}
	streams := http.NewServeMux() // only to tell a streaming route's request; mux serves it
	for _, routes := range cfg.Streams {
		routes(mux)
		routes(streams)
	}

	var h http.Handler = mux
	if cfg.Session != nil {
		h = cfg.Session(h)
	}
	timed := http.TimeoutHandler(h, cfg.Timeout, `{"error":"timeout"}`)
	untimed := h
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := streams.Handler(r); pattern != "" {
			untimed.ServeHTTP(w, r)
			return
		}
		timed.ServeHTTP(w, r)
	})
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

// firstOf serves a read from the first of sources that answers it, and not found when none does.
func firstOf(sources []func(w http.ResponseWriter, r *http.Request) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, answer := range sources {
			if answer(w, r) {
				return
			}
		}
		fhirNotFound(w, r)
	}
}

// fhirNotFound answers every FHIR path the CapabilityStatement does not list.
func fhirNotFound(w http.ResponseWriter, _ *http.Request) {
	fhir.WriteError(w, http.StatusNotFound, "not-found", "This resource or interaction is not supported.")
}
