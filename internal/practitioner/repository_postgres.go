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

func (r *postgresRepository) SetDirectory(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.directory", "public")
}

func (r *postgresRepository) PublicByIDs(ctx context.Context, ids []string) ([]Practitioner, error) {
	uuids := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		var u pgtype.UUID
		if u.Scan(id) == nil {
			uuids = append(uuids, u)
		}
	}
	rows, err := practitionerdb.New(postgres.Conn(ctx, r.db)).PublicPractitionersByIDs(ctx, uuids)
	if err != nil {
		return nil, err
	}
	out := make([]Practitioner, 0, len(rows))
	for _, row := range rows {
		out = append(out, Practitioner{
			ID: row.ID.String(), GivenName: row.GivenName, FamilyName: row.FamilyName, VersionID: row.VersionID,
			Active: row.Active, LastUpdated: row.UpdatedAt.Time,
		})
	}
	return out, nil
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
	return toPractitioner(row), nil
}

func (r *postgresRepository) Create(ctx context.Context, org, givenName, familyName string) (Practitioner, error) {
	var u pgtype.UUID
	if err := u.Scan(org); err != nil {
		return Practitioner{}, errors.New("practitioner: the clinic id is not a UUID")
	}
	row, err := practitionerdb.New(postgres.Conn(ctx, r.db)).CreatePractitioner(ctx, practitionerdb.CreatePractitionerParams{
		OrganizationID: u, GivenName: givenName, FamilyName: familyName,
	})
	if err != nil {
		return Practitioner{}, err
	}
	return toPractitioner(practitionerdb.PractitionerByIDRow(row)), nil
}

func toPractitioner(row practitionerdb.PractitionerByIDRow) Practitioner {
	return Practitioner{
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), GivenName: row.GivenName,
		FamilyName: row.FamilyName, VersionID: row.VersionID, Active: row.Active, LastUpdated: row.UpdatedAt.Time,
	}
}
