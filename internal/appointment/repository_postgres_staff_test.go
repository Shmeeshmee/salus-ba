package appointment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestStaffBookingArriveAndDesk_InTheClinicsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var staffID string
	if err := owner.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role, organization_id)
		VALUES ('desk.a@example.test', 'h', 'admin', $1) RETURNING id::text`, clinicA).Scan(&staffID); err != nil {
		t.Fatal(err)
	}
	in := newAppt(s, s.patientA, start)
	in.Mode, in.BookedBy = schedule.ModeInPerson, staffID
	var booked appointment.Appointment
	err := inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		var err error
		booked, err = r.Create(ctx, in)
		return err
	})
	if err != nil || booked.BookedBy != staffID || booked.Mode != schedule.ModeInPerson {
		t.Fatalf("Create in the clinic's scope = %+v, %v; want booked by the staff account", booked, err)
	}
	at := now.Add(time.Hour)
	var arrived appointment.Appointment
	err = inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		var err error
		arrived, err = r.Arrive(ctx, booked.ID, at)
		return err
	})
	if err != nil || arrived.Status != appointment.StatusArrived || !arrived.ArrivedAt.Equal(at) || arrived.VersionID != booked.VersionID+1 {
		t.Fatalf("Arrive = %+v, %v; want arrived at %v", arrived, err, at)
	}
	err = inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		_, err := r.Arrive(ctx, booked.ID, at)
		return err
	})
	if !errors.Is(err, appointment.ErrClosed) {
		t.Errorf("a second arrival = %v, want ErrClosed", err)
	}
	var day []appointment.Appointment
	err = inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		var err error
		day, err = r.Desk(ctx, clinicA, start.Add(-time.Hour), start.Add(time.Hour))
		return err
	})
	if err != nil || len(day) != 1 || day[0].ID != booked.ID || !day[0].ArrivedAt.Equal(at) {
		t.Errorf("Desk = %+v, %v; want the arrived visit", day, err)
	}
}
