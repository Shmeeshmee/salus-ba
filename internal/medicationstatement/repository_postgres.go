package medicationstatement

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/medicationstatement/medicationstatementdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db (salus_app), using the transaction in the context.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *medicationstatementdb.Queries {
	return medicationstatementdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) ByPatient(ctx context.Context, patientID string) ([]MedicationStatement, error) {
	var p pgtype.UUID
	if p.Scan(patientID) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).MedicationsByPatient(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]MedicationStatement, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row)
	}
	return out, nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (MedicationStatement, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return MedicationStatement{}, ErrNotFound
	}
	row, err := r.q(ctx).MedicationByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return MedicationStatement{}, ErrNotFound
	case err != nil:
		return MedicationStatement{}, err
	}
	return fromRow(row), nil
}

func (r *postgresRepository) Insert(ctx context.Context, a MedicationStatement) error {
	ids, err := uuids(a.ID, a.PatientID, a.Source.OrganizationID, a.Source.EncounterID, a.Source.RecorderRoleID,
		a.Confirmation.RoleID, a.Confirmation.OrganizationID, a.Code.TermID)
	if err != nil {
		return err
	}
	d, err := dosageParams(a.Dosage)
	if err != nil {
		return err
	}
	err = r.q(ctx).InsertMedication(ctx, medicationstatementdb.InsertMedicationParams{
		ID: ids[0], DateAsserted: instant(a.DateAsserted), ConfirmedAt: instant(a.Confirmation.At), PatientID: ids[1],
		OrganizationID: ids[2], EncounterID: ids[3], RecorderRoleID: ids[4], ConfirmedByRoleID: ids[5],
		ConfirmedOrganizationID: ids[6], TermID: ids[7], DoseValue: d.value, Period: d.period, Frequency: d.frequency,
		InProfile: a.InProfile, AsNeeded: a.Dosage.AsNeeded, MedicationText: a.Code.Text, DoseUnit: text(a.Dosage.DoseUnit),
		PeriodUnit: text(a.Dosage.PeriodUnit), Status: string(a.Status),
	})
	return refusal(err)
}

func (r *postgresRepository) Update(ctx context.Context, a MedicationStatement) (int32, error) {
	ids, err := uuids(a.ID, a.Code.TermID, a.Confirmation.RoleID, a.Confirmation.OrganizationID)
	if err != nil {
		return 0, err
	}
	d, err := dosageParams(a.Dosage)
	if err != nil {
		return 0, err
	}
	version, err := r.q(ctx).UpdateMedication(ctx, medicationstatementdb.UpdateMedicationParams{
		UpdatedAt: instant(a.LastUpdated), InProfile: a.InProfile, TermID: ids[1], MedicationText: a.Code.Text,
		DoseValue: d.value, DoseUnit: text(a.Dosage.DoseUnit), Frequency: d.frequency, Period: d.period,
		PeriodUnit: text(a.Dosage.PeriodUnit), AsNeeded: a.Dosage.AsNeeded, Status: string(a.Status),
		ConfirmedAt: instant(a.Confirmation.At), ConfirmedByRoleID: ids[2], ConfirmedOrganizationID: ids[3],
		ID: ids[0], VersionID: a.VersionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, r.missed(ctx, a.ID, a.VersionID)
	}
	return version, refusal(err)
}

func (r *postgresRepository) Delete(ctx context.Context, id string, version int32) error {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return ErrNotFound
	}
	n, err := r.q(ctx).DeleteMedication(ctx, medicationstatementdb.DeleteMedicationParams{ID: u, VersionID: version})
	switch {
	case err != nil:
		return refusal(err)
	case n == 0:
		return r.missed(ctx, id, version)
	}
	return nil
}

