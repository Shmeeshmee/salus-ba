package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// fileRequest inserts a data request as the owner and returns its id.
func fileRequest(t *testing.T, owner *pgxpool.Pool, account, patient, org any, filedBy string) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `INSERT INTO data_request
		(account_id, patient_id, organization_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $2, $3, $4, 'access', '2026-11-24T08:00:00Z', '2026-12-24T08:00:00Z') RETURNING id`,
		account, patient, org, filedBy).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// registerPatient creates, as the owner, a profile the clinic registered and returns its id.
func registerPatient(t *testing.T, owner *pgxpool.Pool, org string) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `INSERT INTO patient (registered_by_organization_id, given_name, family_name, birth_date)
		VALUES ($1, 'Bo', 'Example', '1980-02-03') RETURNING id`, org).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// The patient sees their own requests, a clinic those targeting it, and nobody in the app's role those targeting the
// account but the patient (the operator reads them as the owner); a patient files only for themselves and a clinic only
// for a patient it registered, by its own staff (docs/clinic-cms WAVE5.md, 5.0, S1, S10).
func TestWave5_DataRequestsAreSeenByThePatientAndTheTargetOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	staff := seedStaffAccount(t, owner, clinicA)
	registered := registerPatient(t, owner, clinicA)
	toAccount := fileRequest(t, owner, p[0].account, p[0].patient, nil, p[0].account)
	toClinic := fileRequest(t, owner, p[0].account, p[0].patient, clinicA, p[0].account)
	ofB := fileRequest(t, owner, p[1].account, p[1].patient, clinicB, p[1].account)
	onBehalf := fileRequest(t, owner, nil, registered, clinicA, staff)

	const requests = "SELECT id::text FROM data_request"
	for name, tc := range map[string]struct {
		scope map[string]string
		want  []string
	}{
		"patient A":  {map[string]string{"app.account_id": p[0].account}, sorted(toAccount, toClinic)},
		"clinic A":   {map[string]string{"app.organization_id": clinicA}, sorted(toClinic, onBehalf)},
		"clinic B":   {map[string]string{"app.organization_id": clinicB}, []string{ofB}},
		"no scope":   {nil, nil},
		"patient id": {map[string]string{"app.patient_id": p[0].patient}, nil},
	} {
		if got := idsIn(t, app, tc.scope, requests); !slices.Equal(got, tc.want) {
			t.Errorf("%s sees %v, want %v", name, got, tc.want)
		}
	}
	const insert = `INSERT INTO data_request (account_id, patient_id, organization_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $2, $3, $4, 'erasure', now(), now() + interval '1 month')`
	exec := func(scope map[string]string, args ...any) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, insert, args...)
			return err
		})
	}
	scopeA := map[string]string{"app.account_id": p[0].account}
	clinic := map[string]string{"app.organization_id": clinicA}
	if err := exec(scopeA, p[0].account, p[0].patient, nil, p[0].account); err != nil {
		t.Errorf("patient A files to the account: %v", err)
	}
	if err := exec(clinic, nil, registered, clinicA, staff); err != nil {
		t.Errorf("clinic A files for its registered patient: %v", err)
	}
	for name, tc := range map[string]struct {
		scope map[string]string
		args  []any
	}{
		"patient A for patient B":           {scopeA, []any{p[1].account, p[1].patient, nil, p[1].account}},
		"patient A with B's profile":        {scopeA, []any{p[0].account, p[1].patient, clinicA, p[0].account}},
		"clinic A for a patient's own":      {clinic, []any{nil, p[0].patient, clinicA, staff}},
		"clinic B for A's registered":       {map[string]string{"app.organization_id": clinicB}, []any{nil, registered, clinicB, staff}},
		"clinic A by a patient's account":   {clinic, []any{nil, registered, clinicA, p[0].account}},
		"clinic A for the account target":   {clinic, []any{nil, registered, nil, staff}},
		"clinic A in patient A's name":      {clinic, []any{p[0].account, p[0].patient, clinicA, p[0].account}},
		"patient A filed by someone else":   {scopeA, []any{p[0].account, p[0].patient, nil, staff}},
		"patient A at a clinic, no profile": {scopeA, []any{p[0].account, nil, clinicA, p[0].account}},
	} {
		if err := exec(tc.scope, tc.args...); err == nil {
			t.Errorf("%s: no error, want a refusal", name)
		}
	}
	for name, stmt := range map[string]string{
		"answered":  "INSERT INTO data_request (account_id, filed_by_account_id, kind, status, received_at, due_at, answered_at, answer_code) VALUES ($1, $1, 'access', 'fulfilled', now(), now() + interval '1 month', now(), 'done')",
		"extended":  "INSERT INTO data_request (account_id, filed_by_account_id, kind, received_at, due_at, extended_until, extension_reason) VALUES ($1, $1, 'access', now(), now() + interval '1 month', now() + interval '2 months', 'complex')",
		"backdated": "INSERT INTO data_request (account_id, filed_by_account_id, kind, received_at, due_at) VALUES ($1, $1, 'access', '2020-01-01', '2020-02-01')",
	} {
		err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt, p[0].account)
			return err
		})
		if sqlState(err) != "23514" {
			t.Errorf("patient A files a request %s: %v, want a check violation", name, err)
		}
	}
}

