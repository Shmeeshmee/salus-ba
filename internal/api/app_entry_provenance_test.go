package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// A practitioner's write of an allergy or a medication through api.New (SAL-41) writes its Provenance, served by
// GET /fhir/R4/Provenance/{id} beside the signatures' (one route, several sources): to the practitioner and to the
// patient; the clinic's admin, who does not see the entry, gets not found. The creation and each read are audited.
func TestNew_ServesTheProvenanceOfAClinicalWrite(t *testing.T) {
	s := newStreamApp(t)
	for _, tt := range []struct{ path, body, table, column, typ string }{
		{"/encounters/" + s.encounter + "/allergies", `{"text":"Placeholder substance","clinicalStatus":"active"}`,
			"allergy_intolerance_provenance", "allergy_intolerance_id", "AllergyIntolerance"},
		{"/encounters/" + s.encounter + "/medications", `{"text":"Placeholder medication","status":"active","dosage":{}}`,
			"medication_statement_provenance", "medication_statement_id", "MedicationStatement"},
	} {
		rec := s.d.practitioner.do(http.MethodPost, tt.path, tt.body)
		var written struct{ ID string }
		if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &written) != nil || written.ID == "" {
			t.Fatalf("POST %s = %d %s", tt.path, rec.Code, rec.Body)
		}
		var id string
		if err := s.owner.QueryRow(t.Context(), `SELECT id::text FROM `+tt.table+` WHERE `+tt.column+` = $1`, written.ID).Scan(&id); err != nil {
			t.Fatalf("%s: the Provenance: %v", tt.typ, err)
		}
		path := "/fhir/R4/Provenance/" + id
		target := `"` + tt.typ + `/` + written.ID + `/_history/1"`
		expectHTTP(t, s.d.practitioner, http.MethodGet, path, http.StatusOK, target, `"code":"CREATE"`)
		expectHTTP(t, s.d.patient, http.MethodGet, path, http.StatusOK, target)
		expectHTTP(t, s.d.admin, http.MethodGet, path, http.StatusNotFound, `"not-found"`)
		if n := s.count(t, `SELECT count(*) FROM audit_event WHERE entity_reference = $1 AND outcome = 0`, "Provenance/"+id); n != 3 {
			t.Errorf("%s: audited %d successes on Provenance/%s, want 3 (created, read twice)", tt.typ, n, id)
		}
	}
	expectHTTP(t, s.d.practitioner, http.MethodGet, "/fhir/R4/Provenance/9e000000-0000-4000-8000-0000000000ff", http.StatusNotFound, `"not-found"`)
}
