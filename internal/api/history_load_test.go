//go:build loadtest

package api_test

import (
	"net/http"
	"slices"
	"testing"
	"time"
)

// The readings of the practitioner's view of the patient's history (SAL-40, docs/plans/patient-history.PROGRESS.md;
// make bench-history): through api.New on the compose Postgres, a patient with 300 finished visits with the
// practitioner, each with a signed document, 100 visits at another clinic shared with theirs, each with one, and 500
// more visits there not shared, which the clinic's scope reads through history_shared and the rule leaves out. The
// first page of 30, every page to the last, and the pre-visit summary, each as p50, p95 and p99. Not part of make test.
func TestHistoryReadings(t *testing.T) {
	c := newConsultationApp(t)
	h := seedPatientHistory(t, c)
	ctx := t.Context()
	bulk := func(n int, clinic, role string, shared bool, hour int) {
		t.Helper()
		if _, err := c.owner.Exec(ctx, `WITH a AS (
				INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
				SELECT $1, $2, $3, gen_random_uuid(), tstzrange(t, t + interval '30 minutes'), 'fulfilled'
				FROM generate_series(1, $4) AS i, LATERAL (SELECT timestamptz '2020-01-01' + $7 * interval '1 hour' + i * interval '1 day' AS t) AS s
				RETURNING id, patient_id, practitioner_role_id, organization_id, lower(period) AS start),
			e AS (INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
				SELECT id, organization_id, patient_id, practitioner_role_id, 'finished', start, start + interval '20 minutes' FROM a
				RETURNING id, organization_id, practitioner_role_id, period_end),
			d AS (INSERT INTO document_draft (encounter_id, template_id, status, title, signed_at, signer_role_id, document_reference_id,
					provenance_id, pdf_size, pdf_kek_id, pdf_sha1, pdf_wrapped_key, fields)
				SELECT e.id, t.id, 'final', 'Placeholder report', e.period_end, e.practitioner_role_id, gen_random_uuid(), gen_random_uuid(),
					1, 'kek-test', decode(repeat('00', 20), 'hex'), '\x00'::bytea,
					'[{"key":"diagnosis","type":"coded","label":"Placeholder label","value":"J06.9","missing":false,"required":false}]'
				FROM e JOIN document_template t ON t.organization_id = e.organization_id)
			INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at)
			SELECT $5, e.id, now() FROM e WHERE $6::boolean`, c.patientID, role, clinic, n, h.consent, shared, hour); err != nil {
			t.Fatal(err)
		}
	}
	other := "d0000000-0000-4000-8000-00000000000d"
	bulk(300, inviteClinic, c.d.roleID, false, 0)
	bulk(100, other, h.otherClinicRoleID, true, 2)
	bulk(500, other, h.otherClinicRoleID, false, 4)
	if _, err := c.owner.Exec(ctx, "ANALYZE"); err != nil {
		t.Fatal(err)
	}
	path := "/patients/" + c.patientID + "/history"

	timed := func(name string, n int, get func() string) {
		t.Helper()
		var took []time.Duration
		for range n {
			start := time.Now()
			if body := get(); body == "" {
				t.Fatalf("%s: an empty answer", name)
			}
			took = append(took, time.Since(start))
		}
		slices.Sort(took)
		at := func(p float64) time.Duration { return took[int(p*float64(len(took)-1))] }
		t.Logf("%s (%d runs): p50 %v, p95 %v, p99 %v", name, n, at(0.50).Round(time.Microsecond), at(0.95).Round(time.Microsecond), at(0.99).Round(time.Microsecond))
	}
	get := func(target string) string {
		t.Helper()
		rec := c.d.practitioner.do(http.MethodGet, target, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d %s", target, rec.Code, rec.Body)
		}
		return rec.Body.String()
	}
	timed("the first page of 30", 200, func() string { return get(path) })
	timed("the summary", 200, func() string { return get(path + "/summary?appointment=" + h.appointment) })

	var pages []time.Duration
	items, next := 0, ""
	for {
		var p historyPage
		start := time.Now()
		decode(t, c.d.practitioner.do(http.MethodGet, path+"?limit=100&cursor="+next, ""), &p)
		pages = append(pages, time.Since(start))
		items += len(p.Items)
		if p.Next == "" {
			break
		}
		next = p.Next
	}
	if want := 2*(300+100) + 4; items != want {
		t.Errorf("%d items through every page, want %d", items, want)
	}
	t.Logf("every page of 100 to the last: %d pages, %d items, the last %v, the slowest %v", len(pages), items,
		pages[len(pages)-1].Round(time.Microsecond), slices.Max(pages).Round(time.Microsecond))
}
