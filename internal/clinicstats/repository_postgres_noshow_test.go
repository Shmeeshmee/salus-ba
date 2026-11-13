package clinicstats_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
)

// The no-show reports (SAL-36): each month's appointments booked through the waitlist (the slots refilled), each
// role's booked and no-show appointments beside its fulfilled ones, and the figures per specialty (” for roles
// without one), most booked first. A count stored without a way of booking is a direct booking.
func TestRepository_SumsTheNoShowsAndTheRefilledSlots(t *testing.T) {
	t.Parallel()
	p := newPools(t)
	org, r := seedClinic(t, p.owner, 2)
	jan, feb := day(2026, 1, 5), day(2026, 2, 2)
	replace(t, p.stats, org, jan, feb, []clinicstats.AppointmentCount{
		{Day: jan, Hour: 9, PractitionerRoleID: r[0], SpecialtyCode: gp, Status: "noshow", BookedVia: "direct", Count: 2},
		{Day: jan, Hour: 9, PractitionerRoleID: r[0], SpecialtyCode: gp, Status: "noshow", BookedVia: "waitlist", Count: 1},
		{Day: jan, Hour: 9, PractitionerRoleID: r[0], SpecialtyCode: gp, Status: "fulfilled", Count: 5},
		{Day: jan, Hour: 10, PractitionerRoleID: r[1], Status: "cancelled", BookedVia: "waitlist", Count: 1},
		{Day: feb, Hour: 10, PractitionerRoleID: r[1], Status: "fulfilled", BookedVia: "waitlist", Count: 4},
		{Day: feb, Hour: 11, PractitionerRoleID: r[1], Status: "noshow", BookedVia: "direct", Count: 1},
	}, nil)
	all := clinicstats.AppointmentQuery{From: day(2026, 1, 1), To: day(2026, 2, 28)}
	inScope(t, p.app, org, func(ctx context.Context, repo clinicstats.Repository) error {
		series, err := repo.Series(ctx, org, all)
		want := []clinicstats.Period{
			{Period: "2026-01", Booked: 9, Fulfilled: 5, NoShow: 3, Cancelled: 1, Refilled: 2},
			{Period: "2026-02", Booked: 5, Fulfilled: 4, NoShow: 1, Refilled: 4},
		}
		if err != nil || !slices.Equal(series, want) {
			t.Errorf("Series() = %v, %v; want %v", series, err, want)
		}
		totals, err := repo.Practitioners(ctx, org, all)
		wantTotals := []clinicstats.PractitionerTotal{
			{PractitionerRoleID: r[0], Fulfilled: 5, NoShow: 3, Booked: 8},
			{PractitionerRoleID: r[1], Fulfilled: 4, NoShow: 1, Booked: 6},
		}
		if err != nil || !slices.Equal(totals, wantTotals) {
			t.Errorf("Practitioners() = %v, %v; want %v", totals, err, wantTotals)
		}
		specialties, err := repo.Specialties(ctx, org, all)
		wantSpecialties := []clinicstats.SpecialtyTotal{
			{Specialty: gp, Booked: 8, Fulfilled: 5, NoShow: 3},
			{Specialty: "", Booked: 6, Fulfilled: 4, NoShow: 1, Cancelled: 1},
		}
		if err != nil || !slices.Equal(specialties, wantSpecialties) {
			t.Errorf("Specialties() = %v, %v; want %v", specialties, err, wantSpecialties)
		}
		jan := clinicstats.AppointmentQuery{From: day(2026, 1, 1), To: day(2026, 1, 31), PractitionerRoleID: r[1]}
		if s, err := repo.Specialties(ctx, org, jan); err != nil || !slices.Equal(s, []clinicstats.SpecialtyTotal{{Booked: 1, Cancelled: 1}}) {
			t.Errorf("role 2's January by specialty: %v, %v", s, err)
		}
		return nil
	})
	inScope(t, p.app, "", func(ctx context.Context, repo clinicstats.Repository) error {
		if s, err := repo.Specialties(ctx, org, all); err != nil || len(s) != 0 {
			t.Errorf("no clinic's scope: %v, %v; want none", s, err)
		}
		return nil
	})
}
