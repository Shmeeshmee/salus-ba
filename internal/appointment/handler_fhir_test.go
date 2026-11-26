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
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/fhir"
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
	count  int  // the count SearchPage was asked
	more   bool // SearchPage's answer: more follow
}

func (s *stubReader) Read(_ context.Context, c appointment.Caller, id string) (appointment.Seen, error) {
	s.caller, s.id = c, id
	s.calls++
	if s.err != nil || len(s.out) == 0 {
		return appointment.Seen{}, s.err
	}
	return s.out[0], nil
}

func (s *stubReader) SearchPage(_ context.Context, c appointment.Caller, f appointment.Filter, count int) (appointment.Page, error) {
	s.caller, s.filter, s.count = c, f, count
	s.calls++
	return appointment.Page{Seen: s.out, More: s.more}, s.err
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
		Mode: schedule.ModeInPerson,
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

// The desk's arrival (W6) is a status a search may ask for, as FHIR R4's Appointment.status names it.
func TestHandlerSearch_TakesTheArrivedStatus(t *testing.T) {
	var log bytes.Buffer
	svc := &stubReader{}
	if rec := getAppt(t, svc, as(admin), &log, "/fhir/R4/Appointment?status=arrived"); rec.Code != http.StatusOK || svc.filter.Status != appointment.StatusArrived {
		t.Errorf("status=arrived = %d %s with filter %+v, want 200 searching arrived visits", rec.Code, rec.Body, svc.filter)
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
		"name=Ada", "patient=Patient/" + patientA + "&patient=Patient/" + patientB, "patient=Practitioner/" + practA1,
		"practitioner=PractitionerRole/" + roleA1, "status=proposed", "status=booked&status=cancelled", "status=booked,cancelled",
		"date=2030-01-07", "date=ge2030-01-07", "date=gt2030-01-07T00:00:00Z", "actor=Patient/" + patientA, "patient=",
		"_count=0", "_count=ten", "_count=-1", "_count=5&_count=6", "_cursor=nonsense", "_cursor=" + fhir.NewCursor(time.Time{}, apptA),
		"_cursor=" + fhir.NewCursor(now, apptA) + "&_cursor=" + fhir.NewCursor(now, apptB),
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

// An in-person visit names its location as a Location participant (docs/clinic-cms/WAVE6.md U2); the patient app
// reads it (LC3).
func TestHandlerRead_SendsAnInPersonVisitsLocation(t *testing.T) {
	inPerson := fullA
	inPerson.Appointment.Mode, inPerson.Appointment.LocationID = schedule.ModeInPerson, "a3000000-0000-4000-8000-0000000000a3"
	rec := getAppt(t, &stubReader{out: []appointment.Seen{inPerson}}, as(pract), &bytes.Buffer{}, "/fhir/R4/Appointment/"+apptA)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "appointment_in_person.json", rec.Body.Bytes())
}

// A booking that named a service carries it as a reference to the price list's ChargeItemDefinition (clinic-cms E).
func TestHandlerRead_SendsTheServiceTheBookingNamed(t *testing.T) {
	withService := fullA
	withService.Appointment.ServiceID = serviceVideo
	rec := getAppt(t, &stubReader{out: []appointment.Seen{withService}}, as(pract), &bytes.Buffer{}, "/fhir/R4/Appointment/"+apptA)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "appointment_service.json", rec.Body.Bytes())
}

// _count asks for a page of that many, 200 at most (FHIR lets a server return fewer); none leaves the page to the
// service's rule. _cursor is the position a next link gave, read back into the filter.
func TestHandlerSearch_PassesTheCountAndThePosition(t *testing.T) {
	at := time.Date(2030, 1, 7, 9, 0, 0, 500, time.UTC)
	for _, tt := range []struct {
		query string
		after appointment.Filter
		count int
	}{
		{"", appointment.Filter{}, 0},
		{"_count=1", appointment.Filter{}, 1},
		{"_count=200", appointment.Filter{}, 200},
		{"_count=201", appointment.Filter{}, 200},
		{"_count=007", appointment.Filter{}, 7},
		{"_cursor=" + fhir.NewCursor(at, apptA), appointment.Filter{AfterStart: at, AfterID: apptA}, 0},
		{"_count=3&_cursor=" + fhir.NewCursor(at, apptA), appointment.Filter{AfterStart: at, AfterID: apptA}, 3},
	} {
		var log bytes.Buffer
		svc := &stubReader{}
		rec := getAppt(t, svc, as(admin), &log, "/fhir/R4/Appointment?"+tt.query)
		if rec.Code != http.StatusOK || svc.count != tt.count || !svc.filter.AfterStart.Equal(tt.after.AfterStart) || svc.filter.AfterID != tt.after.AfterID {
			t.Errorf("?%s = %d %s after SearchPage(%+v, %d), want 200 after (%v, %q) and %d", tt.query, rec.Code, rec.Body, svc.filter, svc.count,
				tt.after.AfterStart, tt.after.AfterID, tt.count)
		}
	}
}

// A page with more after it links the next one: the same search, its _count as given, from after the page's last
// appointment (its start and id, opaque). A page of a longer answer has no total: the matches are not counted.
func TestHandlerSearch_LinksTheNextPage(t *testing.T) {
	var log bytes.Buffer
	svc := &stubReader{out: []appointment.Seen{pastB, fullA}, more: true}
	rec := getAppt(t, svc, as(admin), &log, "/fhir/R4/Appointment?patient=Patient/"+patientA+"&_count=2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "search_page.json", rec.Body.Bytes())
	var b struct {
		Total *int `json:"total"`
		Link  []struct {
			Relation string `json:"relation"`
			URL      string `json:"url"`
		} `json:"link"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if b.Total != nil || len(b.Link) != 2 || b.Link[0].Relation != "self" || b.Link[1].Relation != "next" {
		t.Fatalf("bundle %s, want a self and a next link and no total", rec.Body)
	}
	next, err := url.Parse(b.Link[1].URL)
	if err != nil {
		t.Fatal(err)
	}
	at, id, ok := fhir.ParseCursor(next.Query().Get("_cursor"))
	if !strings.HasPrefix(b.Link[1].URL, "https://api.example.test/fhir/R4/Appointment?") || next.Query().Get("_count") != "2" ||
		next.Query().Get("patient") != "Patient/"+patientA || !ok || !at.Equal(fullA.Appointment.Start) || id != apptA {
		t.Errorf("next = %s, want the search from after the page's last appointment", b.Link[1].URL)
	}

	// following it: the page asked for is the next link's, and the last page has no next and no total
	svc = &stubReader{out: []appointment.Seen{fullA}}
	rec = getAppt(t, svc, as(admin), &log, strings.TrimPrefix(b.Link[1].URL, "https://api.example.test"))
	if rec.Code != http.StatusOK || svc.count != 2 || svc.filter.AfterID != apptA || svc.filter.PatientID != patientA ||
		strings.Contains(rec.Body.String(), `"next"`) || strings.Contains(rec.Body.String(), `"total"`) {
		t.Errorf("the next page = %d %s after %+v, %d; want the last page, without a next link or a total", rec.Code, rec.Body, svc.filter, svc.count)
	}
}
