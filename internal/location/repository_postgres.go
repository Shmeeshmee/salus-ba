package location

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/location/locationdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, as salus_app: row-level security shows the clinic set by
// SetOrganization, or the active locations of listed clinics after SetDirectory; it uses the transaction in the
// context when there is one.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *locationdb.Queries {
	return locationdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetDirectory(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.directory", "public")
}

func (r *postgresRepository) Locations(ctx context.Context, org string) ([]Location, error) {
	var o pgtype.UUID
	if o.Scan(org) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).Locations(ctx, o)
	if err != nil {
		return nil, err
	}
	out := make([]Location, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row)
	}
	return out, nil
}

func (r *postgresRepository) PublicLocations(ctx context.Context, org string) ([]Location, error) {
	return r.Locations(ctx, org) // the directory's scope shows the active ones only
}

func (r *postgresRepository) SaveLocation(ctx context.Context, l Location) (Location, bool, error) {
	var org, id pgtype.UUID
	if err := org.Scan(l.OrganizationID); err != nil {
		return Location{}, false, err
	}
	if l.ID == "" {
		row, err := r.q(ctx).InsertLocation(ctx, locationdb.InsertLocationParams{
			OrganizationID: org, Name: l.Name, AddressLine: text(l.AddressLine), PostalCode: text(l.PostalCode),
			CityID: text(l.CityID), Phone: text(l.Phone), TimeZone: text(l.TimeZone), Active: l.Active,
		})
		if err != nil {
			return Location{}, false, refusal(err)
		}
		return fromRow(locationdb.LocationsRow(row)), true, nil
	}
	if id.Scan(l.ID) != nil {
		return Location{}, false, ErrNotFound
	}
	row, err := r.q(ctx).UpdateLocation(ctx, locationdb.UpdateLocationParams{
		Name: l.Name, AddressLine: text(l.AddressLine), PostalCode: text(l.PostalCode), CityID: text(l.CityID),
		Phone: text(l.Phone), TimeZone: text(l.TimeZone), Active: l.Active, ID: id, OrganizationID: org,
		VersionID: int32(l.VersionID - 1), //nolint:gosec // a version is far below 2^31
	})
	if errors.Is(err, pgx.ErrNoRows) { // not the scope's, or at another version
		exists, err := r.q(ctx).LocationExists(ctx, locationdb.LocationExistsParams{ID: id, OrganizationID: org})
		if err != nil || !exists {
			return Location{}, false, errors.Join(ErrNotFound, err)
		}
		return Location{}, false, nil
	}
	if err != nil {
		return Location{}, false, refusal(err)
	}
	return fromRow(locationdb.LocationsRow(row)), true, nil
}

// refusal is ErrInvalid for a write the table refuses (an unknown city, a check), else err.
func refusal(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514") {
		return ErrInvalid
	}
	return err
}

func fromRow(row locationdb.LocationsRow) Location {
	return Location{
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), Name: row.Name, AddressLine: row.AddressLine.String,
		PostalCode: row.PostalCode.String, CityID: row.CityID.String, Phone: row.Phone.String, TimeZone: row.TimeZone,
		VersionID: int(row.VersionID), Active: row.Active,
	}
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

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
