package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A patient the clinic registers has no account and is seen by that clinic only, whatever consent another clinic
// holds (docs/clinic-cms, 3.0, W1); a patient's own profile is unchanged.
func TestWave3_AClinicRegisteredPatientIsThatClinicsOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	scopeA, scopeB := map[string]string{"app.organization_id": clinicA}, map[string]string{"app.organization_id": clinicB}
	var registered string
	err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
		return db.QueryRow(ctx, `INSERT INTO patient (registered_by_organization_id, given_name, family_name, birth_date)
			VALUES ($1, 'Bo', 'Example', '1980-02-03') RETURNING id`, clinicA).Scan(&registered)
	})
	if err != nil {
		t.Fatalf("clinic A registers a patient: %v", err)
	}
	// Even a consent row for clinic B (inserted as the owner) does not show the profile to clinic B.
	if _, err := owner.Exec(t.Context(), `INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())`,
		registered, clinicB); err != nil {
		t.Fatal(err)
	}
	const patients = "SELECT id::text FROM patient"
	if got := idsIn(t, app, scopeA, patients); !slices.Contains(got, registered) {
		t.Errorf("clinic A sees %v, want its registered patient among them", got)
	}
	if got := idsIn(t, app, scopeB, patients); slices.Contains(got, registered) {
		t.Errorf("clinic B sees %v, want not clinic A's registered patient", got)
	}
	if got := idsIn(t, app, map[string]string{"app.patient_id": p[0].patient}, patients); !slices.Equal(got, []string{p[0].patient}) {
		t.Errorf("patient A sees %v, want their own profile only", got)
	}
	for name, stmt := range map[string]string{
		"for another clinic": "INSERT INTO patient (registered_by_organization_id, given_name, family_name, birth_date) VALUES ('" + clinicB + "', 'Bo', 'Example', '1980-02-03')",
		"with an account":    "INSERT INTO patient (account_id, registered_by_organization_id, given_name, family_name, birth_date) VALUES ('" + p[0].account + "', '" + clinicA + "', 'Bo', 'Example', '1980-02-03')",
		"of a patient's own": "UPDATE patient SET given_name = 'Cy' WHERE id = '" + p[0].patient + "' RETURNING id",
	} {
		err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
			var id string
			return db.QueryRow(ctx, stmt).Scan(&id)
		})
		if err == nil {
			t.Errorf("clinic A writes a profile %s: no error, want a refusal", name)
		}
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO patient (given_name, family_name, birth_date) VALUES ('Bo', 'Example', '1980-02-03')`); sqlState(err) != "23514" {
		t.Errorf("a profile with neither an account nor a clinic: %v, want a check violation", err)
	}
}

// Staff record an in-person care consent only for a patient their clinic registered: never for a patient with an
// account, and never in a patient's scope.
func TestWave3_AnInPersonConsentIsOnlyForTheClinicsRegisteredPatient(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	staff := seedStaffAccount(t, owner, clinicA)
	var registered string
	if err := owner.QueryRow(t.Context(), `INSERT INTO patient (registered_by_organization_id, given_name, family_name, birth_date)
		VALUES ($1, 'Bo', 'Example', '1980-02-03') RETURNING id`, clinicA).Scan(&registered); err != nil {
		t.Fatal(err)
	}
	const record = `INSERT INTO consent (patient_id, organization_id, status, date_time, given_in_person, recorded_by_account_id)
		VALUES ($1, $2, 'active', now(), true, $3)`
	exec := func(scope map[string]string, args ...any) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, record, args...)
			return err
		})
	}
	scopeA := map[string]string{"app.organization_id": clinicA}
	if err := exec(scopeA, registered, clinicA, staff); err != nil {
		t.Errorf("clinic A records its registered patient's consent: %v", err)
	}
	if err := exec(map[string]string{"app.organization_id": clinicB}, registered, clinicB, staff); sqlState(err) != "23514" {
		t.Errorf("clinic B records clinic A's registered patient's consent: %v, want a check violation", err)
	}
	if err := exec(map[string]string{"app.organization_id": clinicB}, p[1].patient, clinicB, staff); sqlState(err) != "23514" {
		t.Errorf("clinic B records an in-person consent for a patient with an account: %v, want a check violation", err)
	}
	if err := exec(map[string]string{"app.patient_id": p[1].patient}, p[1].patient, clinicB, staff); sqlState(err) != "23514" {
		t.Errorf("a patient claims an in-person consent: %v, want a check violation", err)
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO consent (patient_id, organization_id, status, date_time, given_in_person)
		VALUES ($1, $2, 'active', now(), true)`, registered, clinicB); sqlState(err) != "23514" {
		t.Errorf("an in-person consent nobody recorded: %v, want a check violation", err)
	}
}