// A request moves forward only, is answered with a code, extended once, never deleted, and its status changes are an
// append-only history; the patient may only withdraw theirs.
func TestWave5_ADataRequestMovesForwardAndKeepsItsHistory(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	id := fileRequest(t, owner, p[0].account, p[0].patient, clinicA, p[0].account)
	ctx := t.Context()
	patient := map[string]string{"app.account_id": p[0].account}
	clinic := map[string]string{"app.organization_id": clinicA}
	update := func(scope map[string]string, set string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			var got string
			return db.QueryRow(ctx, "UPDATE data_request SET "+set+" WHERE id = $1 RETURNING id", id).Scan(&got)
		})
	}
	for name, set := range map[string]string{
		"the patient answers":         "status = 'fulfilled', answered_at = now(), answer_code = 'done'",
		"the patient extends":         "extended_until = due_at + interval '1 month', extension_reason = 'complex'",
		"the clinic changes the kind": "kind = 'erasure'",
		"fulfilled without a code":    "status = 'fulfilled', answered_at = now()",
		"refused with a done code":    "status = 'refused', answered_at = now(), answer_code = 'done'",
		"extended without a reason":   "extended_until = due_at + interval '1 month'",
		"extended before the due":     "extended_until = due_at - interval '1 day', extension_reason = 'complex'",
		"extended over two months":    "extended_until = due_at + interval '3 months', extension_reason = 'complex'",
	} {
		scope := clinic
		if name[:11] == "the patient" {
			scope = patient
		}
		if err := update(scope, set); err == nil {
			t.Errorf("%s: no error, want a refusal", name)
		}
	}
	if err := update(clinic, "status = 'in_progress', extended_until = due_at + interval '2 months', extension_reason = 'many_requests'"); err != nil {
		t.Fatalf("the clinic takes it up and extends it: %v", err)
	}
	if err := update(clinic, "extended_until = due_at + interval '1 month', extension_reason = 'complex'"); err == nil {
		t.Error("a second extension: no error, want a refusal")
	}
	if err := update(clinic, "status = 'withdrawn'"); err == nil {
		t.Error("the clinic withdraws the patient's request: no error, want a refusal")
	}
	if err := update(clinic, "status = 'received'"); err == nil {
		t.Error("back to received: no error, want a refusal")
	}
	if err := update(patient, "status = 'withdrawn', version = version + 1"); err != nil {
		t.Fatalf("the patient withdraws: %v", err)
	}
	if err := update(clinic, "status = 'fulfilled', answered_at = now(), answer_code = 'done'"); err == nil {
		t.Error("a withdrawn request answered: no error, want a refusal")
	}
	if _, err := owner.Exec(ctx, "DELETE FROM data_request WHERE id = $1", id); err == nil {
		t.Error("the owner deletes a request: no error, want a refusal")
	}

	checkHistory(t, owner, app, id, p)
}

// checkHistory checks that the history of patient A's request id at clinic A is written with the request's status,
// by the patient in their scope, seen where the request is seen, and append-only.
func checkHistory(t *testing.T, owner, app *pgxpool.Pool, id string, p [2]patientRow) {
	t.Helper()
	ctx := t.Context()
	patient := map[string]string{"app.account_id": p[0].account}
	clinic := map[string]string{"app.organization_id": clinicA}
	const change = `INSERT INTO data_request_change (data_request_id, status, changed_at, changed_by_account_id) VALUES ($1, $2, now(), $3)`
	err := asApp(t, app, patient, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, change, id, "withdrawn", p[0].account)
		return err
	})
	if err != nil {
		t.Errorf("the patient records the withdrawal: %v", err)
	}
	for name, args := range map[string][]any{
		"a status the request does not have": {id, "fulfilled", p[0].account},
		"as a job":                           {id, "withdrawn", nil},
		"as another account":                 {id, "withdrawn", p[1].account},
	} {
		err := asApp(t, app, patient, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, change, args...)
			return err
		})
		if err == nil {
			t.Errorf("the patient records %s: no error, want a refusal", name)
		}
	}
	err = asApp(t, app, map[string]string{"app.organization_id": clinicB}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, change, id, "withdrawn", nil)
		return err
	})
	if err == nil {
		t.Error("clinic B records a change of clinic A's request: no error, want a refusal")
	}
	if got := idsIn(t, app, clinic, "SELECT data_request_id::text FROM data_request_change"); !slices.Equal(got, []string{id}) {
		t.Errorf("clinic A sees the history of %v, want %v", got, []string{id})
	}
	for _, stmt := range []string{"UPDATE data_request_change SET status = 'received'", "DELETE FROM data_request_change"} {
		if _, err := owner.Exec(ctx, stmt); err == nil {
			t.Errorf("%s: no error, want a refusal", stmt)
		}
	}
}

