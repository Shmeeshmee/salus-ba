package appointment_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seedRoleA2 adds a second practitioner role to clinic A.
func seedRoleA2(t *testing.T, owner *pgxpool.Pool) string {
	t.Helper()
	var role string
	if err := owner.QueryRow(t.Context(), `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
		VALUES ($1, 'Bo', 'Example') RETURNING id, organization_id)
		INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id::text`, clinicA).Scan(&role); err != nil {
		t.Fatal(err)
	}
	return role
}

// The desk's move (docs/clinic-cms WAVE4, M1): the slot, period and practitioner together, in the clinic's scope, not
// counted and not confirmed; a move decided on another version changes nothing; the exclusion constraint refuses a
// time the new practitioner is booked at.
func TestMoveByStaff_MovesToAnotherRoleUncountedAtItsVersionAndTheConstraintGuards(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	roleA2 := seedRoleA2(t, owner)
	a := bookInDB(t, app, s, start)
	to := appointment.StaffMove{Start: start.Add(time.Hour), End: start.Add(90 * time.Minute), SlotID: slotSoon, PractitionerRoleID: roleA2, Version: a.VersionID}
	err := inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		got, err := r.MoveByStaff(ctx, a.ID, to, now)
		if err != nil || !got.Start.Equal(to.Start) || !got.End.Equal(to.End) || got.SlotID != slotSoon || got.PractitionerRoleID != roleA2 ||
			got.RescheduleCount != 0 || !got.ConfirmedAt.IsZero() || got.VersionID != a.VersionID+1 || !got.LastUpdated.Equal(now) {
			t.Errorf("MoveByStaff = %+v, %v; want moved to role A2, uncounted, unconfirmed, at the next version", got, err)
		}
		if _, err := r.MoveByStaff(ctx, a.ID, to, now); !errors.Is(err, appointment.ErrVersionConflict) {
			t.Errorf("MoveByStaff at the old version = %v, want ErrVersionConflict", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	other := bookInDB(t, app, s, start.Add(3*time.Hour)) // role A1's
	err = inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		_, err := r.MoveByStaff(ctx, a.ID, appointment.StaffMove{Start: other.Start, End: other.End, SlotID: slotPast, PractitionerRoleID: s.roleA1, Version: a.VersionID + 1}, now)
		return err
	})
	if !errors.Is(err, appointment.ErrSlotUnavailable) {
		t.Errorf("MoveByStaff onto a booked time = %v, want ErrSlotUnavailable", err)
	}
	// The patient's own scope may not give the visit to another practitioner (appointment_guard_role).
	err = inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		_, err := r.MoveByStaff(ctx, a.ID, appointment.StaffMove{Start: start.Add(5 * time.Hour), End: start.Add(330 * time.Minute), SlotID: slotU, PractitionerRoleID: s.roleA1, Version: a.VersionID + 1}, now)
		return err
	})
	if err == nil {
		t.Error("MoveByStaff in the patient's scope succeeded")
	}
}

// Two desks move one visit at once from the same version: one moves it, the other changes nothing (version_conflict);
// two desks move two visits onto one practitioner's same time: one is refused by the exclusion constraint.
func TestMoveByStaff_ConcurrentMovesGiveOne(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	roleA2 := seedRoleA2(t, owner)
	a := bookInDB(t, app, s, start)
	b := bookInDB(t, app, s, start.Add(time.Hour))
	move := func(id string, version int32, at time.Time, slotID string) error {
		return inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
			_, err := r.MoveByStaff(ctx, id, appointment.StaffMove{Start: at, End: at.Add(30 * time.Minute), SlotID: slotID, PractitionerRoleID: roleA2, Version: version}, now)
			return err
		})
	}
	for name, tc := range map[string]struct {
		want error
		run  [2]func() error
	}{
		"one visit, one version": {want: appointment.ErrVersionConflict, run: [2]func() error{
			func() error { return move(a.ID, a.VersionID, start.Add(4*time.Hour), slotSoon) },
			func() error { return move(a.ID, a.VersionID, start.Add(5*time.Hour), slotPast) },
		}},
		"two visits, one time": {want: appointment.ErrSlotUnavailable, run: [2]func() error{
			func() error { return move(b.ID, b.VersionID, start.Add(8*time.Hour), slotU) },
			func() error { return move(a.ID, a.VersionID+1, start.Add(8*time.Hour), slotFree) },
		}},
	} {
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i, fn := range tc.run {
			wg.Go(func() { errs[i] = fn() })
		}
		wg.Wait()
		ok, refused := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				ok++
			case errors.Is(err, tc.want):
				refused++
			default:
				t.Errorf("%s: %v, want nil or %v", name, err, tc.want)
			}
		}
		if ok != 1 || refused != 1 {
			t.Errorf("%s: %v; want one move and one %v", name, errs, tc.want)
		}
	}
}
