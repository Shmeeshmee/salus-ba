//go:build e2e

package e2e_test

import (
	"net/http"
	"slices"
	"testing"
	"time"
)

// TestSharing_ThePatientSharesTwoOfThreeVisitsAndTheClinicSeesExactlyThose (SAL-42, docs/plans/patient-history.md):
// a patient with three finished visits at clinic B books at clinic A; the preview lists the three, none ticked, each
// as clinic A would see it; they share two; clinic A's practitioner reads the history consent and finds exactly
// those two; a removal and the revocation take effect at once.
//
// The three earlier visits are booked through the API and finished as the owner: their consultations are the
// journey of consultation_test.go, not this one's. SAL-40's view of the patient's history (GET
// /patients/{id}/history) adds the practitioner's read of the two visits once it is on main.
func TestSharing_ThePatientSharesTwoOfThreeVisitsAndTheClinicSeesExactlyThose(t *testing.T) {
	elsewhere, receiving := newBookableClinic(t), newBookableClinic(t)
	p := newPatientWithProfile(t)
	visits := p.finishedVisits(elsewhere, 3)
	p.bookFirstTime(receiving)

	var preview struct {
		Candidates []struct {
			EncounterID, OrganizationID, OrganizationName string
			Selected                                      bool
		}
	}
	p.must(http.StatusOK, http.MethodGet, "/history-sharing/"+receiving.id, nil, &preview)
	if len(preview.Candidates) != 3 {
		t.Fatalf("the preview = %+v, want the three visits at the other clinic", preview)
	}
	for i, c := range preview.Candidates {
		if c.EncounterID != visits[2-i] || c.OrganizationID != elsewhere.id || c.OrganizationName != elsewhere.name || c.Selected {
			t.Errorf("candidate %d = %+v, want visit %s at %q, newest first, not shared", i, c, visits[2-i], elsewhere.name)
		}
	}

	shared := []string{visits[0], visits[2]}
	var sharing struct {
		ConsentID, VersionID string
		Shared               []string
	}
	p.must(http.StatusOK, http.MethodPut, "/history-sharing/"+receiving.id, map[string][]string{"encounterIds": shared}, &sharing)
	if !slices.Equal(sharing.Shared, shared) || sharing.ConsentID == "" {
		t.Fatalf("PUT /history-sharing = %+v, want the two visits shared", sharing)
	}
	if got := sharedWith(t, receiving, sharing.ConsentID); !slices.Equal(got, shared) {
		t.Errorf("clinic A's practitioner reads the history consent listing %v, want exactly %v", got, shared)
	}
	if got := sharedWith(t, elsewhere, sharing.ConsentID); got != nil {
		t.Errorf("the other clinic reads the history consent listing %v, want it not found", got)
	}

	// A removal: the visit is no longer shared from the commit.
	p.mustWith(http.Header{"If-Match": {`W/"` + sharing.VersionID + `"`}}, http.StatusOK, http.MethodPut, "/history-sharing/"+receiving.id,
		map[string][]string{"encounterIds": {visits[2]}}, &sharing)
	if got := sharedWith(t, receiving, sharing.ConsentID); !slices.Equal(got, []string{visits[2]}) {
		t.Errorf("after the removal clinic A reads %v, want only %s", got, visits[2])
	}
	if status, code := p.refusal(http.MethodPut, "/history-sharing/"+receiving.id, map[string][]string{"encounterIds": shared}); status != http.StatusPreconditionFailed || code != "version_conflict" {
		t.Errorf("a change without If-Match = %d %q, want 412 version_conflict", status, code)
	}

	// The revocation: nothing is shared from the commit.
	p.must(http.StatusNoContent, http.MethodDelete, "/history-sharing/"+receiving.id, nil, nil)
	if got := sharedWith(t, receiving, sharing.ConsentID); got != nil {
		t.Errorf("after the revocation clinic A reads %v, want the history consent not found", got)
	}
	var mine struct {
		Sharing []struct{ OrganizationID string }
	}
	p.must(http.StatusOK, http.MethodGet, "/me/history-sharing", nil, &mine)
	if len(mine.Sharing) != 0 {
		t.Errorf("GET /me/history-sharing after the revocation = %+v, want nothing", mine)
	}
}

// finishedVisits books n appointments at the clinic and finishes each as a visit days ago (as the owner: an
// encounter, the appointment fulfilled), oldest first; it returns their encounters.
func (p *patient) finishedVisits(clinic *bookableClinic, n int) []string {
	p.t.Helper()
	first := p.bookFirstTime(clinic)
	ids := []string{first.ID}
	for _, slot := range freeSlots(p.t, p.api, clinic.scheduleID)[:n-1] {
		var booked struct{ ID string }
		p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slot}, &booked)
		ids = append(ids, booked.ID)
	}
	conn := ownerConn(p.t)
	encounters := make([]string, n)
	for i, id := range ids {
		start := time.Now().Add(-time.Duration(n-i) * 24 * time.Hour).UTC()
		if err := conn.QueryRow(p.t.Context(), `WITH a AS (UPDATE appointment SET status = 'fulfilled' WHERE id = $1
				RETURNING id, organization_id, patient_id, practitioner_role_id)
			INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
			SELECT id, organization_id, patient_id, practitioner_role_id, 'finished', $2, $2::timestamptz + interval '20 minutes' FROM a
			RETURNING id`, id, start).Scan(&encounters[i]); err != nil {
			p.t.Fatal(err)
		}
	}
	return encounters
}

// sharedWith is the clinic's practitioner's FHIR read of the history consent: the encounters it lists, and nil when
// it is not found.
func sharedWith(t *testing.T, clinic *bookableClinic, consentID string) []string {
	t.Helper()
	var c struct {
		Provision struct {
			Data []struct{ Reference struct{ Reference string } }
		}
	}
	switch status := clinic.practitioner.do(http.MethodGet, "/fhir/R4/Consent/"+consentID, nil, &c); status {
	case http.StatusNotFound:
		return nil
	case http.StatusOK:
	default:
		t.Fatalf("GET Consent/%s = %d", consentID, status)
	}
	out := []string{}
	for _, d := range c.Provision.Data {
		out = append(out, d.Reference.Reference[len("Encounter/"):])
	}
	return out
}
