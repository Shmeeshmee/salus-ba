package careplan

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/careplan/careplandb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on care_plan and care_plan_activity: row-level security shows only
// the clinic SetOrganization set, or the patient's own booked plans after SetPatient.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *careplandb.Queries {
	return careplandb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) Insert(ctx context.Context, p Plan, createdBy string) (Plan, error) {
	row, err := r.q(ctx).InsertPlan(ctx, careplandb.InsertPlanParams{
		OrganizationID: p.OrganizationID, PatientID: p.PatientID, CreatedBy: createdBy, Title: p.Title, QuoteID: p.QuoteID,
	})
	if err != nil {
		return Plan{}, mapErr(err)
	}
	p.ID, p.CreatedAt, p.Status, p.VersionID = row.ID, row.CreatedAt.Time, StatusDraft, 1
	if err := r.writeSteps(ctx, p.ID, p.Steps); err != nil {
		return Plan{}, err
	}
	return r.ByID(ctx, p.ID)
}

func (r *postgresRepository) Update(ctx context.Context, p Plan, changedBy string) (bool, error) {
	n, err := r.q(ctx).UpdatePlan(ctx, careplandb.UpdatePlanParams{
		Status: string(p.Status), Title: p.Title, QuoteID: p.QuoteID, StartsOn: date(p.StartsOn), BookedAt: stamp(p.BookedAt),
		RevokedAt: stamp(p.RevokedAt), ChangedBy: changedBy, ID: p.ID, VersionID: int32(p.VersionID - 1),
	})
	if err != nil || n == 0 {
		return false, mapErr(err)
	}
	return true, r.writeSteps(ctx, p.ID, p.Steps)
}

// writeSteps stores steps as the plan's: a step with an id is updated, one without inserted, and any other of the
// plan's deleted. Positions are checked at the end of the transaction, so steps may swap them.
func (r *postgresRepository) writeSteps(ctx context.Context, planID string, steps []Step) error {
	keep := []string{}
	for _, s := range steps {
		if s.ID != "" {
			keep = append(keep, s.ID)
		}
	}
	q := r.q(ctx)
	if err := q.DeleteActivitiesExcept(ctx, careplandb.DeleteActivitiesExceptParams{CarePlanID: planID, Keep: keep}); err != nil {
		return mapErr(err)
	}
	for i, s := range steps {
		pos, off := int32(i+1), int32(s.DayOffset)
		if s.ID == "" {
			err := q.InsertActivity(ctx, careplandb.InsertActivityParams{
				CarePlanID: planID, Position: pos, ServiceID: s.ServiceID, DayOffset: off, VisitMode: s.VisitMode, AppointmentID: s.AppointmentID,
			})
			if err != nil {
				return mapErr(err)
			}
			continue
		}
		n, err := q.UpdateActivity(ctx, careplandb.UpdateActivityParams{
			Position: pos, ServiceID: s.ServiceID, DayOffset: off, VisitMode: s.VisitMode, AppointmentID: s.AppointmentID, ID: s.ID, CarePlanID: planID,
		})
		if err != nil {
			return mapErr(err)
		} else if n == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Plan, error) {
	row, err := r.q(ctx).PlanByID(ctx, id)
	if err != nil {
		return Plan{}, mapErr(err)
	}
	out, err := r.withSteps(ctx, []Plan{fromRow(careplandb.PlansForPatientRow(row))})
	if err != nil {
		return Plan{}, err
	}
	return out[0], nil
}

func (r *postgresRepository) ForPatient(ctx context.Context, patientID string) ([]Plan, error) {
	rows, err := r.q(ctx).PlansForPatient(ctx, patientID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]Plan, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row)
	}
	return r.withSteps(ctx, out)
}

// withSteps reads the steps of plans, in their positions.
func (r *postgresRepository) withSteps(ctx context.Context, plans []Plan) ([]Plan, error) {
	ids, at := make([]string, len(plans)), make(map[string]int, len(plans))
	for i, p := range plans {
		ids[i], at[p.ID] = p.ID, i
		plans[i].Steps = []Step{}
	}
	rows, err := r.q(ctx).Activities(ctx, ids)
	if err != nil {
		return nil, mapErr(err)
	}
	for _, a := range rows {
		i := at[a.CarePlanID]
		plans[i].Steps = append(plans[i].Steps, Step{
			ID: a.ID, ServiceID: a.ServiceID, VisitMode: a.VisitMode, AppointmentID: a.AppointmentID, Position: int(a.Position), DayOffset: int(a.DayOffset),
		})
	}
	return plans, nil
}

func fromRow(row careplandb.PlansForPatientRow) Plan {
	p := Plan{
		ID: row.ID, OrganizationID: row.OrganizationID, PatientID: row.PatientID, Status: Status(row.Status), Title: row.Title,
		QuoteID: row.QuoteID, CreatedAt: row.CreatedAt.Time, VersionID: int(row.VersionID),
	}
	if row.StartsOn.Valid {
		p.StartsOn = &row.StartsOn.Time
	}
	if row.BookedAt.Valid {
		p.BookedAt = &row.BookedAt.Time
	}
	if row.RevokedAt.Valid {
		p.RevokedAt = &row.RevokedAt.Time
	}
	return p
}

func date(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func stamp(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// mapErr maps the database's refusals to the domain's: no row, a malformed id or a row-level security refusal (a
// patient without the clinic's active consent, a quote not theirs) is ErrNotFound; a foreign key or check ErrInvalid.
func mapErr(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return ErrNotFound
	case !errors.As(err, &pgErr):
		return err
	case pgErr.Code == "42501", pgErr.Code == "22P02":
		return ErrNotFound
	case pgErr.Code == "23503", pgErr.Code == "23514":
		return ErrInvalid
	}
	return err
}
