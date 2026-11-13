package clinicstats_test

import (
	"net/http"
	"testing"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
)

// SAL-36: the series carry the slots refilled from the waitlist, the practitioners their no-shows and bookings, and
// the admin gets the figures per specialty ("" for roles without one).
func TestHTTPAppointments_ReturnsTheNoShowFigures(t *testing.T) {
	t.Parallel()
	svc := &stubStats{figures: clinicstats.AppointmentFigures{
		Series:          []clinicstats.Period{{Period: "2026-01", Booked: 10, Fulfilled: 7, NoShow: 2, Cancelled: 1, Refilled: 3}},
		Practitioners:   []clinicstats.PractitionerTotal{{PractitionerRoleID: roleA1, Fulfilled: 7, NoShow: 2, Booked: 10}},
		Specialties:     []clinicstats.SpecialtyTotal{{Specialty: gp, Booked: 6, Fulfilled: 5, NoShow: 1}, {Booked: 4, Fulfilled: 2, NoShow: 1, Cancelled: 1}},
		PerPractitioner: true,
	}}
	rec := serveStats(t, svc, true, "/analytics/clinic/appointments?from=2026-01-01&to=2026-01-31")
	want := `{"practitioners":[{"practitionerRole":"` + roleA1 + `","fulfilled":7,"noshow":2,"booked":10}],` +
		`"specialties":[{"specialty":"` + gp + `","booked":6,"fulfilled":5,"noshow":1,"cancelled":0},` +
		`{"specialty":"","booked":4,"fulfilled":2,"noshow":1,"cancelled":1}],` +
		`"series":[{"period":"2026-01","booked":10,"fulfilled":7,"noshow":2,"cancelled":1,"refilled":3}],"heatmap":[]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d\n got %s\nwant %s", rec.Code, rec.Body, want)
	}
}
