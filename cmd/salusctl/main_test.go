package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// ownerDB points salusctl at a fresh database as the owner and returns a count query on it.
func ownerDB(t *testing.T) (*pgxpool.Pool, func(q string, args ...any) int) {
	t.Helper()
	url := pgtest.NewDatabaseURL(t)
	t.Setenv("DATABASE_URL", url)
	db, err := postgres.Open(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db, func(q string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(t.Context(), q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
}

func TestCreateClinic_CreatesTheClinicAndItsFirstAdminsInviteAudited(t *testing.T) {
	_, count := ownerDB(t)
	var out bytes.Buffer

	if err := run(t.Context(), []string{"create-clinic", "-name", "Synthetic clinic", "-admin-email", "Dee@example.test"}, &out); err != nil {
		t.Fatalf("create-clinic: %v", err)
	}
	if n := count(`SELECT count(*) FROM organization o JOIN invite i ON i.organization_id = o.id
		WHERE o.name = 'Synthetic clinic' AND i.role = 'admin' AND i.email = 'dee@example.test' AND i.invited_by IS NULL`); n != 1 {
		t.Errorf("%d clinics with the admin invite, want 1", n)
	}
	if n := count(`SELECT count(*) FROM river_job WHERE kind = 'invite_email' AND queue = 'email'`); n != 1 {
		t.Errorf("%d invite_email jobs, want 1", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_name = 'salusctl' AND agent_account_id IS NULL AND action = 'C'`); n != 2 {
		t.Errorf("%d creates audited as salusctl, want 2 (clinic and invite)", n)
	}
	if strings.Contains(out.String(), "example.test") || !strings.Contains(out.String(), "organization ") || !strings.Contains(out.String(), "invite ") {
		t.Errorf("output %q: want the clinic's and the invite's ids, no e-mail", out.String())
	}
}

func TestCreateClinic_CreatesNothingWhenTheInviteIsRefused(t *testing.T) {
	_, count := ownerDB(t)

	if err := run(t.Context(), []string{"create-clinic", "-name", "Synthetic clinic", "-admin-email", "not an e-mail"}, &bytes.Buffer{}); err == nil {
		t.Fatal("create-clinic with an invalid e-mail: error = nil")
	}
	if n := count("SELECT count(*) FROM organization"); n != 0 {
		t.Errorf("%d clinics, want none: the clinic and its invite are one transaction", n)
	}
}

func TestResetMFA_ClearsAStaffAccountsEnrolmentAndSessions(t *testing.T) {
	db, count := ownerDB(t)
	var staff, patient string
	err := db.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id),
		a AS (INSERT INTO account (email, password_hash, role, organization_id, totp_secret, totp_last_step, mfa_enrolled_at)
		      SELECT 'dee@example.test', 'h', 'admin', id, '\x01', 1, now() FROM o RETURNING id, organization_id)
		INSERT INTO session (account_id, organization_id, token_hash, mfa_verified, expires_at)
		SELECT id, organization_id, '\x02', true, now() + interval '1 hour' FROM a RETURNING account_id`).Scan(&staff)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role, mfa_enrolled_at)
		VALUES ('ana@example.test', 'h', 'patient', now()) RETURNING id`).Scan(&patient); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer

	if err := run(t.Context(), []string{"reset-mfa", staff}, &out); err != nil {
		t.Fatalf("reset-mfa: %v", err)
	}
	if n := count(`SELECT count(*) FROM account WHERE id = $1 AND totp_secret IS NULL AND totp_last_step IS NULL AND mfa_enrolled_at IS NULL`, staff); n != 1 {
		t.Error("the staff account is still enrolled")
	}
	if n := count(`SELECT count(*) FROM session WHERE revoked_at IS NULL`); n != 0 {
		t.Errorf("%d open sessions, want none", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_name = 'salusctl' AND action = 'U' AND outcome = 0 AND entity_reference = $1`, "Account/"+staff); n != 1 {
		t.Errorf("%d audited resets, want 1", n)
	}
	for _, id := range []string{patient, "0b6c8a52-0000-4000-8000-000000000000"} {
		if err := run(t.Context(), []string{"reset-mfa", id}, &bytes.Buffer{}); err == nil {
			t.Errorf("reset-mfa %s: error = nil, want a refusal", id)
		}
	}
	if n := count(`SELECT count(*) FROM account WHERE id = $1 AND mfa_enrolled_at IS NOT NULL`, patient); n != 1 {
		t.Error("the patient's enrolment was reset")
	}
}

func TestRun_RejectsAnUnknownCommandOrMissingArguments(t *testing.T) {
	ownerDB(t)
	for _, args := range [][]string{nil, {"drop-everything"}, {"reset-mfa"}, {"create-clinic", "-name", "x"}, {"create-clinic", "-bogus"}} {
		if err := run(t.Context(), args, &bytes.Buffer{}); err == nil {
			t.Errorf("run(%q): error = nil", args)
		}
	}
	t.Setenv("DATABASE_URL", "")
	if err := run(t.Context(), []string{"reset-mfa", "x"}, &bytes.Buffer{}); err == nil {
		t.Error("run() without DATABASE_URL: error = nil")
	}
}
