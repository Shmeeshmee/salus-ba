package worker

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// The autumn fall-back day, 25 October 2026 in Sarajevo, has 25 hours and the local hour 02:xx twice (00:xx and
// 01:xx UTC): the day counts every appointment and Condition of its 25 hours, and the heatmap's Sunday 02:00 cell and
// the day's diagnoses hold both 02:30s.
func TestClinicStats_TheFallBackDayHas25HoursAndItsRepeatedHourCountsTwice(t *testing.T) {
	t.Parallel()
	p := newStatsPools(t)
	org, roles := seedStatsRoles(t, p.owner, 2)
	first, last := date(2026, 10, 24), date(2026, 10, 26)
	synthetic(t, p.owner, org, roles, first, last, 4)
	// One appointment and one Condition at each 02:30: CEST (00:30Z) and CET (01:30Z).
	_, err := p.owner.Exec(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
		SELECT patient_id, practitioner_role_id, organization_id, gen_random_uuid(), tstzrange(s, s + interval '20 minutes'), 'booked'
		FROM (SELECT * FROM appointment WHERE organization_id = $1 LIMIT 1) a,
			unnest(ARRAY['2026-10-25T00:30:00Z', '2026-10-25T01:30:00Z']::timestamptz[]) s`, org)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.owner.Exec(t.Context(), `INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
			practitioner_role_id, field_key, code, verification_status)
		SELECT s, organization_id, patient_id, encounter_id, document_draft_id, practitioner_role_id, 'fallback-' || s::text, 'I10', 'confirmed'
		FROM (SELECT * FROM condition WHERE organization_id = $1 LIMIT 1) c,
			unnest(ARRAY['2026-10-25T00:30:00Z', '2026-10-25T01:30:00Z']::timestamptz[]) s`, org)
	if err != nil {
		t.Fatal(err)
	}
	b := newClinicStats(p.stats, clock.Fixed{T: localAt(2026, 10, 27, 9, 0)})
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		if err := b.Refresh(t.Context(), org, localAt(d.Year(), d.Month(), d.Day(), 12, 0)); err != nil {
			t.Fatal(err)
		}
	}
	svc := clinicstats.NewService(clinicstats.Ports{Repo: clinicstats.NewPostgresRepository(p.app), Audit: recordNothing{}, Tx: postgres.NewTransactor(p.app)})
	c := clinicstats.Caller{AccountID: "acc", OrganizationID: org, Admin: true}
	day := date(2026, 10, 25)

	var booked, at2, diagnoses int64
	if err := p.owner.QueryRow(t.Context(), `SELECT
			(SELECT count(*) FROM appointment WHERE organization_id = $1
				AND lower(period) >= '2026-10-24T22:00:00Z' AND lower(period) < '2026-10-25T23:00:00Z'),
			(SELECT count(*) FROM appointment WHERE organization_id = $1 AND status <> 'cancelled'
				AND lower(period) >= '2026-10-25T00:00:00Z' AND lower(period) < '2026-10-25T02:00:00Z'),
			(SELECT count(*) FROM condition WHERE organization_id = $1 AND verification_status = 'confirmed'
				AND recorded_at >= '2026-10-24T22:00:00Z' AND recorded_at < '2026-10-25T23:00:00Z')`, org).Scan(&booked, &at2, &diagnoses); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Appointments(t.Context(), c, clinicstats.AppointmentQuery{From: day, To: day})
	if err != nil || len(got.Series) != 1 || got.Series[0].Booked != booked {
		t.Errorf("25 October: %v, %v; want %d appointments over its 25 hours", got.Series, err, booked)
	}
	var cell int64
	for _, h := range got.Heatmap {
		if h.Weekday == 7 && h.Hour == 2 {
			cell = h.Count
		}
	}
	if cell != at2 || at2 < 2 {
		t.Errorf("Sunday 02:00 = %d, want %d (both 02:xx hours)", cell, at2)
	}
	rows, err := svc.Diagnoses(t.Context(), c, clinicstats.DiagnosisQuery{From: day, To: day, By: clinicstats.ByMonth})
	if err != nil || len(rows) != 1 || rows[0].Count != diagnoses {
		t.Errorf("25 October's diagnoses: %v, %v; want %d", rows, err, diagnoses)
	}
}
