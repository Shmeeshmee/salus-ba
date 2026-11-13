package slot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A freed slot is held for one waitlisted patient (free to busy-tentative), booked by their acceptance (to busy) or
// given back (to free), each only from the status before it and only in the slot's clinic (SAL-35).
func TestOffer_MovesASlotThroughTheWaitlistsStatuses(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	id := addSlot(t, owner, scheduleA1, clinicA, monday.Add(time.Hour), slot.StatusFree)
	svc := newService(app, monday)
	ctx := t.Context()

	if _, err := svc.Offer(ctx, clinicB, id); !errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("Offer(another clinic's) = %v, want ErrUnavailable", err)
	}
	if _, err := svc.TakeOffered(ctx, clinicA, id); !errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("TakeOffered(a free slot) = %v, want ErrUnavailable", err)
	}
	if freed, err := svc.Unoffer(ctx, clinicA, id); err != nil || freed {
		t.Errorf("Unoffer(a free slot) = %v, %v; want false", freed, err)
	}
	got, err := svc.Offer(ctx, clinicA, id)
	if err != nil || got.Status != slot.StatusTentative || got.VersionID != 2 || !got.Start.Equal(monday.Add(time.Hour)) {
		t.Fatalf("Offer() = %+v, %v; want the slot busy-tentative at version 2", got, err)
	}
	if _, err := svc.Offer(ctx, clinicA, id); !errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("Offer(an offered slot) = %v, want ErrUnavailable", err)
	}
	if _, err := svc.Hold(ctx, clinicA, id); !errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("Hold(an offered slot) = %v, want ErrUnavailable: a direct booking never takes it", err)
	}
	if freed, err := svc.Unoffer(ctx, clinicB, id); err != nil || freed {
		t.Errorf("Unoffer(another clinic's) = %v, %v; want false", freed, err)
	}
	if freed, err := svc.Unoffer(ctx, clinicA, id); err != nil || !freed {
		t.Fatalf("Unoffer() = %v, %v; want the slot free", freed, err)
	}
	if _, err := svc.Offer(ctx, clinicA, id); err != nil {
		t.Fatal(err)
	}
	taken, err := svc.TakeOffered(ctx, clinicA, id)
	if err != nil || taken.Status != slot.StatusBusy {
		t.Errorf("TakeOffered() = %+v, %v; want the slot busy", taken, err)
	}
	if got := slots(t, owner, scheduleA1); got[0].status != slot.StatusBusy {
		t.Errorf("slots %v, want busy", got)
	}
	if _, err := svc.Offer(ctx, clinicA, "x"); !errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("Offer(not a UUID) = %v, want ErrUnavailable", err)
	}
}

// An offered slot that has started is given back too: no offer can hold it any more, and retention deletes free
// slots only.
func TestUnoffer_FreesAStartedSlot(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	id := addSlot(t, owner, scheduleA1, clinicA, monday.Add(-time.Hour), slot.StatusTentative)
	if freed, err := newService(app, monday).Unoffer(t.Context(), clinicA, id); err != nil || !freed {
		t.Errorf("Unoffer() = %v, %v; want the started slot free", freed, err)
	}
}

// The waitlist reads a slot with its role and locks it until its transaction ends, so two jobs that hand on one slot
// run one after the other.
func TestForWaitlist_LocksTheSlotOfTheClinicWithItsRole(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	id := addSlot(t, owner, scheduleA1, clinicA, monday.Add(time.Hour), slot.StatusBusy)
	svc := newService(app, monday)
	for name, org := range map[string]string{"another clinic's": clinicB} {
		if _, err := svc.ForWaitlist(t.Context(), org, id); !errors.Is(err, slot.ErrUnavailable) {
			t.Errorf("ForWaitlist(%s) = %v, want ErrUnavailable", name, err)
		}
	}
	if _, err := svc.ForWaitlist(t.Context(), clinicA, "x"); !errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("ForWaitlist(not a UUID) = %v, want ErrUnavailable", err)
	}
	locked := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		done <- in(t, app, clinicA, func(ctx context.Context) error {
			b, err := svc.ForWaitlist(ctx, clinicA, id)
			if err == nil && (b.PractitionerRoleID != roleA1 || b.Status != slot.StatusBusy || b.ScheduleID != scheduleA1) {
				err = errors.New("not the slot with its role")
			}
			close(locked)
			<-release
			return err
		})
	}()
	<-locked
	go func() {
		_, err := svc.ForWaitlist(t.Context(), clinicA, id)
		done <- err
	}()
	pgtest.WaitForLockWait(t, owner)
	close(release)
	for range 2 {
		if err := <-done; err != nil {
			t.Error(err)
		}
	}
}
