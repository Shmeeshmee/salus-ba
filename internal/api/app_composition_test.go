package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// A placeholder template: findings (required) and a list of tests, with an English translation. Synthetic.
const templateBody = `{"title":"Placeholder findings report","language":"bs","status":"draft","typeCode":"11488-4",
	"fields":[{"key":"findings","type":"text","label":"Placeholder findings","instruction":"Placeholder instruction","required":true},
	{"key":"tests","type":"list","label":"Placeholder tests","required":false}],
	"translations":[{"language":"en","title":"Placeholder report","fields":{"findings":{"label":"Placeholder findings","instruction":"Placeholder instruction"},
	"tests":{"label":"Placeholder tests"}}}]}`

// The documents of a consultation through api.New (SAL-8), on the compose Postgres, Redis and object store: the
// practitioner keeps a template, drafts it from the stored transcript (the fake model leaves every field
// missing), the draft is pushed on the encounter's stream, they sign it, and its Composition, DocumentReference,
// Provenance and PDF are read back; nobody else drafts, and every write and read is audited.
func TestNew_DraftsSignsAndServesTheEncountersDocuments(t *testing.T) {
	s := newStreamApp(t)
	created := s.d.practitioner.do(http.MethodPost, "/document-templates", templateBody)
	var tpl struct{ ID string }
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &tpl) != nil {
		t.Fatalf("POST /document-templates = %d %s", created.Code, created.Body)
	}
	expectHTTP(t, s.d.admin, http.MethodGet, "/document-templates", http.StatusOK, tpl.ID)
	if _, err := s.owner.Exec(t.Context(), `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, ts_start, ts_end)
		VALUES (gen_random_uuid(), $1, 'patient', 'bs', 'Placeholder original', now(), now())`, s.encounter); err != nil {
		t.Fatal(err)
	}

	path := "/encounters/" + s.encounter + "/drafts"
	body := `{"templateId":"` + tpl.ID + `"}`
	rec := s.d.practitioner.do(http.MethodPost, path, body)
	var draft composition.Event
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &draft) != nil || draft.Status != composition.StatusPreliminary ||
		len(draft.Fields) != 2 || !draft.Fields[0].Missing || draft.VersionID != 1 {
		t.Fatalf("POST %s = %d %s", path, rec.Code, rec.Body)
	}
	if env := s.lastEnvelope(t); env.Type != composition.MessageType || !strings.Contains(string(env.Payload), draft.DraftID) {
		t.Errorf("the stream's last entry = %+v, want the draft", env)
	}
	for _, cl := range []*client{s.d.admin, s.d.patient} {
		if rec := cl.do(http.MethodPost, path, body); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), `"forbidden"`) {
			t.Errorf("POST %s by another = %d %s, want 403 forbidden", path, rec.Code, rec.Body)
		}
	}
	expectHTTP(t, s.d.practitioner, http.MethodGet, path, http.StatusOK, draft.DraftID)

	s.d.practitioner.ifMatch = `W/"1"`
	rec = s.d.practitioner.do(http.MethodPost, "/drafts/"+draft.DraftID+"/sign",
		`{"fields":[{"key":"findings","value":"Placeholder findings"},{"key":"tests","value":"Placeholder one\nPlaceholder two"}]}`)
	s.d.practitioner.ifMatch = ""
	var signed composition.Event
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &signed) != nil || signed.Status != composition.StatusFinal || signed.VersionID != 2 {
		t.Fatalf("sign = %d %s", rec.Code, rec.Body)
	}
	comp := expectHTTP(t, s.d.practitioner, http.MethodGet, "/fhir/R4/Composition/"+draft.DraftID, http.StatusOK, `"status":"final"`, `"mode":"legal"`)
	var ids struct{ DocumentReference, Provenance string }
	if err := s.owner.QueryRow(t.Context(), `SELECT document_reference_id::text, provenance_id::text FROM document_draft WHERE id = $1`,
		draft.DraftID).Scan(&ids.DocumentReference, &ids.Provenance); err != nil {
		t.Fatalf("%v (Composition %s)", err, comp)
	}
	expectHTTP(t, s.d.practitioner, http.MethodGet, "/fhir/R4/DocumentReference/"+ids.DocumentReference, http.StatusOK, `"contentType":"application/pdf"`)
	expectHTTP(t, s.d.practitioner, http.MethodGet, "/fhir/R4/Provenance/"+ids.Provenance, http.StatusOK, `"Composition/`+draft.DraftID+`/_history/2"`)
	pdf := s.d.practitioner.do(http.MethodGet, "/document-references/"+ids.DocumentReference+"/content", "")
	if pdf.Code != http.StatusOK || !strings.HasPrefix(pdf.Body.String(), "%PDF-") || pdf.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("GET content = %d %v", pdf.Code, pdf.Header())
	}
	expectHTTP(t, s.d.admin, http.MethodGet, "/document-references/"+ids.DocumentReference+"/content", http.StatusForbidden)

	for ref, want := range map[string]int{
		"Composition/" + draft.DraftID:               3, // created, updated by the sign, read
		"DocumentReference/" + ids.DocumentReference: 3, // created, read through FHIR, read as its PDF
		"Provenance/" + ids.Provenance:               2,
	} {
		if n := s.count(t, `SELECT count(*) FROM audit_event WHERE entity_reference = $1 AND outcome = 0`, ref); n != want {
			t.Errorf("audited %d successes on %s, want %d", n, ref, want)
		}
	}
}

// lastEnvelope is the encounter's stream's last entry.
func (s streamApp) lastEnvelope(t *testing.T) wire.Envelope {
	t.Helper()
	entries, err := s.rdb.XRevRangeN(t.Context(), transcript.StreamKey(s.encounter), "+", "-", 1).Result()
	if err != nil || len(entries) != 1 {
		t.Fatalf("XREVRANGE = %v, %v", entries, err)
	}
	raw, _ := entries[0].Values[transcript.StreamField].(string)
	env, err := wire.Decode([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return env
}
