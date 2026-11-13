package appointment_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// stubActions answers the actions handler's calls with what it is given and keeps what it was asked.
type stubActions struct {
	err     error
	caller  appointment.Caller
	out     appointment.Appointment
	actions appointment.Actions
	args    []string
}

func (s *stubActions) Confirm(_ context.Context, c appointment.Caller, id string) (appointment.Appointment, error) {
	s.caller, s.args = c, []string{"Confirm", id}
	return s.out, s.err
}

func (s *stubActions) Reschedule(_ context.Context, c appointment.Caller, id, slotID string) (appointment.Appointment, error) {
	s.caller, s.args = c, []string{"Reschedule", id, slotID}
	return s.out, s.err
}

func (s *stubActions) Actions(_ context.Context, c appointment.Caller, id string) (appointment.Actions, error) {
	s.caller, s.args = c, []string{"Actions", id}
	return s.actions, s.err
}

func serveActions(t *testing.T, svc *stubActions, caller func(context.Context) (appointment.Caller, bool, error), method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	appointment.NewActionsHandler(svc, caller, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func TestActionsHandler_ConfirmsReschedulesAndTellsTheActions(t *testing.T) {
	moved := bookedA
	moved.RescheduleCount, moved.VersionID, moved.ConfirmedAt, moved.ConfirmedVia = 1, 2, now, appointment.ViaApp
	movedJSON := strings.Replace(strings.Replace(bookedJSON, `"versionId":1,"rescheduleCount":0`, `"versionId":2,"rescheduleCount":1`, 1),
		`"end":"2030-01-07T09:30:00Z",`, `"end":"2030-01-07T09:30:00Z","confirmedAt":"2030-01-01T08:00:00Z",`, 1)
	movedJSON = strings.Replace(movedJSON, `"status":"booked",`, `"status":"booked","confirmedVia":"app",`, 1)
	for _, tc := range []struct {
		method, path, body, want string
		args                     []string
	}{
		{http.MethodPost, "/appointments/" + apptA + "/confirm", "", movedJSON, []string{"Confirm", apptA}},
		{http.MethodPost, "/appointments/" + apptA + "/reschedule", `{"slotId":"` + slotLater + `"}`, movedJSON, []string{"Reschedule", apptA, slotLater}},
		{http.MethodGet, "/appointments/" + apptA + "/actions", "",
			`{"cancel_until":"2030-01-06T09:00:00Z","confirm":true,"cancel":true,"reschedule":false,"reschedules_left":0}` + "\n", []string{"Actions", apptA}},
	} {
		svc := &stubActions{out: moved, actions: appointment.Actions{Confirm: true, Cancel: true, CancelUntil: start.Add(-24 * time.Hour)}}
		rec := serveActions(t, svc, as(me), tc.method, tc.path, tc.body)
		if rec.Code != http.StatusOK || rec.Body.String() != tc.want || rec.Header().Get("Cache-Control") != "no-store" || svc.caller != me ||
			strings.Join(svc.args, " ") != strings.Join(tc.args, " ") {
			t.Errorf("%s %s = %d %s (args %v),\nwant 200 %s", tc.method, tc.path, rec.Code, rec.Body, svc.args, tc.want)
		}
	}
}

func TestActionsHandler_Errors(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body string
		caller                   func(context.Context) (appointment.Caller, bool, error)
		err                      error
		want                     string
		status                   int
	}{
		{"nobody", http.MethodPost, "/appointments/" + apptA + "/confirm", "", func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, nil },
			nil, "unauthenticated", http.StatusUnauthorized},
		{"no slot", http.MethodPost, "/appointments/" + apptA + "/reschedule", `{}`, as(me), nil, "invalid_request", http.StatusBadRequest},
		{"unknown field", http.MethodPost, "/appointments/" + apptA + "/reschedule", `{"slotId":"x","at":1}`, as(me), nil, "invalid_request", http.StatusBadRequest},
		{"staff", http.MethodPost, "/appointments/" + apptA + "/confirm", "", as(pract), appointment.ErrForbidden, "forbidden", http.StatusForbidden},
		{"not theirs", http.MethodGet, "/appointments/" + apptA + "/actions", "", as(me), appointment.ErrNotFound, "not_found", http.StatusNotFound},
		{"closed", http.MethodPost, "/appointments/" + apptA + "/confirm", "", as(me), appointment.ErrClosed, "appointment_closed", http.StatusConflict},
		{"window", http.MethodPost, "/appointments/" + apptA + "/reschedule", `{"slotId":"` + slotLater + `"}`, as(me), appointment.ErrRescheduleWindowClosed, "reschedule_window_closed", http.StatusConflict},
		{"limit", http.MethodPost, "/appointments/" + apptA + "/reschedule", `{"slotId":"` + slotLater + `"}`, as(me), appointment.ErrRescheduleLimit, "reschedule_limit", http.StatusConflict},
		{"not reschedulable", http.MethodPost, "/appointments/" + apptA + "/reschedule", `{"slotId":"` + slotLater + `"}`, as(me), appointment.ErrNotReschedulable, "not_reschedulable", http.StatusConflict},
		{"taken", http.MethodPost, "/appointments/" + apptA + "/reschedule", `{"slotId":"` + slotLater + `"}`, as(me), appointment.ErrSlotUnavailable, "slot_unavailable", http.StatusConflict},
		{"consent", http.MethodPost, "/appointments/" + apptA + "/reschedule", `{"slotId":"` + slotLater + `"}`, as(me), appointment.ErrConsentRequired, "consent_required", http.StatusConflict},
		{"backend", http.MethodGet, "/appointments/" + apptA + "/actions", "", as(me), errBackend, "internal error", http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveActions(t, &stubActions{err: tc.err}, tc.caller, tc.method, tc.path, tc.body)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), `"`+tc.want+`"`) {
				t.Errorf("%s %s = %d %s, want %d %s", tc.method, tc.path, rec.Code, rec.Body, tc.status, tc.want)
			}
		})
	}
}
