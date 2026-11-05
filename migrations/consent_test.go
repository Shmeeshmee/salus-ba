package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// consentRows is the seeded consents: patient A active with clinic A and revoked with clinic B, patient B
// active with clinic A.
type consentRows struct {
	aWithA, aWithB, bWithA string
}

// seedConsents creates, as the owner, clinics A and B, patients A and B and their consents.
func seedConsents(t *testing.T, owner *pgxpool.Pool) ([2]patientRow, consentRows) {
	t.Helper()
	seedDirectory(t, owner)
	p := seedPatients(t, owner)
	var c consentRows
	const insert = `INSERT INTO consent (patient_id, organization_id, status, date_time, revoked_at)
		VALUES ($1, $2, $3, '2026-11-01T08:00:00Z', $4) RETURNING id`
	for _, row := range []struct {
		id              *string
		revokedAt       any
		patient, clinic string
		status          string
	}{
		{id: &c.aWithA, patient: p[0].patient, clinic: clinicA, status: "active"},
		{id: &c.aWithB, patient: p[0].patient, clinic: clinicB, status: "inactive", revokedAt: "2026-11-02T08:00:00Z"},
		{id: &c.bWithA, patient: p[1].patient, clinic: clinicA, status: "active"},
	} {
		if err := owner.QueryRow(t.Context(), insert, row.patient, row.clinic, row.status, row.revokedAt).Scan(row.id); err != nil {
			t.Fatal(err)
		}
	}
	return p, c
}

// idsIn returns the ids query gives in the scope, sorted.
func idsIn(t *testing.T, app *pgxpool.Pool, scope map[string]string, query string) []string {
	t.Helper()
	var ids []string
	err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
		rows, err := db.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(ids)
	return ids
}

func sorted(ids ...string) []string {
	ids = slices.Clone(ids)
	slices.Sort(ids)
	return ids
}

func TestConsent_AppRoleSeesTheConsentsOfItsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, c := seedConsents(t, owner)

	tests := []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(c.aWithA, c.aWithB)},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(c.aWithA, c.bWithA)},
		{name: "clinic B", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(c.aWithB)},
		{name: "patient A with clinic A set", scope: map[string]string{"app.patient_id": p[0].patient, "app.organization_id": clinicA}, want: sorted(c.aWithA, c.aWithB)},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": p[0].account, "app.organization_id": clinicA}},
	}
	for _, tt := range tests {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM consent"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestConsent_ClinicSeesAPatientOnlyWithAnActiveConsent(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, c := seedConsents(t, owner)
	const patients = "SELECT id::text FROM patient"

	tests := []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "clinic A, both consents active", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(p[0].patient, p[1].patient)},
		{name: "clinic B, consent revoked", scope: map[string]string{"app.organization_id": clinicB}},
		{name: "patient A with clinic A set", scope: map[string]string{"app.patient_id": p[0].patient, "app.organization_id": clinicA}, want: sorted(p[0].patient)},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": p[0].account, "app.organization_id": clinicA}, want: sorted(p[0].patient)},
	}
	for _, tt := range tests {
		if got := idsIn(t, app, tt.scope, patients); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}

	// Patient A revokes, as the application does; once that commits, clinic A no longer sees them.
	err := asApp(t, app, map[string]string{"app.patient_id": p[0].patient}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE consent SET status = 'inactive', revoked_at = now(), version_id = version_id + 1, updated_at = now()
			WHERE id = $1`, c.aWithA)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, patients); !slices.Equal(got, []string{p[1].patient}) {
		t.Errorf("clinic A after patient A's revocation: sees %v, want only patient B", got)
	}
}

func TestConsent_ClinicCannotChangeAPatientOrAConsent(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, c := seedConsents(t, owner)
	scopeA := map[string]string{"app.organization_id": clinicA}
	for name, stmt := range map[string]string{
		"patient": "UPDATE patient SET given_name = 'Cy' WHERE id = '" + p[0].patient + "'",
		"consent": "UPDATE consent SET status = 'inactive', revoked_at = now() WHERE id = '" + c.aWithA + "'",
	} {
		var n int64
		err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, stmt)
			n = tag.RowsAffected()
			return err
		})
		if err != nil || n != 0 {
			t.Errorf("update of a %s in clinic A's scope: %d rows, %v; want 0", name, n, err)
		}
	}
	err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, "INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())", p[0].patient, clinicB)
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("a consent granted in a clinic's scope: %v, want an RLS violation (42501)", err)
	}
}

func TestConsent_AppRoleWritesOnlyThePatientsOwnConsents(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, c := seedConsents(t, owner)
	scopeA := map[string]string{"app.patient_id": p[0].patient}
	const grant = "INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())"
	exec := func(stmt string, args ...any) error {
		return asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt, args...)
			return err
		})
	}

	if err := exec(grant, p[1].patient, clinicB); sqlState(err) != "42501" {
		t.Errorf("a consent for another patient: %v, want an RLS violation (42501)", err)
	}
	if err := exec(grant, p[0].patient, clinicA); sqlState(err) != "23505" {
		t.Errorf("a second active consent with clinic A: %v, want a unique violation (23505)", err)
	}
	if err := exec(grant, p[0].patient, clinicB); err != nil {
		t.Errorf("granting again after a revocation: %v, want a new row", err)
	}
	var n int64
	err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
		tag, err := db.Exec(ctx, "UPDATE consent SET status = 'inactive', revoked_at = now() WHERE id = $1", c.bWithA)
		n = tag.RowsAffected()
		return err
	})
	if err != nil || n != 0 {
		t.Errorf("revoking patient B's consent in patient A's scope: %d rows, %v; want 0", n, err)
	}
	for _, stmt := range []string{
		"DELETE FROM consent", "TRUNCATE consent",
		"UPDATE consent SET patient_id = patient_id", "UPDATE consent SET organization_id = organization_id",
		"UPDATE consent SET date_time = now()",
	} {
		if err := exec(stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.AppRole, err)
		}
	}
}

func TestConsent_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	const insert = "INSERT INTO consent (patient_id, organization_id, status, date_time, revoked_at) VALUES ($1, $2, $3, now(), $4)"
	tests := []struct {
		revokedAt     any
		name, status  string
		clinic, wantE string
	}{
		{name: "an unknown status", status: "draft", clinic: clinicB, wantE: "23514"},
		{name: "active but revoked", status: "active", clinic: clinicB, revokedAt: "2026-11-03T08:00:00Z", wantE: "23514"},
		{name: "inactive but not revoked", status: "inactive", clinic: clinicB, wantE: "23514"},
		{name: "an unknown clinic", status: "active", clinic: "f0000000-0000-4000-8000-00000000000f", wantE: "23503"},
		{name: "a second revoked one", status: "inactive", clinic: clinicB, revokedAt: "2026-11-03T08:00:00Z"},
	}
	for _, tt := range tests {
		_, err := owner.Exec(t.Context(), insert, p[0].patient, tt.clinic, tt.status, tt.revokedAt)
		if got := sqlState(err); got != tt.wantE {
			t.Errorf("%s: SQLSTATE %q (%v), want %q", tt.name, got, err, tt.wantE)
		}
	}
}
