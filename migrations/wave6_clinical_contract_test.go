package migrations_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A lab result is attached by the clinic's staff for a patient with its active care consent, released once by a
// practitioner of the clinic, seen by the patient only then, and never changed after (U5).
func TestWave6_ALabResultIsSeenByThePatientOnlyOnceReleased(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	desk := seedAccount(t, owner, clinicA, "reception", "desk@example.test")
	doctor := seedAccount(t, owner, clinicA, "practitioner", "doctor@example.test")
	scopeA := map[string]string{"app.organization_id": clinicA}
	const attach = `INSERT INTO diagnostic_report (id, organization_id, patient_id, appointment_id, created_by_account_id, title)
		VALUES ($1, $2, $3, $4, $5, 'Complete blood count')`
	const report = "d0000000-0000-4000-8000-00000000000d"
	// A receptionist sees no clinical content: the desk attaches no result.
	if err := execIn(t, app, scopeA, attach, report, clinicA, p[0].patient, a.aFuture, desk); err == nil {
		t.Error("the desk attaches a result: no error")
	}
	if err := execIn(t, app, scopeA, attach, report, clinicA, p[0].patient, a.aFuture, doctor); err != nil {
		t.Fatalf("the practitioner attaches a result: %v", err)
	}
	// Patient B has no consent with clinic A.
	if err := execIn(t, app, scopeA, attach, "d0000000-0000-4000-8000-0000000000de", clinicA, p[1].patient, a.other, doctor); err == nil {
		t.Error("a result for a patient without the clinic's consent: no error")
	}
	const value = `INSERT INTO diagnostic_report_result (diagnostic_report_id, position, test, value_decimal, unit, reference_range, interpretation)
		VALUES ($1, $2, 'Haemoglobin', 151, 'g/L', '130-170', 'N')`
	if err := execIn(t, app, scopeA, value, report, 1); err != nil {
		t.Errorf("the practitioner types in a value: %v", err)
	}
	mine := map[string]string{"app.patient_id": p[0].patient}
	if got := idsIn(t, app, mine, "SELECT id::text FROM diagnostic_report"); got != nil {
		t.Errorf("the patient sees %v before the release, want none", got)
	}
	const release = `UPDATE diagnostic_report SET status = 'final', released_at = now(), released_by_account_id = $2 WHERE id = $1`
	if err := execIn(t, app, scopeA, release, report, desk); err == nil {
		t.Error("the desk releases a result: no error")
	}
	// The release goes on after the patient revoked their consent: consent is asked when the result is written.
	if _, err := owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1 AND organization_id = $2`,
		p[0].patient, clinicA); err != nil {
		t.Fatal(err)
	}
	if err := execIn(t, app, scopeA, release, report, doctor); err != nil {
		t.Fatalf("the practitioner releases the result: %v", err)
	}
	if got := idsIn(t, app, mine, "SELECT id::text FROM diagnostic_report"); !slices.Equal(got, []string{report}) {
		t.Errorf("the patient sees %v after the release, want the report", got)
	}
	if got := idsIn(t, app, mine, "SELECT diagnostic_report_id::text FROM diagnostic_report_result"); !slices.Equal(got, []string{report}) {
		t.Errorf("the patient sees results of %v, want the report's", got)
	}
	if err := execIn(t, app, scopeA, `UPDATE diagnostic_report SET title = 'Other' WHERE id = $1`, report); sqlState(err) != "23514" {
		t.Errorf("a released report changed = %v, want a check violation", err)
	}
	if err := execIn(t, app, scopeA, value, report, 2); err == nil {
		t.Error("a value added to a released report: no error")
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO diagnostic_report (organization_id, patient_id, appointment_id, created_by_account_id, title,
		status, released_at, released_by_account_id) VALUES ($1, $2, $3, $4, 'X', 'final', now(), $4)`, clinicA, p[0].patient, a.aFuture, doctor); sqlState(err) != "23514" {
		t.Errorf("a report inserted released = %v, want a check violation", err)
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicB}, "SELECT id::text FROM diagnostic_report"); got != nil {
		t.Errorf("clinic B sees %v, want none", got)
	}
}

