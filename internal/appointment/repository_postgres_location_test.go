package appointment_test

import (
	"context"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A visit is read with its slot's location and room (docs/clinic-cms/WAVE6.md U2, U3), and no scope moves it to
// another location: the guard sets both from the slot again.
func TestByID_AVisitCarriesItsSlotsLocationAndRoom(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	d := seedDevice(t, owner)
	var other string
	if err := owner.QueryRow(t.Context(), `INSERT INTO location (organization_id, name) VALUES ($1, 'Synthetic annex') RETURNING id::text`,
		clinicA).Scan(&other); err != nil {
		t.Fatal(err)
	}
	in := d.scan(d.roleA1, d.patientA, start.Add(5*time.Hour))
	in.SlotID = d.roomSlot
	created, err := d.book(t, app, in)
	if err != nil {
		t.Fatal(err)
	}
	if created.LocationID != d.site || created.RoomID != d.room {
		t.Errorf("Create() location %q room %q, want the slot's %q %q", created.LocationID, created.RoomID, d.site, d.room)
	}
	err = inScope(t, app, asPatient(d.patientA), func(ctx context.Context, _ appointment.Repository) error {
		_, err := postgres.Conn(ctx, app).Exec(ctx, `UPDATE appointment SET location_id = $2 WHERE id = $1`, created.ID, other)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var read appointment.Appointment
	err = inScope(t, app, asPatient(d.patientA), func(ctx context.Context, r appointment.Repository) (err error) {
		read, err = r.ByID(ctx, created.ID)
		return err
	})
	if err != nil || read.LocationID != d.site || read.RoomID != d.room {
		t.Errorf("ByID() after a direct change: location %q room %q, %v; want the slot's %q %q", read.LocationID, read.RoomID, err, d.site, d.room)
	}
	var video appointment.Appointment
	if video, err = d.book(t, app, newAppt(d.seeded, d.patientB, start)); err != nil || video.LocationID != "" || video.RoomID != "" {
		t.Errorf("a video visit: location %q room %q, %v; want none", video.LocationID, video.RoomID, err)
	}
}
