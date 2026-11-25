package location_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const clinicB = "b0000000-0000-4000-8000-00000000000b"

// roomStore is the repository on a throwaway database with clinic A's siteA and clinic B's siteB.
type roomStore struct {
	t    *testing.T
	app  *pgxpool.Pool
	repo location.Repository
}

func newRoomStore(t *testing.T) roomStore {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	_, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name, listed, updated_at, contributes_since) VALUES
		($1, 'Synthetic clinic A', true, '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z'),
		($2, 'Synthetic clinic B', true, '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z')`, clinicA, clinicB)
	if err == nil {
		_, err = owner.Exec(t.Context(), `INSERT INTO location (id, organization_id, name) VALUES
			($1, $2, 'Synthetic site A'), ($3, $4, 'Synthetic site B')`, siteA, clinicA, siteB, clinicB)
	}
	if err != nil {
		t.Fatal(err)
	}
	return roomStore{t: t, app: app, repo: location.NewPostgresRepository(app)}
}

// in runs fn in a transaction in the clinic's scope, or with set when given (another scope).
func (s roomStore) in(org string, fn func(ctx context.Context) error) error {
	s.t.Helper()
	return postgres.NewTransactor(s.app).WithTx(s.t.Context(), func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		return fn(ctx)
	})
}

func TestPostgres_Rooms_AddChangeAndListInTheClinicsScope(t *testing.T) {
	t.Parallel()
	s := newRoomStore(t)
	var b, a location.Room
	err := s.in(clinicA, func(ctx context.Context) (err error) {
		if b, _, err = s.repo.SaveRoom(ctx, location.Room{LocationID: siteA, Name: "Room B", Active: true}); err != nil {
			return err
		}
		a, _, err = s.repo.SaveRoom(ctx, location.Room{LocationID: siteA, Name: "Room A", Active: true})
		return err
	})
	if err != nil || b.ID == "" || b.VersionID != 1 || b.LocationID != siteA || !b.Active {
		t.Fatalf("insert: %+v, %v", b, err)
	}
	var changed location.Room
	var ok bool
	err = s.in(clinicA, func(ctx context.Context) (err error) {
		changed, ok, err = s.repo.SaveRoom(ctx, location.Room{ID: b.ID, LocationID: siteB, Name: "Room C", VersionID: 2})
		return err
	})
	if err != nil || !ok || changed.Name != "Room C" || changed.VersionID != 2 || changed.Active || changed.LocationID != siteA {
		t.Fatalf("update: %+v, %v, %v", changed, ok, err)
	}
	err = s.in(clinicA, func(ctx context.Context) (err error) {
		_, ok, err = s.repo.SaveRoom(ctx, location.Room{ID: b.ID, Name: "Room D", VersionID: 2})
		return err
	})
	if err != nil || ok {
		t.Fatalf("a stale version: %v, %v, want false", ok, err)
	}
	var got []location.Room
	if err := s.in(clinicA, func(ctx context.Context) (err error) { got, err = s.repo.Rooms(ctx, siteA); return err }); err != nil ||
		len(got) != 2 || got[0].ID != a.ID || got[1].Name != "Room C" {
		t.Fatalf("list: %+v, %v", got, err)
	}
}

func TestPostgres_Rooms_AnotherClinicSeesAndChangesNone(t *testing.T) {
	t.Parallel()
	s := newRoomStore(t)
	var r location.Room
	if err := s.in(clinicA, func(ctx context.Context) (err error) {
		r, _, err = s.repo.SaveRoom(ctx, location.Room{LocationID: siteA, Name: "Room 1", Active: true})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	calls := map[string]func(ctx context.Context) error{
		"list": func(ctx context.Context) error { _, err := s.repo.Rooms(ctx, siteA); return err },
		"add": func(ctx context.Context) error {
			_, _, err := s.repo.SaveRoom(ctx, location.Room{LocationID: siteA, Name: "x"})
			return err
		},
		"change": func(ctx context.Context) error {
			_, _, err := s.repo.SaveRoom(ctx, location.Room{ID: r.ID, Name: "x", VersionID: 2})
			return err
		},
		"bad id": func(ctx context.Context) error { _, err := s.repo.Rooms(ctx, "not-a-uuid"); return err },
	}
	for name, call := range calls {
		if err := s.in(clinicB, call); !errors.Is(err, location.ErrNotFound) {
			t.Errorf("%s from clinic B: %v, want ErrNotFound", name, err)
		}
	}
	// A patient's scope (the app's own) sees no room, even with the clinic set.
	err := postgres.NewTransactor(s.app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.patient_id", "c0000000-0000-4000-8000-00000000000c"); err != nil {
			return err
		}
		if err := s.repo.SetOrganization(ctx, clinicA); err != nil {
			return err
		}
		_, err := s.repo.Rooms(ctx, siteA)
		return err
	})
	if !errors.Is(err, location.ErrNotFound) {
		t.Errorf("a patient's scope: %v, want ErrNotFound", err)
	}
}

func TestPostgres_Devices_AsRooms(t *testing.T) {
	t.Parallel()
	s := newRoomStore(t)
	var d location.Device
	var ok bool
	err := s.in(clinicA, func(ctx context.Context) (err error) {
		if d, _, err = s.repo.SaveDevice(ctx, location.Device{LocationID: siteA, Name: "Ultrasound", Active: true}); err != nil {
			return err
		}
		d, ok, err = s.repo.SaveDevice(ctx, location.Device{ID: d.ID, Name: "Ultrasound 2", Active: true, VersionID: 2})
		return err
	})
	if err != nil || !ok || d.VersionID != 2 || d.Name != "Ultrasound 2" || d.LocationID != siteA {
		t.Fatalf("save: %+v, %v, %v", d, ok, err)
	}
	var got []location.Device
	if err := s.in(clinicA, func(ctx context.Context) (err error) { got, err = s.repo.Devices(ctx, siteA); return err }); err != nil ||
		len(got) != 1 || got[0].ID != d.ID {
		t.Fatalf("list: %+v, %v", got, err)
	}
	for name, call := range map[string]func(ctx context.Context) error{
		"list": func(ctx context.Context) error { _, err := s.repo.Devices(ctx, siteA); return err },
		"change": func(ctx context.Context) error {
			_, _, err := s.repo.SaveDevice(ctx, location.Device{ID: d.ID, Name: "x", VersionID: 3})
			return err
		},
		"add": func(ctx context.Context) error {
			_, _, err := s.repo.SaveDevice(ctx, location.Device{LocationID: siteA, Name: "x"})
			return err
		},
	} {
		if err := s.in(clinicB, call); !errors.Is(err, location.ErrNotFound) {
			t.Errorf("%s from clinic B: %v, want ErrNotFound", name, err)
		}
	}
}

func TestPostgres_LockRoom_TheScopesRoomLocked(t *testing.T) {
	t.Parallel()
	s := newRoomStore(t)
	var r, got location.Room
	if err := s.in(clinicA, func(ctx context.Context) (err error) {
		r, _, err = s.repo.SaveRoom(ctx, location.Room{LocationID: siteA, Name: "Room 1", Active: true})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.in(clinicA, func(ctx context.Context) (err error) { got, err = s.repo.LockRoom(ctx, r.ID); return err }); err != nil || got != r {
		t.Fatalf("LockRoom() = %+v, %v; want %+v", got, err, r)
	}
	for _, id := range []string{r.ID, "not-a-uuid"} {
		if err := s.in(clinicB, func(ctx context.Context) error { _, err := s.repo.LockRoom(ctx, id); return err }); !errors.Is(err, location.ErrNotFound) {
			t.Errorf("LockRoom(%s) from clinic B: %v, want ErrNotFound", id, err)
		}
	}
}

func TestPostgres_Device_TheScopesDevice(t *testing.T) {
	t.Parallel()
	s := newRoomStore(t)
	var d, got location.Device
	if err := s.in(clinicA, func(ctx context.Context) (err error) {
		d, _, err = s.repo.SaveDevice(ctx, location.Device{LocationID: siteA, Name: "Device 1"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.in(clinicA, func(ctx context.Context) (err error) { got, err = s.repo.Device(ctx, d.ID); return err }); err != nil || got != d {
		t.Fatalf("Device() = %+v, %v; want %+v", got, err, d)
	}
	for _, id := range []string{d.ID, "not-a-uuid"} {
		if err := s.in(clinicB, func(ctx context.Context) error { _, err := s.repo.Device(ctx, id); return err }); !errors.Is(err, location.ErrNotFound) {
			t.Errorf("Device(%s) from clinic B: %v, want ErrNotFound", id, err)
		}
	}
}
