package encounter

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/encounter/encounterdb"
	"github.com/Shmeeshmee/salus/pkg/keyset"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db (the salus_app pool), using the transaction in the
// context when there is one; row-level security shows only the scope set.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *encounterdb.Queries {
	return encounterdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) Start(ctx context.Context, e Encounter) (Encounter, bool, error) {
	var appt, org, patient, role pgtype.UUID
	if appt.Scan(e.AppointmentID) != nil || org.Scan(e.OrganizationID) != nil || patient.Scan(e.PatientID) != nil || role.Scan(e.PractitionerRoleID) != nil {
		return Encounter{}, false, errors.New("encounter: an id is not a UUID")
	}
	row, err := r.q(ctx).StartEncounter(ctx, encounterdb.StartEncounterParams{
		AppointmentID: appt, OrganizationID: org, PatientID: patient, PractitionerRoleID: role, PeriodStart: ts(e.PeriodStart),
	})
	if errors.Is(err, pgx.ErrNoRows) { // the appointment has one already
		row, err = r.q(ctx).EncounterByAppointment(ctx, appt)
		return toEncounter(row), false, err
	}
	if err != nil {
		return Encounter{}, false, err
	}
	return toEncounter(row), true, nil
}

func (r *postgresRepository) Finish(ctx context.Context, appointmentID string, at time.Time) (Encounter, bool, error) {
	var appt pgtype.UUID
	if appt.Scan(appointmentID) != nil {
		return Encounter{}, false, nil
	}
	row, err := r.q(ctx).FinishEncounter(ctx, encounterdb.FinishEncounterParams{AppointmentID: appt, At: ts(at)})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Encounter{}, false, nil
	case err != nil:
		return Encounter{}, false, err
	}
	return toEncounter(row), true, nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Encounter, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Encounter{}, ErrNotFound
	}
	row, err := r.q(ctx).EncounterByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Encounter{}, ErrNotFound
	case err != nil:
		return Encounter{}, err
	}
	return toEncounter(row), nil
}

func (r *postgresRepository) Organization(ctx context.Context, id string) (string, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return "", ErrNotFound
	}
	org, err := r.q(ctx).EncounterOrganization(ctx, u)
	switch {
	case err != nil:
		return "", err
	case !org.Valid:
		return "", ErrNotFound
	}
	return org.String(), nil
}

func (r *postgresRepository) Search(ctx context.Context, f Filter) ([]Encounter, error) {
	var p encounterdb.SearchEncountersParams
	if (f.AppointmentID != "" && p.AppointmentID.Scan(f.AppointmentID) != nil) || (f.PatientID != "" && p.PatientID.Scan(f.PatientID) != nil) {
		return nil, nil // an id that is not a UUID matches nothing
	}
	rows, err := r.q(ctx).SearchEncounters(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]Encounter, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEncounter(row))
	}
	return out, nil
}

func (r *postgresRepository) PatientHistory(ctx context.Context, patientID, org string, p keyset.Page) ([]Encounter, error) {
	arg := encounterdb.PatientHistoryEncountersParams{
		FromAt: optTS(p.From), ToAt: optTS(p.To), AfterAt: optTS(p.At), AfterRef: p.Ref, Lim: int32(min(p.Limit, 1000)), //nolint:gosec // bounded
	}
	if arg.PatientID.Scan(patientID) != nil || (org != "" && arg.OrganizationID.Scan(org) != nil) {
		return nil, nil // an id that is not a UUID matches nothing
	}
	rows, err := r.q(ctx).PatientHistoryEncounters(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]Encounter, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEncounter(row))
	}
	return out, nil
}

func (r *postgresRepository) OfPatient(ctx context.Context, patientID string) ([]Encounter, error) {
	var id pgtype.UUID
	if id.Scan(patientID) != nil {
		return nil, nil // an id that is not a UUID matches nothing
	}
	rows, err := r.q(ctx).EncountersOfPatient(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Encounter, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEncounter(row))
	}
	return out, nil
}

// optTS is t, or null when it is zero.
func optTS(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()} }

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func toEncounter(row encounterdb.Encounter) Encounter {
	return Encounter{
		PeriodStart: row.PeriodStart.Time, PeriodEnd: row.PeriodEnd.Time, LastUpdated: row.UpdatedAt.Time,
		ID: row.ID.String(), AppointmentID: row.AppointmentID.String(), OrganizationID: row.OrganizationID.String(),
		PatientID: row.PatientID.String(), PractitionerRoleID: row.PractitionerRoleID.String(),
		Status: Status(row.Status), VersionID: row.VersionID,
	}
}
