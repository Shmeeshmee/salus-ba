package slot

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/slot/slotdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// On the salus_app pool row-level security shows only the scope set; on the retention role's, every clinic.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *slotdb.Queries {
	return slotdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetDirectory(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.directory", "public")
}

func (r *postgresRepository) DeleteFreeExcept(ctx context.Context, scheduleID string, after time.Time, keep []Period) (int64, error) {
	var u pgtype.UUID
	if err := u.Scan(scheduleID); err != nil {
		return 0, err
	}
	starts, ends := bounds(keep)
	return r.q(ctx).DeleteFreeSlotsExcept(ctx, slotdb.DeleteFreeSlotsExceptParams{ScheduleID: u, After: ts(after), Starts: starts, Ends: ends})
}

func (r *postgresRepository) Insert(ctx context.Context, scheduleID, org string, periods []Period) (int64, error) {
	var s, o pgtype.UUID
	if s.Scan(scheduleID) != nil || o.Scan(org) != nil {
		return 0, errors.New("slot: an id is not a UUID")
	}
	starts, ends := bounds(periods)
	return r.q(ctx).InsertSlots(ctx, slotdb.InsertSlotsParams{ScheduleID: s, OrganizationID: o, Starts: starts, Ends: ends})
}

func (r *postgresRepository) Search(ctx context.Context, scheduleID string, from, to time.Time) ([]Slot, error) {
	var u pgtype.UUID
	if u.Scan(scheduleID) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).SearchFreeSlots(ctx, slotdb.SearchFreeSlotsParams{ScheduleID: u, FromAt: ts(from), ToAt: ts(to)})
	if err != nil {
		return nil, err
	}
	out := make([]Slot, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSlot(slotdb.HoldSlotRow(row)))
	}
	return out, nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Slot, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Slot{}, ErrUnavailable
	}
	row, err := r.q(ctx).SlotByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Slot{}, ErrUnavailable
	case err != nil:
		return Slot{}, err
	}
	return toSlot(slotdb.HoldSlotRow(row)), nil
}

func (r *postgresRepository) Hold(ctx context.Context, id string, at time.Time) (Slot, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Slot{}, ErrUnavailable
	}
	row, err := r.q(ctx).HoldSlot(ctx, slotdb.HoldSlotParams{ID: u, UpdatedAt: ts(at)})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Slot{}, ErrUnavailable
	case err != nil:
		return Slot{}, err
	}
	return toSlot(row), nil
}

func (r *postgresRepository) Release(ctx context.Context, id string, at time.Time) (bool, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return false, nil
	}
	n, err := r.q(ctx).ReleaseSlot(ctx, slotdb.ReleaseSlotParams{ID: u, UpdatedAt: ts(at)})
	return n == 1, err
}

func (r *postgresRepository) DeleteFreeEndedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteFreeSlotsEndedBefore(ctx, ts(before))
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

// bounds are the periods' starts and ends, as two arrays of one length.
func bounds(periods []Period) (starts, ends []pgtype.Timestamptz) {
	starts = make([]pgtype.Timestamptz, 0, len(periods))
	ends = make([]pgtype.Timestamptz, 0, len(periods))
	for _, p := range periods {
		starts = append(starts, ts(p.Start))
		ends = append(ends, ts(p.End))
	}
	return starts, ends
}

// toSlot maps a row; every slot query selects the same columns.
func toSlot(row slotdb.HoldSlotRow) Slot {
	return Slot{
		Period: Period{Start: row.StartAt.Time, End: row.EndAt.Time},
		ID:     row.ID.String(), ScheduleID: row.ScheduleID.String(), OrganizationID: row.OrganizationID.String(),
		Status: Status(row.Status), VersionID: row.VersionID, LastUpdated: row.UpdatedAt.Time,
	}
}
