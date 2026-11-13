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

// stubFlags answers the flags with what it is given and keeps what it was asked.
type stubFlags struct {
	err      error
	caller   appointment.Caller
	from, to time.Time
	out      []appointment.NoShowFlag
}

func (s *stubFlags) NoShowFlags(_ context.Context, c appointment.Caller, from, to time.Time) ([]appointment.NoShowFlag, error) {
	s.caller, s.from, s.to = c, from, to
	return s.out, s.err
}

func serveFlags(t *testing.T, svc *stubFlags, caller func(context.Context) (appointment.Caller, bool, error), log *bytes.Buffer, target string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	appointment.NewNoShowHandler(svc, caller, slog.New(slog.NewTextHandler(log, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))
	return rec
}

const flagsRange = "/clinic/noshow-flags?from=2030-01-07T00:00:00%2B01:00&to=2030-01-14T00:00:00%2B01:00"

func TestNoShowHandler_GivesTheFlags(t *testing.T) {
	svc := &stubFlags{out: []appointment.NoShowFlag{{AppointmentID: apptA, NoShows: 3, WindowDays: 180}}}
	rec := serveFlags(t, svc, as(admin), &bytes.Buffer{}, flagsRange)
	want := `{"flags":[{"appointment_id":"` + apptA + `","noshows":3,"window_days":180}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
	if !svc.from.Equal(time.Date(2030, 1, 6, 23, 0, 0, 0, time.UTC)) || !svc.to.Equal(time.Date(2030, 1, 13, 23, 0, 0, 0, time.UTC)) || svc.caller != admin {
		t.Errorf("asked %v to %v as %+v", svc.from, svc.to, svc.caller)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control %q, want no-store", got)
	}
	empty := serveFlags(t, &stubFlags{}, as(admin), &bytes.Buffer{}, flagsRange)
	if want := `{"flags":[]}` + "\n"; empty.Code != http.StatusOK || empty.Body.String() != want {
		t.Errorf("none = %d %s, want 200 %s", empty.Code, empty.Body, want)
	}
}

func TestNoShowHandler_MapsEveryFailure(t *testing.T) {
	log := &bytes.Buffer{}
	for _, tt := range []struct {
		err    error
		caller func(context.Context) (appointment.Caller, bool, error)
		target string
		code   string
		status int
	}{
		{target: flagsRange, caller: func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, nil },
			status: http.StatusUnauthorized, code: "unauthenticated"},
		{target: "/clinic/noshow-flags?from=2030-01-07&to=2030-01-14", caller: as(admin), status: http.StatusBadRequest, code: "invalid_request"},
		{target: "/clinic/noshow-flags?to=2030-01-14T00:00:00Z", caller: as(admin), status: http.StatusBadRequest, code: "invalid_request"},
		{err: appointment.ErrInvalidRange, target: flagsRange, caller: as(admin), status: http.StatusBadRequest, code: "invalid_request"},
		{err: appointment.ErrNotFound, target: flagsRange, caller: as(me), status: http.StatusNotFound, code: "not_found"},
		{err: errors.New("the database at 10.0.0.1 is down"), target: flagsRange, caller: as(admin), status: http.StatusInternalServerError},
	} {
		rec := serveFlags(t, &stubFlags{err: tt.err}, tt.caller, log, tt.target)
		if rec.Code != tt.status || (tt.code != "" && !strings.Contains(rec.Body.String(), `"`+tt.code+`"`)) {
			t.Errorf("%s with %v = %d %s, want %d %s", tt.target, tt.err, rec.Code, rec.Body, tt.status, tt.code)
		}
		if strings.Contains(rec.Body.String(), "10.0.0.1") {
			t.Errorf("the error text reached the client: %s", rec.Body)
		}
	}
}

// The correction's refusal after 30 days has its code (SAL-36).
func TestHandler_TheCorrectionWindowClosed(t *testing.T) {
	svc := &stubAppointments{err: appointment.ErrCorrectionWindowClosed}
	rec := serveAppt(t, svc, as(pract), &bytes.Buffer{}, http.MethodPost, "/appointments/"+apptA+"/status", `{"status":"fulfilled"}`)
	if want := `{"error":"correction_window_closed"}` + "\n"; rec.Code != http.StatusConflict || rec.Body.String() != want {
		t.Errorf("POST = %d %s, want 409 %s", rec.Code, rec.Body, want)
	}
}
