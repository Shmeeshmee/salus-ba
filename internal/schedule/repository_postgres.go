package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/schedule/scheduledb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// Its queries run as salus_app, so row-level security shows only the scope set.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *scheduledb.Queries {
	return scheduledb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetDirectory(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.directory", "public")
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Schedule, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Schedule{}, ErrNotFound
	}
	row, err := r.q(ctx).ScheduleByID(ctx, u)
	return toSchedule(row, err, ErrNotFound)
}

func (r *postgresRepository) ByRole(ctx context.Context, roleID string) (Schedule, error) {
	var u pgtype.UUID
	if u.Scan(roleID) != nil {
		return Schedule{}, ErrNotFound
	}
	row, err := r.q(ctx).ScheduleByRole(ctx, u)
	return toSchedule(scheduledb.ScheduleByIDRow(row), err, ErrNotFound)
}

func (r *postgresRepository) Lock(ctx context.Context, id string) (Schedule, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Schedule{}, ErrNotFound
	}
	row, err := r.q(ctx).LockSchedule(ctx, u)
	return toSchedule(scheduledb.ScheduleByIDRow(row), err, ErrNotFound)
}

func (r *postgresRepository) Listed(ctx context.Context) ([]Schedule, error) {
	rows, err := r.q(ctx).ListedSchedules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Schedule, 0, len(rows))
	for _, row := range rows {
		s, err := toSchedule(scheduledb.ScheduleByIDRow(row), nil, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *postgresRepository) Create(ctx context.Context, s Schedule) (Schedule, error) {
	var org, role pgtype.UUID
	if org.Scan(s.OrganizationID) != nil || role.Scan(s.PractitionerRoleID) != nil {
		return Schedule{}, errors.New("schedule: an id is not a UUID")
	}
	rules, err := rulesJSON(s.Rules)
	if err != nil {
		return Schedule{}, err
	}
	row, err := r.q(ctx).CreateSchedule(ctx, scheduledb.CreateScheduleParams{
		OrganizationID: org, PractitionerRoleID: role, SlotMinutes: s.SlotMinutes, Availability: rules,
	})
	return toSchedule(scheduledb.ScheduleByIDRow(row), err, ErrConflict)
}

func (r *postgresRepository) Update(ctx context.Context, s Schedule, at time.Time) (Schedule, error) {
	var u pgtype.UUID
	if u.Scan(s.ID) != nil {
		return Schedule{}, ErrConflict
	}
	rules, err := rulesJSON(s.Rules)
	if err != nil {
		return Schedule{}, err
	}
	row, err := r.q(ctx).UpdateSchedule(ctx, scheduledb.UpdateScheduleParams{
		ID: u, SlotMinutes: s.SlotMinutes, Availability: rules, VersionID: s.VersionID,
		UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return toSchedule(scheduledb.ScheduleByIDRow(row), err, ErrConflict)
}

// ruleJSON is a rule as stored in schedule.availability.
type ruleJSON struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Day   int    `json:"day"`
}

func rulesJSON(rules []Rule) ([]byte, error) {
	out := make([]ruleJSON, 0, len(rules))
	for _, r := range rules {
		out = append(out, ruleJSON(r))
	}
	return json.Marshal(out)
}

// toSchedule maps a row, or the query's error: noRows for no row. Every schedule query selects the same
// columns.
func toSchedule(row scheduledb.ScheduleByIDRow, err, noRows error) (Schedule, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Schedule{}, noRows
	}
	if err != nil {
		return Schedule{}, err
	}
	var stored []ruleJSON
	if err := json.Unmarshal(row.Availability, &stored); err != nil {
		return Schedule{}, err
	}
	rules := make([]Rule, 0, len(stored))
	for _, r := range stored {
		rules = append(rules, Rule(r))
	}
	return Schedule{
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), PractitionerRoleID: row.PractitionerRoleID.String(),
		SlotMinutes: row.SlotMinutes, VersionID: row.VersionID, Rules: rules, LastUpdated: row.UpdatedAt.Time,
	}, nil
}
