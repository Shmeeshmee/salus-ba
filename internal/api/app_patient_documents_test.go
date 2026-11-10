package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// The patient's documents through api.New (SAL-17), on the compose Postgres, Redis and object store: the patient
// never sees the open draft; once the practitioner signs it, the patient finds it by the encounter, reads its
// Composition and downloads its PDF, also after revoking the consent (their own record), and the signature has
// enqueued their e-mail. The transcript copy is theirs only once the clinic shares it.
func TestNew_ServesThePatientTheirSignedDocumentsAndTheTranscriptCopy(t *testing.T) {
	s := newStreamApp(t)
	created := s.d.practitioner.do(http.MethodPost, "/document-templates", templateBody)
	var tpl struct{ ID string }
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &tpl) != nil {
		t.Fatalf("POST /document-templates = %d %s", created.Code, created.Body)
	}
	rec := s.d.practitioner.do(http.MethodPost, "/encounters/"+s.encounter+"/drafts", `{"templateId":"`+tpl.ID+`"}`)
	var draft struct {
		DraftID string `json:"draft_id"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &draft) != nil {
		t.Fatalf("POST drafts = %d %s", rec.Code, rec.Body)
	}
	search := "/fhir/R4/DocumentReference?encounter=" + s.encounter
	expectHTTP(t, s.d.patient, http.MethodGet, search, http.StatusOK, `"total":0`)
	expectHTTP(t, s.d.patient, http.MethodGet, "/fhir/R4/Composition/"+draft.DraftID, http.StatusNotFound, `"not-found"`)

	s.d.practitioner.ifMatch = `W/"1"`
	rec = s.d.practitioner.do(http.MethodPost, "/drafts/"+draft.DraftID+"/sign", `{"fields":[{"key":"findings","value":"Placeholder findings"},{"key":"tests","value":""}]}`)
	s.d.practitioner.ifMatch = ""
	if rec.Code != http.StatusOK {
		t.Fatalf("sign = %d %s", rec.Code, rec.Body)
	}
	var ref string
	if err := s.owner.QueryRow(t.Context(), `SELECT document_reference_id::text FROM document_draft WHERE id = $1`, draft.DraftID).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	if n := s.count(t, `SELECT count(*) FROM river_job WHERE kind = 'document_signed_email' AND queue = 'email'
		AND args->>'appointment_id' = (SELECT appointment_id::text FROM encounter WHERE id = $1)`, s.encounter); n != 1 {
		t.Errorf("%d document_signed_email jobs, want 1", n)
	}

	for _, revoked := range []bool{false, true} {
		if revoked {
			expectHTTP(t, s.d.patient, http.MethodPost, "/consents/"+s.consent+"/revoke", http.StatusOK)
		}
		expectHTTP(t, s.d.patient, http.MethodGet, search, http.StatusOK, `"total":1`, `"id":"`+ref+`"`, `"Composition/`+draft.DraftID+`"`)
		expectHTTP(t, s.d.patient, http.MethodGet, "/fhir/R4/Composition/"+draft.DraftID, http.StatusOK, `"status":"final"`, "Placeholder findings")
		pdf := s.d.patient.do(http.MethodGet, "/document-references/"+ref+"/content", "")
		if pdf.Code != http.StatusOK || !strings.HasPrefix(pdf.Body.String(), "%PDF-") {
			t.Errorf("revoked %v: the patient's PDF = %d %v", revoked, pdf.Code, pdf.Header())
		}
	}
	expectHTTP(t, s.d.admin, http.MethodGet, search, http.StatusForbidden, `"forbidden"`)
	expectHTTP(t, s.stranger, http.MethodGet, search, http.StatusNotFound)
	if n := s.count(t, `SELECT count(*) FROM audit_event WHERE entity_reference = $1 AND outcome = 0 AND purpose_code = 'PATRQT'`, "DocumentReference/"+ref); n != 2 {
		t.Errorf("audited %d of the patient's PDF reads, want 2", n)
	}

	if _, err := s.owner.Exec(t.Context(), `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, ts_start, ts_end)
		VALUES (gen_random_uuid(), $1, 'patient', 'bs', 'Placeholder original', now(), now())`, s.encounter); err != nil {
		t.Fatal(err)
	}
	transcriptPath := "/encounters/" + s.encounter + "/transcript"
	expectHTTP(t, s.d.patient, http.MethodGet, transcriptPath, http.StatusForbidden, `"forbidden"`)
	if _, err := s.owner.Exec(t.Context(), `UPDATE organization SET patient_transcript_copy = true
		WHERE id = (SELECT organization_id FROM encounter WHERE id = $1)`, s.encounter); err != nil {
		t.Fatal(err)
	}
	expectHTTP(t, s.d.patient, http.MethodGet, transcriptPath, http.StatusOK, "Placeholder original")
}
