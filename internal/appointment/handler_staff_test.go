package appointment_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/patient"
)

// stubStaff answers with what it is given and keeps what it was asked.
type stubStaff struct {
	day     time.Time
	err     error
	booking appointment.StaffBooking
	out     appointment.Appointment
	visits  []appointment.DeskVisit
	calls   []string
	version int32
}

func (s *stubStaff) BookForPatient(_ context.Context, _ appointment.Caller, b appointment.StaffBooking) (appointment.Appointment, error) {
	s.booking, s.calls = b, append(s.calls, "Book")
	return s.out, s.err
}

func (s *stubStaff) CancelForPatient(_ context.Context, _ appointment.Caller, id string, version int32) (appointment.Appointment, error) {
	s.version, s.calls = version, append(s.calls, "Cancel "+id)
	return s.out, s.err
}

func (s *stubStaff) Arrive(_ context.Context, _ appointment.Caller, id string) (appointment.Appointment, error) {
	s.calls = append(s.calls, "Arrive "+id)
	return s.out, s.err
}

func (s *stubStaff) DeskDay(_ context.Context, _ appointment.Caller, day time.Time) (appointment.Desk, error) {
	s.day = day
	if day.IsZero() {
		day = now // the service's today in the clinic's zone
	}
	return appointment.Desk{Day: day, Visits: s.visits}, s.err
}

func serveStaff(t *testing.T, svc *stubStaff, method, target, body string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	h := appointment.NewStaffHandler(svc, func(context.Context) (appointment.Caller, bool, error) { return desk, true, nil },
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
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

func TestStaffHandler_BooksCancelsAndMarksArrival(t *testing.T) {
	t.Parallel()
	out := booked(newID, patientA, roleA1, clinicA, start)
	out.VersionID, out.ArrivedAt = 2, now
	svc := &stubStaff{out: out}
	rec := serveStaff(t, svc, http.MethodPost, "/clinic/appointments", `{"patientId":"`+patientA+`","slotId":"`+slotFree+`"}`, nil)
	if rec.Code != http.StatusCreated || rec.Header().Get("ETag") != `W/"2"` || svc.booking != (appointment.StaffBooking{PatientID: patientA, SlotID: slotFree}) {
		t.Errorf("book = %d %s (ETag %s), asked %+v", rec.Code, rec.Body, rec.Header().Get("ETag"), svc.booking)
	}
	if !strings.Contains(rec.Body.String(), `"arrivedAt":"2030-01-01T08:00:00Z"`) {
		t.Errorf("book = %s, want the arrival time", rec.Body)
	}
	if rec := serveStaff(t, svc, http.MethodPost, "/clinic/appointments/"+newID+"/cancel", "", map[string]string{"If-Match": `W/"1"`}); rec.Code != http.StatusOK || svc.version != 1 {
		t.Errorf("cancel = %d %s at version %d", rec.Code, rec.Body, svc.version)
	}
	if rec := serveStaff(t, svc, http.MethodPost, "/clinic/appointments/"+newID+"/arrive", "", nil); rec.Code != http.StatusOK {
		t.Errorf("arrive = %d %s", rec.Code, rec.Body)
	}
}

func TestStaffHandler_GivesTheDesksDay(t *testing.T) {
	t.Parallel()
	a := booked(newID, patientC, roleA1, clinicA, start)
	p := patient.Patient{ID: patientC, RegisteredBy: clinicA, GivenName: "Cy", FamilyName: "Example", BirthDate: time.Date(1980, 3, 4, 0, 0, 0, 0, time.UTC), VersionID: 1, LastUpdated: now}
	svc := &stubStaff{visits: []appointment.DeskVisit{{Appointment: a, Patient: p, ServiceName: "Synthetic exam"}}}
	rec := serveStaff(t, svc, http.MethodGet, "/clinic/desk/today?date=2030-01-07", "", nil)
	want := `"patient":{"lastUpdated":"2030-01-01T08:00:00Z","id":"` + patientC + `","givenName":"Cy","familyName":"Example","birthDate":"1980-03-04","registeredByClinic":true,"versionId":1}`
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) || !strings.HasPrefix(rec.Body.String(), `{"date":"2030-01-07","visits":[`) ||
		!strings.Contains(rec.Body.String(), `"serviceName":"Synthetic exam"`) ||
		rec.Header().Get("Cache-Control") != "no-store" || !svc.day.Equal(time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("desk = %d %s (day %v)", rec.Code, rec.Body, svc.day)
	}
	empty := &stubStaff{}
	if rec := serveStaff(t, empty, http.MethodGet, "/clinic/desk/today", "", nil); rec.Code != http.StatusOK || rec.Body.String() != `{"date":"2030-01-01","visits":[]}`+"\n" || !empty.day.IsZero() {
		t.Errorf("today = %d %s (day %v)", rec.Code, rec.Body, empty.day)
	}
}

func TestStaffHandler_AnswersEachRefusalWithItsCode(t *testing.T) {
	t.Parallel()
	cancel := "/clinic/appointments/" + newID + "/cancel"
	match := map[string]string{"If-Match": `W/"1"`}
	for name, tc := range map[string]struct {
		err    error
		header map[string]string
		method string
		target string
		body   string
		code   string
		status int
	}{
		"no slot":          {err: nil, method: http.MethodPost, target: "/clinic/appointments", body: `{"patientId":"` + patientA + `"}`, header: nil, status: 400, code: "invalid_request"},
		"an unknown field": {err: nil, method: http.MethodPost, target: "/clinic/appointments", body: `{"patientId":"x","slotId":"y","z":1}`, header: nil, status: 400, code: "invalid_request"},
		"no consent":       {err: appointment.ErrConsentRequired, method: http.MethodPost, target: "/clinic/appointments", body: `{"patientId":"x","slotId":"y"}`, header: nil, status: 403, code: "consent_required"},
		"taken":            {err: appointment.ErrSlotUnavailable, method: http.MethodPost, target: "/clinic/appointments", body: `{"patientId":"x","slotId":"y"}`, header: nil, status: 409, code: "slot_unavailable"},
		"no If-Match":      {err: nil, method: http.MethodPost, target: cancel, body: "", header: nil, status: 400, code: "if_match_required"},
		"a stale version":  {err: appointment.ErrVersionConflict, method: http.MethodPost, target: cancel, body: "", header: match, status: 409, code: "version_conflict"},
		"a video visit":    {err: appointment.ErrNotInPerson, method: http.MethodPost, target: "/clinic/appointments/" + newID + "/arrive", body: "", header: nil, status: 409, code: "not_in_person"},
		"a bad date":       {err: nil, method: http.MethodGet, target: "/clinic/desk/today?date=7.1.2030", body: "", header: nil, status: 400, code: "invalid_request"},
		"not the desk":     {err: appointment.ErrForbidden, method: http.MethodGet, target: "/clinic/desk/today", body: "", header: nil, status: 403, code: "forbidden"},
		"an internal one":  {err: errors.New("boom"), method: http.MethodGet, target: "/clinic/desk/today", body: "", header: nil, status: 500, code: ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec := serveStaff(t, &stubStaff{err: tc.err}, tc.method, tc.target, tc.body, tc.header)
			if rec.Code != tc.status || (tc.code != "" && rec.Body.String() != `{"error":"`+tc.code+`"}`+"\n") {
				t.Errorf("= %d %s, want %d %s", rec.Code, rec.Body, tc.status, tc.code)
			}
		})
	}
}
