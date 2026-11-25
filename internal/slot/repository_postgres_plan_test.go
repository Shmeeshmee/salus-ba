package slot_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A busy schedule's slot reads and its hourly generation touch the rows of their range only, under row-level security
// (docs/clinic-cms/load/REPORT.md, RD2): the week's free slots of the booking, the future free slots the generation
// compares, and its insert of the periods it gives, of which a re-run sends none that is there already. The schedule
// here has a slot every hour for 300 days before and 30 after Monday.
func TestSlotReads_TouchOnlyTheRangesRows(t *testing.T) {
	t.Parallel()
	url := pgtest.NewDatabaseURL(t)
	owner := pgtest.TracedPool(t, url, nil)
	seed(t, owner)
	if _, err := owner.Exec(t.Context(), `INSERT INTO slot (schedule_id, organization_id, period, status)
		SELECT $1, $2, tstzrange($3::timestamptz + h * interval '1 hour', $3::timestamptz + (h + 1) * interval '1 hour'),
			CASE WHEN h % 3 = 0 THEN 'busy' ELSE 'free' END
		FROM generate_series(-300 * 24, 30 * 24 - 1) h`, scheduleA1, clinicA, monday.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), "ANALYZE slot"); err != nil {
		t.Fatal(err)
	}
	given := futureFree(t, owner) // what a re-run's rules give
	for name, tt := range map[string]struct {
		run   func(ctx context.Context, r slot.Repository) error
		scope func(ctx context.Context, r slot.Repository) error
		want  func(p pgtest.Plan) bool
		match string
	}{
		"the week's free slots": {
			scope: func(ctx context.Context, r slot.Repository) error { return r.SetDirectory(ctx) },
			run: func(ctx context.Context, r slot.Repository) error {
				got, err := r.Search(ctx, scheduleA1, monday, monday.Add(7*24*time.Hour))
				if err == nil && len(got) == 0 {
					t.Error("the week's search found no slot")
				}
				return err
			},
			match: "FROM slot", want: func(p pgtest.Plan) bool { return p.Dropped <= 60 },
		},
		"the generation's comparison": {
			scope: func(ctx context.Context, r slot.Repository) error { return r.SetOrganization(ctx, clinicA) },
			run: func(ctx context.Context, r slot.Repository) error {
				_, err := r.DeleteFreeExcept(ctx, scheduleA1, monday, given)
				return err
			},
			// It compares the 720 future slots with the periods given, and reads none of the 7,200 past ones.
			match: "DELETE FROM slot", want: func(p pgtest.Plan) bool { return p.Dropped <= 720 },
		},
		"the generation's insert": {
			scope: func(ctx context.Context, r slot.Repository) error { return r.SetOrganization(ctx, clinicA) },
			run: func(ctx context.Context, r slot.Repository) error {
				n, err := r.Insert(ctx, scheduleA1, clinicA, given)
				if err == nil && n != 0 {
					t.Errorf("a re-run's insert added %d slots", n)
				}
				return err
			},
			match: "INSERT INTO slot", want: func(p pgtest.Plan) bool { return p.Conflicts == 0 },
		},
	} {
		traced := &pgtest.LastQuery{Match: tt.match}
		app := pgtest.TracedPool(t, pgtest.AppURL(t, url), traced)
		repo := slot.NewPostgresRepository(app)
		err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			if err := tt.scope(ctx, repo); err != nil {
				return err
			}
			if err := tt.run(ctx, repo); err != nil {
				return err
			}
			if p := pgtest.Explain(t, ctx, app, traced); !tt.want(p) {
				t.Errorf("%s: %+v", name, p)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

// futureFree is schedule A1's free slots after Monday, as periods of video, read as the owner.
func futureFree(t *testing.T, owner *pgxpool.Pool) []slot.Period {
	t.Helper()
	rows, err := owner.Query(t.Context(), `SELECT lower(period), upper(period) FROM slot WHERE schedule_id = $1 AND status = 'free'
		AND lower(period) > $2 ORDER BY 1`, scheduleA1, monday)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []slot.Period
	for rows.Next() {
		p := slot.Period{Mode: schedule.ModeVideo}
		if err := rows.Scan(&p.Start, &p.End); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