// A treatment plan is written for a patient with the clinic's active care consent from its own price list; the patient
// sees it once booked, never as a draft (U6).
func TestWave6_ATreatmentPlanIsTheClinicsUntilBooked(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	ctx := t.Context()
	doctor := seedAccount(t, owner, clinicA, "practitioner", "doctor@example.test")
	var serviceA, serviceB string
	const service = `INSERT INTO charge_item_definition (organization_id, name, price_minor) VALUES ($1, 'Synthetic service', 5000) RETURNING id`
	if err := owner.QueryRow(ctx, service, clinicA).Scan(&serviceA); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, service, clinicB).Scan(&serviceB); err != nil {
		t.Fatal(err)
	}
	scopeA := map[string]string{"app.organization_id": clinicA}
	const plan = "c0000000-0000-4000-8000-00000000000c"
	const write = `INSERT INTO care_plan (id, organization_id, patient_id, created_by_account_id, title) VALUES ($1, $2, $3, $4, 'Synthetic plan')`
	if err := execIn(t, app, scopeA, write, plan, clinicA, p[0].patient, doctor); err != nil {
		t.Fatalf("clinic A drafts a plan: %v", err)
	}
	if err := execIn(t, app, scopeA, write, "c0000000-0000-4000-8000-0000000000ce", clinicA, p[1].patient, doctor); err == nil {
		t.Error("a plan for a patient without the clinic's consent: no error")
	}
	desk := seedAccount(t, owner, clinicA, "reception", "desk@example.test")
	if err := execIn(t, app, scopeA, write, "c0000000-0000-4000-8000-0000000000cf", clinicA, p[0].patient, desk); err == nil {
		t.Error("the desk drafts a plan: no error")
	}
	// A draft nobody wants is revoked, never booked.
	const dropped = "c0000000-0000-4000-8000-0000000000c1"
	if err := execIn(t, app, scopeA, write, dropped, clinicA, p[0].patient, doctor); err != nil {
		t.Fatal(err)
	}
	if err := execIn(t, app, scopeA, `UPDATE care_plan SET status = 'revoked', revoked_at = now(), revoked_by_account_id = $2 WHERE id = $1`,
		dropped, doctor); err != nil {
		t.Errorf("clinic A revokes a draft: %v", err)
	}
	const step = `INSERT INTO care_plan_activity (care_plan_id, position, charge_item_definition_id, day_offset, visit_mode, appointment_id)
		VALUES ($1, $2, $3, 0, 'video', $4)`
	if err := execIn(t, app, scopeA, step, plan, 1, serviceA, nil); err != nil {
		t.Errorf("a step from the clinic's price list: %v", err)
	}
	if err := execIn(t, app, scopeA, step, plan, 2, serviceB, nil); err == nil {
		t.Error("a step from another clinic's price list: no error")
	}
	// Patient B with clinic A's consent, so their visit is seen in the clinic's scope.
	if _, err := owner.Exec(ctx, `INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())`,
		p[1].patient, clinicA); err != nil {
		t.Fatal(err)
	}
	if err := execIn(t, app, scopeA, step, plan, 2, serviceA, a.other); err == nil {
		t.Error("a step booked as another patient's visit: no error")
	}
	mine := map[string]string{"app.patient_id": p[0].patient}
	if got := idsIn(t, app, mine, "SELECT id::text FROM care_plan"); got != nil {
		t.Errorf("the patient sees the draft %v, want none", got)
	}
	if err := execIn(t, app, scopeA, `UPDATE care_plan SET status = 'active', starts_on = '2030-01-07', booked_at = now() WHERE id = $1`, plan); err != nil {
		t.Fatalf("clinic A books the plan: %v", err)
	}
	if got := idsIn(t, app, mine, "SELECT id::text FROM care_plan"); !slices.Equal(got, []string{plan}) {
		t.Errorf("the patient sees %v once booked, want the plan", got)
	}
	if err := execIn(t, app, mine, `UPDATE care_plan SET title = 'Mine' WHERE id = $1`, plan); err != nil {
		t.Fatal(err)
	}
	var title string
	if err := owner.QueryRow(ctx, `SELECT title FROM care_plan WHERE id = $1`, plan).Scan(&title); err != nil || title != "Synthetic plan" {
		t.Errorf("the patient changed the plan's title to %q, %v", title, err)
	}
}
