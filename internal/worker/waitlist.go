package worker

import (
	"context"
	"errors"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// offerNotDueRetry is how long an offer's expiry that ran before the offer's end waits before it runs again.
const offerNotDueRetry = 30 * time.Second

// slotFreedWorker hands a freed slot to the waitlist (waitlist_slot_freed, enqueued by a cancellation that frees a
// slot still ahead, a reschedule, a decline or a withdrawal; docs/plans/appointments.md, SAL-35): the first matching
// entry is offered it, or it stays free.
type slotFreedWorker struct {
	river.WorkerDefaults[appointment.SlotFreedArgs]
	appointments *appointment.Service
}

func (w *slotFreedWorker) Work(ctx context.Context, job *river.Job[appointment.SlotFreedArgs]) error {
	return w.appointments.HandOn(ctx, job.Args.OrganizationID, job.Args.SlotID)
}

// offerExpireWorker ends an offer at its hold's end and hands its slot on (waitlist_offer_expire, SAL-35); one that
// ran early is snoozed.
type offerExpireWorker struct {
	river.WorkerDefaults[appointment.OfferExpireArgs]
	appointments *appointment.Service
}

func (w *offerExpireWorker) Work(ctx context.Context, job *river.Job[appointment.OfferExpireArgs]) error {
	err := w.appointments.ExpireOffer(ctx, job.Args.OrganizationID, job.Args.OfferID)
	if errors.Is(err, appointment.ErrOfferNotDue) {
		return river.JobSnooze(offerNotDueRetry)
	}
	return err
}
