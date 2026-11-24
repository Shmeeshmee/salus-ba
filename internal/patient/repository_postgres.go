package patient

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/patient/patientdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// Its queries run as salus_app, so row-level security shows only the patient of the account set by
// SetAccount, or the patients with an active consent for the clinic set by SetClinic.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) SetAccount(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.account_id", id)
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) ByAccount(ctx context.Context, accountID string) (Patient, error) {
	var u pgtype.UUID
	if u.Scan(accountID) != nil {
		return Patient{}, ErrNotFound
	}
	row, err := patientdb.New(postgres.Conn(ctx, r.db)).PatientByAccount(ctx, u)
	return toPatient(patientdb.PatientByIDRow(row), err, ErrNotFound)
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Patient, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Patient{}, ErrNotFound
	}
	row, err := patientdb.New(postgres.Conn(ctx, r.db)).PatientByID(ctx, u)
	return toPatient(row, err, ErrNotFound)
}

func (r *postgresRepository) Create(ctx context.Context, p Patient, at time.Time) (Patient, error) {
	var u pgtype.UUID
	if err := u.Scan(p.AccountID); err != nil {
		return Patient{}, errors.New("patient: the account id is not a UUID")
	}
	row, err := patientdb.New(postgres.Conn(ctx, r.db)).CreatePatient(ctx, patientdb.CreatePatientParams{
		AccountID: u, GivenName: p.GivenName, FamilyName: p.FamilyName, BirthDate: pgDate(p.BirthDate),
		Phone: pgText(p.Phone), UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return toPatient(patientdb.PatientByIDRow(row), err, ErrConflict)
}

func (r *postgresRepository) Update(ctx context.Context, p Patient, at time.Time) (Patient, error) {
	var u pgtype.UUID
	if u.Scan(p.AccountID) != nil {
		return Patient{}, ErrConflict
	}
	row, err := patientdb.New(postgres.Conn(ctx, r.db)).UpdatePatient(ctx, patientdb.UpdatePatientParams{
		AccountID: u, VersionID: p.VersionID, GivenName: p.GivenName, FamilyName: p.FamilyName,
		BirthDate: pgDate(p.BirthDate), Phone: pgText(p.Phone), UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return toPatient(patientdb.PatientByIDRow(row), err, ErrConflict)
}

// toPatient maps a row to the model, and no row to noRows.
func toPatient(row patientdb.PatientByIDRow, err, noRows error) (Patient, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Patient{}, noRows
	}
	if err != nil {
		return Patient{}, err
	}
	return Patient{
		ID: row.ID.String(), AccountID: uuidString(row.AccountID), RegisteredBy: uuidString(row.RegisteredByOrganizationID), GivenName: row.GivenName, FamilyName: row.FamilyName,
		BirthDate: row.BirthDate.Time, Phone: row.Phone.String, VersionID: row.VersionID, LastUpdated: row.UpdatedAt.Time,
	}, nil
}

func (r *postgresRepository) Register(ctx context.Context, p Patient, org string, at time.Time) (Patient, error) {
	var o pgtype.UUID
	if err := o.Scan(org); err != nil {
		return Patient{}, errors.New("patient: the clinic id is not a UUID")
	}
	row, err := patientdb.New(postgres.Conn(ctx, r.db)).RegisterPatient(ctx, patientdb.RegisterPatientParams{
		OrganizationID: o, GivenName: p.GivenName, FamilyName: p.FamilyName, BirthDate: pgDate(p.BirthDate),
		Phone: pgText(p.Phone), UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return toPatient(patientdb.PatientByIDRow(row), err, errors.New("patient: register returned no row"))
}

func (r *postgresRepository) Correct(ctx context.Context, p Patient, org string, at time.Time) (Patient, error) {
	var id, o pgtype.UUID
	if id.Scan(p.ID) != nil || o.Scan(org) != nil {
		return Patient{}, ErrConflict
	}
	row, err := patientdb.New(postgres.Conn(ctx, r.db)).CorrectRegisteredPatient(ctx, patientdb.CorrectRegisteredPatientParams{
		ID: id, OrganizationID: o, VersionID: p.VersionID, GivenName: p.GivenName, FamilyName: p.FamilyName,
		BirthDate: pgDate(p.BirthDate), Phone: pgText(p.Phone), UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return toPatient(patientdb.PatientByIDRow(row), err, ErrConflict)
}

func (r *postgresRepository) SearchClinic(ctx context.Context, org, roleID string, q ClinicSearch, limit int32) ([]Patient, error) {
	var o, role pgtype.UUID
	if err := o.Scan(org); err != nil {
		return nil, errors.New("patient: the clinic id is not a UUID")
	}
	if roleID != "" && role.Scan(roleID) != nil {
		return nil, nil
	}
	born := pgtype.Date{Time: q.BirthDate, Valid: !q.BirthDate.IsZero()}
	rows, err := patientdb.New(postgres.Conn(ctx, r.db)).SearchClinicPatients(ctx, patientdb.SearchClinicPatientsParams{
		OrganizationID: o, Pattern: likePattern(q.Name), BirthDate: born, PractitionerRoleID: role, MaxRows: limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Patient, 0, len(rows))
	for _, row := range rows {
		p, err := toPatient(patientdb.PatientByIDRow(row), nil, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *postgresRepository) Matching(ctx context.Context, org string, in []Patient) ([]Patient, error) {
	var o pgtype.UUID
	if err := o.Scan(org); err != nil {
		return nil, errors.New("patient: the clinic id is not a UUID")
	}
	keys := make([]string, len(in))
	for i, p := range in {
		keys[i] = strings.ToLower(p.GivenName) + "\x1f" + strings.ToLower(p.FamilyName) + "\x1f" + p.BirthDate.Format(dateLayout)
	}
	rows, err := patientdb.New(postgres.Conn(ctx, r.db)).MatchingClinicPatients(ctx, patientdb.MatchingClinicPatientsParams{OrganizationID: o, Keys: keys})
	if err != nil {
		return nil, err
	}
	out := make([]Patient, 0, len(rows))
	for _, row := range rows {
		p, err := toPatient(patientdb.PatientByIDRow(row), nil, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *postgresRepository) RegisteredBy(ctx context.Context, org string) ([]string, error) {
	var o pgtype.UUID
	if err := o.Scan(org); err != nil {
		return nil, errors.New("patient: the clinic id is not a UUID")
	}
	rows, err := patientdb.New(postgres.Conn(ctx, r.db)).RegisteredByClinic(ctx, o)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, id := range rows {
		out = append(out, id.String())
	}
	return out, nil
}

func (r *postgresRepository) Restrict(ctx context.Context, patientID, org, by, requestID string, at time.Time) error {
	var p, o, b, req pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(org) != nil || b.Scan(by) != nil || (requestID != "" && req.Scan(requestID) != nil) {
		return ErrNotFound
	}
	q := patientdb.New(postgres.Conn(ctx, r.db))
	n, err := q.RestrictPatient(ctx, patientdb.RestrictPatientParams{
		RestrictedAt: pgtype.Timestamptz{Time: at, Valid: true}, PatientID: p, OrganizationID: o,
		RestrictedByAccountID: b, DataRequestID: req,
	})
	if err != nil || n == 1 {
		return err
	}
	// Nothing inserted: restricted already, or the request is not the clinic's and the patient's.
	restricted, err := q.PatientRestricted(ctx, patientdb.PatientRestrictedParams{PatientID: p, OrganizationID: o})
	if err == nil && !restricted {
		return ErrNotFound
	}
	return err
}

func (r *postgresRepository) Unrestrict(ctx context.Context, patientID, org string) error {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(org) != nil {
		return nil
	}
	return patientdb.New(postgres.Conn(ctx, r.db)).UnrestrictPatient(ctx, patientdb.UnrestrictPatientParams{PatientID: p, OrganizationID: o})
}

func (r *postgresRepository) Restricted(ctx context.Context, patientID, org string) (bool, error) {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(org) != nil {
		return false, nil
	}
	return patientdb.New(postgres.Conn(ctx, r.db)).PatientRestricted(ctx, patientdb.PatientRestrictedParams{PatientID: p, OrganizationID: o})
}

// likePattern is %name%, with LIKE's wildcards and escape in name taken literally.
func likePattern(name string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(name) + "%"
}

// uuidString is u as text, or empty when it is NULL.
func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}

func pgDate(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

// pgText is s, or NULL when it is empty.
func pgText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func (r *postgresRepository) RestrictForErasure(ctx context.Context, patientID, org string, at time.Time) error {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(org) != nil {
		return ErrNotFound
	}
	return patientdb.New(postgres.Conn(ctx, r.db)).RestrictForErasure(ctx, patientdb.RestrictForErasureParams{
		RestrictedAt: pgtype.Timestamptz{Time: at, Valid: true}, PatientID: p, OrganizationID: o,
	})
}
