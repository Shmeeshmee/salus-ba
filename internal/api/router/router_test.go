package router_test

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/api/router"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

func newRouter(checks ...router.Check) http.Handler {
	return router.New(router.Config{
		Logger:  slog.New(slog.DiscardHandler),
		Checks:  checks,
		Timeout: time.Second,
	})
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

func ok(context.Context) error { return nil }

// assertGolden compares got with testdata/<name>; -update rewrites the file.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("%s differs from the response:\n got: %s\nwant: %s", path, got, want)
	}
}

func TestHealthz_IsOKWithoutTouchingDependencies(t *testing.T) {
	failing := router.Check{Name: "postgres", Ping: func(context.Context) error { return errors.New("down") }}

	rec := get(t, newRouter(failing), "/healthz")

	if rec.Code != http.StatusOK || rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Errorf("GET /healthz = %d %q, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body)
	}
}

func TestReadyz_IsOKWhenEveryCheckPasses(t *testing.T) {
	rec := get(t, newRouter(router.Check{Name: "postgres", Ping: ok}, router.Check{Name: "redis", Ping: ok}), "/readyz")

	if rec.Code != http.StatusOK || rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Errorf("GET /readyz = %d %q, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body)
	}
}

func TestReadyz_NamesFailingChecksWithoutTheirErrors(t *testing.T) {
	down := router.Check{Name: "postgres", Ping: func(context.Context) error {
		return errors.New("dial tcp 10.0.0.7:5432: connection refused")
	}}

	rec := get(t, newRouter(down, router.Check{Name: "redis", Ping: ok}), "/readyz")

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if body := rec.Body.String(); body != "{\"failed\":[\"postgres\"],\"status\":\"unavailable\"}\n" {
		t.Errorf("body = %q", body)
	}
}

func TestReadyz_GivesChecksADeadline(t *testing.T) {
	var hasDeadline bool
	check := router.Check{Name: "postgres", Ping: func(ctx context.Context) error {
		_, hasDeadline = ctx.Deadline()
		return nil
	}}

	get(t, newRouter(check), "/readyz")

	if !hasDeadline {
		t.Error("the check's context has no deadline")
	}
}

func TestMetadata_ServesTheCapabilityStatement(t *testing.T) {
	rec := get(t, newRouter(), "/fhir/R4/metadata")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/fhir+json" {
		t.Errorf("Content-Type = %q, want application/fhir+json", ct)
	}
	assertGolden(t, "metadata.json", rec.Body.Bytes())
}

func TestFHIR_UnknownPathIsANotFoundOperationOutcome(t *testing.T) {
	rec := get(t, newRouter(), "/fhir/R4/Patient/7c9e6679-7425-40de-944b-e07fc1f90ae7")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/fhir+json" {
		t.Errorf("Content-Type = %q, want application/fhir+json", ct)
	}
	assertGolden(t, "fhir_not_found.json", rec.Body.Bytes())
}

func TestRouter_SetsARequestID(t *testing.T) {
	if id := get(t, newRouter(), "/healthz").Header().Get("X-Request-ID"); id == "" {
		t.Error("no X-Request-ID on the response")
	}
}

func TestRouter_TimesOutSlowRequests(t *testing.T) {
	slow := router.Check{Name: "postgres", Ping: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	h := router.New(router.Config{Logger: slog.New(slog.DiscardHandler), Checks: []router.Check{slow}, Timeout: 20 * time.Millisecond})

	rec := get(t, h, "/readyz")

	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "timeout") {
		t.Errorf("GET /readyz with a hanging check = %d %q, want 503 timeout", rec.Code, rec.Body)
	}
}
