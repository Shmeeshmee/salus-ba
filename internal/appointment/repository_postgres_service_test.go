package appointment_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seedService inserts a service of the price list of org and returns its id.
func seedService(t *testing.T, owner *pgxpool.Pool, org string) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `INSERT INTO charge_item_definition (organization_id, name, price_minor) VALUES ($1, 'Synthetic service', 5000)
		RETURNING id::text`, org).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCreate_KeepsTheServiceOfItsClinicAndARescheduleKeepsIt(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	service := seedService(t, owner, clinicA)
	var created, moved appointment.Appointment
	err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		in := newAppt(s, s.patientA, start)
		in.ServiceID = service
		var err error
		created, err = r.Create(ctx, in)
		return err
	})
	if err != nil || created.ServiceID != service {
		t.Fatalf("Create() = %+v, %v; want the service kept", created, err)
	}
	err = inSelf(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.SelfServiceRepository) error {
		var err error
		moved, err = r.Reschedule(ctx, created.ID, appointment.Move{Start: start.Add(time.Hour), End: start.Add(90 * time.Minute), SlotID: slotSoon}, now)
		return err
	})
	if err != nil || moved.ServiceID != service {
		t.Errorf("Reschedule() = %+v, %v; want the service kept", moved, err)
	}
	none := bookInDB(t, app, s, start.Add(3*time.Hour))
	if none.ServiceID != "" {
		t.Errorf("Create() without a service = %q, want none", none.ServiceID)
	}
}

func TestCreate_RefusesAServiceOfAnotherClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	if _, err := owner.Exec(t.Context(), "INSERT INTO organization (id, name, listed) VALUES ($1, 'Other synthetic clinic', true)", clinicB); err != nil {
		t.Fatal(err)
	}
	other := seedService(t, owner, clinicB)
	err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		in := newAppt(s, s.patientA, start)
		in.ServiceID = other
		_, err := r.Create(ctx, in)
		return err
	})
	if err == nil {
		t.Fatal("Create() with another clinic's service passed, want the foreign key to refuse it")
	}
}

func TestVisits_GivesTheClinicsAppointmentsStartingInTheRange(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	in := bookInDB(t, app, s, start)
	bookInDB(t, app, s, start.Add(48*time.Hour)) // after the range
	var got []appointment.Visit
	err := inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		var err error
		got, err = r.Visits(ctx, clinicA, start, start.Add(24*time.Hour))
		return err
	})
	want := []appointment.Visit{{ID: in.ID, PractitionerRoleID: s.roleA1, Mode: schedule.ModeVideo, Status: appointment.StatusBooked}}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("Visits() = %+v, %v; want %+v", got, err, want)
	}
}
