package patient_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/patient"
)

// stubClinic answers with what it is given and keeps what it was asked.
type stubClinic struct {
	err    error
	search patient.ClinicSearch
	asked  string // the patient id a rights route named
	in     patient.Patient
	caller patient.Caller
	out    []patient.Patient
}

func (s *stubClinic) SearchClinic(_ context.Context, c patient.Caller, q patient.ClinicSearch) ([]patient.Patient, error) {
	s.caller, s.search = c, q
	return s.out, s.err
}

func (s *stubClinic) Import(context.Context, patient.Caller, []byte, bool) ([]patient.ImportRow, error) {
	return nil, s.err
}

func (s *stubClinic) Register(_ context.Context, c patient.Caller, in patient.Patient) (patient.Patient, error) {
	s.caller, s.in = c, in
	if len(s.out) == 0 {
		return patient.Patient{}, s.err
	}
	return s.out[0], s.err
}

var registeredC = patient.Patient{ID: registeredID, RegisteredBy: clinicA, GivenName: "Cy", FamilyName: "Example",
	BirthDate: date(1980, 3, 4), Phone: "+38761000002", VersionID: 1, LastUpdated: now}

const registeredJSON = `{"lastUpdated":"2026-11-04T09:30:00Z","id":"` + registeredID + `","givenName":"Cy","familyName":"Example","birthDate":"1980-03-04","phone":"+38761000002","registeredByClinic":true,"consentActive":false,"versionId":1}`

func serveClinic(t *testing.T, svc *stubClinic, signedIn bool, log *bytes.Buffer, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := patient.NewClinicHandler(svc, func(context.Context) (patient.Caller, bool, error) { return admin, signedIn, nil },
		slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body)))
	return rec
}

func TestClinicHandler_SearchesByNameAndBirthDate(t *testing.T) {
	t.Parallel()
	svc := &stubClinic{out: []patient.Patient{registeredC}}
	rec := serveClinic(t, svc, true, &bytes.Buffer{}, http.MethodGet, "/clinic/patients?name=Cy&birthDate=1980-03-04", "")
	if rec.Code != http.StatusOK || rec.Body.String() != "["+registeredJSON+"]\n" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET = %d %s (%s), want 200 [%s] no-store", rec.Code, rec.Body, rec.Header().Get("Cache-Control"), registeredJSON)
	}
	if svc.caller != admin || svc.search != (patient.ClinicSearch{Name: "Cy", BirthDate: date(1980, 3, 4)}) {
		t.Errorf("asked %+v as %+v", svc.search, svc.caller)
	}
	empty := &stubClinic{}
	if rec := serveClinic(t, empty, true, &bytes.Buffer{}, http.MethodGet, "/clinic/patients?name=Nobody", ""); rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Errorf("no match = %d %s, want 200 []", rec.Code, rec.Body)
	}
}

func TestClinicHandler_RegistersWithTheConsentGivenInPerson(t *testing.T) {
	t.Parallel()
	svc := &stubClinic{out: []patient.Patient{registeredC}}
	body := `{"givenName":"Cy","familyName":"Example","birthDate":"1980-03-04","phone":"+38761000002","consentGivenInPerson":true}`
	rec := serveClinic(t, svc, true, &bytes.Buffer{}, http.MethodPost, "/clinic/patients", body)
	if rec.Code != http.StatusCreated || rec.Body.String() != registeredJSON+"\n" || rec.Header().Get("ETag") != `W/"1"` {
		t.Errorf("POST = %d %s (ETag %s), want 201 %s W/\"1\"", rec.Code, rec.Body, rec.Header().Get("ETag"), registeredJSON)
	}
	want := patient.Patient{GivenName: "Cy", FamilyName: "Example", BirthDate: date(1980, 3, 4), Phone: "+38761000002"}
	if svc.in != want {
		t.Errorf("registered %+v, want %+v", svc.in, want)
	}
}

func TestClinicHandler_AnswersEachRefusalWithItsCodeAndLogsNoField(t *testing.T) {
	t.Parallel()
	const valid = `{"givenName":"Cy","familyName":"Example","birthDate":"1980-03-04","consentGivenInPerson":true}`
	for name, tc := range map[string]struct {
		err      error
		method   string
		target   string
		body     string
		code     string
		status   int
		signedIn bool
	}{
		"signed out":          {err: nil, signedIn: false, method: http.MethodGet, target: "/clinic/patients?name=Cy", body: "", status: 401, code: "unauthenticated"},
		"no name":             {err: nil, signedIn: true, method: http.MethodGet, target: "/clinic/patients", body: "", status: 400, code: "invalid_request"},
		"a bad birth date":    {err: nil, signedIn: true, method: http.MethodGet, target: "/clinic/patients?name=Cy&birthDate=4.3.1980", body: "", status: 400, code: "invalid_request"},
		"a short name":        {err: patient.ErrInvalidSearch, signedIn: true, method: http.MethodGet, target: "/clinic/patients?name=C", body: "", status: 400, code: "invalid_request"},
		"forbidden":           {err: patient.ErrForbidden, signedIn: true, method: http.MethodPost, target: "/clinic/patients", body: valid, status: 403, code: "forbidden"},
		"no consent":          {err: nil, signedIn: true, method: http.MethodPost, target: "/clinic/patients", body: strings.Replace(valid, "true", "false", 1), status: 400, code: "invalid_request"},
		"consent missing":     {err: nil, signedIn: true, method: http.MethodPost, target: "/clinic/patients", body: `{"givenName":"Cy","familyName":"Example","birthDate":"1980-03-04"}`, status: 400, code: "invalid_request"},
		"an unknown field":    {err: nil, signedIn: true, method: http.MethodPost, target: "/clinic/patients", body: strings.Replace(valid, "}", `,"email":"x"}`, 1), status: 400, code: "invalid_request"},
		"a bad register date": {err: nil, signedIn: true, method: http.MethodPost, target: "/clinic/patients", body: strings.Replace(valid, "1980-03-04", "x", 1), status: 400, code: "invalid_birth_date"},
		"a bad phone":         {err: patient.ErrInvalidPhone, signedIn: true, method: http.MethodPost, target: "/clinic/patients", body: valid, status: 400, code: "invalid_phone"},
		"an internal error":   {err: errors.New("boom Cy Example"), signedIn: true, method: http.MethodPost, target: "/clinic/patients", body: valid, status: 500, code: ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var log bytes.Buffer
			rec := serveClinic(t, &stubClinic{err: tc.err}, tc.signedIn, &log, tc.method, tc.target, tc.body)
			if rec.Code != tc.status || (tc.code != "" && rec.Body.String() != `{"error":"`+tc.code+`"}`+"\n") {
				t.Errorf("= %d %s, want %d %s", rec.Code, rec.Body, tc.status, tc.code)
			}
			if strings.Contains(rec.Body.String(), "Cy") {
				t.Errorf("the answer carries a name: %s", rec.Body)
			}
		})
	}
}
