package datarequest_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/datarequest"
)

type stubExports struct {
	err    error
	export datarequest.Export
	body   string
	calls  []string
}

func (s *stubExports) Export(_ context.Context, p datarequest.Patient, id string) (datarequest.Export, error) {
	s.calls = append(s.calls, p.AccountID+" "+id)
	return s.export, s.err
}

func (s *stubExports) ExportArchive(_ context.Context, p datarequest.Patient, id string) (io.ReadCloser, int64, error) {
	s.calls = append(s.calls, p.AccountID+" "+id)
	if s.err != nil {
		return nil, 0, s.err
	}
	return io.NopCloser(strings.NewReader(s.body)), int64(len(s.body)), nil
}

func serveExport(t *testing.T, svc *stubExports, signedIn bool, path string) *httptest.ResponseRecorder {
	t.Helper()
	caller := func(context.Context) (datarequest.Patient, bool) {
		return datarequest.Patient{AccountID: accountA}, signedIn
	}
	mux := http.NewServeMux()
	datarequest.NewExportHandler(svc, caller, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

func TestExportHandler_Status(t *testing.T) {
	t.Parallel()
	expires := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	for name, tc := range map[string]struct {
		svc      *stubExports
		body     string
		status   int
		signedIn bool
	}{
		"queued":        {&stubExports{export: datarequest.Export{Status: datarequest.ExportQueued}}, `{"status":"queued"}`, 200, true},
		"ready":         {&stubExports{export: datarequest.Export{Status: datarequest.ExportReady, ExpiresAt: expires}}, `{"expiresAt":"2026-12-01T09:00:00Z","status":"ready"}`, 200, true},
		"not found":     {&stubExports{err: datarequest.ErrNotFound}, `{"error":"not_found"}`, 404, true},
		"internal":      {&stubExports{err: errors.New("down: secret detail")}, "", 500, true},
		"not signed in": {&stubExports{}, `{"error":"unauthenticated"}`, 401, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec := serveExport(t, tc.svc, tc.signedIn, "/me/data-requests/"+request1+"/export")
			if rec.Code != tc.status {
				t.Fatalf("status %d, want %d", rec.Code, tc.status)
			}
			if got := strings.TrimSpace(rec.Body.String()); tc.body != "" && got != tc.body {
				t.Errorf("body %s, want %s", got, tc.body)
			}
			if strings.Contains(rec.Body.String(), "secret detail") {
				t.Error("the internal error reached the client")
			}
			if tc.signedIn && (len(tc.svc.calls) != 1 || tc.svc.calls[0] != accountA+" "+request1) {
				t.Errorf("called %v, want the patient's request", tc.svc.calls)
			}
			if tc.status == 200 && rec.Header().Get("Cache-Control") != "no-store" {
				t.Error("cacheable")
			}
		})
	}
}

func TestExportHandler_Download(t *testing.T) {
	t.Parallel()
	svc := &stubExports{body: "zip bytes"}
	rec := serveExport(t, svc, true, "/me/data-requests/"+request1+"/export/archive")
	if rec.Code != 200 || rec.Body.String() != "zip bytes" || rec.Header().Get("Content-Type") != "application/zip" ||
		rec.Header().Get("Content-Length") != "9" || rec.Header().Get("Cache-Control") != "no-store" ||
		rec.Header().Get("Content-Disposition") != `attachment; filename="salus-export-`+request1+`.zip"` {
		t.Errorf("%d %q %v", rec.Code, rec.Body.String(), rec.Header())
	}
	for name, tc := range map[string]struct {
		svc      *stubExports
		signedIn bool
		status   int
	}{
		"not ready":     {&stubExports{err: datarequest.ErrNotFound}, true, 404},
		"not signed in": {&stubExports{}, false, 401},
	} {
		if rec := serveExport(t, tc.svc, tc.signedIn, "/me/data-requests/"+request1+"/export/archive"); rec.Code != tc.status {
			t.Errorf("%s: %d, want %d", name, rec.Code, tc.status)
		}
	}
}
