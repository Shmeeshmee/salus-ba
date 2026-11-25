package location_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/location"
)

const (
	clinicA    = "a0000000-0000-4000-8000-00000000000a"
	siteA      = "a3000000-0000-4000-8000-0000000000a3"
	siteB      = "b3000000-0000-4000-8000-0000000000b3" // clinic B's
	adminAccID = "a1000000-0000-4000-8000-0000000000a1"
)

var (
	roomAdmin = location.Staff{AccountID: adminAccID, OrganizationID: clinicA, Admin: true}
	roomStaff = location.Staff{AccountID: "a2000000-0000-4000-8000-0000000000a2", OrganizationID: clinicA}
	noClinic  = location.Staff{AccountID: "c1000000-0000-4000-8000-0000000000c1"}
	errBroken = errors.New("fake: broken")
)

type txKey struct{}

type fakeTransactor struct{}

func (fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

type fakeAudit struct {
	events []auditevent.AuditEvent
}

func (a *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	a.events = append(a.events, e)
	return nil
}

// fakeRepo holds rooms and devices and shows only those of the clinic set on the transaction, as row-level security
// does. Locations are siteA of clinic A and siteB of another clinic.
type fakeRepo struct {
	location.Repository
	err     error
	clinic  string
	rooms   []location.Room
	devices []location.Device
	outside bool
}

func (r *fakeRepo) seen(ctx context.Context) {
	if ctx.Value(txKey{}) == nil {
		r.outside = true
	}
}

func (r *fakeRepo) SetOrganization(ctx context.Context, id string) error {
	r.seen(ctx)
	r.clinic = id
	return r.err
}

func (r *fakeRepo) visible(locationID string) bool {
	return r.clinic == clinicA && locationID == siteA
}

func (r *fakeRepo) Rooms(ctx context.Context, locationID string) ([]location.Room, error) {
	r.seen(ctx)
	if !r.visible(locationID) {
		return nil, location.ErrNotFound
	}
	out := slices.Clone(r.rooms)
	slices.SortFunc(out, func(a, b location.Room) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *fakeRepo) SaveRoom(ctx context.Context, in location.Room) (location.Room, bool, error) {
	r.seen(ctx)
	if r.err != nil {
		return location.Room{}, false, r.err
	}
	if in.ID == "" {
		if !r.visible(in.LocationID) {
			return location.Room{}, false, location.ErrNotFound
		}
		in.ID, in.VersionID = "a4000000-0000-4000-8000-00000000000"+strconv.Itoa(len(r.rooms)), 1
		r.rooms = append(r.rooms, in)
		return in, true, nil
	}
	for i, old := range r.rooms {
		if old.ID != in.ID || r.clinic != clinicA {
			continue
		}
		if old.VersionID != in.VersionID-1 {
			return location.Room{}, false, nil
		}
		in.LocationID = old.LocationID
		r.rooms[i] = in
		return in, true, nil
	}
	return location.Room{}, false, location.ErrNotFound
}

func (r *fakeRepo) LockRoom(ctx context.Context, id string) (location.Room, error) {
	r.seen(ctx)
	for _, x := range r.rooms {
		if x.ID == id && r.clinic == clinicA {
			return x, r.err
		}
	}
	return location.Room{}, location.ErrNotFound
}

func (r *fakeRepo) Devices(ctx context.Context, locationID string) ([]location.Device, error) {
	r.seen(ctx)
	if !r.visible(locationID) {
		return nil, location.ErrNotFound
	}
	out := slices.Clone(r.devices)
	slices.SortFunc(out, func(a, b location.Device) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *fakeRepo) SaveDevice(ctx context.Context, in location.Device) (location.Device, bool, error) {
	saved := r.rooms
	r.rooms = make([]location.Room, len(r.devices))
	for i, d := range r.devices {
		r.rooms[i] = location.Room(d)
	}
	out, ok, err := r.SaveRoom(ctx, location.Room(in))
	r.devices = make([]location.Device, len(r.rooms))
	for i, x := range r.rooms {
		r.devices[i] = location.Device(x)
	}
	r.rooms = saved
	return location.Device(out), ok, err
}

func (r *fakeRepo) Device(ctx context.Context, id string) (location.Device, error) {
	r.seen(ctx)
	for _, x := range r.devices {
		if x.ID == id && r.clinic == clinicA {
			return x, r.err
		}
	}
	return location.Device{}, location.ErrNotFound
}
