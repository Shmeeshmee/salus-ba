package diagnosticreport_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
)

// stubReports answers every call with out and err, and keeps the draft and version it was given.
type stubReports struct {
	err     error
	draft   diagnosticreport.Draft
	out     diagnosticreport.Report
	version int
}

func (s *stubReports) Attach(_ context.Context, _ diagnosticreport.Staff, _ string, d diagnosticreport.Draft) (diagnosticreport.Report, error) {
	s.draft = d
	return s.out, s.err
}

func (s *stubReports) Change(_ context.Context, _ diagnosticreport.Staff, _ string, d diagnosticreport.Draft) (diagnosticreport.Report, error) {
	s.draft = d
	return s.out, s.err
}

func (s *stubReports) Release(_ context.Context, _ diagnosticreport.Staff, _ string, v int) (diagnosticreport.Report, error) {
	s.version = v
	return s.out, s.err
}

func (s *stubReports) ForVisit(context.Context, diagnosticreport.Staff, string) ([]diagnosticreport.Report, error) {
	return []diagnosticreport.Report{s.out}, s.err
}

func (s *stubReports) Read(context.Context, diagnosticreport.Staff, string) (diagnosticreport.Report, error) {
	return s.out, s.err
}

func (s *stubReports) Mine(context.Context, diagnosticreport.Patient) ([]diagnosticreport.Report, error) {
	return []diagnosticreport.Report{s.out}, s.err
}

func (s *stubReports) MyReport(context.Context, diagnosticreport.Patient, string) (diagnosticreport.Report, error) {
	return s.out, s.err
}

func serve(t *testing.T, svc *stubReports, staff bool, method, target, body string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	session := diagnosticreport.Session{Patient: thePatient}
	if staff {
		session = diagnosticreport.Session{Staff: thePractitioner}
	}
	h := diagnosticreport.NewHTTPHandler(svc, func(context.Context) (diagnosticreport.Session, bool, error) { return session, true, nil },
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandler_AttachEncodesTheReport(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	svc := &stubReports{out: diagnosticreport.Report{
		CreatedAt: now, ID: "00000000-0000-4000-8000-000000000099", OrganizationID: clinic, PatientID: patientID, AppointmentID: visitID,
		Status: diagnosticreport.StatusPreliminary, Title: "Complete blood count", EffectiveDate: &day, VersionID: 1,
		Results: []diagnosticreport.Result{{Test: "Haemoglobin", Value: ptr(13.5), Unit: "g/dL", Interpretation: "N"}},
	}}
	rec := serve(t, svc, true, "POST", "/clinic/appointments/"+visitID+"/diagnostic-reports",
		`{"title":"Complete blood count","effectiveDate":"2026-11-20","results":[{"test":"Haemoglobin","value":13.5,"unit":"g/dL","interpretation":"N"}]}`, nil)
	if rec.Code != http.StatusCreated || rec.Header().Get("ETag") != `W/"1"` {
		t.Fatalf("%d %s %s", rec.Code, rec.Header(), rec.Body)
	}
	want := `{"createdAt":"2026-11-25T09:00:00Z","id":"00000000-0000-4000-8000-000000000099","organizationId":"` + clinic +
		`","patientId":"` + patientID + `","appointmentId":"` + visitID + `","status":"preliminary","title":"Complete blood count",` +
		`"effectiveDate":"2026-11-20","results":[{"value":13.5,"test":"Haemoglobin","unit":"g/dL","interpretation":"N"}],"versionId":1,"hasPdf":false}`
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body\n%s\nwant\n%s", got, want)
	}
	if svc.draft.EffectiveDate == nil || !svc.draft.EffectiveDate.Equal(day) || *svc.draft.Results[0].Value != 13.5 {
		t.Errorf("draft %+v", svc.draft)
	}
}

func TestHandler_Answers(t *testing.T) {
	t.Parallel()
	report := "/clinic/diagnostic-reports/" + visitID
	ifMatch := map[string]string{"If-Match": `W/"2"`}
	cases := []struct {
		err                              error
		header                           map[string]string
		name, method, target, body, code string
		status                           int
		staff                            bool
	}{
		{nil, nil, "an unknown field", "POST", "/clinic/appointments/" + visitID + "/diagnostic-reports", `{"title":"T","results":[],"x":1}`, "invalid_request", 400, true},
		{nil, nil, "no results", "POST", "/clinic/appointments/" + visitID + "/diagnostic-reports", `{"title":"T"}`, "invalid_request", 400, true},
		{nil, nil, "a bad date", "POST", "/clinic/appointments/" + visitID + "/diagnostic-reports", `{"title":"T","results":[],"effectiveDate":"20.11.2026"}`, "invalid_request", 400, true},
		{diagnosticreport.ErrInvalid, nil, "invalid", "POST", "/clinic/appointments/" + visitID + "/diagnostic-reports", `{"title":"","results":[]}`, "invalid_diagnostic_report", 400, true},
		{diagnosticreport.ErrForbidden, nil, "a receptionist attaches", "POST", "/clinic/appointments/" + visitID + "/diagnostic-reports", `{"title":"T","results":[]}`, "forbidden", 403, true},
		{diagnosticreport.ErrForbidden, nil, "a receptionist lists", "GET", "/clinic/appointments/" + visitID + "/diagnostic-reports", "", "not_found", 404, true},
		{nil, nil, "a malformed id", "GET", "/clinic/diagnostic-reports/x", "", "not_found", 404, true},
		{nil, nil, "a patient on a clinic route", "GET", report, "", "not_found", 404, false},
		{nil, nil, "no If-Match", "PUT", report, `{"title":"T","results":[]}`, "invalid_request", 400, true},
		{diagnosticreport.ErrReleased, ifMatch, "released", "PUT", report, `{"title":"T","results":[]}`, "report_released", 409, true},
		{diagnosticreport.ErrVersionConflict, ifMatch, "stale", "POST", report + "/release", "", "version_conflict", 412, true},
		{nil, nil, "release without If-Match", "POST", report + "/release", "", "invalid_request", 400, true},
		{diagnosticreport.ErrNotFound, nil, "not the patient's", "GET", "/me/diagnostic-reports/" + visitID, "", "not_found", 404, false},
		{context.DeadlineExceeded, nil, "a fault", "GET", "/me/diagnostic-reports", "", "", 500, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := serve(t, &stubReports{err: tc.err}, tc.staff, tc.method, tc.target, tc.body, tc.header)
			if rec.Code != tc.status || (tc.code != "" && !strings.Contains(rec.Body.String(), `"`+tc.code+`"`)) {
				t.Errorf("%d %s, want %d %s", rec.Code, rec.Body, tc.status, tc.code)
			}
		})
	}
}

