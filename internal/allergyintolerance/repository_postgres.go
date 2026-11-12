package allergyintolerance

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance/allergyintolerancedb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db (salus_app), using the transaction in the context.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *allergyintolerancedb.Queries {
	return allergyintolerancedb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) ByPatient(ctx context.Context, patientID string) ([]AllergyIntolerance, error) {
	var p pgtype.UUID
	if p.Scan(patientID) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).AllergiesByPatient(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]AllergyIntolerance, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row)
	}
	return out, nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (AllergyIntolerance, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return AllergyIntolerance{}, ErrNotFound
	}
	row, err := r.q(ctx).AllergyByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return AllergyIntolerance{}, ErrNotFound
	case err != nil:
		return AllergyIntolerance{}, err
	}
	return fromRow(row), nil
}

func (r *postgresRepository) Insert(ctx context.Context, a AllergyIntolerance) error {
	ids, err := uuids(a.ID, a.PatientID, a.Source.OrganizationID, a.Source.EncounterID, a.Source.RecorderRoleID,
		a.Confirmation.RoleID, a.Confirmation.OrganizationID, a.Code.TermID)
	if err != nil {
		return err
	}
	err = r.q(ctx).InsertAllergy(ctx, allergyintolerancedb.InsertAllergyParams{
		ID: ids[0], RecordedDate: instant(a.RecordedDate), ConfirmedAt: instant(a.Confirmation.At), PatientID: ids[1],
		OrganizationID: ids[2], EncounterID: ids[3], RecorderRoleID: ids[4], ConfirmedByRoleID: ids[5],
		ConfirmedOrganizationID: ids[6], TermID: ids[7], InProfile: a.InProfile, CodeText: a.Code.Text,
		Category: text(string(a.Category)), ReactionText: text(a.Reaction), Severity: text(string(a.Severity)),
		ClinicalStatus: string(a.ClinicalStatus), VerificationStatus: string(a.VerificationStatus),
	})
	return refusal(err)
}

func (r *postgresRepository) Update(ctx context.Context, a AllergyIntolerance) (int32, error) {
	ids, err := uuids(a.ID, a.Code.TermID, a.Confirmation.RoleID, a.Confirmation.OrganizationID)
	if err != nil {
		return 0, err
	}
	version, err := r.q(ctx).UpdateAllergy(ctx, allergyintolerancedb.UpdateAllergyParams{
		UpdatedAt: instant(a.LastUpdated), InProfile: a.InProfile, TermID: ids[1], CodeText: a.Code.Text,
		Category: text(string(a.Category)), ReactionText: text(a.Reaction), Severity: text(string(a.Severity)),
		ClinicalStatus: string(a.ClinicalStatus), VerificationStatus: string(a.VerificationStatus),
		ConfirmedAt: instant(a.Confirmation.At), ConfirmedByRoleID: ids[2], ConfirmedOrganizationID: ids[3],
		ID: ids[0], VersionID: a.VersionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrVersionConflict
	}
	return version, refusal(err)
}

func (r *postgresRepository) Delete(ctx context.Context, id string, version int32) error {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return ErrNotFound
	}
	n, err := r.q(ctx).DeleteAllergy(ctx, allergyintolerancedb.DeleteAllergyParams{ID: u, VersionID: version})
	switch {
	case err != nil:
		return refusal(err)
	case n == 0:
		return ErrVersionConflict
	}
	return nil
}

func fromRow(row allergyintolerancedb.AllergyIntolerance) AllergyIntolerance {
	return AllergyIntolerance{
		RecordedDate: row.RecordedDate.Time.UTC(), LastUpdated: row.UpdatedAt.Time.UTC(), ID: row.ID.String(),
		Confirmation: Confirmation{At: timeOf(row.ConfirmedAt), RoleID: idOf(row.ConfirmedByRoleID), OrganizationID: idOf(row.ConfirmedOrganizationID)},
		PatientID:    row.PatientID.String(),
		Source: Source{OrganizationID: idOf(row.OrganizationID), EncounterID: idOf(row.EncounterID),
			RecorderRoleID: idOf(row.RecorderRoleID)},
		Code:           Code{TermID: idOf(row.TermID), Text: row.CodeText},
		Reaction:       row.ReactionText.String,
		Category:       Category(row.Category.String),
		Severity:       Severity(row.Severity.String),
		ClinicalStatus: ClinicalStatus(row.ClinicalStatus), VerificationStatus: VerificationStatus(row.VerificationStatus),
		VersionID: row.VersionID, InProfile: row.InProfile,
	}
}

// refusal maps what Postgres refuses to the domain's errors: a scope that may not write the row (row-level
// security, the trigger) to ErrForbidden, a value the table does not take to ErrInvalid.
func refusal(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}
	switch pgErr.Code {
	case "42501":
		return ErrForbidden
	case "23514", "23503", "22001":
		return ErrInvalid
	}
	return err
}

// uuids parses the ids, an empty one as null.
func uuids(ids ...string) ([]pgtype.UUID, error) {
	out := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		if id == "" {
			continue
		}
		if err := out[i].Scan(id); err != nil {
			return nil, ErrInvalid
		}
	}
	return out, nil
}

func idOf(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}

func timeOf(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

func instant(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
