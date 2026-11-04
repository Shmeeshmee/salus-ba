package patient

import (
	"context"
	"errors"
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
// SetAccount.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) SetAccount(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.account_id", id)
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
		ID: row.ID.String(), AccountID: row.AccountID.String(), GivenName: row.GivenName, FamilyName: row.FamilyName,
		BirthDate: row.BirthDate.Time, Phone: row.Phone.String, VersionID: row.VersionID, LastUpdated: row.UpdatedAt.Time,
	}, nil
}

func pgDate(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

// pgText is s, or NULL when it is empty.
func pgText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