// A clinic restricts a patient's record at itself only, by its own staff; another clinic sees nothing of it.
func TestWave5_ARestrictionIsPerClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	staff := seedStaffAccount(t, owner, clinicA)
	const insert = `INSERT INTO patient_restriction (patient_id, organization_id, restricted_at, restricted_by_account_id) VALUES ($1, $2, now(), $3)`
	exec := func(org string, args ...any) error {
		return asApp(t, app, map[string]string{"app.organization_id": org}, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, insert, args...)
			return err
		})
	}
	if err := exec(clinicA, p[0].patient, clinicA, staff); err != nil {
		t.Fatalf("clinic A restricts patient A's record: %v", err)
	}
	if err := exec(clinicB, p[0].patient, clinicA, staff); err == nil {
		t.Error("clinic B restricts at clinic A: no error, want a refusal")
	}
	ofB := fileRequest(t, owner, p[1].account, p[1].patient, clinicB, p[1].account)
	err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `INSERT INTO patient_restriction (patient_id, organization_id, restricted_at, restricted_by_account_id, data_request_id)
			VALUES ($1, $2, now(), $3, $4)`, p[1].patient, clinicA, staff, ofB)
		return err
	})
	if err == nil {
		t.Error("clinic A restricts for clinic B's request: no error, want a refusal")
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO patient_restriction (patient_id, organization_id, restricted_at) VALUES ($1, $2, now())`,
		p[1].patient, clinicA); err != nil {
		t.Fatal(err)
	}
	if err := exec(clinicB, p[0].patient, clinicB, staff); err == nil {
		t.Error("clinic B restricts naming clinic A's staff: no error, want a refusal")
	}
	const restricted = "SELECT patient_id::text FROM patient_restriction"
	if got, want := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, restricted), sorted(p[0].patient, p[1].patient); !slices.Equal(got, want) {
		t.Errorf("clinic A sees %v, want %v", got, want)
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicB}, restricted); len(got) != 0 {
		t.Errorf("clinic B sees %v, want none", got)
	}
	if got := idsIn(t, app, map[string]string{"app.account_id": p[0].account}, restricted); len(got) != 0 {
		t.Errorf("the patient's scope sees %v, want none", got)
	}
	var lifted []string
	err = asApp(t, app, map[string]string{"app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
		rows, err := db.Query(ctx, "DELETE FROM patient_restriction RETURNING patient_id::text")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			lifted = append(lifted, id)
		}
		return rows.Err()
	})
	if err != nil || !slices.Equal(lifted, []string{p[0].patient}) {
		t.Errorf("clinic A lifts %v, %v; want its own restriction only, not the erasure's", lifted, err)
	}
}

// An account is closed with its erasure date, or open with neither; the patient's objection to analytics is theirs to
// set.
func TestWave5_AccountClosureAndTheAnalyticsObjection(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p := seedPatients(t, owner)
	ctx := t.Context()
	for set, want := range map[string]string{
		"closed_at = now()":   "23514",
		"erase_after = now()": "23514",
		"closed_at = now(), erase_after = now() - interval '1 day'":   "23514",
		"closed_at = now(), erase_after = now() + interval '30 days'": "",
	} {
		if _, err := owner.Exec(ctx, "UPDATE account SET closed_at = NULL, erase_after = NULL WHERE id = $1", p[0].account); err != nil {
			t.Fatal(err)
		}
		_, err := owner.Exec(ctx, "UPDATE account SET "+set+" WHERE id = $1", p[0].account)
		if sqlState(err) != want {
			t.Errorf("%s: %v, want %q", set, err, want)
		}
	}
	err := asApp(t, app, map[string]string{"app.account_id": p[1].account}, func(ctx context.Context, db postgres.Querier) error {
		var id string
		return db.QueryRow(ctx, "UPDATE patient SET analytics_objected_at = now() WHERE id = $1 RETURNING id", p[1].patient).Scan(&id)
	})
	if err != nil {
		t.Errorf("patient B objects: %v", err)
	}
}
