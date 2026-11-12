package organization

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/organization/organizationdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// Its queries run as salus_app, so row-level security shows only the clinic set by SetOrganization.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *organizationdb.Queries {
	return organizationdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetDirectory(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.directory", "public")
}

func (r *postgresRepository) Listed(ctx context.Context) ([]Organization, error) {
	rows, err := r.q(ctx).ListedOrganizations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Organization, 0, len(rows))
	for _, row := range rows {
		// The directory shows no location and no contribution.
		out = append(out, toOrganization(organizationdb.OrganizationByIDRow{
			ID: row.ID, UpdatedAt: row.UpdatedAt, VersionID: row.VersionID, Active: row.Active, Listed: row.Listed, Name: row.Name,
			TimeZone: row.TimeZone, DefaultLanguage: row.DefaultLanguage, PatientTranscriptCopy: row.PatientTranscriptCopy,
		}))
	}
	return out, nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Organization, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Organization{}, ErrNotFound
	}
	row, err := r.q(ctx).OrganizationByID(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrNotFound
	}
	if err != nil {
		return Organization{}, err
	}
	return toOrganization(row), nil
}

func (r *postgresRepository) Update(ctx context.Context, o Organization, at time.Time) (Organization, error) {
	var u pgtype.UUID
	if u.Scan(o.ID) != nil {
		return Organization{}, ErrConflict
	}
	row, err := r.q(ctx).UpdateOrganization(ctx, organizationdb.UpdateOrganizationParams{
		ID: u, Name: o.Name, Listed: o.Listed, TimeZone: o.TimeZone, DefaultLanguage: organizationdb.Language(o.DefaultLanguage), VersionID: o.VersionID,
		PatientTranscriptCopy: o.PatientTranscriptCopy, UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
		RegionID: text(o.Region), CityID: text(o.City), LocationSince: instant(o.LocationSince),
		AnalyticsContributes: o.AnalyticsContributes, ContributesSince: instant(o.ContributesSince),
	})
	if foreignKeyViolation(err) {
		return Organization{}, ErrInvalidLocation
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrConflict
	}
	if err != nil {
		return Organization{}, err
	}
	return toOrganization(organizationdb.OrganizationByIDRow(row)), nil
}

func (r *postgresRepository) Create(ctx context.Context, name string) (Organization, error) {
	row, err := r.q(ctx).CreateOrganization(ctx, name)
	if err != nil {
		return Organization{}, err
	}
	return toOrganization(organizationdb.OrganizationByIDRow(row)), nil
}

func toOrganization(row organizationdb.OrganizationByIDRow) Organization {
	return Organization{
		ID: row.ID.String(), Name: row.Name, TimeZone: row.TimeZone, VersionID: row.VersionID, Active: row.Active,
		Listed: row.Listed, DefaultLanguage: string(row.DefaultLanguage), LastUpdated: row.UpdatedAt.Time,
		PatientTranscriptCopy: row.PatientTranscriptCopy, Region: row.RegionID.String, City: row.CityID.String,
		LocationSince: row.LocationSince.Time, AnalyticsContributes: row.AnalyticsContributes, ContributesSince: row.ContributesSince.Time,
	}
}

// text is s as a nullable column, NULL for "".
func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

// instant is t as a nullable timestamptz, NULL for the zero time.
func instant(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()} }

type postgresStatsRepository struct {
	db postgres.Querier
}

// NewPostgresStatsRepository returns the StatsRepository on db, the salus_stats pool.
func NewPostgresStatsRepository(db postgres.Querier) StatsRepository {
	return &postgresStatsRepository{db: db}
}

func (r *postgresStatsRepository) Locations(ctx context.Context) ([]Location, error) {
	rows, err := organizationdb.New(postgres.Conn(ctx, r.db)).ListLocationsForStats(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Location, len(rows))
	for i, row := range rows {
		out[i] = Location{
			ID: row.ID.String(), TimeZone: row.TimeZone, Region: row.RegionID.String, City: row.CityID.String,
			LocationSince: row.LocationSince.Time, Contributes: row.AnalyticsContributes, ContributesSince: row.ContributesSince.Time,
		}
	}
	return out, nil
}

// foreignKeyViolation reports whether err is Postgres refusing a location that is not a city of a leaf region
// (organization_region_fkey, organization_city_fkey).
func foreignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
