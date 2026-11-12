package main

import (
	"bytes"
	"testing"
)

// salusctl conditions unconfirmed lists the ids of the Conditions backfilled from documents signed before the
// confirmation existed (SAL-19 "unmapped ones flagged"), one per line, then their count; ids only.
func TestConditions_ListsTheUnconfirmed(t *testing.T) {
	db, _ := ownerDB(t)
	var id string
	err := db.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id),
		pr AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM pr RETURNING id, organization_id),
		a AS (INSERT INTO account (email, password_hash, role) VALUES ('synthetic@example.test', 'h', 'patient') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id),
		ap AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			SELECT p.id, r.id, r.organization_id, gen_random_uuid(), tstzrange('2026-11-02T09:00:00Z', '2026-11-02T09:30:00Z') FROM p, r RETURNING *),
		e AS (INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
			SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', lower(period) FROM ap RETURNING *),
		tm AS (INSERT INTO document_template (organization_id, title, fields) SELECT id, 'Placeholder', '[{"key":"d"}]' FROM o RETURNING id),
		d AS (INSERT INTO document_draft (encounter_id, template_id, status, fields)
			SELECT e.id, tm.id, 'entered-in-error', '[{"key":"d","missing":true}]' FROM e, tm RETURNING id)
		INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id, practitioner_role_id,
			field_key, code, verification_status)
		SELECT now(), e.organization_id, e.patient_id, e.id, d.id, e.practitioner_role_id, 'd', 'J00', 'unconfirmed' FROM e, d
		RETURNING id::text`).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(t.Context(), []string{"conditions", "unconfirmed"}, &out); err != nil {
		t.Fatal(err)
	}
	if want := id + "\n1 unconfirmed Condition(s)\n"; out.String() != want {
		t.Errorf("output %q, want %q", out.String(), want)
	}
	for _, args := range [][]string{{"conditions"}, {"conditions", "confirm"}} {
		if err := run(t.Context(), args, &bytes.Buffer{}); err == nil {
			t.Errorf("run(%q): error = nil", args)
		}
	}
}
