package appointment

import "github.com/Shmeeshmee/salus/internal/slot"

// sameMode tells whether the slot b is of a's visit mode (docs/plans/clinic-cms.md, ADR 0016): an appointment keeps
// the mode it was booked in, so a reschedule takes only a slot of that mode. A mode not set is video, as everything
// booked before the clinic system.
func sameMode(a Appointment, b slot.Bookable) bool {
	return visitMode(a.Mode) == visitMode(b.Mode)
}
