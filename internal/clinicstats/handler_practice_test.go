package clinicstats_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

func (s *stubStats) Practice(_ context.Context, c clinicstats.Caller, q clinicstats.PracticeQuery) ([]clinicstats.PracticeRow, error) {
	s.caller, s.practice = c, q
	return []clinicstats.PracticeRow{{PractitionerRoleID: roleA1, Mode: schedule.ModeVideo, MinutesOffered: 150, Visits: 2, ReceiptedMinor: 7500}}, s.err
}

func TestHTTPPractice_ReturnsTheRows(t *testing.T) {
	t.Parallel()
	svc := &stubStats{}
	rec := serveStats(t, svc, true, "/clinic/reports/practice?from=2026-03-01T23:00:00Z&to=2026-03-08T23:00:00Z")
	want := `{"from":"2026-03-01T23:00:00Z","to":"2026-03-08T23:00:00Z","currency":"BAM","rows":[{"practitionerRole":"` + roleA1 +
		`","visitMode":"video","minutesOffered":150,"visits":2,"receiptedMinor":7500}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("GET practice = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
	from := time.Date(2026, 3, 1, 23, 0, 0, 0, time.UTC)
	if svc.caller != admin || !svc.practice.From.Equal(from) || !svc.practice.To.Equal(from.Add(7*24*time.Hour)) {
		t.Errorf("asked %+v by %+v, want the period by the admin", svc.practice, svc.caller)
	}
}

func TestHTTPPractice_AnswersEveryRefusal(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		err    error
		target string
		signed bool
		status int
	}{
		"signed out":   {target: "/clinic/reports/practice?from=2026-03-01T00:00:00Z&to=2026-03-02T00:00:00Z", status: http.StatusUnauthorized},
		"no times":     {target: "/clinic/reports/practice", signed: true, status: http.StatusBadRequest},
		"a date only":  {target: "/clinic/reports/practice?from=2026-03-01&to=2026-03-02", signed: true, status: http.StatusBadRequest},
		"not an admin": {err: clinicstats.ErrForbidden, target: "/clinic/reports/practice?from=2026-03-01T00:00:00Z&to=2026-03-02T00:00:00Z", signed: true, status: http.StatusForbidden},
		"a bad range":  {err: clinicstats.ErrInvalidQuery, target: "/clinic/reports/practice?from=2026-03-02T00:00:00Z&to=2026-03-01T00:00:00Z", signed: true, status: http.StatusBadRequest},
		"a failure":    {err: errors.New("synthetic failure"), target: "/clinic/reports/practice?from=2026-03-01T00:00:00Z&to=2026-03-02T00:00:00Z", signed: true, status: http.StatusInternalServerError},
	} {
		if rec := serveStats(t, &stubStats{err: tc.err}, tc.signed, tc.target); rec.Code != tc.status {
			t.Errorf("%s: GET practice = %d %s, want %d", name, rec.Code, rec.Body, tc.status)
		}
	}
}
