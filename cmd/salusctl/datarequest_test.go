package main

import (
	"bytes"
	"strings"
	"testing"
)

// salusctl datarequest (docs/clinic-cms/WAVE5.md D5): the operator lists, reads, answers and extends the requests to
// the account, never a clinic's, each audited as salusctl; it prints opaque ids and codes, never an e-mail.
func TestDataRequest_TheOperatorHandlesTheRequestsToTheAccount(t *testing.T) {
	db, count := ownerDB(t)
	ctx := t.Context()
	var account, toAccount, overdue, toClinic string
	if err := db.QueryRow(ctx, `INSERT INTO account (email, password_hash, role) VALUES ('dr-patient@example.test', 'h', 'patient') RETURNING id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	insert := func(org, due string) string {
		var id string
		err := db.QueryRow(ctx, `WITH p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date)
			SELECT $1, 'Ana', 'Synthetic', '1980-01-01' WHERE NOT EXISTS (SELECT FROM patient WHERE account_id = $1) RETURNING id)
			INSERT INTO data_request (account_id, patient_id, organization_id, filed_by_account_id, kind, received_at, due_at)
			SELECT $1, COALESCE((SELECT id FROM p), (SELECT id FROM patient WHERE account_id = $1)), NULLIF($2, '')::uuid, $1, 'access', now() - interval '40 days', now() + $3::interval
			RETURNING id`, account, org, due).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO data_request_change (data_request_id, changed_by_account_id, status, changed_at)
			SELECT id, filed_by_account_id, 'received', received_at FROM data_request WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	var clinic string
	if err := db.QueryRow(ctx, `INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id`).Scan(&clinic); err != nil {
		t.Fatal(err)
	}
	toAccount, overdue, toClinic = insert("", "5 days"), insert("", "-1 day"), insert(clinic, "5 days")

	var out bytes.Buffer
	if err := run(ctx, []string{"datarequest", "list"}, &out); err != nil {
		t.Fatalf("datarequest list: %v", err)
	}
	if !strings.Contains(out.String(), toAccount) || !strings.Contains(out.String(), overdue) || strings.Contains(out.String(), toClinic) ||
		strings.Contains(out.String(), "example.test") || strings.Contains(out.String(), "Ana") {
		t.Errorf("datarequest list printed %q: want the two to the account, no clinic's, no personal data", out.String())
	}
	out.Reset()
	if err := run(ctx, []string{"datarequest", "show", toAccount}, &out); err != nil {
		t.Fatalf("datarequest show: %v", err)
	}
	if !strings.Contains(out.String(), "version 1") || !strings.Contains(out.String(), "history ") {
		t.Errorf("datarequest show printed %q: want its version and history", out.String())
	}
	if err := run(ctx, []string{"datarequest", "show", toClinic}, &bytes.Buffer{}); err == nil {
		t.Error("datarequest show of a clinic's request: error = nil")
	}
	if err := run(ctx, []string{"datarequest", "extend", overdue, "1", "complex"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "overdue") {
		t.Errorf("extending an overdue request: error = %v, want overdue (S12)", err)
	}
	if err := run(ctx, []string{"datarequest", "extend", toAccount, "1", "complex"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("datarequest extend: %v", err)
	}
	out.Reset()
	if err := run(ctx, []string{"datarequest", "answer", toAccount, "2", "done"}, &out); err != nil {
		t.Fatalf("datarequest answer: %v", err)
	}
	if !strings.Contains(out.String(), "fulfilled") || !strings.Contains(out.String(), "answer done") {
		t.Errorf("datarequest answer printed %q", out.String())
	}
	if n := count(`SELECT count(*) FROM data_request WHERE id = $1 AND status = 'fulfilled' AND answer_code = 'done' AND extension_reason = 'complex'`, toAccount); n != 1 {
		t.Errorf("answered and extended: %d, want 1", n)
	}
	if n := count(`SELECT count(*) FROM data_request_change WHERE data_request_id = $1 AND status = 'fulfilled' AND changed_by_account_id IS NULL`, toAccount); n != 1 {
		t.Errorf("%d history entries of the answer by no account, want 1", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_name = 'salusctl' AND agent_account_id IS NULL AND entity_name = 'data_request'`); n != 4 {
		t.Errorf("%d accesses audited as salusctl, want 4 (list, show, extend, answer)", n)
	}
	for _, args := range [][]string{{"datarequest"}, {"datarequest", "bogus"}, {"datarequest", "answer", toAccount, "x", "done"}, {"datarequest", "extend", toAccount, "x", "complex"}} {
		if err := run(ctx, args, &bytes.Buffer{}); err == nil {
			t.Errorf("%v: error = nil", args)
		}
	}
}

// salusctl datarequest reopen-account reopens a closed account before its erasure: its erasure request is withdrawn,
// audited as salusctl; once reopened there is nothing to reopen.
func TestDataRequest_ReopensAClosedAccount(t *testing.T) {
	db, count := ownerDB(t)
	ctx := t.Context()
	var account, erasure string
	if err := db.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, closed_at, erase_after)
		VALUES ('closed@example.test', 'h', 'patient', now() - interval '2 days', now() + interval '19 days') RETURNING id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO data_request (account_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $1, 'erasure', now() - interval '2 days', now() + interval '28 days') RETURNING id`, account).Scan(&erasure); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `UPDATE data_request SET status = 'in_progress', version = 2 WHERE id = $1`, erasure); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(ctx, []string{"datarequest", "reopen-account", account}, &out); err != nil {
		t.Fatalf("datarequest reopen-account: %v", err)
	}
	if !strings.Contains(out.String(), erasure) || strings.Contains(out.String(), "example.test") {
		t.Errorf("printed %q: want the withdrawn request's id, no e-mail", out.String())
	}
	if n := count(`SELECT count(*) FROM account WHERE id = $1 AND closed_at IS NULL AND erase_after IS NULL`, account); n != 1 {
		t.Errorf("reopened accounts: %d, want 1", n)
	}
	if n := count(`SELECT count(*) FROM data_request WHERE id = $1 AND status = 'withdrawn'`, erasure); n != 1 {
		t.Errorf("withdrawn erasure requests: %d, want 1", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_name = 'salusctl' AND action = 'U'`); n != 2 {
		t.Errorf("%d updates audited as salusctl, want 2 (the account, the request)", n)
	}
	if err := run(ctx, []string{"datarequest", "reopen-account", account}, &bytes.Buffer{}); err == nil {
		t.Error("reopening twice: error = nil")
	}
}