// arrived is for an in-person visit, with its time; the change log takes it between booked and fulfilled.
func TestWave3_ArrivedIsAnInPersonVisitsWithItsTime(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	_, _, a := seedAppointments(t, owner)
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `UPDATE appointment SET status = 'arrived', arrived_at = now() WHERE id = $1`, a.aFuture); sqlState(err) != "23514" {
		t.Errorf("a video visit arrives: %v, want a check violation", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE appointment SET visit_mode = 'in-person' WHERE id = $1`, a.aFuture); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `UPDATE appointment SET status = 'arrived' WHERE id = $1`, a.aFuture); sqlState(err) != "23514" {
		t.Errorf("an arrival without its time: %v, want a check violation", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE appointment SET status = 'arrived', arrived_at = now() WHERE id = $1`, a.aFuture); err != nil {
		t.Errorf("an in-person visit arrives: %v", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE appointment SET status = 'fulfilled' WHERE id = $1`, a.aFuture); err != nil {
		t.Errorf("an arrived visit is fulfilled, keeping its arrival: %v", err)
	}
}

// seedStaffAccount creates, as the owner, a receptionist account of the clinic org (the role 3.0 adds) and returns its
// id.
func seedStaffAccount(t *testing.T, owner *pgxpool.Pool, org string) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role, organization_id)
		VALUES ('desk@example.test', 'h', 'reception', $1) RETURNING id`, org).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// Staff move a booked visit in their clinic's scope (F2), as its patient does; the patient alone marks no arrival, and
// a booking names only its own clinic's staff, in the clinic's scope.
func TestWave3_StaffMoveBookAndMarkArrivalInTheClinicsScopeOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, dir, a := seedAppointments(t, owner)
	ctx := t.Context()
	staffA, staffB := seedStaffAccount(t, owner, clinicA), seedStaffAccount2(t, owner, clinicB)
	scopeA := map[string]string{"app.organization_id": clinicA}
	var schedule, slot string
	if err := owner.QueryRow(ctx, `INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
		VALUES ($1, $2, 30, '[]') RETURNING id`, clinicA, dir[clinicA].role).Scan(&schedule); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO slot (schedule_id, organization_id, period, status, visit_mode)
		VALUES ($1, $2, tstzrange('2030-02-04T10:00:00Z', '2030-02-04T10:30:00Z'), 'busy', 'video') RETURNING id`,
		schedule, clinicA).Scan(&slot); err != nil {
		t.Fatal(err)
	}
	exec := func(scope map[string]string, stmt string, args ...any) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt, args...)
			return err
		})
	}
	if err := exec(scopeA, `UPDATE appointment SET period = tstzrange('2030-02-04T10:00:00Z', '2030-02-04T10:30:00Z'), slot_id = $2,
		reschedule_count = reschedule_count + 1, version_id = version_id + 1 WHERE id = $1`, a.aFuture, slot); err != nil {
		t.Errorf("clinic A moves its patient's booked visit: %v", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE appointment SET visit_mode = 'in-person' WHERE id = $1`, a.aPast); err != nil {
		t.Fatal(err)
	}
	const arrive = `UPDATE appointment SET status = 'arrived', arrived_at = now() WHERE id = $1`
	if err := exec(map[string]string{"app.patient_id": p[0].patient}, arrive, a.aPast); sqlState(err) != "42501" {
		t.Errorf("the patient marks their own arrival: %v, want refused (42501)", err)
	}
	if err := exec(scopeA, arrive, a.aPast); err != nil {
		t.Errorf("clinic A marks an in-person visit arrived: %v", err)
	}
	if err := exec(scopeA, `UPDATE appointment SET arrived_at = now() - interval '1 hour' WHERE id = $1`, a.aPast); sqlState(err) != "23514" {
		t.Errorf("an arrival changed: %v, want a check violation", err)
	}
	const book = `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, booked_by_account_id)
		VALUES ($1, $2, $3, gen_random_uuid(), tstzrange('2030-03-04T10:00:00Z', '2030-03-04T10:30:00Z'), $4)`
	if err := exec(scopeA, book, p[0].patient, dir[clinicA].role, clinicA, staffB); sqlState(err) != "42501" {
		t.Errorf("clinic A books naming clinic B's staff: %v, want refused (42501)", err)
	}
	if err := exec(map[string]string{"app.patient_id": p[0].patient}, book, p[0].patient, dir[clinicA].role, clinicA, staffA); sqlState(err) != "42501" {
		t.Errorf("a patient's booking names staff: %v, want refused (42501)", err)
	}
	if err := exec(scopeA, book, p[0].patient, dir[clinicA].role, clinicA, staffA); err != nil {
		t.Errorf("clinic A books for its patient: %v", err)
	}
	if err := exec(scopeA, book, p[1].patient, dir[clinicA].role, clinicA, staffA); sqlState(err) != "42501" {
		t.Errorf("clinic A books for a patient without its consent: %v, want an RLS violation (42501)", err)
	}
}

// A quote's lines are written only with the quote, never added to a sent one.
func TestWave3_ASentQuoteGainsNoLine(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	staff := seedStaffAccount(t, owner, clinicA)
	scopeA := map[string]string{"app.organization_id": clinicA}
	const line = `INSERT INTO quote_line (quote_id, position, name, quantity, unit_price_minor, total_minor) VALUES ($1, $2, 'Placeholder', 1, 100, 100)`
	var quote string
	err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
		if err := db.QueryRow(ctx, `INSERT INTO quote (organization_id, patient_id, created_by_account_id, issued_at, valid_until, total_minor)
			VALUES ($1, $2, $3, now(), now() + interval '30 days', 100) RETURNING id`, clinicA, p[0].patient, staff).Scan(&quote); err != nil {
			return err
		}
		_, err := db.Exec(ctx, line, quote, 1)
		return err
	})
	if err != nil {
		t.Fatalf("clinic A sends a quote with its line: %v", err)
	}
	err = asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, line, quote, 2)
		return err
	})
	if sqlState(err) != "23514" {
		t.Errorf("a line added to a sent quote: %v, want a check violation", err)
	}
	if got := idsIn(t, app, map[string]string{"app.patient_id": p[0].patient}, "SELECT id::text FROM quote"); !slices.Equal(got, []string{quote}) {
		t.Errorf("patient A sees quotes %v, want their own", got)
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicB}, "SELECT id::text FROM quote"); len(got) != 0 {
		t.Errorf("clinic B sees quotes %v, want none", got)
	}
}

// seedStaffAccount2 is seedStaffAccount with another e-mail, for a second clinic.
func seedStaffAccount2(t *testing.T, owner *pgxpool.Pool, org string) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role, organization_id)
		VALUES ('desk.b@example.test', 'h', 'reception', $1) RETURNING id`, org).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
