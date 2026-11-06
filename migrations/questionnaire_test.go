package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// questionnaires is one clinic's synthetic questionnaire rows: an active version 1 and a draft version 2 of
// one series, and its role's default.
type questionnaires struct {
	series, active, draft string
}

// seedQuestionnaires creates, as the owner, clinics A and B (seedDirectory), each with one series at an
// active version 1 and a draft version 2, and the series as its role's default.
func seedQuestionnaires(t *testing.T, owner *pgxpool.Pool) map[string]questionnaires {
	t.Helper()
	seeded := map[string]questionnaires{}
	for org, d := range seedDirectory(t, owner) {
		var q questionnaires
		err := owner.QueryRow(t.Context(), `WITH a AS (
				INSERT INTO questionnaire (organization_id, series_id, version, status, title, published_at)
				VALUES ($1, gen_random_uuid(), 1, 'active', 'Placeholder questionnaire', now()) RETURNING id, series_id),
			d AS (INSERT INTO questionnaire (organization_id, series_id, version, status, title)
				SELECT $1, series_id, 2, 'draft', 'Placeholder questionnaire' FROM a RETURNING id),
			f AS (INSERT INTO questionnaire_default (practitioner_role_id, organization_id, series_id)
				SELECT $2, $1, series_id FROM a)
			SELECT (SELECT series_id FROM a), (SELECT id FROM a), (SELECT id FROM d)`, org, d.role).Scan(&q.series, &q.active, &q.draft)
		if err != nil {
			t.Fatal(err)
		}
		seeded[org] = q
	}
	return seeded
}

func TestQuestionnaire_TheClinicScopeSeesOnlyItsOwnRows(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedQuestionnaires(t, owner)

	for _, table := range []string{"questionnaire", "questionnaire_default"} {
		for org, want := range map[string]int{"": 0, clinicA: map[string]int{"questionnaire": 2, "questionnaire_default": 1}[table]} {
			var n, other int
			err := inClinic(t, app, org, func(ctx context.Context, db postgres.Querier) error {
				return db.QueryRow(ctx, "SELECT count(*), count(*) FILTER (WHERE organization_id <> $1) FROM "+table, clinicA).Scan(&n, &other)
			})
			if err != nil || n != want || other != 0 {
				t.Errorf("%s with clinic %q: %d rows, %d of another clinic, %v; want %d and 0", table, org, n, other, err, want)
			}
		}
	}
}

func TestQuestionnaire_APatientScopeSeesNoneOfTheLibrary(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedQuestionnaires(t, owner)
	scopes := []map[string]string{
		{"app.patient_id": "c2000000-0000-4000-8000-0000000000c1", "app.organization_id": clinicA},
		{"app.account_id": "c1000000-0000-4000-8000-0000000000c1", "app.organization_id": clinicA},
	}
	for _, scope := range scopes {
		for _, table := range []string{"questionnaire", "questionnaire_default"} {
			var n int
			err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
				return db.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n)
			})
			if err != nil || n != 0 {
				t.Errorf("%s in %v: %d rows, %v; want none", table, scope, n, err)
			}
		}
	}
}

func TestQuestionnaire_TheAppRoleWritesOnlyWhatTheLifecycleNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seeded := seedQuestionnaires(t, owner)
	a, b := seeded[clinicA], seeded[clinicB]

	allowed := []string{
		"UPDATE questionnaire SET title = 'Renamed placeholder', description = 'x', items = '[]', version_id = version_id + 1, updated_at = now() WHERE id = '" + a.draft + "'",
		"UPDATE questionnaire SET status = 'retired', retired_at = now() WHERE id = '" + a.active + "'",
		"UPDATE questionnaire_default SET series_id = gen_random_uuid(), version_id = version_id + 1, updated_at = now()",
		"DELETE FROM questionnaire WHERE id = '" + a.draft + "'",
		"DELETE FROM questionnaire_default",
		"INSERT INTO questionnaire (organization_id, series_id, version, status, title) VALUES ('" + clinicA + "', gen_random_uuid(), 1, 'draft', 'Placeholder')",
	}
	for _, stmt := range allowed {
		if err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		}); err != nil {
			t.Errorf("%s as %s: %v", stmt, pgtest.AppRole, err)
		}
	}
	denied := map[string]string{
		"UPDATE questionnaire SET organization_id = '" + clinicB + "'":         "42501",
		"UPDATE questionnaire SET series_id = gen_random_uuid()":               "42501",
		"UPDATE questionnaire SET version = 9":                                 "42501",
		"UPDATE questionnaire SET created_at = now()":                          "42501",
		"UPDATE questionnaire_default SET organization_id = '" + clinicB + "'": "42501",
		"TRUNCATE questionnaire":                                               "42501",
		"TRUNCATE questionnaire_default":                                       "42501",
		"INSERT INTO questionnaire (organization_id, series_id, version, status, title) VALUES ('" + clinicB + "', gen_random_uuid(), 1, 'draft', 'x')": "42501",
	}
	for stmt, want := range denied {
		err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if got := sqlState(err); got != want {
			t.Errorf("%s as %s: %v, want SQLSTATE %s", stmt, pgtest.AppRole, err, want)
		}
	}

	// A published version is never deleted, whatever a query asks; another clinic's rows are not touched.
	var left int
	err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
		if _, err := db.Exec(ctx, "DELETE FROM questionnaire"); err != nil {
			return err
		}
		return db.QueryRow(ctx, "SELECT count(*) FROM questionnaire WHERE id = $1", a.active).Scan(&left)
	})
	if err != nil || left != 1 {
		t.Errorf("DELETE FROM questionnaire left the published version %d times, %v; want it kept", left, err)
	}
	var others int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM questionnaire WHERE id IN ($1, $2)", b.active, b.draft).Scan(&others); err != nil || others != 2 {
		t.Errorf("clinic B's rows after clinic A's writes: %d, %v; want 2", others, err)
	}
}

