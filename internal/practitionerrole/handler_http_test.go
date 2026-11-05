package practitionerrole_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// stubSpecialties answers SetSpecialty with what it is given and keeps what it was asked.
type stubSpecialties struct {
	err     error
	caller  practitionerrole.Caller
	id      string
	code    string
	out     practitionerrole.PractitionerRole
	version int32
	called  bool
}

func (s *stubSpecialties) SetSpecialty(_ context.Context, c practitionerrole.Caller, id, code string, version int32) (practitionerrole.PractitionerRole, error) {
	s.called, s.caller, s.id, s.code, s.version = true, c, id, code, version
	return s.out, s.err
}

// putSpecialty sends PUT /practitioner-roles/{roleA1}/specialty with body and If-Match ifMatch ("" for none)
// as adminA, or as nobody when signedIn is false, logging to log.
func putSpecialty(t *testing.T, svc *stubSpecialties, signedIn bool, log *bytes.Buffer, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := practitionerrole.NewHTTPHandler(svc, func(context.Context) (practitionerrole.Caller, bool, error) { return adminA, signedIn, nil },
		slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/practitioner-roles/"+roleA1+"/specialty", strings.NewReader(body))
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const cardiologyBody = `{"code":"394579002"}`

func TestHTTPHandler_SetsTheSpecialtyAtTheVersionOfIfMatch(t *testing.T) {
	updated := role(roleA1, clinicA, "a3000000-0000-4000-8000-0000000000a1")
	updated.SpecialtyCode, updated.VersionID = cardiology, 4
	svc := &stubSpecialties{out: updated}
	rec := putSpecialty(t, svc, true, &bytes.Buffer{}, `W/"3"`, cardiologyBody)

	want := `{"specialty":{"system":"http://snomed.info/sct","code":"394579002","display":"Cardiology","text":"Cardiology"},"id":"` + roleA1 + `","versionId":4}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("PUT = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
	if got := rec.Header().Get("ETag"); got != `W/"4"` {
		t.Errorf("ETag %q, want W/\"4\"", got)
	}
	if svc.caller != adminA || svc.id != roleA1 || svc.code != cardiology || svc.version != 3 {
		t.Errorf("SetSpecialty(%+v, %q, %q, %d), want the caller, the URL's id, the code and version 3", svc.caller, svc.id, svc.code, svc.version)
	}
}

func TestHTTPHandler_RefusesARequestItCannotReadBeforeCallingTheService(t *testing.T) {
	tests := []struct {
		name, ifMatch, body string
	}{
		{name: "no If-Match", body: cardiologyBody},
		{name: "an If-Match that is not a version", ifMatch: `W/"three"`, body: cardiologyBody},
		{name: "an If-Match without quotes", ifMatch: `3`, body: cardiologyBody},
		{name: "no body", ifMatch: `W/"3"`},
		{name: "no code", ifMatch: `W/"3"`, body: `{}`},
		{name: "an unknown field", ifMatch: `W/"3"`, body: `{"code":"394579002","display":"Cardiology"}`},
		{name: "a code that is not a string", ifMatch: `W/"3"`, body: `{"code":394579002}`},
		{name: "two objects", ifMatch: `W/"3"`, body: cardiologyBody + cardiologyBody},
		{name: "a body over 1 KiB", ifMatch: `W/"3"`, body: `{"code":"` + strings.Repeat("1", 1<<10) + `"}`},
	}
	for _, tt := range tests {
		svc := &stubSpecialties{}
		rec := putSpecialty(t, svc, true, &bytes.Buffer{}, tt.ifMatch, tt.body)
		if rec.Code != http.StatusBadRequest || rec.Body.String() != `{"error":"invalid_request"}`+"\n" || svc.called {
			t.Errorf("%s: PUT = %d %s (service called %v), want 400 invalid_request", tt.name, rec.Code, rec.Body, svc.called)
		}
	}
}

func TestHTTPHandler_MapsTheServiceErrors(t *testing.T) {
	tests := []struct {
		err    error
		code   string
		status int
	}{
		{practitionerrole.ErrInvalidSpecialty, "invalid_specialty", http.StatusBadRequest},
		{practitionerrole.ErrForbidden, "forbidden", http.StatusForbidden},
		{practitionerrole.ErrNotFound, "not_found", http.StatusNotFound},
		{practitionerrole.ErrConflict, "version_conflict", http.StatusPreconditionFailed},
	}
	for _, tt := range tests {
		rec := putSpecialty(t, &stubSpecialties{err: tt.err}, true, &bytes.Buffer{}, `W/"3"`, cardiologyBody)
		if rec.Code != tt.status || rec.Body.String() != `{"error":"`+tt.code+`"}`+"\n" {
			t.Errorf("%v: PUT = %d %s, want %d %s", tt.err, rec.Code, rec.Body, tt.status, tt.code)
		}
	}
}

func TestHTTPHandler_AsksForASignInWithoutCallingTheService(t *testing.T) {
	svc := &stubSpecialties{}
	rec := putSpecialty(t, svc, false, &bytes.Buffer{}, `W/"3"`, cardiologyBody)
	if rec.Code != http.StatusUnauthorized || rec.Body.String() != `{"error":"unauthenticated"}`+"\n" || svc.called {
		t.Errorf("PUT signed out = %d %s (service called %v), want 401 unauthenticated", rec.Code, rec.Body, svc.called)
	}
}

func TestHTTPHandler_SendsAFixedErrorAndLogsNoSpecialty(t *testing.T) {
	var logs bytes.Buffer
	rec := putSpecialty(t, &stubSpecialties{err: errors.New("synthetic failure")}, true, &logs, `W/"3"`, cardiologyBody)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "synthetic") {
		t.Errorf("PUT with a failing service = %d %s, want 500 with a fixed message", rec.Code, rec.Body)
	}
	if logs.Len() == 0 || strings.Contains(logs.String(), cardiology) || strings.Contains(logs.String(), "Cardiology") {
		t.Errorf("log = %q, want the error logged without the specialty", logs.String())
	}

	h := practitionerrole.NewHTTPHandler(&stubSpecialties{}, func(context.Context) (practitionerrole.Caller, bool, error) {
		return practitionerrole.Caller{}, false, errors.New("synthetic caller failure")
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec = httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/practitioner-roles/"+roleA1+"/specialty", strings.NewReader(cardiologyBody))
	req.Header.Set("If-Match", `W/"3"`)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("PUT when the caller cannot be resolved = %d, want 500", rec.Code)
	}
}

func TestHTTPHandler_ListsTheSpecialtiesToAnyone(t *testing.T) {
	svc := &stubSpecialties{}
	h := practitionerrole.NewHTTPHandler(svc, func(context.Context) (practitionerrole.Caller, bool, error) {
		return practitionerrole.Caller{}, false, nil
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/specialties", nil))

	var want strings.Builder
	want.WriteString(`{"specialties":[`)
	for i, s := range practitionerrole.Specialties() {
		if i > 0 {
			want.WriteString(",")
		}
		fmt.Fprintf(&want, `{"system":"http://snomed.info/sct","code":%q,"display":%q,"text":%q}`, s.Code, s.Display, s.Text)
	}
	want.WriteString("]}\n")
	if rec.Code != http.StatusOK || rec.Body.String() != want.String() {
		t.Errorf("GET /specialties = %d %s, want 200 %s", rec.Code, rec.Body, want.String())
	}
	if svc.called {
		t.Error("the list called the service; it is the fixed list")
	}
}
