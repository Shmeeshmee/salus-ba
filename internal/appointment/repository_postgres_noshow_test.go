package appointment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A correction moves the appointment from the status it names into the other, at its next version, in the clinic's
// scope; an appointment in another status (booked, cancelled, corrected meanwhile) is closed.
func TestCorrectStatus_MovesOnlyFromTheStatusItNames(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var a, b appointment.Appointment
	at := now.Add(time.Hour)
	err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		if a, err = r.Create(ctx, newAppt(s, s.patientA, start)); err != nil {
			return err
		}
		b, err = r.Create(ctx, newAppt(s, s.patientA, start.Add(time.Hour)))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	noShows := appointment.NewPostgresNoShowRepository(app)
	err = inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		if _, err := noShows.CorrectStatus(ctx, a.ID, appointment.StatusBooked, appointment.StatusFulfilled, at); !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("CorrectStatus() of a booked one named as booked = %v, want ErrClosed", err)
		}
		if _, err := r.SetStatus(ctx, a.ID, appointment.StatusNoShow, at); err != nil {
			return err
		}
		got, err := noShows.CorrectStatus(ctx, a.ID, appointment.StatusNoShow, appointment.StatusFulfilled, at.Add(time.Minute))
		if err != nil || got.Status != appointment.StatusFulfilled || got.VersionID != 3 || !got.LastUpdated.Equal(at.Add(time.Minute)) {
			t.Errorf("CorrectStatus() = %+v, %v; want fulfilled at version 3", got, err)
		}
		if _, err := noShows.CorrectStatus(ctx, a.ID, appointment.StatusNoShow, appointment.StatusFulfilled, at); !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("a second CorrectStatus() from noshow = %v, want ErrClosed", err)
		}
		if _, err := noShows.CorrectStatus(ctx, b.ID, appointment.StatusNoShow, appointment.StatusFulfilled, at); !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("CorrectStatus() of a booked one = %v, want ErrClosed", err)
		}
		if _, err := noShows.CorrectStatus(ctx, "x", appointment.StatusNoShow, appointment.StatusFulfilled, at); !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("CorrectStatus() of an id that is not a UUID = %v, want ErrClosed", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
