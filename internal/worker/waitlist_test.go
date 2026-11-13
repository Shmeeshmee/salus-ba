package worker

import (
	"testing"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// Until the waitlist (SAL-35), a freed slot's job completes and changes nothing.
func TestSlotFreed_CompletesUntilTheWaitlistExists(t *testing.T) {
	t.Parallel()
	w := &slotFreedWorker{}
	if err := w.Work(t.Context(), &river.Job[appointment.SlotFreedArgs]{Args: appointment.SlotFreedArgs{SlotID: "s", OrganizationID: "o"}}); err != nil {
		t.Errorf("Work() = %v, want nil", err)
	}
}
