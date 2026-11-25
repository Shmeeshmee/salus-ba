package location

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/location/locationdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *locationdb.Queries {
	return locationdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

// The locations are LC1's.

func (r *postgresRepository) Locations(context.Context, string) ([]Location, error) {
	return nil, errors.ErrUnsupported
}

func (r *postgresRepository) PublicLocations(context.Context, string) ([]Location, error) {
	return nil, errors.ErrUnsupported
}

func (r *postgresRepository) SaveLocation(context.Context, Location) (Location, bool, error) {
	return Location{}, false, errors.ErrUnsupported
}

// The rooms and devices (RM1).

func (r *postgresRepository) Rooms(ctx context.Context, locationID string) ([]Room, error) {
	var u pgtype.UUID
	if u.Scan(locationID) != nil {
		return nil, ErrNotFound
	}
	q := r.q(ctx)
	if ok, err := q.LocationVisible(ctx, u); err != nil || !ok {
		return nil, notVisible(err)
	}
	rows, err := q.Rooms(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Room, 0, len(rows))
	for _, row := range rows {
		out = append(out, toRoom(row))
	}
	return out, nil
}

func (r *postgresRepository) Devices(ctx context.Context, locationID string) ([]Device, error) {
	var u pgtype.UUID
	if u.Scan(locationID) != nil {
		return nil, ErrNotFound
	}
	q := r.q(ctx)
	if ok, err := q.LocationVisible(ctx, u); err != nil || !ok {
		return nil, notVisible(err)
	}
	rows, err := q.Devices(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Device, 0, len(rows))
	for _, row := range rows {
		out = append(out, Device(toRoom(locationdb.RoomsRow(row))))
	}
	return out, nil
}

func (r *postgresRepository) LockRoom(ctx context.Context, id string) (Room, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Room{}, ErrNotFound
	}
	rows, err := r.q(ctx).LockRoom(ctx, u)
	if err != nil || len(rows) == 0 {
		return Room{}, notVisible(err)
	}
	return toRoom(locationdb.RoomsRow(rows[0])), nil
}

func (r *postgresRepository) SaveRoom(ctx context.Context, in Room) (Room, bool, error) {
	q := r.q(ctx)
	if in.ID == "" {
		var loc pgtype.UUID
		if loc.Scan(in.LocationID) != nil {
			return Room{}, false, ErrNotFound
		}
		rows, err := q.InsertRoom(ctx, locationdb.InsertRoomParams{LocationID: loc, Name: in.Name, Active: in.Active})
		if err != nil || len(rows) == 0 {
			return Room{}, false, notVisible(err)
		}
		return toRoom(locationdb.RoomsRow(rows[0])), true, nil
	}
	var id pgtype.UUID
	if id.Scan(in.ID) != nil {
		return Room{}, false, ErrNotFound
	}
	rows, err := q.UpdateRoom(ctx, locationdb.UpdateRoomParams{ID: id, Name: in.Name, Active: in.Active, VersionID: int32(in.VersionID - 1)})
	if err != nil {
		return Room{}, false, err
	}
	if len(rows) == 1 {
		return toRoom(locationdb.RoomsRow(rows[0])), true, nil
	}
	return Room{}, false, stale(q.RoomVisible(ctx, id))
}

func (r *postgresRepository) SaveDevice(ctx context.Context, in Device) (Device, bool, error) {
	q := r.q(ctx)
	if in.ID == "" {
		var loc pgtype.UUID
		if loc.Scan(in.LocationID) != nil {
			return Device{}, false, ErrNotFound
		}
		rows, err := q.InsertDevice(ctx, locationdb.InsertDeviceParams{LocationID: loc, Name: in.Name, Active: in.Active})
		if err != nil || len(rows) == 0 {
			return Device{}, false, notVisible(err)
		}
		return Device(toRoom(locationdb.RoomsRow(rows[0]))), true, nil
	}
	var id pgtype.UUID
	if id.Scan(in.ID) != nil {
		return Device{}, false, ErrNotFound
	}
	rows, err := q.UpdateDevice(ctx, locationdb.UpdateDeviceParams{ID: id, Name: in.Name, Active: in.Active, VersionID: int32(in.VersionID - 1)})
	if err != nil {
		return Device{}, false, err
	}
	if len(rows) == 1 {
		return Device(toRoom(locationdb.RoomsRow(rows[0]))), true, nil
	}
	return Device{}, false, stale(q.DeviceVisible(ctx, id))
}

// notVisible is ErrNotFound for a row the scope does not show, or the query's error.
func notVisible(err error) error {
	if err != nil {
		return fmt.Errorf("location: %w", err)
	}
	return ErrNotFound
}

// stale is the answer to an update that matched no row: nil (a version that moved on) when the row is in the scope,
// else ErrNotFound.
func stale(visible bool, err error) error {
	if err != nil || !visible {
		return notVisible(err)
	}
	return nil
}

func toRoom(row locationdb.RoomsRow) Room {
	return Room{ID: row.ID.String(), LocationID: row.LocationID.String(), Name: row.Name, Active: row.Active, VersionID: int(row.VersionID)}
}
