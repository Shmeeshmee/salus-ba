package patient_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/patient"
)

// importStub is the clinic stub with Import's answer.
type importStub struct {
	stubClinic
	data []byte
	rows []patient.ImportRow
	save bool
}

func (s *importStub) Import(_ context.Context, c patient.Caller, data []byte, save bool) ([]patient.ImportRow, error) {
	s.caller, s.data, s.save = c, data, save
	return s.rows, s.err
}

func serveImport(t *testing.T, svc *importStub, log *bytes.Buffer, query string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	h := patient.NewClinicHandler(svc, func(context.Context) (patient.Caller, bool, error) { return admin, true, nil }, slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	w, err := mw.CreateFormFile("file", "patients.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/clinic/patients/import"+query, &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

var (
	validRow = patient.ImportRow{Line: 2, Duplicate: true, Patient: patient.Patient{GivenName: "Cy", FamilyName: "Sample", BirthDate: date(1980, 3, 4), Phone: "+38761000002"}}
	wrongRow = patient.ImportRow{Line: 3, Errors: []string{"birth_date_invalid"}, Patient: patient.Patient{GivenName: "Di", FamilyName: "Sample"}}
)

func TestClinicHandler_ImportsAFileAndSaysWhetherItWasSaved(t *testing.T) {
	t.Parallel()
	saved := validRow
	saved.Patient.ID = registeredID
	for name, tc := range map[string]struct {
		query string
		want  string
		rows  []patient.ImportRow
		save  bool
	}{
		"a preview": {"", `{"rows":[{"patient":{"givenName":"Cy","familyName":"Sample","birthDate":"1980-03-04","phone":"+38761000002"},"errors":[],"line":2,"duplicate":true},` +
			`{"errors":["birth_date_invalid"],"line":3,"duplicate":false}],"saved":false}` + "\n", []patient.ImportRow{validRow, wrongRow}, false},
		"saved": {"?save=true", `{"rows":[{"patient":{"id":"` + registeredID + `","givenName":"Cy","familyName":"Sample","birthDate":"1980-03-04","phone":"+38761000002"},"errors":[],"line":2,"duplicate":true}],"saved":true}` + "\n",
			[]patient.ImportRow{saved}, true},
		"an error, not saved": {"?save=true", `"saved":false`, []patient.ImportRow{validRow, wrongRow}, true},
	} {
		svc := &importStub{rows: tc.rows}
		rec := serveImport(t, svc, &bytes.Buffer{}, tc.query, []byte("ime;prezime\n"))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tc.want) || svc.save != tc.save || string(svc.data) != "ime;prezime\n" ||
			svc.caller != admin || rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: %d %s, save %v; want 200 %s", name, rec.Code, rec.Body, svc.save, tc.want)
		}
	}
}

func TestClinicHandler_RefusesAnImportWithItsCodeAndLogsNoRow(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		err    error
		query  string
		code   string
		data   []byte
		status int
	}{
		"unreadable":     {err: patient.ErrInvalidImport, status: 400, code: "invalid_import"},
		"forbidden":      {err: patient.ErrForbidden, status: 403, code: "forbidden"},
		"a bad save":     {query: "?save=maybe", status: 400, code: "invalid_request"},
		"too large":      {data: bytes.Repeat([]byte("x"), 3<<20), status: 413, code: "too_large"},
		"internal error": {err: errors.New("boom Cy Sample"), status: 500},
	} {
		var log bytes.Buffer
		rec := serveImport(t, &importStub{err: tc.err}, &log, tc.query, tc.data)
		if rec.Code != tc.status || (tc.code != "" && rec.Body.String() != `{"error":"`+tc.code+`"}`+"\n") {
			t.Errorf("%s: = %d %s, want %d %s", name, rec.Code, rec.Body, tc.status, tc.code)
		}
		if strings.Contains(rec.Body.String(), "Cy") {
			t.Errorf("%s: the answer carries a name: %s", name, rec.Body)
		}
	}
	h := patient.NewClinicHandler(&importStub{}, func(context.Context) (patient.Caller, bool, error) { return patient.Caller{}, false, nil }, slog.New(slog.DiscardHandler))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/clinic/patients/import", strings.NewReader("")))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out = %d, want 401", rec.Code)
	}
	rec = serveImport(t, &importStub{}, &bytes.Buffer{}, "", nil)
	if rec.Code != http.StatusOK { // an empty part is the service's to refuse
		t.Errorf("an empty file = %d, want the service's answer", rec.Code)
	}
}
