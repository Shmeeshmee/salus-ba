package chargeitemdefinition_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

// stubService answers every call with its values and remembers what it was asked.
type stubService struct {
	err      error
	caller   chargeitemdefinition.Caller
	org      string
	language string
	mode     schedule.VisitMode
	out      []chargeitemdefinition.Definition
	rows     []chargeitemdefinition.ImportRow
	file     chargeitemdefinition.ImportFile
	saved    chargeitemdefinition.Definition
	save     bool
}

func (s *stubService) Import(_ context.Context, c chargeitemdefinition.Caller, f chargeitemdefinition.ImportFile, save bool) ([]chargeitemdefinition.ImportRow, error) {
	s.caller, s.file, s.save = c, f, save
	return s.rows, s.err
}

func (s *stubService) List(_ context.Context, c chargeitemdefinition.Caller) ([]chargeitemdefinition.Definition, error) {
	s.caller = c
	return s.out, s.err
}

func (s *stubService) Create(_ context.Context, c chargeitemdefinition.Caller, d chargeitemdefinition.Definition) (chargeitemdefinition.Definition, error) {
	s.caller, s.saved = c, d
	d.ID, d.VersionID, d.Currency = "d0000000-0000-4000-8000-000000000001", 1, "BAM"
	return d, s.err
}

func (s *stubService) Update(_ context.Context, c chargeitemdefinition.Caller, d chargeitemdefinition.Definition) (chargeitemdefinition.Definition, error) {
	s.caller, s.saved = c, d
	d.VersionID, d.Currency = d.VersionID+1, "BAM"
	return d, s.err
}

func (s *stubService) Offered(_ context.Context, org string, mode schedule.VisitMode, language string) ([]chargeitemdefinition.Definition, error) {
	s.org, s.mode, s.language = org, mode, language
	return s.out, s.err
}

