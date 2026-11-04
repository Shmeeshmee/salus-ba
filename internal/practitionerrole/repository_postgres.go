package practitionerrole

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/practitionerrole/practitionerroledb"
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

func (r *postgresRepository) q(ctx context.Context) *practitionerroledb.Queries {
	return practitionerroledb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (PractitionerRole, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return PractitionerRole{}, ErrNotFound
	}
	row, err := r.q(ctx).PractitionerRoleByID(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return PractitionerRole{}, ErrNotFound
	}
	if err != nil {
		return PractitionerRole{}, err
	}
	return toRole(practitionerroledb.PractitionerRolesByOrganizationRow(row)), nil
}

func (r *postgresRepository) ByOrganization(ctx context.Context, org string) ([]PractitionerRole, error) {
	var u pgtype.UUID
	if u.Scan(org) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).PractitionerRolesByOrganization(ctx, u)
	if err != nil {
		return nil, err
	}
	roles := make([]PractitionerRole, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, toRole(row))
	}
	return roles, nil
}

// toRole maps a row; both role queries select the same columns.
func toRole(row practitionerroledb.PractitionerRolesByOrganizationRow) PractitionerRole {
	return PractitionerRole{
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), PractitionerID: row.PractitionerID.String(),
		VersionID: row.VersionID, Active: row.Active, LastUpdated: row.UpdatedAt.Time,
	}
}
