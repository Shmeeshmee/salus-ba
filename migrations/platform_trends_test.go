package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A clinic's location is a city under a leaf region, set together with the time it was set; a clinic contributes
// to the platform figures by default, since the time its row was made.
func TestPlatformTrends_AClinicsLocationIsACityInItsRegion(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)
	var contributes, since bool
	err := owner.QueryRow(t.Context(), `SELECT analytics_contributes, contributes_since IS NOT NULL FROM organization WHERE id = $1`, clinicA).
		Scan(&contributes, &since)
	if err != nil || !contributes || !since {
		t.Errorf("a new clinic contributes %v since %v, %v; want true with a time", contributes, since, err)
	}
	set := func(region, city any, since string) error {
		return asApp(t, app, map[string]string{"app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `UPDATE organization SET region_id = $1, city_id = $2, location_since = `+since+`,
				analytics_contributes = false, contributes_since = now() WHERE id = $3`, region, city, clinicA)
			return err
		})
	}
	if err := set("BA-09", "Q1270190", "now()"); err != nil {
		t.Errorf("Centar in Kanton Sarajevo: %v, want it set", err)
	}
	for name, tt := range map[string]struct {
		region, city any
		since, state string
	}{
		"a city of another region": {"BA-01", "Q1270190", "now()", "23503"},
		"a region as the city":     {"BA-BIH", "BA-09", "now()", "23503"},
		"a city without a region":  {nil, "Q1270190", "now()", "23514"},
		"a region without a city":  {"BA-09", nil, "now()", "23514"},
		"no time":                  {"BA-09", "Q1270190", "NULL", "23514"},
	} {
		if err := set(tt.region, tt.city, tt.since); sqlState(err) != tt.state {
			t.Errorf("%s: %v, want SQLSTATE %s", name, err, tt.state)
		}
	}
	if err := set(nil, nil, "NULL"); err != nil {
		t.Errorf("clearing the location: %v", err)
	}
}

// The builds read the location and the opt-out and write the published figures as salus_stats, once: neither it nor
// the app changes or deletes them, and the app only reads them; the retention role deletes old cells and rises and
// nothing else.
func TestPlatformTrends_TheTablesGrants(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewDatabaseURL(t)
	owner, stats := open(t, dbURL), open(t, pgtest.StatsURL(t, dbURL))
	app, retention := open(t, pgtest.AppURL(t, dbURL)), open(t, pgtest.RetentionURL(t, dbURL))
	seedDirectory(t, owner)
	if _, err := stats.Exec(t.Context(), `SELECT id, time_zone, region_id, city_id, location_since, analytics_contributes,
			contributes_since FROM organization`); err != nil {
		t.Errorf("salus_stats reads the location: %v", err)
	}
	if _, err := stats.Exec(t.Context(), `INSERT INTO trends_period (period, kind, status, code_level, min_count) VALUES ('2026-10', 'month', 'published', true, 10)`); err != nil {
		t.Fatalf("salus_stats writes a period: %v", err)
	}
	for _, stmt := range []string{
		`INSERT INTO trends_cell (period, tbl, region, grp, count, suppressed) VALUES ('2026-10', 'gd', 'BA-09', 'flu', 15, false)`,
		`INSERT INTO trends_cell (period, tbl, region, count, suppressed) VALUES ('2026-10', 'gs', 'BA-09', NULL, true)`,
		`INSERT INTO trends_rise (period, region, grp) VALUES ('2026-10', 'BA-09', 'flu')`,
	} {
		if _, err := stats.Exec(t.Context(), stmt); err != nil {
			t.Errorf("%s = %v, want it written", stmt, err)
		}
	}
	for _, q := range []string{`SELECT * FROM trends_period`, `SELECT * FROM trends_cell`, `SELECT * FROM trends_rise`} {
		if _, err := app.Exec(t.Context(), q); err != nil {
			t.Errorf("the app: %s = %v, want it read", q, err)
		}
	}
	if _, err := app.Exec(t.Context(), `INSERT INTO trends_period (period, kind, status, code_level, min_count)
			VALUES ('2026-09', 'month', 'published', true, 10)`); sqlState(err) != "42501" {
		t.Errorf("the app writes a period: %v, want permission denied", err)
	}
	for _, stmt := range []string{
		`UPDATE trends_cell SET count = 20`, `DELETE FROM trends_cell`, `UPDATE trends_period SET status = 'refused'`,
		`DELETE FROM trends_period`, `UPDATE trends_rise SET grp = 'x'`, `DELETE FROM trends_rise`,
	} {
		for name, db := range map[string]*pgxpool.Pool{"the app": app, "salus_stats": stats} {
			if _, err := db.Exec(t.Context(), stmt); sqlState(err) != "42501" {
				t.Errorf("%s: %s = %v, want permission denied", name, stmt, err)
			}
		}
	}
	for stmt, want := range map[string]string{
		`DELETE FROM trends_cell WHERE created_at < now() + interval '1 day'`: "",
		`DELETE FROM trends_rise WHERE created_at < now() + interval '1 day'`: "",
		`DELETE FROM trends_period`:         "42501",
		`UPDATE trends_cell SET count = 20`: "42501",
	} {
		if _, err := retention.Exec(t.Context(), stmt); sqlState(err) != want {
			t.Errorf("salus_retention: %s = %v, want %q", stmt, err, want)
		}
	}
	for _, stmt := range []string{
		`INSERT INTO trends_cell (period, tbl, count, suppressed) VALUES ('2026-10', 'gd', 12, false)`,
		`INSERT INTO trends_cell (period, tbl, count, suppressed) VALUES ('2026-10', 'gd', NULL, false)`,
		`INSERT INTO trends_cell (period, tbl, count, suppressed) VALUES ('2026-10', 'xx', 10, false)`,
		`INSERT INTO trends_period (period, kind, status, code_level, min_count) VALUES ('2026-13', 'month', 'published', true, 10)`,
		`INSERT INTO trends_period (period, kind, status, code_level, min_count) VALUES ('2026-11', 'month', 'published', true, 9)`,
	} {
		if _, err := owner.Exec(t.Context(), stmt); sqlState(err) != "23514" {
			t.Errorf("%s = %v, want a check violation", stmt, err)
		}
	}
}
