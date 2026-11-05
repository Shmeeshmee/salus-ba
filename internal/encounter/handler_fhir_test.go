package encounter_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// assertGolden compares got with testdata/<name>; -update rewrites the file.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("%s differs from the response:\n got: %s\nwant: %s", path, got, want)
	}
}

// stubReader answers with what it is given and keeps what it was asked.
type stubReader struct {
	err    error
	caller appointment.Caller
	id     string
	filter encounter.Filter
	out    []encounter.Encounter
	calls  int
}

func (s *stubReader) Read(_ context.Context, c appointment.Caller, id string) (encounter.Encounter, error) {
	s.caller, s.id = c, id
	s.calls++
	if s.err != nil || len(s.out) == 0 {
		return encounter.Encounter{}, s.err
	}
	return s.out[0], nil
}

func (s *stubReader) Search(_ context.Context, c appointment.Caller, f encounter.Filter) ([]encounter.Encounter, error) {
	s.caller, s.filter = c, f
	s.calls++
	return s.out, s.err
}

func as(c appointment.Caller) func(context.Context) (appointment.Caller, bool, error) {
	return func(context.Context) (appointment.Caller, bool, error) { return c, true, nil }
}

func getEnc(t *testing.T, svc *stubReader, caller func(context.Context) (appointment.Caller, bool, error), log *bytes.Buffer, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	encounter.NewHandler(svc, caller, "https://api.example.test/fhir/R4", slog.New(slog.NewJSONHandler(log, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

const (
	encA  = "ec000000-0000-4000-8000-0000000000a1"
	encA2 = "ec000000-0000-4000-8000-0000000000a2"
)

var (
	inProgress = encounter.Encounter{
		ID: encA, AppointmentID: apptA, OrganizationID: clinicA, PatientID: patientA, PractitionerRoleID: roleA1,
		Status: encounter.StatusInProgress, PeriodStart: joined, VersionID: 1, LastUpdated: joined,
	}
	finished = encounter.Encounter{
		ID: encA2, AppointmentID: apptA2, OrganizationID: clinicA, PatientID: patientA, PractitionerRoleID: roleA2,
		Status: encounter.StatusFinished, PeriodStart: joined, PeriodEnd: joined.Add(25 * time.Minute), VersionID: 2,
		LastUpdated: joined.Add(25 * time.Minute),
	}
)

// The read sends the Encounter (class VR, the patient, the practitioner role as primary performer, the
// appointment, the period, the clinic) with its version as the ETag, not to be stored.
func TestHandlerRead_SendsTheEncounter(t *testing.T) {
	for _, tt := range []struct {
		golden, etag string
		e            encounter.Encounter
	}{
		{golden: "encounter_in_progress.json", etag: `W/"1"`, e: inProgress},
		{golden: "encounter_finished.json", etag: `W/"2"`, e: finished},
	} {
		var log bytes.Buffer
		svc := &stubReader{out: []encounter.Encounter{tt.e}}
		rec := getEnc(t, svc, as(pract), &log, "/fhir/R4/Encounter/"+tt.e.ID)
		if rec.Code != http.StatusOK || rec.Header().Get("ETag") != tt.etag || rec.Header().Get("Cache-Control") != "no-store" ||
			rec.Header().Get("Content-Type") != "application/fhir+json" {
			t.Fatalf("status = %d %s %v, want 200 with %s, not stored", rec.Code, rec.Body, rec.Header(), tt.etag)
		}
		if svc.caller != pract || svc.id != tt.e.ID {
			t.Errorf("Read(%+v, %q), want the caller and the id of the URL", svc.caller, svc.id)
		}
		assertGolden(t, tt.golden, rec.Body.Bytes())
	}
}

// The search passes the appointment, the patient and the practitioner (as references or bare ids) and sends
// a searchset with its self link; no parameters search everything the caller may see.
func TestHandlerSearch_PassesTheFilterAndSendsASearchset(t *testing.T) {
	var log bytes.Buffer
	svc := &stubReader{out: []encounter.Encounter{inProgress, finished}}
	rec := getEnc(t, svc, as(admin), &log, "/fhir/R4/Encounter?patient=Patient/"+patientA+"&practitioner="+practA1+"&appointment=Appointment/"+apptA)
	want := encounter.Filter{AppointmentID: apptA, PatientID: patientA, PractitionerID: practA1}
	if rec.Code != http.StatusOK || svc.caller != admin || svc.filter != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status = %d %s after Search(%+v, %+v), want 200 and %+v", rec.Code, rec.Body, svc.caller, svc.filter, want)
	}
	assertGolden(t, "search.json", rec.Body.Bytes())

	svc = &stubReader{}
	rec = getEnc(t, svc, as(me), &log, "/fhir/R4/Encounter")
	if rec.Code != http.StatusOK || svc.filter != (encounter.Filter{}) || !strings.Contains(rec.Body.String(), `"total":0`) {
		t.Errorf("no parameters = %d %s after %+v, want an empty filter and searchset", rec.Code, rec.Body, svc.filter)
	}
}

func TestHandlerSearch_RejectsAnUnknownOrRepeatedParameterWithoutCallingTheService(t *testing.T) {
	for _, query := range []string{
		"_count=10", "status=finished", "date=ge2030-01-07T00:00:00Z", "subject=Patient/" + patientA,
		"patient=Patient/" + patientA + "&patient=Patient/" + patientB, "patient=Practitioner/" + practA1, "patient=",
		"practitioner=PractitionerRole/" + roleA1, "appointment=Encounter/" + encA, "appointment=" + apptA + "," + apptA2,
		"participant=Practitioner/" + practA1, "%zz",
	} {
		var log bytes.Buffer
		svc := &stubReader{}
		rec := getEnc(t, svc, as(admin), &log, "/fhir/R4/Encounter?"+query)
		if rec.Code != http.StatusBadRequest || svc.calls != 0 {
			t.Errorf("?%s: status = %d after %d calls, want 400 and none", query, rec.Code, svc.calls)
		}
		assertGolden(t, "unsupported_search.json", rec.Body.Bytes())
	}
}

func TestHandler_MapsTheServiceErrorsAndAsksForASignIn(t *testing.T) {
	for _, tt := range []struct {
		err    error
		path   string
		golden string
		status int
	}{
		{encounter.ErrNotFound, "/fhir/R4/Encounter/" + encA, "not_found.json", http.StatusNotFound},
		{encounter.ErrForbidden, "/fhir/R4/Encounter?practitioner=" + practA2, "forbidden.json", http.StatusForbidden},
	} {
		var log bytes.Buffer
		rec := getEnc(t, &stubReader{err: tt.err}, as(pract), &log, tt.path)
		if rec.Code != tt.status {
			t.Errorf("GET %s with %v = %d, want %d", tt.path, tt.err, rec.Code, tt.status)
		}
		assertGolden(t, tt.golden, rec.Body.Bytes())
	}
	for _, path := range []string{"/fhir/R4/Encounter/" + encA, "/fhir/R4/Encounter?patient=" + patientA} {
		var log bytes.Buffer
		svc := &stubReader{}
		rec := getEnc(t, svc, func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, nil }, &log, path)
		if rec.Code != http.StatusUnauthorized || svc.calls != 0 {
			t.Errorf("GET %s signed out = %d after %d calls, want 401 and none", path, rec.Code, svc.calls)
		}
		assertGolden(t, "unauthenticated.json", rec.Body.Bytes())
	}
}

func TestHandler_SendsAFixedOperationOutcomeAndLogsTheErrorOnly(t *testing.T) {
	failing := errors.New("db down")
	for _, path := range []string{"/fhir/R4/Encounter/" + encA, "/fhir/R4/Encounter?patient=" + patientA} {
		for name, caller := range map[string]func(context.Context) (appointment.Caller, bool, error){
			"the service": as(admin),
			"the caller":  func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, failing },
		} {
			var log bytes.Buffer
			rec := getEnc(t, &stubReader{err: failing}, caller, &log, path)
			if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") || !strings.Contains(log.String(), "db down") {
				t.Errorf("GET %s, %s failing = %d %s, log %q; want 500 without the error, the error logged", path, name, rec.Code, rec.Body, log.String())
			}
			if strings.Contains(log.String(), patientA) {
				t.Errorf("GET %s: the log holds the patient's id from the query: %s", path, log.String())
			}
		}
	}
}
