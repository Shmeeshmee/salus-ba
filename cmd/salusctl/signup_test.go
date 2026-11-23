package main

import (
	"bytes"
	"strings"
	"testing"
)

// salusctl signup list | approve (docs/clinic-cms, O1): the operator lists a self-signed-up clinic in the directory.
func TestSignup_ListsTheVerifiedSignupsAndApprovesOne(t *testing.T) {
	db, count := ownerDB(t)
	var org string
	err := db.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id)
		INSERT INTO clinic_signup (email, clinic_name, language, token_hash, verified_at, organization_id)
		SELECT 'admin@example.test', 'Synthetic clinic', 'en', sha256('one'), now(), id FROM o RETURNING organization_id`).Scan(&org)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(t.Context(), []string{"signup", "list"}, &out); err != nil {
		t.Fatalf("signup list: %v", err)
	}
	if !strings.Contains(out.String(), org) || !strings.Contains(out.String(), "waiting") || strings.Contains(out.String(), "example.test") {
		t.Errorf("signup list printed %q: want the clinic's id waiting, no e-mail", out.String())
	}
	out.Reset()
	if err := run(t.Context(), []string{"signup", "approve", org}, &out); err != nil {
		t.Fatalf("signup approve: %v", err)
	}
	if n := count(`SELECT count(*) FROM organization o JOIN clinic_signup s ON s.organization_id = o.id
		WHERE o.id = $1 AND o.listed AND s.approved_at IS NOT NULL`, org); n != 1 {
		t.Errorf("clinic listed and approved: %d, want 1", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_name = 'salusctl' AND action = 'U' AND entity_reference = 'Organization/' || $1`, org); n != 1 {
		t.Errorf("%d listings audited as salusctl, want 1", n)
	}
	if err := run(t.Context(), []string{"signup", "approve", org}, &bytes.Buffer{}); err == nil {
		t.Error("signup approve twice: error = nil")
	}
	for _, args := range [][]string{{"signup"}, {"signup", "approve"}, {"signup", "bogus"}} {
		if err := run(t.Context(), args, &bytes.Buffer{}); err == nil {
			t.Errorf("%v: error = nil", args)
		}
	}
}