// missed says why a write of the entry id at version matched no row, read again in the transaction: ErrNotFound when
// the scope no longer sees it, ErrVersionConflict when it is at another version, else ErrForbidden (the scope sees it
// but may not write it).
func (r *postgresRepository) missed(ctx context.Context, id string, version int32) error {
	cur, err := r.ByID(ctx, id)
	switch {
	case err != nil:
		return err
	case cur.VersionID != version:
		return ErrVersionConflict
	}
	return ErrForbidden
}

func fromRow(row medicationstatementdb.MedicationStatement) MedicationStatement {
	return MedicationStatement{
		DateAsserted: row.DateAsserted.Time.UTC(), LastUpdated: row.UpdatedAt.Time.UTC(), ID: row.ID.String(),
		Confirmation: Confirmation{At: timeOf(row.ConfirmedAt), RoleID: idOf(row.ConfirmedByRoleID), OrganizationID: idOf(row.ConfirmedOrganizationID)},
		PatientID:    row.PatientID.String(),
		Source: Source{OrganizationID: idOf(row.OrganizationID), EncounterID: idOf(row.EncounterID),
			RecorderRoleID: idOf(row.RecorderRoleID)},
		Code:   Code{TermID: idOf(row.TermID), Text: row.MedicationText},
		Status: Status(row.Status),
		Dosage: Dosage{DoseUnit: row.DoseUnit.String, PeriodUnit: row.PeriodUnit.String, DoseValue: float(row.DoseValue),
			Period: float(row.Period), Frequency: row.Frequency.Int32, AsNeeded: row.AsNeeded},
		VersionID: row.VersionID, InProfile: row.InProfile,
	}
}

// dosageRow is a Dosage's numbers as the table holds them, null when absent.
type dosageRow struct {
	value, period pgtype.Numeric
	frequency     pgtype.Int4
}

func dosageParams(d Dosage) (dosageRow, error) {
	var out dosageRow
	if d.DoseValue != 0 {
		if err := out.value.Scan(strconv.FormatFloat(d.DoseValue, 'f', -1, 64)); err != nil {
			return dosageRow{}, ErrInvalid
		}
	}
	if d.Period != 0 {
		if err := out.period.Scan(strconv.FormatFloat(d.Period, 'f', -1, 64)); err != nil {
			return dosageRow{}, ErrInvalid
		}
	}
	out.frequency = pgtype.Int4{Int32: d.Frequency, Valid: d.Frequency != 0}
	return out, nil
}

// float is n as a float64, 0 for null.
func float(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
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

func (r *postgresRepository) InsertProvenance(ctx context.Context, p Provenance) error {
	ids, err := uuids(p.ID, p.EntryID, p.RoleID, p.OrganizationID, p.PatientID)
	if err != nil {
		return err
	}
	return refusal(r.q(ctx).InsertMedicationProvenance(ctx, medicationstatementdb.InsertMedicationProvenanceParams{
		ID: ids[0], Recorded: instant(p.Recorded), MedicationStatementID: ids[1], PractitionerRoleID: ids[2], OrganizationID: ids[3],
		PatientID: ids[4], TargetVersion: p.Version, Activity: p.Activity, AgentType: p.AgentType,
	}))
}

func (r *postgresRepository) Provenance(ctx context.Context, id string) (Provenance, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Provenance{}, ErrNotFound
	}
	row, err := r.q(ctx).MedicationProvenanceByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Provenance{}, ErrNotFound
	case err != nil:
		return Provenance{}, err
	}
	return Provenance{
		Recorded: row.Recorded.Time.UTC(), ID: row.ID.String(), EntryID: row.MedicationStatementID.String(),
		RoleID: idOf(row.PractitionerRoleID), OrganizationID: idOf(row.OrganizationID), PatientID: idOf(row.PatientID),
		Activity: row.Activity, AgentType: row.AgentType, Version: row.TargetVersion,
	}, nil
}

func (r *postgresRepository) HasProvenance(ctx context.Context, id string) (bool, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return false, nil
	}
	return r.q(ctx).MedicationHasProvenance(ctx, u)
}
