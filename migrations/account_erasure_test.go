package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

var erasureScope = map[string]string{"app.erasure": "on"}

// closeAccount closes the account as the owner, its erasure date a day ago.
func closeAccount(t *testing.T, owner *pgxpool.Pool, account string) {
	t.Helper()
	if _, err := owner.Exec(t.Context(), `UPDATE account SET closed_at = now() - interval '22 days',
		erase_after = now() - interval '1 day' WHERE id = $1`, account); err != nil {
		t.Fatal(err)
	}
}

// The erasure scope files, moves and answers the account's own erasure request of a closed account, and nothing else
// (docs/clinic-cms/WAVE5.md S5, S6, D3).
func TestAccountErasure_TheErasureScopeHandlesTheClosedAccountsErasureRequestOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p := seedPatients(t, owner)
	closeAccount(t, owner, p[0].account)
	const insert = `INSERT INTO data_request (account_id, patient_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $2, $1, $3, now(), now() + interval '1 month') RETURNING id`
	file := func(scope map[string]string, account, patient, kind string) (string, error) {
		var id string
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			return db.QueryRow(ctx, insert, account, patient, kind).Scan(&id)
		})
		return id, err
	}
	id, err := file(erasureScope, p[0].account, p[0].patient, "erasure")
	if err != nil {
		t.Fatalf("the closed account's erasure request: %v", err)
	}
	for name, tc := range map[string]struct {
		scope            map[string]string
		account, patient string
		kind             string
	}{
		"an open account":     {erasureScope, p[1].account, p[1].patient, "erasure"},
		"another kind":        {erasureScope, p[0].account, p[0].patient, "access"},
		"without the setting": {nil, p[0].account, p[0].patient, "erasure"},
		"with a clinic set":   {map[string]string{"app.erasure": "on", "app.organization_id": clinicA}, p[0].account, p[0].patient, "erasure"},
	} {
		if _, err := file(tc.scope, tc.account, tc.patient, tc.kind); sqlState(err) != "42501" {
			t.Errorf("%s: %v, want 42501", name, err)
		}
	}
	err = asApp(t, app, erasureScope, func(ctx context.Context, db postgres.Querier) error {
		for _, step := range []string{
			"INSERT INTO data_request_change (changed_at, data_request_id, changed_by_account_id, status) VALUES (now(), $1, $2, 'received')",
			"UPDATE data_request SET status = 'in_progress', version = 2 WHERE id = $1 AND $2::uuid IS NOT NULL",
			"INSERT INTO data_request_change (changed_at, data_request_id, status) VALUES (now(), $1, 'in_progress') RETURNING $2::uuid",
			`UPDATE data_request SET status = 'fulfilled', answer_code = 'done_partly_kept', answered_at = now(), version = 3
			 WHERE id = $1 AND $2::uuid IS NOT NULL`,
			"INSERT INTO data_request_change (changed_at, data_request_id, status) VALUES (now(), $1, 'fulfilled') RETURNING $2::uuid",
		} {
			if _, err := db.Exec(ctx, step, id, p[0].account); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("the job moves and answers it: %v", err)
	}
	if got := idsIn(t, app, nil, "SELECT id::text FROM data_request"); len(got) != 0 {
		t.Errorf("no scope sees %v", got)
	}
	if n := count(t, owner, "SELECT count(*) FROM data_request_change"); n != 3 {
		t.Errorf("%d history entries, want 3", n)
	}
}

// An erased account keeps nothing of its login (S6).
func TestAccountErasure_AnErasedAccountIsBlank(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	p := seedPatients(t, owner)
	closeAccount(t, owner, p[0].account)
	ctx := t.Context()
	if _, err := owner.Exec(ctx, "UPDATE account SET erased_at = now() WHERE id = $1", p[0].account); sqlState(err) != "23514" {
		t.Errorf("erased with the e-mail kept: %v, want 23514", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE account SET erased_at = now(), email = id::text || '@erased.invalid', password_hash = ''
		WHERE id = $1`, p[1].account); sqlState(err) != "23514" {
		t.Errorf("an open account erased: %v, want 23514", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE account SET erased_at = now(), email = id::text || '@erased.invalid', password_hash = ''
		WHERE id = $1`, p[0].account); err != nil {
		t.Errorf("erased blank: %v", err)
	}
}

// The erasure restricts the record a clinic keeps of a due account's profile, in that clinic's scope, by no staff.
func TestAccountErasure_TheJobRestrictsTheDueProfileAtAClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	closeAccount(t, owner, p[0].account)
	restrict := func(scope map[string]string, patient string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO patient_restriction (restricted_at, patient_id, organization_id)
				VALUES (now(), $1, $2)`, patient, clinicA)
			return err
		})
	}
	clinic := map[string]string{"app.erasure": "on", "app.organization_id": clinicA}
	if err := restrict(clinic, p[0].patient); err != nil {
		t.Errorf("the due profile: %v", err)
	}
	if err := restrict(clinic, p[1].patient); sqlState(err) != "42501" {
		t.Errorf("an open account's profile: %v, want 42501", err)
	}
	if err := restrict(map[string]string{"app.organization_id": clinicA}, p[0].patient); sqlState(err) != "42501" {
		t.Errorf("without the erasure setting: %v, want 42501", err)
	}
}

func count(t *testing.T, db *pgxpool.Pool, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(t.Context(), query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
