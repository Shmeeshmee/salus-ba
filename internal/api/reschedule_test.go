package api_test

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// Three reschedules of one appointment at once, to three free slots: they run one after the other on the appointment's
// lock, so the clinic's limit (2) lets two through and refuses the third; the appointment holds exactly one slot, and
// every slot is busy exactly when an appointment holds it.
func TestReschedule_ConcurrentReschedulesOfOneAppointmentRespectTheLimit(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedClinic(t, owner, 1, 72*time.Hour, 73*time.Hour, 74*time.Hour, 75*time.Hour)
	b := newBooking(t, app)
	me := appointment.Caller{AccountID: c.accounts[0], PatientID: c.patients[0]}
	a, err := b.appointments.Book(t.Context(), me, c.slots[0])
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make([]error, 3)
	for i, sl := range c.slots[1:] {
		wg.Go(func() { _, errs[i] = b.appointments.Reschedule(t.Context(), me, a.ID, sl) })
	}
	wg.Wait()
	var ok, limited int
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, appointment.ErrRescheduleLimit):
			limited++
		default:
			t.Errorf("Reschedule() = %v", err)
		}
	}
	if ok != 2 || limited != 1 {
		t.Errorf("%d moved and %d refused by the limit, want 2 and 1", ok, limited)
	}
	checkInStep(t, owner, 0)
	var count, busy int
	if err := owner.QueryRow(t.Context(), `SELECT (SELECT reschedule_count FROM appointment WHERE id = $1),
		(SELECT count(*) FROM slot WHERE status = 'busy')`, a.ID).Scan(&count, &busy); err != nil || count != 2 || busy != 1 {
		t.Errorf("reschedule_count %d, busy slots %d (%v); want 2 and 1", count, busy, err)
	}
	var changes int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM appointment_change WHERE appointment_id = $1 AND kind = 'rescheduled'", a.ID).Scan(&changes); err != nil || changes != 2 {
		t.Errorf("%d reschedules in the history (%v), want 2", changes, err)
	}
}

// A reschedule onto a slot races another patient's booking of it, and a cancellation of the moving appointment races
// its reschedule, for many rounds: one of each pair wins, and slots and appointments stay in step.
func TestReschedule_RacingABookingOfTheSlotAndACancellationKeepSlotsInStep(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	var offsets []time.Duration
	for i := range 20 {
		offsets = append(offsets, 48*time.Hour+time.Duration(i)*time.Hour)
	}
	c := seedClinic(t, owner, 3, offsets...)
	b := newBooking(t, app)
	mover, booker, canceller := appointment.Caller{AccountID: c.accounts[0], PatientID: c.patients[0]},
		appointment.Caller{AccountID: c.accounts[1], PatientID: c.patients[1]}, appointment.Caller{AccountID: c.accounts[2], PatientID: c.patients[2]}
	for round := range 5 {
		base := c.slots[4*round : 4*round+4]
		moving, err := b.appointments.Book(t.Context(), mover, base[0])
		if err != nil {
			t.Fatal(err)
		}
		cancelling, err := b.appointments.Book(t.Context(), canceller, base[1])
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		var moveErr, bookErr, moveCancelled, cancelErr error
		wg.Go(func() { _, moveErr = b.appointments.Reschedule(t.Context(), mover, moving.ID, base[2]) })
		wg.Go(func() { _, bookErr = b.appointments.Book(t.Context(), booker, base[2]) })
		wg.Go(func() { _, moveCancelled = b.appointments.Reschedule(t.Context(), canceller, cancelling.ID, base[3]) })
		wg.Go(func() { _, cancelErr = b.appointments.Cancel(t.Context(), canceller, cancelling.ID) })
		wg.Wait()
		if (moveErr == nil) == (bookErr == nil) || !slices.ContainsFunc([]error{moveErr, bookErr}, func(e error) bool { return errors.Is(e, appointment.ErrSlotUnavailable) }) {
			t.Errorf("round %d: reschedule %v, booking %v; want exactly one to take the slot", round, moveErr, bookErr)
		}
		if cancelErr != nil || (moveCancelled != nil && !errors.Is(moveCancelled, appointment.ErrClosed)) {
			t.Errorf("round %d: cancellation %v, reschedule %v; want the cancellation, and the reschedule before it or refused", round, cancelErr, moveCancelled)
		}
		checkInStep(t, owner, round)
	}
}

// When the role is booked at the new slot's time through another slot (overlapping schedules), the exclusion
// constraint refuses the move: the transaction is rolled back (the new slot stays free) and the refusal is audited in
// a transaction of its own, as a booking's.
func TestReschedule_TheExclusionConstraintsRefusalIsAuditedInItsOwnTransaction(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedClinic(t, owner, 2, 48*time.Hour, 72*time.Hour)
	b := newBooking(t, app)
	me := appointment.Caller{AccountID: c.accounts[0], PatientID: c.patients[0]}
	a, err := b.appointments.Book(t.Context(), me, c.slots[0])
	if err != nil {
		t.Fatal(err)
	}
	// Patient B is booked with the role at the second slot's time, on a slot of another schedule (inserted as the owner).
	if _, err := owner.Exec(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at, updated_at)
		SELECT $1, $2, organization_id, gen_random_uuid(), period, now(), now() FROM slot WHERE id = $3`, c.patients[1], c.role, c.slots[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := b.appointments.Reschedule(t.Context(), me, a.ID, c.slots[1]); !errors.Is(err, appointment.ErrSlotUnavailable) {
		t.Fatalf("Reschedule() = %v, want ErrSlotUnavailable", err)
	}
	var status string
	var refused int
	if err := owner.QueryRow(t.Context(), `SELECT (SELECT status FROM slot WHERE id = $1),
		(SELECT count(*) FROM audit_event WHERE agent_account_id = $2 AND subtype_code = 'update' AND outcome = '4')`, c.slots[1], c.accounts[0]).
		Scan(&status, &refused); err != nil || status != "free" || refused != 1 {
		t.Errorf("slot %s, %d refusals audited (%v); want free and 1", status, refused, err)
	}
}
