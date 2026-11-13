package appointment

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/slot"
)

// purchase pays for the booking of the slot b, the step every booking takes (a direct one, a waitlist acceptance:
// docs/plans/appointments.md "Payments", ADR 0015). No price exists before the payments epic (SAL-23), so nothing is
// charged and nothing changes for today's booking; SAL-35 fills it with the price lookup and pkg/payments.
func (s *Service) purchase(_ context.Context, _ slot.Bookable) error { return nil }

// settle decides what a patient's cancellation (and, with SAL-34, a reschedule) of a owes or is owed: the seam of
// purchase protection (SAL-37). Nothing is ever paid before SAL-23, so nothing is refunded.
func (s *Service) settle(_ context.Context, _ Appointment) error { return nil }
