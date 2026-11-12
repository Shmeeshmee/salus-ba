package trends_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/trends"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// A build writes its period once, as salus_stats, with its cells and flags; the app reads them back; the retention
// role deletes old cells and flags and keeps the period.
func TestRepository_WritesOnceReadsAndPurges(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewDatabaseURL(t)
	statsPool, err := postgres.Open(t.Context(), pgtest.StatsURL(t, dbURL))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(statsPool.Close)
	stats, tx := trends.NewPostgresRepository(statsPool), postgres.NewTransactor(statsPool)
	rows := []trends.Row{
		{Table: trends.TableGD, Region: "BA-09", Entity: "BA-BIH", Count: 15},
		{Table: trends.TableGD, Region: "BA-01", Entity: "BA-BIH", Suppressed: true},
		{Table: trends.TableHeatmap, Weekday: "1", Hour: "9", Count: 20},
	}
	save := func(ctx context.Context, p trends.Period) error {
		return tx.WithTx(ctx, func(ctx context.Context) error {
			if err := stats.SavePeriod(ctx, p); err != nil {
				return err
			}
			if err := stats.SaveCells(ctx, p.Period, rows); err != nil {
				return err
			}
			return stats.SaveFlags(ctx, p.Period, []trends.Flag{{Region: "BA-09", Group: "flu"}})
		})
	}
	month := trends.Period{Period: "2026-10", Kind: trends.KindMonth, Status: trends.StatusPublished, MinCount: 10, CodeLevel: true}
	if err := save(t.Context(), month); err != nil {
		t.Fatal(err)
	}
	if err := save(t.Context(), month); !errors.Is(err, trends.ErrFrozen) {
		t.Errorf("a second save = %v, want ErrFrozen", err)
	}
	appPool, err := postgres.Open(t.Context(), pgtest.AppURL(t, dbURL))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(appPool.Close)
	app := trends.NewPostgresRepository(appPool)
	if built, err := app.Built(t.Context(), "2026-10"); err != nil || !built {
		t.Errorf("Built = %v, %v", built, err)
	}
	ps, err := app.Periods(t.Context(), trends.KindMonth)
	if err != nil || len(ps) != 1 || ps[0].Period != "2026-10" || !ps[0].CodeLevel || ps[0].MinCount != 10 || ps[0].Built.IsZero() {
		t.Errorf("Periods = %+v, %v", ps, err)
	}
	got, err := app.Cells(t.Context(), trends.TableGD, "2026-01", "2026-12")
	want := []trends.Row{
		{Period: "2026-10", Table: trends.TableGD, Region: "BA-01", Entity: "BA-BIH", Suppressed: true},
		{Period: "2026-10", Table: trends.TableGD, Region: "BA-09", Entity: "BA-BIH", Count: 15},
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("Cells = %+v, %v;\nwant %+v", got, err, want)
	}
	flags, err := app.Flags(t.Context(), []string{"2026-10"})
	if err != nil || len(flags) != 1 || flags[0] != (trends.Flag{Week: "2026-10", Region: "BA-09", Group: "flu"}) {
		t.Errorf("Flags = %+v, %v", flags, err)
	}
	retentionPool, err := postgres.Open(t.Context(), pgtest.RetentionURL(t, dbURL))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(retentionPool.Close)
	purge := trends.NewPostgresRepository(retentionPool)
	if n, err := trends.NewCellPurger(purge).Purge(t.Context(), time.Now().Add(-time.Hour)); err != nil || n != 0 {
		t.Errorf("purging nothing recent = %d, %v", n, err)
	}
	if n, err := trends.NewCellPurger(purge).Purge(t.Context(), time.Now().Add(time.Hour)); err != nil || n != 3 {
		t.Errorf("purging the cells = %d, %v; want 3", n, err)
	}
	if n, err := trends.NewFlagPurger(purge).Purge(t.Context(), time.Now().Add(time.Hour)); err != nil || n != 1 {
		t.Errorf("purging the flags = %d, %v; want 1", n, err)
	}
	if built, err := app.Built(t.Context(), "2026-10"); err != nil || !built {
		t.Errorf("after the purge Built = %v, %v; want the period kept, so it is never built again", built, err)
	}
}
