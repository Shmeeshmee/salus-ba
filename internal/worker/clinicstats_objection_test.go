package worker

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// clinicFigures are what the clinic's admin reads of the range: the appointments and the diagnoses by code.
type clinicFigures struct {
	Diagnoses    []clinicstats.Count
	Appointments clinicstats.AppointmentFigures
}

// refreshFigures refreshes every day of the range and reads the figures back as the clinic's admin.
func refreshFigures(t *testing.T, p statsPools, org string, first, last time.Time) clinicFigures {
	t.Helper()
	b := newClinicStats(p.stats, clock.Fixed{T: localAt(2026, 5, 1, 9, 0)})
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		if err := b.Refresh(t.Context(), org, localAt(d.Year(), d.Month(), d.Day(), 12, 0)); err != nil {
			t.Fatal(err)
		}
	}
	svc := clinicstats.NewService(clinicstats.Ports{Repo: clinicstats.NewPostgresRepository(p.app), Audit: recordNothing{}, Tx: postgres.NewTransactor(p.app)})
	c := clinicstats.Caller{AccountID: "acc", OrganizationID: org, Admin: true}
	appts, err := svc.Appointments(t.Context(), c, clinicstats.AppointmentQuery{From: first, To: last})
	if err != nil {
		t.Fatal(err)
	}
	diags, err := svc.Diagnoses(t.Context(), c, clinicstats.DiagnosisQuery{From: first, To: last, By: clinicstats.ByCode})
	if err != nil {
		t.Fatal(err)
	}
	return clinicFigures{Appointments: appts, Diagnoses: diags}
}

// secondPatient gives the clinic, as the owner, another synthetic patient with a role of their own: an appointment and
// a confirmed Condition on each day of the range, and returns the patient's id.
func secondPatient(t *testing.T, p statsPools, org string, first, last time.Time) string {
	t.Helper()
	var patient string
	err := p.owner.QueryRow(context.Background(), `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
			pt AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Bo', 'Example', '1991-02-02' FROM a RETURNING id),
			pr AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Cy', 'Example second') RETURNING id),
			r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT $1, id FROM pr RETURNING id),
			ap AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
				SELECT pt.id, r.id, $1, gen_random_uuid(), tstzrange(d + interval '10 hours', d + interval '10 hours 20 minutes'), 'fulfilled'
				FROM pt, r, generate_series($2::date, $3::date, interval '1 day') d RETURNING *),
			e AS (INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
				SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', lower(period) FROM ap RETURNING *),
			tm AS (INSERT INTO document_template (organization_id, title, fields)
				VALUES ($1, 'Placeholder template', '[{"key":"diagnosis","type":"coded","label":"Placeholder"}]') RETURNING id),
			dd AS (INSERT INTO document_draft (encounter_id, template_id, status, fields)
				SELECT e.id, tm.id, 'entered-in-error', '[{"key":"diagnosis","missing":true}]' FROM e, tm RETURNING id, encounter_id),
			c AS (INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
					practitioner_role_id, field_key, code, verification_status)
				SELECT e.period_start, $1, e.patient_id, e.id, dd.id, e.practitioner_role_id, 'second', 'I10', 'confirmed'
				FROM e JOIN dd ON dd.encounter_id = e.id RETURNING 1)
		SELECT (SELECT id::text FROM pt) FROM (SELECT count(*) FROM c) n`, org, first, last).Scan(&patient)
	if err != nil {
		t.Fatal(err)
	}
	return patient
}

// An objection to the analytics (docs/clinic-cms/WAVE5.md S9) and a restricted record at the clinic (S7) leave the
// patient out of the aggregates built after it: the figures of a clinic with the patient's appointments and diagnoses
// equal, field for field, those of the clinic without them; lifted, the patient counts again.
func TestClinicStats_AnObjectingOrRestrictedPatientIsLeftOut(t *testing.T) {
	t.Parallel()
	p := newStatsPools(t)
	org, roles := seedStatsRoles(t, p.owner, 2)
	first, last := date(2026, 3, 1), date(2026, 3, 14)
	synthetic(t, p.owner, org, roles, first, last, 3)
	without := refreshFigures(t, p, org, first, last)
	second := secondPatient(t, p, org, first, last)
	for name, q := range map[string][2]string{
		"objection": {`UPDATE patient SET analytics_objected_at = now() WHERE id = $1`, `UPDATE patient SET analytics_objected_at = NULL WHERE id = $1`},
		"restriction": {`INSERT INTO patient_restriction (restricted_at, patient_id, organization_id) VALUES (now(), $1, $2)`,
			`DELETE FROM patient_restriction WHERE patient_id = $1 AND organization_id = $2`},
	} {
		exec := func(sql string) {
			t.Helper()
			args := []any{second}
			if name == "restriction" {
				args = append(args, org)
			}
			if _, err := p.owner.Exec(t.Context(), sql, args...); err != nil {
				t.Fatal(err)
			}
		}
		exec(q[0])
		if got := refreshFigures(t, p, org, first, last); !reflect.DeepEqual(got, without) {
			t.Errorf("%s: figures with the patient\n got %+v\nwant %+v (without them)", name, got, without)
		}
		exec(q[1])
		if got := refreshFigures(t, p, org, first, last); reflect.DeepEqual(got, without) {
			t.Errorf("%s lifted: the patient's appointments and diagnoses are not counted", name)
		}
	}
}
