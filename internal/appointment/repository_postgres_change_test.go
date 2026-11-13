package appointment_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The history is appended for an appointment the scope sees, with every kind's fields, and read back oldest first.
func TestAddChangeAndChanges_KeepTheHistoryOfAnAppointmentTheScopeSees(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var a appointment.Appointment
	err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		a, err = r.Create(ctx, newAppt(s, s.patientA, start))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	moved := appointment.Change{
		At: now, AppointmentID: a.ID, Kind: appointment.ChangeRescheduled, Via: appointment.ViaApp, ByAccountID: s.accountA,
		FromSlotID: slotFree, ToSlotID: slotSoon, FromStart: start, FromEnd: start.Add(30 * time.Minute),
		ToStart: start.Add(time.Hour), ToEnd: start.Add(90 * time.Minute),
	}
	set := appointment.Change{
		At: now.Add(time.Minute), AppointmentID: a.ID, Kind: appointment.ChangeStatusSet, Via: appointment.ViaSystem,
		Agent: "consultation_window_close", FromStatus: appointment.StatusBooked, ToStatus: appointment.StatusNoShow,
	}
	// The patient's app writes in the patient's scope, a job in the clinic's (the policy ties via to the scope).
	for _, w := range []struct {
		scope func(ctx context.Context, r appointment.Repository) error
		c     *appointment.Change
	}{{asPatient(s.patientA), &moved}, {asClinic, &set}} {
		if err := inScope(t, app, w.scope, func(ctx context.Context, r appointment.Repository) error {
			var err error
			w.c.ID, err = r.AddChange(ctx, *w.c)
			return err
		}); err != nil {
			t.Fatalf("AddChange(%s) = %v", w.c.Kind, err)
		}
	}
	var got []appointment.Change
	err = inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		var err error
		got, err = r.Changes(ctx, a.ID)
		return err
	})
	if err != nil || len(got) != 2 {
		t.Fatalf("Changes() = %+v, %v; want two", got, err)
	}
	for i, want := range []appointment.Change{moved, set} {
		g := got[i]
		if g.ID != want.ID || !g.At.Equal(want.At) || g.Kind != want.Kind || g.Via != want.Via || g.ByAccountID != want.ByAccountID ||
			g.Agent != want.Agent || g.FromSlotID != want.FromSlotID || g.ToSlotID != want.ToSlotID || g.FromStatus != want.FromStatus ||
			g.ToStatus != want.ToStatus || !g.FromStart.Equal(want.FromStart) || !g.ToEnd.Equal(want.ToEnd) {
			t.Errorf("change %d = %+v,\nwant %+v", i, g, want)
		}
	}
	// Patient B's scope does not see the appointment: no change is written, and none is read.
	err = inScope(t, app, asPatient(s.patientB), func(ctx context.Context, r appointment.Repository) error {
		if _, err := r.AddChange(ctx, set); !errors.Is(err, appointment.ErrNotFound) {
			t.Errorf("AddChange() in patient B's scope = %v, want ErrNotFound", err)
		}
		got, err := r.Changes(ctx, a.ID)
		if err == nil && len(got) != 0 {
			t.Errorf("patient B reads %+v", got)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A job stores a link for its appointment; the retention role deletes it once its period after the expiry is over.
func TestInsertTokenAndDeleteTokensExpiredBefore(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var a appointment.Appointment
	err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		a, err = r.Create(ctx, newAppt(s, s.patientA, start))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	tokens := appointment.NewPostgresTokenRepository(app)
	tok := appointment.Token{
		CreatedAt: now, StartAt: a.Start, AppointmentID: a.ID, OrganizationID: clinicA, PatientID: s.patientA, AccountID: s.accountA,
		Language: "bs", Channel: appointment.ChannelEmail, Hash: sha256.Sum256([]byte("synthetic")),
	}
	err = inScope(t, app, func(ctx context.Context, r appointment.Repository) error {
		if err := r.SetClinic(ctx, clinicA); err != nil {
			return err
		}
		return r.SetNotice(ctx, a.ID)
	}, func(ctx context.Context, _ appointment.Repository) error { return tokens.InsertToken(ctx, tok) })
	if err != nil {
		t.Fatalf("InsertToken() = %v", err)
	}
	if err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		return tokens.InsertToken(ctx, appointment.Token{AppointmentID: "not a uuid"})
	}); err == nil {
		t.Error("a link with an id that is not a UUID was stored")
	}
	purger := appointment.NewTokenPurger(appointment.NewPostgresTokenRepository(owner))
	if n, err := purger.Purge(t.Context(), a.Start); err != nil || n != 0 {
		t.Errorf("Purge(start) = %d, %v; want 0", n, err)
	}
	if n, err := purger.Purge(t.Context(), a.Start.Add(time.Second)); err != nil || n != 1 {
		t.Errorf("Purge(after start) = %d, %v; want 1", n, err)
	}
}
