package clinicexport_test

import (
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

type stubExports struct {
	err    error
	body   string
	ids    []string
	record clinicexport.Record
	export clinicexport.Export
}

func (s *stubExports) PatientRecord(_ context.Context, _ clinicexport.Caller, id string) (clinicexport.Record, error) {
	s.ids = append(s.ids, id)
	return s.record, s.err
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

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// The hand-over's download: the record as a FHIR collection to save, named by the patient's id (docs/clinic-cms R2),
// now with the patient's visit and receipt at the clinic. Validated against R4 by make fhir-validate.
func TestHandler_SendsAPatientsRecordAsAFHIRCollectionToSave(t *testing.T) {
	t.Parallel()
	const id = "d8000000-0000-4000-8000-0000000000d8"
	const clinic = "a0000000-0000-4000-8000-00000000000a"
	at := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)
	start := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	stub := &stubExports{record: clinicexport.Record{
		Patient: patient.ClinicRecord{
			Patient: patient.Patient{ID: id, GivenName: "Cy", FamilyName: "Example", BirthDate: time.Date(1980, 3, 4, 0, 0, 0, 0, time.UTC),
				Phone: "+38761000002", RegisteredBy: clinic, VersionID: 1, LastUpdated: at},
			Consents: []consent.Consent{{ID: "f4000000-0000-4000-8000-0000000000f4", PatientID: id, OrganizationID: clinic, Kind: consent.KindCare,
				Status: consent.StatusInactive, DateTime: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), RevokedAt: at, LastUpdated: at, VersionID: 2, GivenInPerson: true}},
		},
		Visits: []appointment.Seen{{PractitionerID: "c1000000-0000-4000-8000-0000000000c1", Appointment: appointment.Appointment{
			ID: "a1000000-0000-4000-8000-0000000000a1", PatientID: id, OrganizationID: clinic, PractitionerRoleID: "b1000000-0000-4000-8000-0000000000b1",
			SlotID: "e1000000-0000-4000-8000-0000000000e1", Start: start, End: start.Add(30 * time.Minute), Status: appointment.StatusFulfilled,
			Mode: schedule.ModeInPerson, Created: at, VersionID: 3, LastUpdated: start.Add(time.Hour)}}},
		Receipts: []invoice.Invoice{{ID: "e2000000-0000-4000-8000-0000000000e2", OrganizationID: clinic, PatientID: id, Number: 7,
			IssuedAt: start.Add(45 * time.Minute), TotalMinor: 5000, Currency: "BAM"}},
	}}
	rec := serve(t, stub, true, http.MethodGet, "/clinic/patients/"+id+"/record")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/fhir+json" || rec.Header().Get("Cache-Control") != "no-store" ||
		rec.Header().Get("Content-Disposition") != `attachment; filename="patient-`+id+`.json"` {
		t.Fatalf("GET = %d %v, want 200 a FHIR attachment named by the id", rec.Code, rec.Header())
	}
	if len(stub.ids) != 1 || stub.ids[0] != id {
		t.Errorf("asked for %v", stub.ids)
	}
	path := filepath.Join("testdata", "clinic_record.json")
	if *update {
		if err := os.WriteFile(path, rec.Body.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil || rec.Body.String() != string(want) {
		t.Errorf("%s differs (%v):\n got: %s\nwant: %s", path, err, rec.Body, want)
	}
}

func TestHandler_AnswersARecordsRefusalWithItsCode(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		err    error
		code   string
		status int
	}{
		"not the admin":    {err: patient.ErrForbidden, status: http.StatusForbidden, code: "forbidden"},
		"not the clinic's": {err: patient.ErrNotFound, status: http.StatusNotFound, code: "not_found"},
		"an error":         {err: errors.New("boom Cy Example"), status: http.StatusInternalServerError},
	} {
		rec := serve(t, &stubExports{err: tc.err}, true, http.MethodGet, "/clinic/patients/pt1/record")
		if rec.Code != tc.status || (tc.code != "" && strings.TrimSpace(rec.Body.String()) != `{"error":"`+tc.code+`"}`) || strings.Contains(rec.Body.String(), "Cy") {
			t.Errorf("%s: %d %s, want %d %s", name, rec.Code, rec.Body, tc.status, tc.code)
		}
	}
}
