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
)

// stubAppointments answers with what it is given and keeps what it was asked.
type stubAppointments struct {
	err    error
	caller appointment.Caller
	out    appointment.Appointment
	args   []string
}

func (s *stubAppointments) Book(_ context.Context, c appointment.Caller, slotID string) (appointment.Appointment, error) {
	s.caller, s.args = c, []string{"Book", slotID}
	return s.out, s.err
}

func (s *stubAppointments) Cancel(_ context.Context, c appointment.Caller, id string) (appointment.Appointment, error) {
	s.caller, s.args = c, []string{"Cancel", id}
	return s.out, s.err
}

func (s *stubAppointments) SetStatus(_ context.Context, c appointment.Caller, id string, status appointment.Status) (appointment.Appointment, error) {
	s.caller, s.args = c, []string{"SetStatus", id, string(status)}
	return s.out, s.err
}

var (
	bookedA = appointment.Appointment{
		ID: apptA, PatientID: patientA, PractitionerRoleID: roleA1, OrganizationID: clinicA, SlotID: slotFree,
		Start: start, End: start.Add(30 * time.Minute), Created: now, LastUpdated: now, Status: appointment.StatusBooked, VersionID: 1,
	}
	bookedJSON = `{"start":"2030-01-07T09:00:00Z","end":"2030-01-07T09:30:00Z","id":"` + apptA + `","organizationId":"` + clinicA +
		`","practitionerRoleId":"` + roleA1 + `","slotId":"` + slotFree + `","status":"booked","versionId":1}` + "\n"
)