// serve sends the request to the handler on svc, signed in as admin unless anonymous.
func serve(t *testing.T, svc *stubService, anonymous bool, method, path, body string, header ...string) *httptest.ResponseRecorder {
	t.Helper()
	caller := func(context.Context) (chargeitemdefinition.Caller, bool, error) { return admin, !anonymous, nil }
	mux := http.NewServeMux()
	chargeitemdefinition.NewHandler(svc, caller, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes(mux)
	r := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	for i := 0; i+1 < len(header); i += 2 {
		r.Header.Set(header[i], header[i+1])
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

const (
	servicePath = "/clinic/services/d0000000-0000-4000-8000-000000000001"
	saveBody    = `{"name":"Placeholder exam","code":"EX-1","priceMinor":5000,"durationMinutes":30,"vatRateBp":1700,` +
		`"modes":["in-person"],"active":true,"translations":[{"language":"de","name":"Platzhalter"}]}`
	minimalBody = `{"name":"Placeholder exam","priceMinor":0,"modes":["video"],"active":false,"translations":[]}`
)

func TestHandler_ListsTheClinicsServices(t *testing.T) {
	svc := &stubService{out: []chargeitemdefinition.Definition{
		{ID: "d1", Name: "Placeholder exam", Code: "EX-1", Currency: "BAM", PriceMinor: 5000, DurationMinutes: 30, VATRateBP: ptr(1700),
			Modes: []schedule.VisitMode{schedule.ModeInPerson}, Active: true, VersionID: 3,
			Translations: []chargeitemdefinition.Translation{{Language: "de", Name: "Platzhalter"}}},
		{ID: "d2", Name: "Placeholder call", Currency: "BAM", Modes: []schedule.VisitMode{schedule.ModeVideo}, VersionID: 1},
	}}
	w := serve(t, svc, false, "GET", "/clinic/services", "")
	want := `[{"vatRateBp":1700,"id":"d1","name":"Placeholder exam","code":"EX-1","currency":"BAM","modes":["in-person"],` +
		`"translations":[{"language":"de","name":"Platzhalter"}],"priceMinor":5000,"durationMinutes":30,"versionId":3,"active":true},` +
		`{"id":"d2","name":"Placeholder call","currency":"BAM","modes":["video"],"translations":[],"priceMinor":0,"versionId":1,"active":false}]`
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != want || w.Header().Get("Cache-Control") != "no-store" || svc.caller != admin {
		t.Errorf("GET = %d %s (Cache-Control %q), want 200 %s", w.Code, w.Body, w.Header().Get("Cache-Control"), want)
	}
	svc.out = nil
	if w := serve(t, svc, false, "GET", "/clinic/services", ""); strings.TrimSpace(w.Body.String()) != "[]" {
		t.Errorf("GET of an empty list = %s, want []", w.Body)
	}
}

func TestHandler_CreatesAndUpdatesAService(t *testing.T) {
	svc := &stubService{}
	w := serve(t, svc, false, "POST", "/clinic/services", saveBody)
	if w.Code != http.StatusCreated || w.Header().Get("ETag") != `W/"1"` || !strings.Contains(w.Body.String(), `"versionId":1`) {
		t.Fatalf("POST = %d %s, ETag %q; want 201 at version 1", w.Code, w.Body, w.Header().Get("ETag"))
	}
	d := svc.saved
	if d.Name != "Placeholder exam" || d.Code != "EX-1" || d.PriceMinor != 5000 || d.DurationMinutes != 30 || d.VATRateBP == nil || *d.VATRateBP != 1700 ||
		len(d.Modes) != 1 || d.Modes[0] != schedule.ModeInPerson || !d.Active || len(d.Translations) != 1 || d.Translations[0].Language != "de" {
		t.Errorf("POST asked for %+v", d)
	}
	w = serve(t, svc, false, "PUT", servicePath, minimalBody, "If-Match", `W/"4"`)
	if w.Code != http.StatusOK || w.Header().Get("ETag") != `W/"5"` {
		t.Fatalf("PUT = %d %s, ETag %q; want 200 at version 5", w.Code, w.Body, w.Header().Get("ETag"))
	}
	if d := svc.saved; d.ID != "d0000000-0000-4000-8000-000000000001" || d.VersionID != 4 || d.VATRateBP != nil || d.Active || d.DurationMinutes != 0 {
		t.Errorf("PUT asked for %+v", d)
	}
}

func TestHandler_RefusesWhatItCannotRead(t *testing.T) {
	for name, tc := range map[string]struct{ method, path, body, ifMatch string }{
		"not JSON":             {"POST", "/clinic/services", "{", ""},
		"an unknown field":     {"POST", "/clinic/services", `{"name":"x","priceMinor":1,"modes":["video"],"active":true,"translations":[],"id":"1"}`, ""},
		"no name":              {"POST", "/clinic/services", `{"priceMinor":1,"modes":["video"],"active":true,"translations":[]}`, ""},
		"no price":             {"POST", "/clinic/services", `{"name":"x","modes":["video"],"active":true,"translations":[]}`, ""},
		"no modes":             {"POST", "/clinic/services", `{"name":"x","priceMinor":1,"active":true,"translations":[]}`, ""},
		"no active":            {"POST", "/clinic/services", `{"name":"x","priceMinor":1,"modes":["video"],"translations":[]}`, ""},
		"no translations":      {"POST", "/clinic/services", `{"name":"x","priceMinor":1,"modes":["video"],"active":true}`, ""},
		"two documents":        {"POST", "/clinic/services", minimalBody + minimalBody, ""},
		"no If-Match":          {"PUT", servicePath, minimalBody, ""},
		"a malformed If-Match": {"PUT", servicePath, minimalBody, `W/"x"`},
	} {
		svc := &stubService{}
		w := serve(t, svc, false, tc.method, tc.path, tc.body, "If-Match", tc.ifMatch)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"invalid_request"`) || svc.saved.Name != "" {
			t.Errorf("%s: %d %s, want 400 invalid_request and no call", name, w.Code, w.Body)
		}
	}
}

func TestHandler_MapsTheServicesErrors(t *testing.T) {
	for err, want := range map[error]struct {
		code   string
		status int
	}{
		chargeitemdefinition.ErrNotFound:  {"not_found", http.StatusNotFound},
		chargeitemdefinition.ErrForbidden: {"forbidden", http.StatusForbidden},
		chargeitemdefinition.ErrInvalid:   {"invalid_service", http.StatusBadRequest},
		chargeitemdefinition.ErrConflict:  {"version_conflict", http.StatusPreconditionFailed},
		errBroken:                         {"internal", http.StatusInternalServerError},
	} {
		svc := &stubService{err: err}
		for _, w := range []*httptest.ResponseRecorder{
			serve(t, svc, false, "GET", "/clinic/services", ""),
			serve(t, svc, false, "POST", "/clinic/services", minimalBody),
			serve(t, svc, false, "PUT", servicePath, minimalBody, "If-Match", `W/"1"`),
		} {
			var body map[string]string
			if json.Unmarshal(w.Body.Bytes(), &body) != nil || w.Code != want.status || (want.status != 500 && body["error"] != want.code) ||
				strings.Contains(w.Body.String(), "fake") {
				t.Errorf("%v: %d %s, want %d %s", err, w.Code, w.Body, want.status, want.code)
			}
		}
	}
	for _, w := range []*httptest.ResponseRecorder{
		serve(t, &stubService{}, true, "GET", "/clinic/services", ""),
		serve(t, &stubService{}, true, "POST", "/clinic/services", minimalBody),
		serve(t, &stubService{}, true, "PUT", servicePath, minimalBody, "If-Match", `W/"1"`),
	} {
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), `"unauthenticated"`) {
			t.Errorf("signed out: %d %s, want 401 unauthenticated", w.Code, w.Body)
		}
	}
}

// The prices at booking are public, as the directory: no session is asked for.
func TestHandler_ShowsAnyoneTheOfferedServices(t *testing.T) {
	svc := &stubService{out: []chargeitemdefinition.Definition{
		{ID: "d1", Name: "Platzhalter", Code: "EX-1", Currency: "BAM", PriceMinor: 5000, DurationMinutes: 30, VATRateBP: ptr(1700), Active: true, VersionID: 3},
		{ID: "d2", Name: "Placeholder call", Currency: "BAM"},
	}}
	w := serve(t, svc, true, "GET", "/organizations/"+clinicA+"/services?mode=in-person&language=de", "")
	want := `[{"id":"d1","name":"Platzhalter","currency":"BAM","priceMinor":5000,"durationMinutes":30},{"id":"d2","name":"Placeholder call","currency":"BAM","priceMinor":0}]`
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != want {
		t.Errorf("GET = %d %s, want 200 %s", w.Code, w.Body, want)
	}
	if svc.org != clinicA || svc.mode != schedule.ModeInPerson || svc.language != "de" {
		t.Errorf("asked for %s %s %s", svc.org, svc.mode, svc.language)
	}
	for err, want := range map[error]struct {
		code   string
		status int
	}{
		chargeitemdefinition.ErrInvalid:  {"invalid_request", http.StatusBadRequest},
		chargeitemdefinition.ErrNotFound: {"not_found", http.StatusNotFound},
		errBroken:                        {"", http.StatusInternalServerError},
	} {
		w := serve(t, &stubService{err: err}, true, "GET", "/organizations/"+clinicA+"/services?mode=video", "")
		if w.Code != want.status || !strings.Contains(w.Body.String(), want.code) {
			t.Errorf("%v: %d %s, want %d %s", err, w.Code, w.Body, want.status, want.code)
		}
	}
}
