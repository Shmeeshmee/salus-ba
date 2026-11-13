package clinicstats_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
)

// stubStats is the handler's service: fixed answers, and the caller and query it was asked with.
type stubStats struct {
	err     error
	caller  clinicstats.Caller
	appts   clinicstats.AppointmentQuery
	diags   clinicstats.DiagnosisQuery
	figures clinicstats.AppointmentFigures
}

func (s *stubStats) Appointments(_ context.Context, c clinicstats.Caller, q clinicstats.AppointmentQuery) (clinicstats.AppointmentFigures, error) {
	s.caller, s.appts = c, q
	return s.figures, s.err
}

func (s *stubStats) Diagnoses(_ context.Context, c clinicstats.Caller, q clinicstats.DiagnosisQuery) ([]clinicstats.Count, error) {
	s.caller, s.diags = c, q
	return []clinicstats.Count{{Key: "respiratory-infections", Count: 31}, {Key: "", Count: 2}}, s.err
}

func serveStats(t *testing.T, svc clinicstats.Stats, signed bool, target string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	caller := func(context.Context) (clinicstats.Caller, bool, error) { return admin, signed, nil }
	clinicstats.NewHTTPHandler(svc, caller, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))
	return rec
}

func TestHTTPAppointments_ReturnsTheFigures(t *testing.T) {
	t.Parallel()
	svc := &stubStats{figures: clinicstats.AppointmentFigures{
		Series:          []clinicstats.Period{{Period: "2026-01", Booked: 120, Fulfilled: 98, NoShow: 6, Cancelled: 16}},
		Heatmap:         []clinicstats.HeatCell{{Weekday: 1, Hour: 9, Count: 14}},
		Practitioners:   []clinicstats.PractitionerTotal{{PractitionerRoleID: roleA1, Fulfilled: 40}},
		PerPractitioner: true,
	}}
	rec := serveStats(t, svc, true, "/analytics/clinic/appointments?from=2026-01-01&to=2026-12-31&practitionerRole="+roleA1+"&specialty="+gp)
	want := `{"practitioners":[{"practitionerRole":"` + roleA1 + `","fulfilled":40}],` +
		`"series":[{"period":"2026-01","booked":120,"fulfilled":98,"noshow":6,"cancelled":16}],"heatmap":[{"weekday":1,"hour":9,"count":14}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
	q := clinicstats.AppointmentQuery{From: day(2026, 1, 1), To: day(2026, 12, 31), PractitionerRoleID: roleA1, Specialty: gp}
	if svc.appts != q || svc.caller != admin {
		t.Errorf("asked %+v as %+v, want %+v", svc.appts, svc.caller, q)
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q", got)
	}
}

// A practitioner who is not an admin gets no practitioners at all, and empty lists are lists.
func TestHTTPAppointments_LeavesOutThePractitionersForAPractitioner(t *testing.T) {
	t.Parallel()
	rec := serveStats(t, &stubStats{}, true, "/analytics/clinic/appointments?from=2026-01-01&to=2026-01-31")
	if want := `{"series":[],"heatmap":[]}` + "\n"; rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
}

func TestHTTPDiagnoses_ReturnsTheRows(t *testing.T) {
	t.Parallel()
	svc := &stubStats{}
	rec := serveStats(t, svc, true, "/analytics/clinic/diagnoses?from=2026-01-01&to=2026-12-31&by=code&group=respiratory-infections"+
		"&subgroup=common-cold&specialty="+gp+"&practitionerRole="+roleA1)
	want := `{"rows":[{"key":"respiratory-infections","count":31},{"key":"","count":2}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
	q := clinicstats.DiagnosisQuery{From: day(2026, 1, 1), To: day(2026, 12, 31), By: clinicstats.ByCode, Group: "respiratory-infections",
		Subgroup: "common-cold", Specialty: gp, PractitionerRoleID: roleA1}
	if svc.diags != q {
		t.Errorf("asked %+v, want %+v", svc.diags, q)
	}
}

func TestHTTPStats_MapsTheErrors(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"/analytics/clinic/appointments?by=", "/analytics/clinic/diagnoses?by=group"} {
		for name, tt := range map[string]struct {
			err    error
			query  string
			body   string
			want   int
			signed bool
		}{
			"signed out":    {signed: false, want: http.StatusUnauthorized, body: `{"error":"unauthenticated"}`},
			"forbidden":     {signed: true, err: clinicstats.ErrForbidden, want: http.StatusForbidden, body: `{"error":"forbidden"}`},
			"invalid":       {signed: true, err: clinicstats.ErrInvalidQuery, want: http.StatusBadRequest, body: `{"error":"invalid_request"}`},
			"a bad date":    {signed: true, query: "&from=2026-13-01&to=2026-01-31", want: http.StatusBadRequest, body: `{"error":"invalid_request"}`},
			"a time":        {signed: true, query: "&from=2026-01-01&to=2026-01-31T10:00:00Z", want: http.StatusBadRequest, body: `{"error":"invalid_request"}`},
			"a failure":     {signed: true, err: errors.New("down"), want: http.StatusInternalServerError},
			"the signed in": {signed: true, want: http.StatusOK},
		} {
			if tt.query == "" {
				tt.query = "&from=2026-01-01&to=2026-01-31"
			}
			rec := serveStats(t, &stubStats{err: tt.err}, tt.signed, target+tt.query)
			if rec.Code != tt.want || (tt.body != "" && rec.Body.String() != tt.body+"\n") {
				t.Errorf("%s %s = %d %s, want %d %s", target, name, rec.Code, rec.Body, tt.want, tt.body)
			}
		}
	}
	mux := http.NewServeMux()
	clinicstats.NewHTTPHandler(&stubStats{}, func(context.Context) (clinicstats.Caller, bool, error) {
		return clinicstats.Caller{}, false, errors.New("down")
	}, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/analytics/clinic/diagnoses", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("a failing caller: %d, want 500", rec.Code)
	}
}