func TestQuestionnaire_HoldsOneDraftAndOneActiveVersionPerSeries(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	q := seedQuestionnaires(t, owner)[clinicA]
	ctx := t.Context()

	for name, stmt := range map[string]string{
		"a second draft":  "INSERT INTO questionnaire (organization_id, series_id, version, status, title) VALUES ($1, $2, 3, 'draft', 'x')",
		"a second active": "INSERT INTO questionnaire (organization_id, series_id, version, status, title, published_at) VALUES ($1, $2, 3, 'active', 'x', now())",
		"a version twice": "INSERT INTO questionnaire (organization_id, series_id, version, status, title, published_at, retired_at) VALUES ($1, $2, 1, 'retired', 'x', now(), now())",
	} {
		if _, err := owner.Exec(ctx, stmt, clinicA, q.series); sqlState(err) != "23505" {
			t.Errorf("%s: %v, want unique_violation", name, err)
		}
	}
	if _, err := owner.Exec(ctx, `INSERT INTO questionnaire (organization_id, series_id, version, status, title, published_at, retired_at)
		VALUES ($1, $2, 3, 'retired', 'x', now(), now()), ($1, $2, 4, 'retired', 'x', now(), now())`, clinicA, q.series); err != nil {
		t.Errorf("two retired versions: %v, want accepted", err)
	}
}

func TestQuestionnaire_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	q := seedQuestionnaires(t, owner)[clinicA]
	for name, stmt := range map[string]string{
		"an unknown status":             "UPDATE questionnaire SET status = 'published' WHERE id = $1",
		"active without a publish time": "UPDATE questionnaire SET published_at = NULL WHERE id = $1",
		"retired without a retire time": "UPDATE questionnaire SET status = 'retired' WHERE id = $1",
		"a retire time while active":    "UPDATE questionnaire SET retired_at = now() WHERE id = $1",
		"version 0":                     "UPDATE questionnaire SET version = 0 WHERE id = $1",
		"items that are no list":        "UPDATE questionnaire SET items = '{}' WHERE id = $1",
		"a blank title":                 "UPDATE questionnaire SET title = ' ' WHERE id = $1",
		"an unknown clinic":             "UPDATE questionnaire SET organization_id = gen_random_uuid() WHERE id = $1",
	} {
		if _, err := owner.Exec(t.Context(), stmt, q.active); sqlState(err) == "" {
			t.Errorf("%s: accepted, want a violation", name)
		}
	}
	if _, err := owner.Exec(t.Context(), "UPDATE questionnaire SET published_at = now() WHERE id = $1", q.draft); sqlState(err) == "" {
		t.Error("a draft with a publish time: accepted, want a violation")
	}
	for name, stmt := range map[string]string{
		"a role of another clinic": "UPDATE questionnaire_default SET organization_id = '" + clinicB + "' WHERE organization_id = '" + clinicA + "'",
		"an unknown role":          "UPDATE questionnaire_default SET practitioner_role_id = gen_random_uuid() WHERE organization_id = '" + clinicA + "'",
	} {
		if _, err := owner.Exec(t.Context(), stmt); sqlState(err) == "" {
			t.Errorf("%s: accepted, want a violation", name)
		}
	}
}
