package migrations_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// clinicStatInserts insert one row of each clinic aggregate (00043, docs/plans/analytics.md §4) of the clinic with
// its role.
var clinicStatInserts = []string{
	`INSERT INTO clinic_appointment_stat (organization_id, practitioner_role_id, day, hour, specialty_code, status, count)
		VALUES ($1, $2, '2026-11-02', 9, '', 'fulfilled', 3)`,
	`INSERT INTO clinic_diagnosis_stat (organization_id, practitioner_role_id, day, hour, specialty_code, group_code,
		subgroup_code, code, count) VALUES ($1, $2, '2026-11-02', 9, '', 'respiratory-infections', 'common-cold', 'J00', 2)`,
}

// salus_stats writes the clinic aggregates of every clinic, past row-level security: it inserts and deletes rows
// (a refresh), never updates one. salus_app reads only the rows of the clinic set, and writes none.
func TestClinicStats_OnlyTheStatsRoleWritesAndAClinicReadsOnlyItsOwn(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewDatabaseURL(t)
	owner := open(t, dbURL)
	stats, app := open(t, pgtest.StatsURL(t, dbURL)), open(t, pgtest.AppURL(t, dbURL))
	d := seedDirectory(t, owner)
	for _, org := range []string{clinicA, clinicB} {
		for _, q := range clinicStatInserts {
			if _, err := stats.Exec(t.Context(), q, org, d[org].role); err != nil {
				t.Fatalf("salus_stats: %s = %v", q, err)
			}
		}
	}
	for _, table := range []string{"clinic_appointment_stat", "clinic_diagnosis_stat"} {
		for org, want := range map[string]int{clinicA: 1, clinicB: 1, "": 0} {
			var n, other int
			err := inClinic(t, app, org, func(ctx context.Context, db postgres.Querier) error {
				return db.QueryRow(ctx, "SELECT count(*), count(*) FILTER (WHERE organization_id::text <> $1) FROM "+table, org).Scan(&n, &other)
			})
			if err != nil || n != want || other != 0 {
				t.Errorf("%s, clinic %q: %d rows, %d of another clinic, %v; want %d and 0", table, org, n, other, err, want)
			}
		}
		for _, stmt := range []string{"UPDATE " + table + " SET count = 1", "DELETE FROM " + table} {
			err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
				_, err := db.Exec(ctx, stmt)
				return err
			})
			if sqlState(err) != "42501" {
				t.Errorf("salus_app: %s = %v, want permission denied", stmt, err)
			}
		}
		if _, err := stats.Exec(t.Context(), "UPDATE "+table+" SET count = 1"); sqlState(err) != "42501" {
			t.Errorf("salus_stats: UPDATE %s = %v, want permission denied", table, err)
		}
		tag, err := stats.Exec(t.Context(), "DELETE FROM "+table+" WHERE organization_id = $1 AND day = '2026-11-02'", clinicA)
		if err != nil || tag.RowsAffected() != 1 {
			t.Errorf("salus_stats deletes clinic A's day of %s: %v, %v; want one row", table, tag, err)
		}
	}
	err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, clinicStatInserts[0], clinicA, d[clinicA].role)
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("salus_app inserts an aggregate: %v, want permission denied", err)
	}
}

// A row's role is of its clinic, its counts are positive, its hour a local hour, its status an appointment's.
func TestClinicStats_RefusesRowsThatCannotBe(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	d := seedDirectory(t, owner)
	appt := `INSERT INTO clinic_appointment_stat (organization_id, practitioner_role_id, day, hour, specialty_code, status, count)
		VALUES ($1, $2, '2026-11-02', $3, '', $4, $5)`
	diag := `INSERT INTO clinic_diagnosis_stat (organization_id, practitioner_role_id, day, hour, specialty_code, group_code,
		subgroup_code, code, count) VALUES ($1, $2, '2026-11-02', 9, '', $3, NULL, 'J00', 1)`
	for name, tt := range map[string]struct {
		stmt string
		args []any
	}{
		"another clinic's role": {appt, []any{clinicA, d[clinicB].role, 9, "booked", 1}},
		"a zero count":          {appt, []any{clinicA, d[clinicA].role, 9, "booked", 0}},
		"hour 24":               {appt, []any{clinicA, d[clinicA].role, 24, "booked", 1}},
		"an unknown status":     {appt, []any{clinicA, d[clinicA].role, 9, "proposed", 1}},
		"an unknown group":      {diag, []any{clinicA, d[clinicA].role, "no-such-group"}},
	} {
		if _, err := owner.Exec(t.Context(), tt.stmt, tt.args...); sqlState(err) != "23503" && sqlState(err) != "23514" {
			t.Errorf("%s: %v, want a constraint violation", name, err)
		}
	}
	if _, err := owner.Exec(t.Context(), appt, clinicA, d[clinicA].role, 23, "noshow", 1); err != nil {
		t.Errorf("a valid row: %v", err)
	}
}
