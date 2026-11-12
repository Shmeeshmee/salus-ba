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

// organizationFK is the constraint an unknown clinic violates, per kind.
var organizationFK = map[Kind]string{
	KindCare: "consent_organization_id_fkey", KindTranscription: "transcription_consent_organization_id_fkey",
	KindHistory: "history_consent_organization_id_fkey",
}

// insufficientPrivilege is the SQLSTATE of a row-level security policy's refusal of a row.
const insufficientPrivilege = "42501"

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

func (r *postgresRepository) Grant(ctx context.Context, kind Kind, patientID, organizationID string, at time.Time) (Consent, error) {
	var p, o pgtype.UUID
	if err := p.Scan(patientID); err != nil {
		return Consent{}, errors.New("consent: the patient id is not a UUID")
	}
	if o.Scan(organizationID) != nil {
		return Consent{}, ErrClinicNotFound
	}
	q := consentdb.New(postgres.Conn(ctx, r.db))
	ts := pgtype.Timestamptz{Time: at, Valid: true}
	var row consentdb.GrantConsentRow
	var err error
	if kind == KindTranscription {
		var t consentdb.GrantTranscriptionConsentRow
		t, err = q.GrantTranscriptionConsent(ctx, consentdb.GrantTranscriptionConsentParams{PatientID: p, OrganizationID: o, DateTime: ts})
		row = consentdb.GrantConsentRow(t)
	} else {
		row, err = q.GrantConsent(ctx, consentdb.GrantConsentParams{PatientID: p, OrganizationID: o, DateTime: ts})
	}
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.ConstraintName == organizationFK[kind] {
		return Consent{}, ErrClinicNotFound
	}
	return toConsent(kind, row, err, ErrActive)
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Consent, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Consent{}, ErrNotFound
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).ConsentByID(ctx, u)
	c, err := toConsent(Kind(row.Kind), consentdb.GrantConsentRow{
		ID: row.ID, UpdatedAt: row.UpdatedAt, DateTime: row.DateTime, RevokedAt: row.RevokedAt, PatientID: row.PatientID,
		OrganizationID: row.OrganizationID, VersionID: row.VersionID, Status: row.Status,
	}, err, ErrNotFound)
	if err != nil {
		return Consent{}, err
	}
	return r.withShared(ctx, c)
}

func (r *postgresRepository) Revoke(ctx context.Context, kind Kind, id string, at time.Time) (Consent, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Consent{}, ErrInactive
	}
	q := consentdb.New(postgres.Conn(ctx, r.db))
	ts := pgtype.Timestamptz{Time: at, Valid: true}
	switch kind {
	case KindTranscription:
		row, err := q.RevokeTranscriptionConsent(ctx, consentdb.RevokeTranscriptionConsentParams{ID: u, RevokedAt: ts})
		return toConsent(kind, consentdb.GrantConsentRow(row), err, ErrInactive)
	case KindHistory:
		row, err := q.RevokeHistoryConsent(ctx, consentdb.RevokeHistoryConsentParams{ID: u, RevokedAt: ts})
		return toConsent(kind, consentdb.GrantConsentRow(row), err, ErrInactive)
	}
	row, err := q.RevokeConsent(ctx, consentdb.RevokeConsentParams{ID: u, RevokedAt: ts})
	return toConsent(kind, consentdb.GrantConsentRow(row), err, ErrInactive)
}

func (r *postgresRepository) RevokeTranscription(ctx context.Context, patientID, organizationID string, at time.Time) (Consent, error) {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(organizationID) != nil {
		return Consent{}, ErrInactive
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).RevokeActiveTranscriptionConsent(ctx, consentdb.RevokeActiveTranscriptionConsentParams{
		PatientID: p, OrganizationID: o, RevokedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return toConsent(KindTranscription, consentdb.GrantConsentRow(row), err, ErrInactive)
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
		c, _ := toConsent(Kind(row.Kind), consentdb.GrantConsentRow{
			ID: row.ID, UpdatedAt: row.UpdatedAt, DateTime: row.DateTime, RevokedAt: row.RevokedAt, PatientID: row.PatientID,
			OrganizationID: row.OrganizationID, VersionID: row.VersionID, Status: row.Status,
		}, nil, nil)
		consents = append(consents, c)
	}
	return r.withAllShared(ctx, consents)
}

func (r *postgresRepository) State(ctx context.Context, patientID, organizationID string) (access.Consent, error) {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(organizationID) != nil {
		return access.Consent{}, nil
	}
	q := consentdb.New(postgres.Conn(ctx, r.db))
	row, err := q.ConsentState(ctx, consentdb.ConsentStateParams{PatientID: p, OrganizationID: o})
	if err != nil {
		return access.Consent{}, err
	}
	st := access.Consent{Active: row.Active, RevokedAt: row.RevokedAt.Time, Transcription: row.Transcription, History: row.History}
	if !st.History {
		return st, nil
	}
	shared, err := q.HistoryShared(ctx, consentdb.HistorySharedParams{PatientID: p, OrganizationID: o})
	if err != nil {
		return access.Consent{}, err
	}
	st.Shared = make([]string, 0, len(shared))
	for _, e := range shared {
		st.Shared = append(st.Shared, e.String())
	}
	return st, nil
}

// toConsent maps a row of kind to the model, and no row to noRows.
func toConsent(kind Kind, row consentdb.GrantConsentRow, err, noRows error) (Consent, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Consent{}, noRows
	}
	if err != nil {
		return Consent{}, err
	}
	return Consent{
		ID: row.ID.String(), PatientID: row.PatientID.String(), OrganizationID: row.OrganizationID.String(), Kind: kind,
		Status: Status(row.Status), DateTime: row.DateTime.Time, RevokedAt: row.RevokedAt.Time,
		LastUpdated: row.UpdatedAt.Time, VersionID: row.VersionID,
	}, nil
}
