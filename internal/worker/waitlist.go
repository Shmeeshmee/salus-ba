package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// slotFreedWorker completes waitlist_slot_freed, enqueued by every cancellation that frees a slot still ahead: the
// seam of the waitlist (docs/plans/appointments.md, SAL-35), which replaces it with the offer to the first matching
// entry. Until then the slot simply stays free in the directory.
type slotFreedWorker struct {
	river.WorkerDefaults[appointment.SlotFreedArgs]
}

func (w *slotFreedWorker) Work(context.Context, *river.Job[appointment.SlotFreedArgs]) error {
	return nil
}
