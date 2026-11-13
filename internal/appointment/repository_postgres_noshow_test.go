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

// The no-shows of several patients are counted in one query, in the clinic's scope: only noshow appointments of the
// clinic that started in [since, until]; a patient without any is left out.
func TestNoShowsSince_CountsEachPatientsNoShowsInTheWindow(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	if _, err := owner.Exec(t.Context(), "INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())", s.patientB, clinicA); err != nil {
		t.Fatal(err)
	}
	day := 24 * time.Hour
	for _, x := range []struct {
		patient string
		at      time.Time
		status  string
	}{
		{s.patientA, now.Add(-10 * day), "noshow"},
		{s.patientA, now.Add(-20 * day), "noshow"},
		{s.patientA, now.Add(-200 * day), "noshow"}, // before the window
		{s.patientA, now.Add(-30 * day), "fulfilled"},
		{s.patientB, now.Add(-5 * day), "noshow"},
		{s.patientB, now.Add(5 * day), "booked"},
	} {
		if _, err := owner.Exec(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
			VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes'), $5)`,
			x.patient, s.roleA1, clinicA, x.at, x.status); err != nil {
			t.Fatal(err)
		}
	}
	var other string
	if err := owner.QueryRow(t.Context(), `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id)
		INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Bo', 'Example', '1990-01-01' FROM a RETURNING id::text`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	noShows := appointment.NewPostgresNoShowRepository(app)
	err := inScope(t, app, asClinic, func(ctx context.Context, _ appointment.Repository) error {
		got, err := noShows.NoShowsSince(ctx, clinicA, []string{s.patientA, s.patientB, other, "x"}, now.Add(-180*day), now)
		if err != nil || len(got) != 2 || got[s.patientA] != 2 || got[s.patientB] != 1 {
			t.Errorf("NoShowsSince() = %v, %v; want patient A 2, patient B 1", got, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Without the clinic's scope nothing is seen.
	err = inScope(t, app, func(context.Context, appointment.Repository) error { return nil }, func(ctx context.Context, _ appointment.Repository) error {
		if got, err := noShows.NoShowsSince(ctx, clinicA, []string{s.patientA}, now.Add(-180*day), now); err != nil || len(got) != 0 {
			t.Errorf("outside the scope: %v, %v; want none", got, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
