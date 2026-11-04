package auth_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The cut-off of the retention deletes; every row is one second either side of it.
var cutOff = time.Date(2026, 10, 5, 3, 0, 0, 0, time.UTC)

func TestDeleteSessionsEndedBefore_DeletesSessionsRevokedOrExpiredBeforeTheCutOffAsTheRetentionRole(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	sec, far := time.Second, 10*24*time.Hour
	// name -> expires_at, revoked_at (zero: not revoked), relative to the cut-off
	sessions := map[string][2]time.Duration{
		"expired 1 s before":            {-sec, 0},
		"expired 1 s after":             {sec, 0},
		"revoked 1 s before":            {far, -sec},
		"revoked 1 s after":             {far, sec},
		"expired before, revoked after": {-sec, sec},
	}
	var account string
	if err := owner.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role) VALUES ('ana@example.test', 'h', 'patient') RETURNING id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	for name, at := range sessions {
		var revoked *time.Time
		if at[1] != 0 {
			r := cutOff.Add(at[1])
			revoked = &r
		}
		if _, err := owner.Exec(t.Context(), `INSERT INTO session (account_id, token_hash, mfa_verified, expires_at, revoked_at, created_at)
			VALUES ($1, $2, false, $3, $4, $5)`, account, []byte(name), cutOff.Add(at[0]), revoked, cutOff.Add(-far)); err != nil {
			t.Fatal(err)
		}
	}

	n, err := auth.NewPostgresRepository(ret).DeleteSessionsEndedBefore(t.Context(), cutOff)

	if err != nil || n != 3 {
		t.Fatalf("DeleteSessionsEndedBefore() = %d, %v; want 3 rows", n, err)
	}
	rows, err := owner.Query(t.Context(), "SELECT token_hash FROM session ORDER BY token_hash")
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for rows.Next() {
		var h []byte
		if err := rows.Scan(&h); err != nil {
			t.Fatal(err)
		}
		kept = append(kept, string(h))
	}
	if len(kept) != 2 || kept[0] != "expired 1 s after" || kept[1] != "revoked 1 s after" {
		t.Errorf("kept %q, want the two sessions that ended after the cut-off", kept)
	}
}

func TestDeleteInvitesEndedBefore_DeletesInvitesAcceptedOrExpiredBeforeTheCutOffInEveryClinic(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	if _, err := owner.Exec(t.Context(), "INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic'), ($2, 'Other clinic')",
		clinicID, otherClinicID); err != nil {
		t.Fatal(err)
	}
	sec, far := time.Second, 10*24*time.Hour
	// e-mail -> clinic, expires_at, accepted_at (zero: open), relative to the cut-off
	invites := []struct {
		email, clinic     string
		expires, accepted time.Duration
	}{
		{"expired-before@example.test", clinicID, -sec, 0},
		{"expired-after@example.test", clinicID, sec, 0},
		{"accepted-before@example.test", otherClinicID, far, -sec},
		{"accepted-after@example.test", otherClinicID, far, sec},
	}
	for _, inv := range invites {
		var accepted *time.Time
		if inv.accepted != 0 {
			a := cutOff.Add(inv.accepted)
			accepted = &a
		}
		if _, err := owner.Exec(t.Context(), `INSERT INTO invite (organization_id, expires_at, accepted_at, role, email)
			VALUES ($1, $2, $3, 'admin', $4)`, inv.clinic, cutOff.Add(inv.expires), accepted, inv.email); err != nil {
			t.Fatal(err)
		}
	}

	n, err := auth.NewPostgresRepository(ret).DeleteInvitesEndedBefore(t.Context(), cutOff)

	if err != nil || n != 2 {
		t.Fatalf("DeleteInvitesEndedBefore() = %d, %v; want 2 rows, one in each clinic", n, err)
	}
	var kept string
	if err := owner.QueryRow(t.Context(), "SELECT string_agg(email, ',' ORDER BY email) FROM invite").Scan(&kept); err != nil {
		t.Fatal(err)
	}
	if kept != "accepted-after@example.test,expired-after@example.test" {
		t.Errorf("kept %s, want the two invites that ended after the cut-off", kept)
	}
}
