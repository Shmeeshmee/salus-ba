//go:build e2e

package e2e_test

import (
	"net/http"
	"slices"
	"testing"
)

// practitionerSeesThePatientsHistory is the practitioner's view of the patient's history after the consultation
// (SAL-40, ADR 0014), as the web app reads it: the visit they held and the document they signed in it, with the
// diagnosis they confirmed; the pre-visit summary of the appointment with that visit as the last one. Each view is
// an AuditEvent of the practitioner's (read as the owner, DATABASE_URL). The patient and a stranger have no such view.
func practitionerSeesThePatientsHistory(t *testing.T, clinic *bookableClinic, p *patient, appointmentID, code string) {
	t.Helper()
	enc := encounterOf(t, p, appointmentID).ID
	path := "/patients/" + p.patientID + "/history"
	var page struct {
		Items []struct {
			Type, Reference, EncounterID string
			Diagnoses                    []string
			Shared                       bool
		}
	}
	clinic.practitioner.must(http.StatusOK, http.MethodGet, path, nil, &page)
	var refs []string
	diagnosed := false
	for _, it := range page.Items {
		refs = append(refs, it.Reference)
		if it.EncounterID != enc || it.Shared {
			t.Errorf("history item %+v, want one of the visit %s, not shared", it, enc)
		}
		diagnosed = diagnosed || (it.Type == "document" && slices.Equal(it.Diagnoses, []string{code}))
	}
	if !slices.Contains(refs, "Encounter/"+enc) || !diagnosed {
		t.Errorf("the practitioner's history of the patient = %v, want the visit and its signed document with %s", refs, code)
	}
	var summary struct {
		LastVisit *struct{ Reference string }
		OpenItems []struct{ Type string }
	}
	clinic.practitioner.must(http.StatusOK, http.MethodGet, path+"/summary?appointment="+appointmentID, nil, &summary)
	if summary.LastVisit == nil || summary.LastVisit.Reference != "Encounter/"+enc {
		t.Errorf("the summary's last visit = %+v, want the visit %s", summary.LastVisit, enc)
	}
	if status, got := p.refusal(http.MethodGet, path, nil); status != http.StatusNotFound || got != "not_found" {
		t.Errorf("the patient's own GET of the practitioner's view = %d %q, want 404 not_found", status, got)
	}
	var views int
	if err := ownerConn(t).QueryRow(t.Context(), `SELECT count(*) FROM audit_event
		WHERE entity_reference = $1 AND outcome = 0 AND organization_id = $2 AND purpose_code = 'TREAT'`,
		"Patient/"+p.patientID, clinic.id).Scan(&views); err != nil || views < 2 {
		t.Errorf("audited %d views of the patient's history and summary (%v), want both", views, err)
	}
}
