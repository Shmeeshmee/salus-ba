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

// An arrived visit is closed as a booked one is: fulfilled or a no-show, or cancelled (00065's transitions).
func TestCancelAndSetStatus_CloseAnArrivedVisit(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var staffID string
	if err := owner.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role, organization_id)
		VALUES ('desk.b@example.test', 'h', 'reception', $1) RETURNING id::text`, clinicA).Scan(&staffID); err != nil {
		t.Fatal(err)
	}
	var visits [2]appointment.Appointment
	at := now.Add(time.Hour)
	err := inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		for i := range visits {
			in := newAppt(s, s.patientA, start.Add(time.Duration(i)*time.Hour))
			in.Mode, in.BookedBy = schedule.ModeInPerson, staffID
			a, err := r.Create(ctx, in)
			if err != nil {
				return err
			}
			if visits[i], err = r.Arrive(ctx, a.ID, at); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		done, err := r.SetStatus(ctx, visits[0].ID, appointment.StatusFulfilled, at)
		if err != nil || done.Status != appointment.StatusFulfilled || !done.ArrivedAt.Equal(at) {
			t.Errorf("SetStatus(fulfilled) of an arrived visit = %+v, %v; want fulfilled, its arrival kept", done, err)
		}
		cancelled, err := r.Cancel(ctx, visits[1].ID, appointment.ReasonProvider, "", at)
		if err != nil || cancelled.Status != appointment.StatusCancelled {
			t.Errorf("Cancel() of an arrived visit = %+v, %v; want cancelled", cancelled, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