func serveAppt(t *testing.T, svc *stubAppointments, caller func(context.Context) (appointment.Caller, bool, error), log *bytes.Buffer, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := appointment.NewHTTPHandler(svc, caller, slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func as(c appointment.Caller) func(context.Context) (appointment.Caller, bool, error) {
	return func(context.Context) (appointment.Caller, bool, error) { return c, true, nil }
}

func TestHandler_Books(t *testing.T) {
	svc := &stubAppointments{out: bookedA}
	rec := serveAppt(t, svc, as(me), &bytes.Buffer{}, http.MethodPost, "/appointments", `{"slotId":"`+slotFree+`"}`)
	if rec.Code != http.StatusCreated || rec.Body.String() != bookedJSON || svc.caller != me || svc.args[1] != slotFree {
		t.Errorf("POST /appointments = %d %s (caller %+v, %v),\nwant 201 %s", rec.Code, rec.Body, svc.caller, svc.args, bookedJSON)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control %q, want no-store", got)
	}
}

func TestHandler_CancelsAndSetsTheStatus(t *testing.T) {
	cancelled := bookedA
	cancelled.Status, cancelled.Reason, cancelled.CancelledAt, cancelled.CancelledBy, cancelled.VersionID = appointment.StatusCancelled, appointment.ReasonPatient, now, accountA, 2
	svc := &stubAppointments{out: cancelled}
	rec := serveAppt(t, svc, as(me), &bytes.Buffer{}, http.MethodPost, "/appointments/"+apptA+"/cancel", "")
	if rec.Code != http.StatusOK || svc.args[1] != apptA || !strings.Contains(rec.Body.String(), `"status":"cancelled"`) ||
		!strings.Contains(rec.Body.String(), `"cancellationReason":"pat"`) || !strings.Contains(rec.Body.String(), `"cancelledAt":"2030-01-01T08:00:00Z"`) ||
		strings.Contains(rec.Body.String(), accountA) {
		t.Errorf("POST cancel = %d %s (%v), want 200 cancelled with the reason and time, without who", rec.Code, rec.Body, svc.args)
	}

	svc = &stubAppointments{out: bookedA}
	rec = serveAppt(t, svc, as(pract), &bytes.Buffer{}, http.MethodPost, "/appointments/"+apptA+"/status", `{"status":"noshow"}`)
	if rec.Code != http.StatusOK || svc.caller != pract || svc.args[1] != apptA || svc.args[2] != "noshow" {
		t.Errorf("POST status = %d %s (%v), want 200 with noshow asked", rec.Code, rec.Body, svc.args)
	}
}

func TestHandler_MapsEveryErrorToItsCode(t *testing.T) {
	for err, want := range map[error]struct {
		code   string
		status int
	}{
		appointment.ErrNotFound:                 {"not_found", http.StatusNotFound},
		appointment.ErrForbidden:                {"forbidden", http.StatusForbidden},
		appointment.ErrProfileRequired:          {"profile_required", http.StatusConflict},
		appointment.ErrConsentRequired:          {"consent_required", http.StatusConflict},
		appointment.ErrSlotUnavailable:          {"slot_unavailable", http.StatusConflict},
		appointment.ErrCancellationWindowClosed: {"cancellation_window_closed", http.StatusConflict},
		appointment.ErrClosed:                   {"appointment_closed", http.StatusConflict},
		appointment.ErrNotStarted:               {"appointment_not_started", http.StatusConflict},
		appointment.ErrInvalidStatus:            {"invalid_status", http.StatusBadRequest},
	} {
		svc := &stubAppointments{err: err}
		rec := serveAppt(t, svc, as(me), &bytes.Buffer{}, http.MethodPost, "/appointments/"+apptA+"/cancel", "")
		if rec.Code != want.status || rec.Body.String() != `{"error":"`+want.code+`"}`+"\n" {
			t.Errorf("%v = %d %s, want %d %s", err, rec.Code, rec.Body, want.status, want.code)
		}
	}
}

func TestHandler_RefusesBadRequestsAndNobodyBeforeTheService(t *testing.T) {
	nobody := func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, nil }
	for _, tt := range []struct {
		caller     func(context.Context) (appointment.Caller, bool, error)
		name, path string
		body, code string
		status     int
	}{
		{as(me), "an empty slot", "/appointments", `{"slotId":""}`, "invalid_request", http.StatusBadRequest},
		{as(me), "an unknown field", "/appointments", `{"slotId":"x","reason":"pain"}`, "invalid_request", http.StatusBadRequest},
		{as(me), "not JSON", "/appointments", `slot`, "invalid_request", http.StatusBadRequest},
		{as(pract), "no status", "/appointments/" + apptA + "/status", `{}`, "invalid_request", http.StatusBadRequest},
		{nobody, "nobody books", "/appointments", `{"slotId":"x"}`, "unauthenticated", http.StatusUnauthorized},
		{nobody, "nobody cancels", "/appointments/" + apptA + "/cancel", ``, "unauthenticated", http.StatusUnauthorized},
		{nobody, "nobody marks", "/appointments/" + apptA + "/status", `{"status":"noshow"}`, "unauthenticated", http.StatusUnauthorized},
	} {
		svc := &stubAppointments{}
		rec := serveAppt(t, svc, tt.caller, &bytes.Buffer{}, http.MethodPost, tt.path, tt.body)
		if rec.Code != tt.status || rec.Body.String() != `{"error":"`+tt.code+`"}`+"\n" || svc.args != nil {
			t.Errorf("%s = %d %s (service asked %v), want %d %s before the service", tt.name, rec.Code, rec.Body, svc.args, tt.status, tt.code)
		}
	}
}

func TestHandler_LogsAnInternalErrorWithoutIDsOrDetail(t *testing.T) {
	var log bytes.Buffer
	svc := &stubAppointments{err: errors.New("connection reset")}
	rec := serveAppt(t, svc, as(me), &log, http.MethodPost, "/appointments", `{"slotId":"`+slotFree+`"}`)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "connection reset") {
		t.Errorf("POST /appointments on a failure = %d %s, want 500 without the error text", rec.Code, rec.Body)
	}
	if !strings.Contains(log.String(), "connection reset") || strings.Contains(log.String(), accountA) || strings.Contains(log.String(), slotFree) {
		t.Errorf("log %q, want the error and neither the account nor the slot", log.String())
	}
	failing := func(context.Context) (appointment.Caller, bool, error) {
		return appointment.Caller{}, false, errors.New("down")
	}
	if rec := serveAppt(t, &stubAppointments{}, failing, &log, http.MethodPost, "/appointments", `{"slotId":"x"}`); rec.Code != http.StatusInternalServerError {
		t.Errorf("POST /appointments when the caller cannot be resolved = %d, want 500", rec.Code)
	}
}
