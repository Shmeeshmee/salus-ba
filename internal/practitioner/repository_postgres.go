package practitioner

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/practitioner/practitionerdb"
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

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Practitioner, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Practitioner{}, ErrNotFound
	}
	row, err := practitionerdb.New(postgres.Conn(ctx, r.db)).PractitionerByID(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return Practitioner{}, ErrNotFound
	}
	if err != nil {
		return Practitioner{}, err
	}
	return Practitioner{
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), GivenName: row.GivenName,
		FamilyName: row.FamilyName, VersionID: row.VersionID, Active: row.Active, LastUpdated: row.UpdatedAt.Time,
	}, nil
}
