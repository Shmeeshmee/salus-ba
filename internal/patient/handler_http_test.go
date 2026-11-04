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

// stubProfiles answers with what it is given and keeps what it was asked.
type stubProfiles struct {
	err    error
	caller patient.Caller
	in     patient.Patient
	out    patient.Patient
	calls  []string
}

func (s *stubProfiles) Me(_ context.Context, c patient.Caller) (patient.Patient, error) {
	s.caller = c
	s.calls = append(s.calls, "Me")
	return s.out, s.err
}

func (s *stubProfiles) SaveMe(_ context.Context, c patient.Caller, in patient.Patient) (patient.Patient, error) {
	s.caller, s.in = c, in
	s.calls = append(s.calls, "SaveMe")
	return s.out, s.err
}

var savedA = patient.Patient{ID: patientA, AccountID: accountA, GivenName: "Ada", FamilyName: "Example",
	BirthDate: date(1990, 1, 1), Phone: "+38760000001", VersionID: 2, LastUpdated: now}

const savedJSON = `{"lastUpdated":"2026-11-04T09:30:00Z","id":"` + patientA + `","givenName":"Ada","familyName":"Example","birthDate":"1990-01-01","phone":"+38760000001","versionId":2}` + "\n"

// serve runs one request through the handler as caller (nobody when signedIn is false), logging to log.
func serve(t *testing.T, svc *stubProfiles, caller patient.Caller, signedIn bool, log *bytes.Buffer, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := patient.NewHandler(svc, func(context.Context) (patient.Caller, bool) { return caller, signedIn },
		slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/me/patient", strings.NewReader(body)))
	return rec
}

func TestHandler_GetsTheProfile(t *testing.T) {
	svc := &stubProfiles{out: savedA}
	rec := serve(t, svc, me, true, &bytes.Buffer{}, http.MethodGet, "")
	if rec.Code != http.StatusOK || rec.Body.String() != savedJSON || svc.caller != me {
		t.Errorf("GET = %d %s (caller %+v), want 200 %s", rec.Code, rec.Body, svc.caller, savedJSON)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control %q, want no-store", got)
	}
}

func TestHandler_LeavesOutAPhoneThatIsNotThere(t *testing.T) {
	p := savedA
	p.Phone = ""
	rec := serve(t, &stubProfiles{out: p}, me, true, &bytes.Buffer{}, http.MethodGet, "")
	if strings.Contains(rec.Body.String(), "phone") {
		t.Errorf("GET = %s, want no phone", rec.Body)
	}
}

func TestHandler_CreatesWithoutAVersionAndUpdatesWithOne(t *testing.T) {
	tests := []struct {
		body    string
		version int32
		status  int
	}{
		{body: `{"givenName":"Ada","familyName":"Example","birthDate":"1990-01-01","phone":"+38760000001"}`, status: http.StatusCreated},
		{body: `{"givenName":"Ada","familyName":"Example","birthDate":"1990-01-01","versionId":1}`, version: 1, status: http.StatusOK},
	}
	for _, tt := range tests {
		svc := &stubProfiles{out: savedA}
		rec := serve(t, svc, me, true, &bytes.Buffer{}, http.MethodPut, tt.body)
		if rec.Code != tt.status || rec.Body.String() != savedJSON {
			t.Errorf("PUT %s = %d %s, want %d with the saved profile", tt.body, rec.Code, rec.Body, tt.status)
		}
		want := patient.Patient{GivenName: "Ada", FamilyName: "Example", BirthDate: date(1990, 1, 1), VersionID: tt.version}
		if tt.version == 0 {
			want.Phone = "+38760000001"
		}
		if svc.in != want || svc.caller != me {
			t.Errorf("PUT %s passed %+v as %+v, want %+v as %+v", tt.body, svc.in, svc.caller, want, me)
		}
	}
}

func TestHandler_MapsEveryRefusalToItsCode(t *testing.T) {
	const valid = `{"givenName":"Ada","familyName":"Example","birthDate":"1990-01-01"}`
	tests := []struct {
		err                      error
		name, method, body, want string
		status                   int
		signedOut                bool
	}{
		{name: "signed out GET", method: http.MethodGet, signedOut: true, status: 401, want: "unauthenticated"},
		{name: "signed out PUT", method: http.MethodPut, body: valid, signedOut: true, status: 401, want: "unauthenticated"},
		{name: "no profile", method: http.MethodGet, err: patient.ErrNotFound, status: 404, want: "not_found"},
		{name: "staff GET", method: http.MethodGet, err: patient.ErrForbidden, status: 403, want: "forbidden"},
		{name: "staff PUT", method: http.MethodPut, body: valid, err: patient.ErrForbidden, status: 403, want: "forbidden"},
		{name: "stale version", method: http.MethodPut, body: valid, err: patient.ErrConflict, status: 409, want: "version_conflict"},
		{name: "invalid name", method: http.MethodPut, body: valid, err: patient.ErrInvalidName, status: 400, want: "invalid_name"},
		{name: "invalid birth date", method: http.MethodPut, body: valid, err: patient.ErrInvalidBirthDate, status: 400, want: "invalid_birth_date"},
		{name: "invalid phone", method: http.MethodPut, body: valid, err: patient.ErrInvalidPhone, status: 400, want: "invalid_phone"},
		{name: "not JSON", method: http.MethodPut, body: `{`, status: 400, want: "invalid_request"},
		{name: "an unknown field", method: http.MethodPut, body: `{"givenName":"Ada","gender":"x"}`, status: 400, want: "invalid_request"},
		{name: "two objects", method: http.MethodPut, body: valid + valid, status: 400, want: "invalid_request"},
		{name: "a birth date with a time", method: http.MethodPut, body: `{"givenName":"Ada","familyName":"Example","birthDate":"1990-01-01T00:00:00Z"}`, status: 400, want: "invalid_birth_date"},
		{name: "no birth date", method: http.MethodPut, body: `{"givenName":"Ada","familyName":"Example"}`, status: 400, want: "invalid_birth_date"},
		{name: "a failure", method: http.MethodGet, err: errors.New("connection reset"), status: 500, want: "internal error"},
	}
	for _, tt := range tests {
		svc := &stubProfiles{err: tt.err}
		rec := serve(t, svc, me, !tt.signedOut, &bytes.Buffer{}, tt.method, tt.body)
		if want := `{"error":"` + tt.want + `"}` + "\n"; rec.Code != tt.status || rec.Body.String() != want {
			t.Errorf("%s: %d %s, want %d %s", tt.name, rec.Code, rec.Body, tt.status, want)
		}
		if tt.err == nil && len(svc.calls) != 0 {
			t.Errorf("%s: the service was called (%v) for a request refused by the handler", tt.name, svc.calls)
		}
	}
}

func TestHandler_LogsNoProfileField(t *testing.T) {
	var log bytes.Buffer
	body := `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07","phone":"+38760000777"}`
	for _, err := range []error{nil, errors.New("connection reset"), patient.ErrConflict} {
		svc := &stubProfiles{err: err, out: patient.Patient{ID: patientA, GivenName: "Zorana", FamilyName: "Syntheticova",
			BirthDate: date(1977, 7, 7), Phone: "+38760000777", VersionID: 1, LastUpdated: now}}
		serve(t, svc, me, true, &log, http.MethodPut, body)
		serve(t, svc, me, true, &log, http.MethodGet, "")
		serve(t, svc, me, true, &log, http.MethodPut, `{"givenName":"Zorana","birthDate":"1977-07-07x"}`)
	}
	if !strings.Contains(log.String(), "internal error") {
		t.Fatalf("log %q, want the internal errors logged", log.String())
	}
	for _, field := range []string{"Zorana", "Syntheticova", "1977", "+38760000777", "38760000777"} {
		if strings.Contains(log.String(), field) {
			t.Errorf("log holds %q:\n%s", field, log.String())
		}
	}
}
