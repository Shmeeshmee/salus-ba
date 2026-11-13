package clinicstats_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
)

// A refresh counts the appointments booked through the waitlist apart from the direct ones (SAL-36): the slots
// refilled from the waitlist.
func TestRefresh_CountsTheWaitlistBookingsApart(t *testing.T) {
	t.Parallel()
	f := newBuildFixture(t, local(2026, 11, 12, 9, 0))
	f.src.starts[clinicA] = []appointment.Start{
		{Start: local(2026, 11, 3, 9, 0), PractitionerRoleID: roleA1, Status: appointment.StatusNoShow, BookedVia: appointment.BookedDirect},
		{Start: local(2026, 11, 3, 9, 20), PractitionerRoleID: roleA1, Status: appointment.StatusNoShow, BookedVia: appointment.BookedWaitlist},
		{Start: local(2026, 11, 3, 9, 40), PractitionerRoleID: roleA1, Status: appointment.StatusNoShow, BookedVia: appointment.BookedWaitlist},
	}
	if err := f.b.Refresh(t.Context(), clinicA, local(2026, 11, 3, 12, 0)); err != nil {
		t.Fatal(err)
	}
	d := day(2026, 11, 3)
	want := []clinicstats.AppointmentCount{
		{Day: d, Hour: 9, PractitionerRoleID: roleA1, SpecialtyCode: gp, Status: "noshow", BookedVia: "direct", Count: 1},
		{Day: d, Hour: 9, PractitionerRoleID: roleA1, SpecialtyCode: gp, Status: "noshow", BookedVia: "waitlist", Count: 2},
	}
	got := f.repo.appts[clinicA]
	slices.SortFunc(got, func(a, b clinicstats.AppointmentCount) int { return len(a.BookedVia) - len(b.BookedVia) })
	if !slices.Equal(got, want) {
		t.Errorf("appointments\n got %+v\nwant %+v", got, want)
	}
}
