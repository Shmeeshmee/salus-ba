package appointment

import (
	"context"
	"time"
)

// reminderBefore is the lead of the one reminder e-mail until SAL-33's reminders replace it.
const reminderBefore = 24 * time.Hour

// scheduleReminders schedules the reminders of the booked appointment a in the transaction in ctx (docs/plans/
// appointments.md, the seam of SAL-33). Until SAL-33: the one appointment_reminder_email 24 hours before the start,
// when that is ahead of now.
func (s *Service) scheduleReminders(ctx context.Context, a Appointment, now time.Time) error {
	if at := a.Start.Add(-reminderBefore); at.After(now) {
		return s.Jobs.EnqueueReminder(ctx, a.ID, a.OrganizationID, at)
	}
	return nil
}

// cancelReminders cancels the scheduled reminders of the appointment id in the transaction in ctx (the seam of
// SAL-33). Until SAL-33 there is nothing to cancel: the one reminder e-mail sends only while the appointment is
// booked.
func (s *Service) cancelReminders(_ context.Context, _ string) error { return nil }

// slotFreed hands the slot of the clinic org to the waitlist (waitlist_slot_freed, the seam of SAL-35) when the
// change that released it freed it, that is, while it is still ahead.
func (s *Service) slotFreed(ctx context.Context, freed bool, org, slotID string) error {
	if !freed {
		return nil
	}
	return s.Jobs.EnqueueSlotFreed(ctx, slotID, org)
}
