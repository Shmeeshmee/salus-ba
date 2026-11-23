package chargeitemdefinition_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

// upload posts data as the form's part named field to the import route, signed in as admin unless anonymous.
func upload(t *testing.T, svc *stubService, anonymous bool, query, field string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	m := multipart.NewWriter(&body)
	w, err := m.CreateFormFile(field, "services.csv")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write(data)
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	caller := func(context.Context) (chargeitemdefinition.Caller, bool, error) { return admin, !anonymous, nil }
	mux := http.NewServeMux()
	chargeitemdefinition.NewHandler(svc, caller, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes(mux)
	r := httptest.NewRequestWithContext(t.Context(), "POST", "/clinic/services/import"+query, &body)
	r.Header.Set("Content-Type", m.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func TestHandler_ImportsAFileAndSaysWhetherItWasSaved(t *testing.T) {
	valid := chargeitemdefinition.ImportRow{Line: 2, Service: chargeitemdefinition.Definition{
		ID: "d1", Name: "Placeholder exam", Code: "EX-1", Currency: "BAM", PriceMinor: 2550, DurationMinutes: 30,
		VATRateBP: ptr(1700), Modes: []schedule.VisitMode{schedule.ModeInPerson}, Active: true,
	}}
	wrong := chargeitemdefinition.ImportRow{Line: 3, Errors: []string{"name_missing"}, Service: chargeitemdefinition.Definition{PriceMinor: 10}}
	for name, tc := range map[string]struct {
		query string
		want  string
		rows  []chargeitemdefinition.ImportRow
		save  bool
	}{
		"a preview": {"", `{"rows":[{"service":{"vatRateBp":1700,"name":"Placeholder exam","code":"EX-1",` +
			`"currency":"BAM","modes":["in-person"],"translations":[],"priceMinor":2550,"durationMinutes":30,"active":true},"errors":[],"line":2}],"saved":false}`,
			[]chargeitemdefinition.ImportRow{valid}, false},
		"saved": {"?save=true", `"saved":true`, []chargeitemdefinition.ImportRow{valid}, true},
		"an error, not saved": {"?save=true", `{"rows":[{"service":{"vatRateBp":1700,"name":"Placeholder exam","code":"EX-1",` +
			`"currency":"BAM","modes":["in-person"],"translations":[],"priceMinor":2550,"durationMinutes":30,"active":true},"errors":[],"line":2},` +
			`{"errors":["name_missing"],"line":3}],"saved":false}`, []chargeitemdefinition.ImportRow{valid, wrong}, true},
	} {
		svc := &stubService{rows: tc.rows}
		w := upload(t, svc, false, tc.query, "file", []byte("name;price\n"))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), tc.want) || svc.save != tc.save ||
			string(svc.file.Data) != "name;price\n" || svc.file.Name != "services.csv" || svc.caller != admin ||
			w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: %d %s, save %v; want 200 %s", name, w.Code, w.Body, svc.save, tc.want)
		}
	}
}

func TestHandler_RefusesAnImportItCannotRead(t *testing.T) {
	for name, tc := range map[string]struct {
		err    error
		code   string
		field  string
		query  string
		data   []byte
		status int
	}{
		"no file part":    {nil, "invalid_request", "other", "", []byte("x"), http.StatusBadRequest},
		"a bad save flag": {nil, "invalid_request", "file", "?save=maybe", []byte("x"), http.StatusBadRequest},
		"too large":       {nil, "too_large", "file", "", bytes.Repeat([]byte("x"), 3<<20), http.StatusRequestEntityTooLarge},
		"unreadable":      {chargeitemdefinition.ErrInvalidImport, "invalid_import", "file", "", []byte("x"), http.StatusBadRequest},
		"not the admin":   {chargeitemdefinition.ErrForbidden, "forbidden", "file", "", []byte("x"), http.StatusForbidden},
		"broken":          {errBroken, "internal", "file", "", []byte("x"), http.StatusInternalServerError},
	} {
		w := upload(t, &stubService{err: tc.err}, false, tc.query, tc.field, tc.data)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.code) || strings.Contains(w.Body.String(), "fake") {
			t.Errorf("%s: %d %s, want %d %s", name, w.Code, w.Body, tc.status, tc.code)
		}
	}
	if w := upload(t, &stubService{}, true, "", "file", []byte("x")); w.Code != http.StatusUnauthorized {
		t.Errorf("signed out: %d, want 401", w.Code)
	}
	mux := http.NewServeMux()
	chargeitemdefinition.NewHandler(&stubService{}, func(context.Context) (chargeitemdefinition.Caller, bool, error) { return admin, true, nil },
		slog.New(slog.NewTextHandler(io.Discard, nil))).Routes(mux)
	r := httptest.NewRequestWithContext(t.Context(), "POST", "/clinic/services/import", strings.NewReader("name;price"))
	r.Header.Set("Content-Type", "text/csv")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_request") {
		t.Errorf("not multipart: %d %s, want 400 invalid_request", w.Code, w.Body)
	}
}
