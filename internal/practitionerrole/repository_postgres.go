package practitionerrole

import (
	"context"
	"errors"
	"time"

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

func (r *postgresRepository) SetDirectory(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.directory", "public")
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

func (r *postgresRepository) PublicByOrganization(ctx context.Context, org string) ([]PractitionerRole, error) {
	var u pgtype.UUID
	if u.Scan(org) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).PublicPractitionerRolesByOrganization(ctx, u)
	if err != nil {
		return nil, err
	}
	roles := make([]PractitionerRole, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, toRole(practitionerroledb.PractitionerRolesByOrganizationRow(row)))
	}
	return roles, nil
}

func (r *postgresRepository) Create(ctx context.Context, org, practitionerID string) (PractitionerRole, error) {
	var o, p pgtype.UUID
	if o.Scan(org) != nil || p.Scan(practitionerID) != nil {
		return PractitionerRole{}, errors.New("practitionerrole: an id is not a UUID")
	}
	row, err := r.q(ctx).CreatePractitionerRole(ctx, practitionerroledb.CreatePractitionerRoleParams{OrganizationID: o, PractitionerID: p})
	if err != nil {
		return PractitionerRole{}, err
	}
	return toRole(practitionerroledb.PractitionerRolesByOrganizationRow(row)), nil
}

func (r *postgresRepository) SetSpecialty(ctx context.Context, id, code string, version int32, at time.Time) (PractitionerRole, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return PractitionerRole{}, ErrConflict
	}
	row, err := r.q(ctx).SetPractitionerRoleSpecialty(ctx, practitionerroledb.SetPractitionerRoleSpecialtyParams{
		ID: u, SpecialtyCode: pgtype.Text{String: code, Valid: true}, VersionID: version,
		UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return PractitionerRole{}, ErrConflict
	}
	if err != nil {
		return PractitionerRole{}, err
	}
	return toRole(practitionerroledb.PractitionerRolesByOrganizationRow(row)), nil
}

// toRole maps a row; every role query selects the same columns.
func toRole(row practitionerroledb.PractitionerRolesByOrganizationRow) PractitionerRole {
	return PractitionerRole{
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), PractitionerID: row.PractitionerID.String(),
		SpecialtyCode: row.SpecialtyCode.String, VersionID: row.VersionID, Active: row.Active, LastUpdated: row.UpdatedAt.Time,
	}
}
