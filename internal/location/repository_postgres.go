package location

import (
	"context"
	"errors"

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
// SetOrganization, or the active locations of listed clinics after SetDirectory.
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

// The rooms and devices are RM1's (docs/clinic-cms/WAVE6.md).

func (r *postgresRepository) Rooms(context.Context, string) ([]Room, error) {
	return nil, errors.ErrUnsupported
}

func (r *postgresRepository) SaveRoom(context.Context, Room) (Room, bool, error) {
	return Room{}, false, errors.ErrUnsupported
}

func (r *postgresRepository) Devices(context.Context, string) ([]Device, error) {
	return nil, errors.ErrUnsupported
}

func (r *postgresRepository) SaveDevice(context.Context, Device) (Device, bool, error) {
	return Device{}, false, errors.ErrUnsupported
}
