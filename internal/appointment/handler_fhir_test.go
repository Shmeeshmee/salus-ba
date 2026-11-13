package appointment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
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
	filter appointment.Filter
	out    []appointment.Seen
	calls  int
}

func (s *stubReader) Read(_ context.Context, c appointment.Caller, id string) (appointment.Seen, error) {
	s.caller, s.id = c, id
	s.calls++
	if s.err != nil || len(s.out) == 0 {
		return appointment.Seen{}, s.err
	}
	return s.out[0], nil
}

func (s *stubReader) Search(_ context.Context, c appointment.Caller, f appointment.Filter) ([]appointment.Seen, error) {
	s.caller, s.filter = c, f
	s.calls++
	return s.out, s.err
}

func getAppt(t *testing.T, svc *stubReader, caller func(context.Context) (appointment.Caller, bool, error), log *bytes.Buffer, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	appointment.NewHandler(svc, caller, "https://api.example.test/fhir/R4", slog.New(slog.NewJSONHandler(log, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

var (
	fullA = appointment.Seen{Appointment: bookedA, Patient: nameA, PractitionerID: practA1, Level: access.Full}
	// pastB is a past appointment after a revocation, as the clinic sees it: the name, times and status.
	pastB = appointment.Seen{Appointment: appointment.Appointment{
		ID: apptB, PatientID: patientB, PractitionerRoleID: roleA1, OrganizationID: clinicA, Status: appointment.StatusBooked,
		Start: now.Add(-2 * time.Hour), End: now.Add(-90 * time.Minute), VersionID: 1, LastUpdated: now.Add(-24 * time.Hour),
	}, Patient: appointment.Name{Given: "Bo", Family: "Sample"}, PractitionerID: practA1, Level: access.Name}
)

func cancelledA() appointment.Seen {
	s := fullA
	s.Appointment.Status, s.Appointment.Reason, s.Appointment.CancelledAt, s.Appointment.VersionID = appointment.StatusCancelled, appointment.ReasonPatient, now, 2
	s.Appointment.CancelledBy = accountA
	return s
}

func TestHandlerRead_SendsTheAppointmentWithItsParticipantsMetaAndETag(t *testing.T) {
	var log bytes.Buffer
	svc := &stubReader{out: []appointment.Seen{fullA}}

	rec := getAppt(t, svc, as(pract), &log, "/fhir/R4/Appointment/"+apptA)

	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"1"` || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status = %d %s %v, want 200 at version 1, not stored", rec.Code, rec.Body, rec.Header())
	}
	if svc.caller != pract || svc.id != apptA {
		t.Errorf("Read(%+v, %q), want the caller and the id of the URL", svc.caller, svc.id)
	}
	assertGolden(t, "appointment.json", rec.Body.Bytes())
}

// After a revocation the clinic sees a past appointment at the name, times and status only: no slot, no
// created time, no cancellation details.
func TestHandlerRead_SendsAPastAppointmentAtTheNameLevelWithoutItsDetails(t *testing.T) {
	var log bytes.Buffer
	rec := getAppt(t, &stubReader{out: []appointment.Seen{pastB}}, as(pract), &log, "/fhir/R4/Appointment/"+apptB)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "appointment_name_level.json", rec.Body.Bytes())

	noName := pastB
	noName.Patient = appointment.Name{}
	rec = getAppt(t, &stubReader{out: []appointment.Seen{noName}}, as(pract), &log, "/fhir/R4/Appointment/"+apptB)
	if strings.Contains(rec.Body.String(), "display") {
		t.Errorf("without the name = %s, want no display", rec.Body)
	}
}

func TestHandlerSearch_PassesTheFilterAndSendsASearchset(t *testing.T) {
	from, to := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC), time.Date(2030, 1, 14, 0, 0, 0, 0, time.UTC)
	query := "practitioner=Practitioner/" + practA1 + "&date=ge2030-01-07T00:00:00Z&date=" + url.QueryEscape("lt2030-01-14T01:00:00+01:00") +
		"&status=cancelled&patient=Patient/" + patientA
	var log bytes.Buffer
	svc := &stubReader{out: []appointment.Seen{cancelledA()}}

	rec := getAppt(t, svc, as(admin), &log, "/fhir/R4/Appointment?"+query)

	want := appointment.Filter{PractitionerID: practA1, PatientID: patientA, Status: appointment.StatusCancelled, From: from, To: to}
	if rec.Code != http.StatusOK || svc.caller != admin || svc.filter.PractitionerID != want.PractitionerID || svc.filter.PatientID != want.PatientID ||
		svc.filter.Status != want.Status || !svc.filter.From.Equal(from) || !svc.filter.To.Equal(to) || svc.filter.PractitionerRoleID != "" {
		t.Fatalf("status = %d %s after Search(%+v, %+v), want 200 and %+v", rec.Code, rec.Body, svc.caller, svc.filter, want)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}
	assertGolden(t, "search.json", rec.Body.Bytes())

	svc = &stubReader{}
	rec = getAppt(t, svc, as(me), &log, "/fhir/R4/Appointment")
	if rec.Code != http.StatusOK || svc.filter != (appointment.Filter{}) || !strings.Contains(rec.Body.String(), `"total":0`) {
		t.Errorf("no parameters = %d %s after %+v, want an empty filter and searchset", rec.Code, rec.Body, svc.filter)
	}
}

func TestHandlerSearch_RejectsAnUnknownOrRepeatedParameterWithoutCallingTheService(t *testing.T) {
	for _, query := range []string{
		"_count=10", "name=Ada", "patient=Patient/" + patientA + "&patient=Patient/" + patientB, "patient=Practitioner/" + practA1,
		"practitioner=PractitionerRole/" + roleA1, "status=proposed", "status=booked&status=cancelled", "status=booked,cancelled",
		"date=2030-01-07", "date=ge2030-01-07", "date=gt2030-01-07T00:00:00Z", "actor=Patient/" + patientA, "patient=",
	} {
		var log bytes.Buffer
		svc := &stubReader{}

		rec := getAppt(t, svc, as(admin), &log, "/fhir/R4/Appointment?"+query)

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
		{appointment.ErrNotFound, "/fhir/R4/Appointment/" + apptA, "not_found.json", http.StatusNotFound},
		{appointment.ErrForbidden, "/fhir/R4/Appointment?practitioner=" + practA2, "forbidden.json", http.StatusForbidden},
	} {
		var log bytes.Buffer
		rec := getAppt(t, &stubReader{err: tt.err}, as(pract), &log, tt.path)
		if rec.Code != tt.status {
			t.Errorf("GET %s with %v = %d, want %d", tt.path, tt.err, rec.Code, tt.status)
		}
		assertGolden(t, tt.golden, rec.Body.Bytes())
	}
	for _, path := range []string{"/fhir/R4/Appointment/" + apptA, "/fhir/R4/Appointment?patient=" + patientA} {
		var log bytes.Buffer
		svc := &stubReader{}
		rec := getAppt(t, svc, func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, nil }, &log, path)
		if rec.Code != http.StatusUnauthorized || svc.calls != 0 {
			t.Errorf("GET %s signed out = %d after %d calls, want 401 and none", path, rec.Code, svc.calls)
		}
		assertGolden(t, "unauthenticated.json", rec.Body.Bytes())
	}
}

func TestHandler_SendsAFixedOperationOutcomeAndLogsNoName(t *testing.T) {
	failing := errors.New("db down")
	for _, path := range []string{"/fhir/R4/Appointment/" + apptA, "/fhir/R4/Appointment?status=booked"} {
		for name, caller := range map[string]func(context.Context) (appointment.Caller, bool, error){
			"the service": as(admin),
			"the caller":  func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, failing },
		} {
			var log bytes.Buffer
			rec := getAppt(t, &stubReader{err: failing, out: []appointment.Seen{fullA}}, caller, &log, path)
			if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") || log.Len() == 0 {
				t.Errorf("GET %s, %s failing = %d %s, log %q; want 500 without the error, the error logged", path, name, rec.Code, rec.Body, log.String())
			}
			if strings.Contains(log.String(), "Ada") || strings.Contains(log.String(), "Example") {
				t.Errorf("GET %s: the log holds the name: %s", path, log.String())
			}
		}
	}
}

// The patient participant is tentative until they confirm, accepted once they did, and declined once they cancelled;
// the practitioner and the role stay accepted (docs/plans/appointments.md Q3).
func TestHandlerRead_ThePatientParticipantFollowsTheConfirmation(t *testing.T) {
	confirmed := fullA
	confirmed.Appointment.ConfirmedAt, confirmed.Appointment.ConfirmedVia = now, appointment.ViaLink
	byStaff := cancelledA()
	byStaff.Appointment.Reason = appointment.ReasonProvider
	byRevocation := cancelledA()
	byRevocation.Appointment.CancelledBy = "" // the consent_revoked job: reason pat, by no account
	confirmedThenRevoked := byRevocation
	confirmedThenRevoked.Appointment.ConfirmedAt = now
	for _, tt := range []struct {
		name string
		want string
		seen appointment.Seen
	}{
		{"unconfirmed", "tentative", fullA},
		{"confirmed", "accepted", confirmed},
		{"cancelled by the patient", "declined", cancelledA()},
		{"cancelled by staff", "tentative", byStaff},
		{"cancelled by the revocation", "tentative", byRevocation},
		{"confirmed, then cancelled by the revocation", "accepted", confirmedThenRevoked},
	} {
		var log bytes.Buffer
		rec := getAppt(t, &stubReader{out: []appointment.Seen{tt.seen}}, as(pract), &log, "/fhir/R4/Appointment/"+apptA)
		var res struct {
			Participant []struct {
				Status string `json:"status"`
			} `json:"participant"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || len(res.Participant) != 3 {
			t.Fatalf("%s: %s, %v", tt.name, rec.Body, err)
		}
		if got := []string{res.Participant[0].Status, res.Participant[1].Status, res.Participant[2].Status}; !slices.Equal(got, []string{tt.want, "accepted", "accepted"}) {
			t.Errorf("%s: participants %v, want the patient %s and the others accepted", tt.name, got, tt.want)
		}
	}
}