func TestHandler_ReleaseAndReads(t *testing.T) {
	t.Parallel()
	svc := &stubReports{out: diagnosticreport.Report{ID: visitID, VersionID: 3, Status: diagnosticreport.StatusFinal}}
	rec := serve(t, svc, true, "POST", "/clinic/diagnostic-reports/"+visitID+"/release", "", map[string]string{"If-Match": `W/"2"`})
	if rec.Code != 200 || svc.version != 2 || rec.Header().Get("ETag") != `W/"3"` {
		t.Fatalf("release %d %d %s", rec.Code, svc.version, rec.Header())
	}
	for _, target := range []string{"/me/diagnostic-reports", "/me/diagnostic-reports/" + visitID} {
		rec := serve(t, svc, false, "GET", target, "", nil)
		if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" || !strings.Contains(rec.Body.String(), `"status":"final"`) {
			t.Errorf("%s: %d %s", target, rec.Code, rec.Body)
		}
	}
	for _, target := range []string{"/clinic/appointments/" + visitID + "/diagnostic-reports", "/clinic/diagnostic-reports/" + visitID} {
		if rec := serve(t, svc, true, "GET", target, "", nil); rec.Code != 200 {
			t.Errorf("%s: %d %s", target, rec.Code, rec.Body)
		}
	}
	d := `{"title":"T","results":[{"test":"A","valueText":"negative"}]}`
	if rec := serve(t, svc, true, "PUT", "/clinic/diagnostic-reports/"+visitID, d, map[string]string{"If-Match": `W/"2"`}); rec.Code != 200 || svc.draft.Version != 2 {
		t.Errorf("change: %d %+v", rec.Code, svc.draft)
	}
}
