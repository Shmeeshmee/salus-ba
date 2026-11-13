package clinicstats_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// pools are the owner's, salus_app's and salus_stats' logins on one throwaway database.
type pools struct {
	owner, app, stats *pgxpool.Pool
}

func newPools(t testing.TB) pools {
	t.Helper()
	url := pgtest.NewDatabaseURL(t)
	return pools{owner: openPool(t, url), app: openPool(t, pgtest.AppURL(t, url)), stats: openPool(t, pgtest.StatsURL(t, url))}
}

func openPool(t testing.TB, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// seedClinic creates, as the owner, a clinic in Sarajevo with n practitioner roles, the first a general
// practitioner, and returns its id and the roles' ids.
func seedClinic(t testing.TB, owner *pgxpool.Pool, n int) (string, []string) {
	t.Helper()
	ctx := context.Background()
	var org string
	if err := owner.QueryRow(ctx, `INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	rows, err := owner.Query(ctx, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
			SELECT $1, 'Cy', 'Example ' || i FROM generate_series(1, $2) i RETURNING id)
		INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT $1, id FROM p RETURNING id::text`, org, n)
	if err != nil {
		t.Fatal(err)
	}
	roles, err := collect(rows)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(roles)
	if _, err := owner.Exec(ctx, `UPDATE practitioner_role SET specialty_code = $2 WHERE id = $1`, roles[0], gp); err != nil {
		t.Fatal(err)
	}
	return org, roles
}

func collect(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}) ([]string, error) {
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// replace runs a refresh's writes on salus_stats in one transaction under the clinic's lock.
func replace(t *testing.T, stats *pgxpool.Pool, org string, first, last time.Time, a []clinicstats.AppointmentCount, d []clinicstats.DiagnosisCount) {
	t.Helper()
	repo := clinicstats.NewPostgresBuildRepository(stats)
	err := postgres.NewTransactor(stats).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.Lock(ctx, org); err != nil {
			return err
		}
		return repo.Replace(ctx, org, first, last, a, d)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// inScope runs fn on the read repository in a salus_app transaction scoped to the clinic.
func inScope(t *testing.T, app *pgxpool.Pool, scope string, fn func(ctx context.Context, r clinicstats.Repository) error) {
	t.Helper()
	repo := clinicstats.NewPostgresRepository(app)
	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, scope); err != nil {
			return err
		}
		return fn(ctx, repo)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// salus_stats replaces exactly the clinic's days it names: the other days and the other clinic keep their rows;
// an empty subgroup is stored as none.
func TestBuildRepository_ReplacesTheClinicsDays(t *testing.T) {
	t.Parallel()
	p := newPools(t)
	a, ra := seedClinic(t, p.owner, 1)
	b, rb := seedClinic(t, p.owner, 1)
	for _, c := range []struct{ org, role string }{{a, ra[0]}, {b, rb[0]}} {
		replace(t, p.stats, c.org, day(2026, 11, 1), day(2026, 11, 3), []clinicstats.AppointmentCount{
			{Day: day(2026, 11, 1), Hour: 9, PractitionerRoleID: c.role, Status: "booked", Count: 1},
			{Day: day(2026, 11, 2), Hour: 9, PractitionerRoleID: c.role, Status: "booked", Count: 2},
			{Day: day(2026, 11, 3), Hour: 9, PractitionerRoleID: c.role, Status: "booked", Count: 3},
		}, []clinicstats.DiagnosisCount{
			{Day: day(2026, 11, 2), Hour: 9, PractitionerRoleID: c.role, Group: "other", Code: "I10", Count: 1},
		})
	}
	replace(t, p.stats, a, day(2026, 11, 2), day(2026, 11, 2), []clinicstats.AppointmentCount{
		{Day: day(2026, 11, 2), Hour: 10, PractitionerRoleID: ra[0], SpecialtyCode: gp, Status: "fulfilled", Count: 5},
	}, nil)

	var got []string
	rows, err := p.owner.Query(t.Context(), `SELECT organization_id::text || ' ' || day || ' ' || hour || ' ' || status || ' ' || count
		FROM clinic_appointment_stat ORDER BY organization_id = $1 DESC, day, hour`, a)
	if err == nil {
		got, err = collect(rows)
	}
	want := []string{
		a + " 2026-11-01 9 booked 1", a + " 2026-11-02 10 fulfilled 5", a + " 2026-11-03 9 booked 3",
		b + " 2026-11-01 9 booked 1", b + " 2026-11-02 9 booked 2", b + " 2026-11-03 9 booked 3",
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("rows %v, %v\nwant %v", got, err, want)
	}
	var diags, nullSubgroups int
	if err := p.owner.QueryRow(t.Context(), `SELECT count(*), count(*) FILTER (WHERE subgroup_code IS NULL) FROM clinic_diagnosis_stat`).
		Scan(&diags, &nullSubgroups); err != nil || diags != 1 || nullSubgroups != 1 {
		t.Errorf("%d diagnosis rows, %d without a subgroup, %v; want clinic B's one, without", diags, nullSubgroups, err)
	}
}

// seedFigures stores a clinic's counts for the reads: two roles over two months.
func seedFigures(t *testing.T, stats *pgxpool.Pool, org string, r []string) {
	t.Helper()
	monday, sunday := day(2026, 1, 5), day(2026, 2, 1)
	replace(t, stats, org, monday, sunday, []clinicstats.AppointmentCount{
		{Day: monday, Hour: 9, PractitionerRoleID: r[0], SpecialtyCode: gp, Status: "fulfilled", Count: 4},
		{Day: monday, Hour: 9, PractitionerRoleID: r[0], SpecialtyCode: gp, Status: "cancelled", Count: 2},
		{Day: monday, Hour: 10, PractitionerRoleID: r[1], Status: "noshow", Count: 1},
		{Day: sunday, Hour: 23, PractitionerRoleID: r[1], Status: "booked", Count: 3},
		{Day: sunday, Hour: 23, PractitionerRoleID: r[1], Status: "fulfilled", Count: 1},
	}, []clinicstats.DiagnosisCount{
		{Day: monday, Hour: 9, PractitionerRoleID: r[0], SpecialtyCode: gp, Group: "respiratory-infections", Subgroup: "common-cold", Code: "J00", Count: 3},
		{Day: monday, Hour: 9, PractitionerRoleID: r[0], SpecialtyCode: gp, Group: "respiratory-infections", Code: "J18.9", Count: 1},
		{Day: sunday, Hour: 23, PractitionerRoleID: r[1], Group: "other", Code: "I10", Count: 5},
	})
}

// The reads sum the clinic's counts in the range with the filters: by month, by weekday and hour (not
// cancelled), per role (fulfilled), and the diagnoses by every breakdown.
func TestRepository_SumsTheClinicsCounts(t *testing.T) {
	t.Parallel()
	p := newPools(t)
	org, r := seedClinic(t, p.owner, 2)
	seedFigures(t, p.stats, org, r)
	all := clinicstats.AppointmentQuery{From: day(2026, 1, 1), To: day(2026, 2, 28)}
	inScope(t, p.app, org, func(ctx context.Context, repo clinicstats.Repository) error {
		series, err := repo.Series(ctx, org, all)
		want := []clinicstats.Period{{Period: "2026-01", Booked: 7, Fulfilled: 4, NoShow: 1, Cancelled: 2}, {Period: "2026-02", Booked: 4, Fulfilled: 1}}
		if err != nil || !slices.Equal(series, want) {
			t.Errorf("Series() = %v, %v; want %v", series, err, want)
		}
		heat, err := repo.Heatmap(ctx, org, all)
		wantHeat := []clinicstats.HeatCell{{Weekday: 1, Hour: 9, Count: 4}, {Weekday: 1, Hour: 10, Count: 1}, {Weekday: 7, Hour: 23, Count: 4}}
		if err != nil || !slices.Equal(heat, wantHeat) {
			t.Errorf("Heatmap() = %v, %v; want %v", heat, err, wantHeat)
		}
		totals, err := repo.Practitioners(ctx, org, all)
		wantTotals := []clinicstats.PractitionerTotal{{PractitionerRoleID: r[0], Fulfilled: 4}, {PractitionerRoleID: r[1], Fulfilled: 1}}
		if err != nil || !slices.Equal(totals, wantTotals) {
			t.Errorf("Practitioners() = %v, %v; want %v", totals, err, wantTotals)
		}
		filtered := all
		filtered.Specialty = gp
		if s, err := repo.Series(ctx, org, filtered); err != nil || len(s) != 1 || s[0].Booked != 6 {
			t.Errorf("the general practitioners' Series() = %v, %v; want January's 6", s, err)
		}
		filtered = clinicstats.AppointmentQuery{From: day(2026, 2, 1), To: day(2026, 2, 1), PractitionerRoleID: r[1]}
		if s, err := repo.Series(ctx, org, filtered); err != nil || len(s) != 1 || s[0].Booked != 4 {
			t.Errorf("role 2's 1 February: %v, %v; want 4", s, err)
		}
		return nil
	})
	year := clinicstats.DiagnosisQuery{From: day(2026, 1, 1), To: day(2026, 12, 31)}
	for name, tt := range map[string]struct {
		q    func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery
		want []clinicstats.Count
	}{
		"by group": {func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery { q.By = clinicstats.ByGroup; return q },
			[]clinicstats.Count{{Key: "other", Count: 5}, {Key: "respiratory-infections", Count: 4}}},
		"by subgroup": {func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery { q.By = clinicstats.BySubgroup; return q },
			[]clinicstats.Count{{Key: "", Count: 6}, {Key: "common-cold", Count: 3}}},
		"by code in a group": {func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery {
			q.By, q.Group = clinicstats.ByCode, "respiratory-infections"
			return q
		}, []clinicstats.Count{{Key: "J00", Count: 3}, {Key: "J18.9", Count: 1}}},
		"by code in a subgroup": {func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery {
			q.By, q.Subgroup = clinicstats.ByCode, "common-cold"
			return q
		}, []clinicstats.Count{{Key: "J00", Count: 3}}},
		"by role": {func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery {
			q.By = clinicstats.ByPractitionerRole
			return q
		},
			[]clinicstats.Count{{Key: r[1], Count: 5}, {Key: r[0], Count: 4}}},
		"by specialty": {func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery {
			q.By = clinicstats.BySpecialty
			return q
		},
			[]clinicstats.Count{{Key: "", Count: 5}, {Key: gp, Count: 4}}},
		"by month of a role": {func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery {
			q.By, q.PractitionerRoleID = clinicstats.ByMonth, r[1]
			return q
		}, []clinicstats.Count{{Key: "2026-02", Count: 5}}},
		"by month": {func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery { q.By = clinicstats.ByMonth; return q },
			[]clinicstats.Count{{Key: "2026-01", Count: 4}, {Key: "2026-02", Count: 5}}},
		"by group of a specialty": {func(q clinicstats.DiagnosisQuery) clinicstats.DiagnosisQuery {
			q.By, q.Specialty = clinicstats.ByGroup, gp
			return q
		}, []clinicstats.Count{{Key: "respiratory-infections", Count: 4}}},
	} {
		inScope(t, p.app, org, func(ctx context.Context, repo clinicstats.Repository) error {
			got, err := repo.Diagnoses(ctx, org, tt.q(year))
			if err != nil || !slices.Equal(got, tt.want) {
				t.Errorf("%s: Diagnoses() = %v, %v; want %v", name, got, err, tt.want)
			}
			return nil
		})
	}
}

// Row-level security: in another clinic's scope a read of this clinic's figures finds nothing, and so does a read
// with no clinic set.
func TestRepository_AClinicsScopeReadsNoneOfAnothersRows(t *testing.T) {
	t.Parallel()
	p := newPools(t)
	a, ra := seedClinic(t, p.owner, 2)
	b, _ := seedClinic(t, p.owner, 1)
	seedFigures(t, p.stats, a, ra)
	all := clinicstats.AppointmentQuery{From: day(2026, 1, 1), To: day(2026, 12, 31)}
	for _, scope := range []string{b, ""} {
		if scope == "" {
			// No clinic set: SetClinic is never called with an empty id by the service; read without it.
			err := postgres.NewTransactor(p.app).WithTx(t.Context(), func(ctx context.Context) error {
				s, err := clinicstats.NewPostgresRepository(p.app).Series(ctx, a, all)
				if err == nil && len(s) != 0 {
					t.Errorf("no scope reads %v", s)
				}
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			continue
		}
		inScope(t, p.app, scope, func(ctx context.Context, repo clinicstats.Repository) error {
			s, _ := repo.Series(ctx, a, all)
			h, _ := repo.Heatmap(ctx, a, all)
			pr, _ := repo.Practitioners(ctx, a, all)
			d, err := repo.Diagnoses(ctx, a, clinicstats.DiagnosisQuery{From: all.From, To: all.To, By: clinicstats.ByCode})
			if len(s)+len(h)+len(pr)+len(d) != 0 || err != nil {
				t.Errorf("clinic B's scope reads clinic A's %v %v %v %v, %v", s, h, pr, d, err)
			}
			return nil
		})
	}
}
