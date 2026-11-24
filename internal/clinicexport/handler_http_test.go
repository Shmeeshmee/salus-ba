package clinicexport_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/clinicexport"
)

type stubExports struct {
	err    error
	body   string
	ids    []string
	export clinicexport.Export
}

func (s *stubExports) Request(context.Context, clinicexport.Caller) (clinicexport.Export, error) {
	return s.export, s.err
}

func (s *stubExports) Get(_ context.Context, _ clinicexport.Caller, id string) (clinicexport.Export, error) {
	s.ids = append(s.ids, id)
	return s.export, s.err
}

func (s *stubExports) Download(_ context.Context, _ clinicexport.Caller, id string) (io.ReadCloser, int64, error) {
	s.ids = append(s.ids, id)
	if s.err != nil {
		return nil, 0, s.err
	}
	return io.NopCloser(strings.NewReader(s.body)), int64(len(s.body)), nil
}

func serve(t *testing.T, svc clinicexport.Exports, signedIn bool, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	caller := func(context.Context) (clinicexport.Caller, bool, error) { return admin, signedIn, nil }
	mux := http.NewServeMux()
	clinicexport.NewHandler(svc, caller, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, nil))
	return rec
}

func TestHandler_RequestGetAndDownload(t *testing.T) {
	t.Parallel()
	id := "44444444-4444-4444-8444-444444444444"
	queued := clinicexport.Export{ID: id, Status: clinicexport.StatusQueued, RequestedAt: asked, ExpiresAt: asked.Add(clinicexport.Window)}
	rec := serve(t, &stubExports{export: queued}, true, http.MethodPost, "/clinic/export")
	want := `{"requestedAt":"2026-11-24T09:00:00Z","id":"` + id + `","status":"queued"}`
	if rec.Code != http.StatusAccepted || strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("POST = %d %s; want 202 %s", rec.Code, rec.Body, want)
	}

	ready := queued
	ready.Status = clinicexport.StatusReady
	stub := &stubExports{export: ready}
	rec = serve(t, stub, true, http.MethodGet, "/clinic/export/"+id)
	want = `{"requestedAt":"2026-11-24T09:00:00Z","expiresAt":"2026-12-01T09:00:00Z","id":"` + id + `","status":"ready"}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want || rec.Header().Get("Cache-Control") != "no-store" || stub.ids[0] != id {
		t.Errorf("GET = %d %s %v; want 200 %s, no-store", rec.Code, rec.Body, rec.Header(), want)
	}

	rec = serve(t, &stubExports{body: "PK-archive"}, true, http.MethodGet, "/clinic/export/"+id+"/archive")
	if rec.Code != http.StatusOK || rec.Body.String() != "PK-archive" || rec.Header().Get("Content-Type") != "application/zip" ||
		rec.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(rec.Header().Get("Content-Disposition"), "attachment") {
		t.Errorf("archive = %d %q %v; want the zip as an attachment", rec.Code, rec.Body, rec.Header())
	}
}

func TestHandler_Errors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		err      error
		code     string
		status   int
		signedIn bool
	}{
		{signedIn: false, status: http.StatusUnauthorized, code: "unauthenticated"},
		{err: clinicexport.ErrForbidden, signedIn: true, status: http.StatusForbidden, code: "forbidden"},
		{err: clinicexport.ErrNotFound, signedIn: true, status: http.StatusNotFound, code: "not_found"},
		{err: errors.New("db down: secret detail"), signedIn: true, status: http.StatusInternalServerError},
	} {
		for _, req := range [][2]string{{http.MethodPost, "/clinic/export"}, {http.MethodGet, "/clinic/export/x"}, {http.MethodGet, "/clinic/export/x/archive"}} {
			rec := serve(t, &stubExports{err: tc.err}, tc.signedIn, req[0], req[1])
			if rec.Code != tc.status || (tc.code != "" && !strings.Contains(rec.Body.String(), `"`+tc.code+`"`)) || strings.Contains(rec.Body.String(), "secret") {
				t.Errorf("%s %s with %v = %d %s; want %d %s", req[0], req[1], tc.err, rec.Code, rec.Body, tc.status, tc.code)
			}
		}
	}
}
