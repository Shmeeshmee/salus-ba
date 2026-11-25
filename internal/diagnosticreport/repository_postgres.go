package diagnosticreport

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/diagnosticreport/diagnosticreportdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db (salus_app), using the transaction in the context when there is
// one: row-level security shows only the clinic SetOrganization set, or the patient's released reports after
// SetPatient.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *diagnosticreportdb.Queries {
	return diagnosticreportdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) Insert(ctx context.Context, rep Report, createdBy string) (Report, error) {
	ids, err := uuids(rep.OrganizationID, rep.PatientID, rep.AppointmentID, createdBy)
	if err != nil {
		return Report{}, err
	}
	row, err := r.q(ctx).InsertReport(ctx, diagnosticreportdb.InsertReportParams{
		OrganizationID: ids[0], PatientID: ids[1], AppointmentID: ids[2], CreatedByAccountID: ids[3],
		Title: rep.Title, LoincCode: text(rep.LOINC), EffectiveDate: date(rep.EffectiveDate),
	})
	if refusedByPolicy(err) {
		return Report{}, ErrNotFound // row-level security: no active care consent, or not the clinic's visit
	}
	if err != nil {
		return Report{}, err
	}
	if err := r.insertResults(ctx, row.ID, rep.Results); err != nil {
		return Report{}, err
	}
	out := fromRow(row)
	out.Results = rep.Results
	return out, nil
}

func (r *postgresRepository) Update(ctx context.Context, rep Report) (bool, error) {
	id, ok := parse(rep.ID)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).UpdateReport(ctx, diagnosticreportdb.UpdateReportParams{
		Title: rep.Title, LoincCode: text(rep.LOINC), EffectiveDate: date(rep.EffectiveDate), ID: id,
		VersionID: int32(rep.VersionID - 1), //nolint:gosec // a version is small
	})
	if refusedByPolicy(err) {
		return false, ErrForbidden // the clinic's policy re-checks the author: one no longer the clinic's admin or practitioner
	}
	if err != nil || n == 0 {
		return false, err
	}
	if err := r.q(ctx).DeleteResults(ctx, id); err != nil {
		return false, err
	}
	return true, r.insertResults(ctx, id, rep.Results)
}

func (r *postgresRepository) Release(ctx context.Context, id, releasedBy string, version int, at time.Time) (bool, error) {
	ids, err := uuids(id, releasedBy)
	if err != nil {
		return false, nil //nolint:nilerr // an id that is no UUID names no report
	}
	n, err := r.q(ctx).ReleaseReport(ctx, diagnosticreportdb.ReleaseReportParams{
		ReleasedAt: pgtype.Timestamptz{Time: at, Valid: true}, ReleasedByAccountID: ids[1], ID: ids[0],
		VersionID: int32(version), //nolint:gosec // a version is small
	})
	if refusedByPolicy(err) {
		return false, ErrForbidden // as Update's
	}
	return n == 1, err
}

// refusedByPolicy reports whether err is row-level security's refusal of a write (42501).
func refusedByPolicy(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42501"
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Report, error) {
	u, ok := parse(id)
	if !ok {
		return Report{}, ErrNotFound
	}
	row, err := r.q(ctx).ReportByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Report{}, ErrNotFound
	case err != nil:
		return Report{}, err
	}
	out, err := r.withResults(ctx, []diagnosticreportdb.DiagnosticReport{row})
	if err != nil {
		return Report{}, err
	}
	return out[0], nil
}

func (r *postgresRepository) ForAppointment(ctx context.Context, appointmentID string) ([]Report, error) {
	u, ok := parse(appointmentID)
	if !ok {
		return []Report{}, nil
	}
	rows, err := r.q(ctx).ReportsForAppointment(ctx, u)
	if err != nil {
		return nil, err
	}
	return r.withResults(ctx, rows)
}

func (r *postgresRepository) ForPatient(ctx context.Context, patientID string) ([]Report, error) {
	u, ok := parse(patientID)
	if !ok {
		return []Report{}, nil
	}
	rows, err := r.q(ctx).ReportsForPatient(ctx, u)
	if err != nil {
		return nil, err
	}
	return r.withResults(ctx, rows)
}

func (r *postgresRepository) insertResults(ctx context.Context, id pgtype.UUID, results []Result) error {
	for i, res := range results {
		value, err := numeric(res.Value)
		if err != nil {
			return err
		}
		if err := r.q(ctx).InsertResult(ctx, diagnosticreportdb.InsertResultParams{
			DiagnosticReportID: id, Position: int32(i + 1), Test: res.Test, LoincCode: text(res.LOINC), //nolint:gosec // at most MaxResults
			ValueDecimal: value, Unit: text(res.Unit), ValueText: text(res.ValueText), ReferenceRange: text(res.ReferenceRange),
			Interpretation: text(string(res.Interpretation)),
		}); err != nil {
			return err
		}
	}
	return nil
}

// withResults maps rows to reports with their values, read in one query after the rows were collected.
func (r *postgresRepository) withResults(ctx context.Context, rows []diagnosticreportdb.DiagnosticReport) ([]Report, error) {
	out := make([]Report, len(rows))
	ids := make([]pgtype.UUID, len(rows))
	at := map[string]int{}
	for i, row := range rows {
		out[i], ids[i] = fromRow(row), row.ID
		at[out[i].ID] = i
	}
	if len(rows) == 0 {
		return out, nil
	}
	results, err := r.q(ctx).ResultsOf(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, res := range results {
		i := at[res.DiagnosticReportID.String()]
		out[i].Results = append(out[i].Results, fromResult(res))
	}
	return out, nil
}

func fromRow(row diagnosticreportdb.DiagnosticReport) Report {
	out := Report{
		CreatedAt: row.CreatedAt.Time, ID: row.ID.String(), OrganizationID: row.OrganizationID.String(),
		PatientID: row.PatientID.String(), AppointmentID: row.AppointmentID.String(), Status: Status(row.Status),
		Title: row.Title, LOINC: row.LoincCode.String, Results: []Result{}, VersionID: int(row.VersionID),
	}
	if row.ReleasedAt.Valid {
		t := row.ReleasedAt.Time
		out.ReleasedAt = &t
	}
	if row.EffectiveDate.Valid {
		t := row.EffectiveDate.Time
		out.EffectiveDate = &t
	}
	return out
}

func fromResult(row diagnosticreportdb.DiagnosticReportResult) Result {
	out := Result{
		Test: row.Test, LOINC: row.LoincCode.String, Unit: row.Unit.String, ValueText: row.ValueText.String,
		ReferenceRange: row.ReferenceRange.String, Interpretation: Interpretation(row.Interpretation.String),
	}
	if row.ValueDecimal.Valid {
		if f, err := row.ValueDecimal.Float64Value(); err == nil && f.Valid {
			out.Value = &f.Float64
		}
	}
	return out
}

func numeric(v *float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if v == nil {
		return n, nil
	}
	if err := n.Scan(strconv.FormatFloat(*v, 'f', -1, 64)); err != nil {
		return n, fmt.Errorf("diagnosticreport: a value that is no number: %w", err)
	}
	return n, nil
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func date(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func parse(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	return u, u.Scan(s) == nil && u.Valid
}

// uuids parses ids the service made or read: one that is no UUID is a fault.
func uuids(ids ...string) ([]pgtype.UUID, error) {
	out := make([]pgtype.UUID, len(ids))
	for i, s := range ids {
		u, ok := parse(s)
		if !ok {
			return nil, errors.New("diagnosticreport: an id that is no UUID")
		}
		out[i] = u
	}
	return out, nil
}
