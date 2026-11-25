package appointment_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The rows of the device tests (U3, U4 of docs/clinic-cms/WAVE6.md, RM1): clinic A's site with a room and a device,
// a service that needs the device, a second role, and a slot of role A1 in the room.
type withDevice struct {
	seeded
	site, room, device, service, roleA2, roomSlot string
}

func seedDevice(t *testing.T, owner *pgxpool.Pool) withDevice {
	t.Helper()
	d := withDevice{seeded: seedDB(t, owner)}
	ctx := t.Context()
	err := owner.QueryRow(ctx, `WITH l AS (INSERT INTO location (organization_id, name) VALUES ($1, 'Synthetic site') RETURNING id, organization_id),
		r AS (INSERT INTO room (organization_id, location_id, name) SELECT organization_id, id, 'Room 1' FROM l RETURNING id),
		v AS (INSERT INTO device (organization_id, location_id, name) SELECT organization_id, id, 'Ultrasound' FROM l RETURNING id),
		c AS (INSERT INTO charge_item_definition (organization_id, name, price_minor, device_id) SELECT $1, 'Synthetic scan', 5000, id FROM v RETURNING id)
		SELECT (SELECT id::text FROM l), (SELECT id::text FROM r), (SELECT id::text FROM v), (SELECT id::text FROM c)`, clinicA).
		Scan(&d.site, &d.room, &d.device, &d.service)
	if err == nil {
		err = owner.QueryRow(ctx, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Di', 'Example') RETURNING id, organization_id)
			INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id::text`, clinicA).Scan(&d.roleA2)
	}
	if err == nil {
		err = owner.QueryRow(ctx, `WITH s AS (INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
				VALUES ($1, $2, 30, '[]') RETURNING id)
			INSERT INTO slot (schedule_id, organization_id, period, visit_mode, location_id, room_id)
			SELECT id, $1, tstzrange($3, $4), 'in-person', $5, $6 FROM s RETURNING id::text`,
			clinicA, d.roleA1, start.Add(5*time.Hour), start.Add(5*time.Hour+30*time.Minute), d.site, d.room).Scan(&d.roomSlot)
	}
	if err == nil {
		_, err = owner.Exec(ctx, "INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())", d.patientB, clinicA)
	}
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// scan is an in-person visit with the device's service, of role for patient at at, half an hour.
func (d withDevice) scan(role, patient string, at time.Time) appointment.Appointment {
	a := newAppt(d.seeded, patient, at)
	a.PractitionerRoleID, a.ServiceID, a.Mode = role, d.service, schedule.ModeInPerson
	return a
}

func (d withDevice) book(t *testing.T, app *pgxpool.Pool, a appointment.Appointment) (appointment.Appointment, error) {
	t.Helper()
	var out appointment.Appointment
	err := inScope(t, app, asPatient(a.PatientID), func(ctx context.Context, r appointment.Repository) (err error) {
		out, err = r.Create(ctx, a)
		return err
	})
	return out, err
}

// resources reads, as the owner, the device, room and location of the visit, "" for none.
func resources(t *testing.T, owner *pgxpool.Pool, id string) (device, room, site string) {
	t.Helper()
	if err := owner.QueryRow(t.Context(), `SELECT coalesce(device_id::text, ''), coalesce(room_id::text, ''), coalesce(location_id::text, '')
		FROM appointment WHERE id = $1`, id).Scan(&device, &room, &site); err != nil {
		t.Fatal(err)
	}
	return device, room, site
}

func TestCreate_TwoVisitsThatNeedOneDeviceNeverOverlap(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	d := seedDevice(t, owner)
	first, err := d.book(t, app, d.scan(d.roleA1, d.patientA, start))
	if err != nil {
		t.Fatal(err)
	}
	if device, _, _ := resources(t, owner, first.ID); device != d.device {
		t.Fatalf("the visit's device %q, want its service's %s", device, d.device)
	}
	if _, err := d.book(t, app, d.scan(d.roleA2, d.patientB, start.Add(15*time.Minute))); !errors.Is(err, appointment.ErrResourceBusy) {
		t.Errorf("another practitioner's overlapping visit on the device: %v, want ErrResourceBusy", err)
	}
	if _, err := d.book(t, app, d.scan(d.roleA2, d.patientB, start.Add(30*time.Minute))); err != nil {
		t.Errorf("back to back: %v, want booked", err)
	}
	video := d.scan(d.roleA2, d.patientB, start.Add(-15*time.Minute))
	video.Mode = schedule.ModeVideo
	v, err := d.book(t, app, video)
	if err != nil {
		t.Fatalf("a video visit of the service: %v, want booked (no device)", err)
	}
	if device, _, _ := resources(t, owner, v.ID); device != "" {
		t.Errorf("a video visit's device %q, want none", device)
	}
	plain := newAppt(d.seeded, d.patientB, start.Add(-2*time.Hour))
	plain.PractitionerRoleID, plain.Mode = d.roleA2, schedule.ModeInPerson
	if p, err := d.book(t, app, plain); err != nil {
		t.Errorf("an in-person visit without a service: %v", err)
	} else if device, _, _ := resources(t, owner, p.ID); device != "" {
		t.Errorf("a visit without a service has device %q", device)
	}
}

func TestCancel_AVisitCancelledFreesItsDevice(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	d := seedDevice(t, owner)
	first, err := d.book(t, app, d.scan(d.roleA1, d.patientA, start))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE appointment SET status = 'cancelled', cancelled_at = now(), cancellation_reason = 'pat' WHERE id = $1`, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.book(t, app, d.scan(d.roleA2, d.patientB, start)); err != nil {
		t.Errorf("the device of a cancelled visit: %v, want booked", err)
	}
}

func TestReschedule_IntoABusyDeviceIsRefused(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	d := seedDevice(t, owner)
	if _, err := d.book(t, app, d.scan(d.roleA2, d.patientB, start.Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	mine, err := d.book(t, app, d.scan(d.roleA1, d.patientA, start))
	if err != nil {
		t.Fatal(err)
	}
	err = inSelf(t, app, asPatient(d.patientA), func(ctx context.Context, r appointment.SelfServiceRepository) error {
		_, err := r.Reschedule(ctx, mine.ID, appointment.Move{Start: start.Add(time.Hour), End: start.Add(90 * time.Minute), SlotID: slotSoon}, now)
		return err
	})
	if !errors.Is(err, appointment.ErrResourceBusy) {
		t.Errorf("Reschedule() onto the device's busy time: %v, want ErrResourceBusy", err)
	}
}

// The patient's scope may update a visit's row (its move), but never its device, room or location: the guard keeps
// them as the service and slot give them.
func TestUpdate_TheGuardKeepsTheVisitsDeviceAndRoom(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	d := seedDevice(t, owner)
	in := d.scan(d.roleA1, d.patientA, start.Add(5*time.Hour))
	in.SlotID = d.roomSlot
	a, err := d.book(t, app, in)
	if err != nil {
		t.Fatal(err)
	}
	if device, room, site := resources(t, owner, a.ID); device != d.device || room != d.room || site != d.site {
		t.Fatalf("booked in the room's slot: device %q room %q site %q, want the service's device and the slot's room and site", device, room, site)
	}
	err = inScope(t, app, asPatient(d.patientA), func(ctx context.Context, _ appointment.Repository) error {
		_, err := postgres.Conn(ctx, app).Exec(ctx, `UPDATE appointment SET device_id = NULL, room_id = NULL WHERE id = $1`, a.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if device, room, _ := resources(t, owner, a.ID); device != d.device || room != d.room {
		t.Errorf("after the patient's update: device %q room %q, want both kept", device, room)
	}
}

// As the booking's (TestCreate_TwoConcurrentOverlappingBookingsOfOneRoleGiveOne): two practitioners booking one
// device at once give one visit and one resource_busy, never a deadlock's slot_unavailable.
func TestCreate_TwoConcurrentBookingsOfOneDeviceGiveOne(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	d := seedDevice(t, owner)
	for round := range 10 {
		at := start.Add(time.Duration(round) * time.Hour)
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i, x := range []struct{ role, patient string }{{d.roleA1, d.patientA}, {d.roleA2, d.patientB}} {
			wg.Go(func() {
				_, errs[i] = d.book(t, app, d.scan(x.role, x.patient, at.Add(time.Duration(i)*15*time.Minute)))
			})
		}
		wg.Wait()
		var ok, busy int
		for _, err := range errs {
			switch {
			case err == nil:
				ok++
			case errors.Is(err, appointment.ErrResourceBusy):
				busy++
			default:
				t.Fatalf("round %d: Create() error = %v", round, err)
			}
		}
		if ok != 1 || busy != 1 {
			t.Errorf("round %d: %d booked and %d refused, want one of each", round, ok, busy)
		}
	}
}
