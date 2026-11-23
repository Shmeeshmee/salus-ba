package appointment_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// Bookings by staff, arrival and the desk's day are the wave 3 contract (docs/clinic-cms, 3.0): refused until
// package F2 builds them.
func TestStaff_RefusesUntilPackageF2(t *testing.T) {
	t.Parallel()
	var s appointment.Service
	c := appointment.Caller{AccountID: "a", OrganizationID: "o", Admin: true}
	ctx := t.Context()
	_, book := s.BookForPatient(ctx, c, appointment.StaffBooking{PatientID: "p", SlotID: "s"})
	_, move := s.MoveForPatient(ctx, c, "id", "s")
	_, cancel := s.CancelForPatient(ctx, c, "id")
	_, arrive := s.Arrive(ctx, c, "id")
	_, desk := s.DeskDay(ctx, c, time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC))
	for name, err := range map[string]error{"BookForPatient": book, "MoveForPatient": move, "CancelForPatient": cancel, "Arrive": arrive, "DeskDay": desk} {
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s = %v, want errors.ErrUnsupported", name, err)
		}
	}
	if appointment.StatusArrived != "arrived" {
		t.Errorf("StatusArrived = %q", appointment.StatusArrived)
	}
}
