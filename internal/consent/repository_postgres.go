package consent

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/consent/consentdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// organizationFK is the constraint an unknown clinic violates.
const organizationFK = "consent_organization_id_fkey"

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// Its queries run as salus_app, so row-level security shows only the consents of the scope set by SetPatient
// or SetClinic.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) Grant(ctx context.Context, patientID, organizationID string, at time.Time) (Consent, error) {
	var p, o pgtype.UUID
	if err := p.Scan(patientID); err != nil {
		return Consent{}, errors.New("consent: the patient id is not a UUID")
	}
	if o.Scan(organizationID) != nil {
		return Consent{}, ErrClinicNotFound
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).GrantConsent(ctx, consentdb.GrantConsentParams{
		PatientID: p, OrganizationID: o, DateTime: pgtype.Timestamptz{Time: at, Valid: true},
	})
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.ConstraintName == organizationFK {
		return Consent{}, ErrClinicNotFound
	}
	return toConsent(consentdb.ConsentByIDRow(row), err, ErrActive)
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Consent, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Consent{}, ErrNotFound
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).ConsentByID(ctx, u)
	return toConsent(row, err, ErrNotFound)
}

func (r *postgresRepository) Revoke(ctx context.Context, id string, at time.Time) (Consent, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Consent{}, ErrInactive
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).RevokeConsent(ctx, consentdb.RevokeConsentParams{
		ID: u, RevokedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return toConsent(consentdb.ConsentByIDRow(row), err, ErrInactive)
}

func (r *postgresRepository) ByPatient(ctx context.Context, patientID string) ([]Consent, error) {
	var u pgtype.UUID
	if u.Scan(patientID) != nil {
		return nil, nil
	}
	rows, err := consentdb.New(postgres.Conn(ctx, r.db)).ConsentsByPatient(ctx, u)
	if err != nil {
		return nil, err
	}
	consents := make([]Consent, 0, len(rows))
	for _, row := range rows {
		c, _ := toConsent(consentdb.ConsentByIDRow(row), nil, nil)
		consents = append(consents, c)
	}
	return consents, nil
}

func (r *postgresRepository) State(ctx context.Context, patientID, organizationID string) (access.Consent, error) {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(organizationID) != nil {
		return access.Consent{}, nil
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).ConsentState(ctx, consentdb.ConsentStateParams{PatientID: p, OrganizationID: o})
	if err != nil {
		return access.Consent{}, err
	}
	return access.Consent{Active: row.Active, RevokedAt: row.RevokedAt.Time}, nil
}

// toConsent maps a row to the model, and no row to noRows.
func toConsent(row consentdb.ConsentByIDRow, err, noRows error) (Consent, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Consent{}, noRows
	}
	if err != nil {
		return Consent{}, err
	}
	return Consent{
		ID: row.ID.String(), PatientID: row.PatientID.String(), OrganizationID: row.OrganizationID.String(),
		Status: Status(row.Status), DateTime: row.DateTime.Time, RevokedAt: row.RevokedAt.Time,
		LastUpdated: row.UpdatedAt.Time, VersionID: row.VersionID,
	}, nil
}
