package trends

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/trends/trendsdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one: the
// salus_stats pool for a build in cmd/worker, the owner's for salusctl, salus_app's for the view (which only reads),
// salus_retention's for the retention job (which only deletes).
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *trendsdb.Queries {
	return trendsdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) Built(ctx context.Context, period string) (bool, error) {
	return r.q(ctx).PeriodBuilt(ctx, period)
}

func (r *postgresRepository) SavePeriod(ctx context.Context, p Period) error {
	n, err := r.q(ctx).InsertPeriod(ctx, trendsdb.InsertPeriodParams{
		Period: p.Period, Kind: p.Kind, Status: p.Status, CodeLevel: p.CodeLevel, MinCount: int32(p.MinCount),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrFrozen
	}
	return nil
}

func (r *postgresRepository) SaveCells(ctx context.Context, period string, rows []Row) error {
	var a trendsdb.InsertCellsParams
	a.Period = period
	for _, row := range rows {
		count := int32(row.Count)
		if row.Suppressed {
			count = -1
		}
		a.Tbls, a.Entities, a.Regions, a.Cities = append(a.Tbls, row.Table), append(a.Entities, row.Entity), append(a.Regions, row.Region), append(a.Cities, row.City)
		a.Groups, a.Subgroups, a.Codes = append(a.Groups, row.Group), append(a.Subgroups, row.Subgroup), append(a.Codes, row.Code)
		a.Specialties, a.Weekdays, a.Hours = append(a.Specialties, row.Specialty), append(a.Weekdays, row.Weekday), append(a.Hours, row.Hour)
		a.Counts = append(a.Counts, count)
	}
	return r.q(ctx).InsertCells(ctx, a)
}

func (r *postgresRepository) SaveFlags(ctx context.Context, period string, flags []Flag) error {
	a := trendsdb.InsertRisesParams{Period: period}
	for _, f := range flags {
		a.Regions, a.Groups = append(a.Regions, f.Region), append(a.Groups, f.Group)
	}
	return r.q(ctx).InsertRises(ctx, a)
}

func (r *postgresRepository) Periods(ctx context.Context, kind string) ([]Period, error) {
	rows, err := r.q(ctx).PeriodsOfKind(ctx, kind)
	if err != nil {
		return nil, err
	}
	out := make([]Period, len(rows))
	for i, p := range rows {
		out[i] = Period{Built: p.CreatedAt.Time, Period: p.Period, Kind: p.Kind, Status: p.Status, MinCount: int(p.MinCount), CodeLevel: p.CodeLevel}
	}
	return out, nil
}

func (r *postgresRepository) Cells(ctx context.Context, table, from, to string) ([]Row, error) {
	rows, err := r.q(ctx).CellsOfTable(ctx, trendsdb.CellsOfTableParams{Tbl: table, FromPeriod: from, ToPeriod: to})
	if err != nil {
		return nil, err
	}
	out := make([]Row, len(rows))
	for i, c := range rows {
		out[i] = Row{
			Period: c.Period, Table: c.Tbl, Entity: c.Entity, Region: c.Region, City: c.City, Group: c.Grp, Subgroup: c.Subgroup,
			Code: c.Code, Specialty: c.Specialty, Weekday: c.Weekday, Hour: c.Hour, Count: int(c.Count.Int32), Suppressed: c.Suppressed,
		}
	}
	return out, nil
}

func (r *postgresRepository) Flags(ctx context.Context, weeks []string) ([]Flag, error) {
	rows, err := r.q(ctx).RisesOfPeriods(ctx, weeks)
	if err != nil {
		return nil, err
	}
	out := make([]Flag, len(rows))
	for i, f := range rows {
		out[i] = Flag{Week: f.Period, Region: f.Region, Group: f.Grp}
	}
	return out, nil
}

func (r *postgresRepository) DeleteCellsBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteCellsBefore(ctx, pgtype.Timestamptz{Time: before, Valid: true})
}

func (r *postgresRepository) DeleteFlagsBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteRisesBefore(ctx, pgtype.Timestamptz{Time: before, Valid: true})
}
