package api_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

// historySeed is, as the owner, a finished visit of the patient with the practitioner at their clinic with a signed
// document holding a diagnosis, and two finished visits at another clinic with a signed document each, the first
// shared with the practitioner's clinic through an active history consent.
type historySeed struct {
	appointment, own, ownDoc   string
	shared, sharedDoc          string
	unshared, unsharedDoc      string
	consent, otherClinicRoleID string
}

func seedPatientHistory(t *testing.T, c consultationApp) historySeed {
	t.Helper()
	ctx := t.Context()
	var h historySeed
	h.appointment = c.book(t, wallNow().Add(-48*time.Hour))
	q := func(dst *string, sql string, args ...any) {
		t.Helper()
		if err := c.owner.QueryRow(ctx, sql, args...).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	q(&h.own, `INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
		SELECT id, organization_id, patient_id, practitioner_role_id, 'finished', lower(period), lower(period) + interval '20 minutes'
		FROM appointment WHERE id = $1 RETURNING id::text`, h.appointment)
	q(&h.otherClinicRoleID, `WITH o AS (INSERT INTO organization (id, name) VALUES ('d0000000-0000-4000-8000-00000000000d', 'Synthetic clinic D') RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Di', 'Example' FROM o RETURNING id, organization_id)
		INSERT INTO practitioner_role (organization_id, practitioner_id, specialty_code) SELECT organization_id, id, '419192003' FROM p RETURNING id::text`)
	visit := func(at string) string {
		var id string
		q(&id, `WITH a AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
				VALUES ($1, $2, 'd0000000-0000-4000-8000-00000000000d', gen_random_uuid(), tstzrange($3::timestamptz, $3::timestamptz + interval '30 minutes'), 'fulfilled')
				RETURNING id, patient_id, practitioner_role_id, organization_id, lower(period) AS start)
			INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
			SELECT id, organization_id, patient_id, practitioner_role_id, 'finished', start, start + interval '20 minutes' FROM a RETURNING id::text`,
			c.patientID, h.otherClinicRoleID, at)
		return id
	}
	h.shared, h.unshared = visit("2026-09-01T09:00:00Z"), visit("2026-08-01T09:00:00Z")
	for _, clinic := range []string{inviteClinic, "d0000000-0000-4000-8000-00000000000d"} {
		if _, err := c.owner.Exec(ctx, `INSERT INTO document_template (organization_id, title, fields)
			VALUES ($1, 'Placeholder template', '[{"key":"diagnosis","type":"coded","label":"Placeholder label"}]')`, clinic); err != nil {
			t.Fatal(err)
		}
	}
	doc := func(encounter, code string) string {
		var id string
		q(&id, `INSERT INTO document_draft (encounter_id, template_id, status, title, signed_at, signer_role_id, document_reference_id,
				provenance_id, pdf_size, pdf_kek_id, pdf_sha1, pdf_wrapped_key, fields)
			SELECT e.id, t.id, 'final', 'Placeholder report', e.period_end, e.practitioner_role_id, gen_random_uuid(), gen_random_uuid(),
			       1, 'kek-test', decode(repeat('00', 20), 'hex'), '\x00'::bytea,
			       jsonb_build_array(jsonb_build_object('key', 'diagnosis', 'type', 'coded', 'label', 'Placeholder label', 'value', $2::text, 'missing', false, 'required', false))
			FROM encounter e JOIN document_template t ON t.organization_id = e.organization_id WHERE e.id = $1 RETURNING id::text`, encounter, code)
		return id
	}
	h.ownDoc, h.sharedDoc, h.unsharedDoc = doc(h.own, "J06.9"), doc(h.shared, "K21.0"), doc(h.unshared, "F32.1")
	// The own document's diagnosis as 00042 backfilled one signed before SAL-19, unconfirmed; the shared one's
	// confirmed at its signing.
	for doc, status := range map[string]string{h.ownDoc: "unconfirmed", h.sharedDoc: "confirmed"} {
		if _, err := c.owner.Exec(ctx, `INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
				practitioner_role_id, field_key, code, verification_status)
			SELECT d.signed_at, e.organization_id, e.patient_id, e.id, d.id, d.signer_role_id, 'diagnosis', d.fields->0->>'value', $2
			FROM document_draft d JOIN encounter e ON e.id = d.encounter_id WHERE d.id = $1`, doc, status); err != nil {
			t.Fatal(err)
		}
	}
	q(&h.consent, `INSERT INTO history_consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now()) RETURNING id::text`,
		c.patientID, inviteClinic)
	if _, err := c.owner.Exec(ctx, `INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at) VALUES ($1, $2, now())`, h.consent, h.shared); err != nil {
		t.Fatal(err)
	}
	return h
}

type historyPage struct {
	Next  string `json:"next"`
	Items []struct {
		Clinic struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"clinic"`
		Type        string `json:"type"`
		Reference   string `json:"reference"`
		EncounterID string `json:"encounterId"`
		Specialty   string `json:"specialty"`
		Diagnoses   []struct {
			Code      string `json:"code"`
			Label     string `json:"label"`
			Confirmed bool   `json:"confirmed"`
		} `json:"diagnoses"`
		Shared bool `json:"shared"`
	} `json:"items"`
}

func (p historyPage) refs() []string {
	var out []string
	for _, it := range p.Items {
		out = append(out, it.Reference)
	}
	return out
}

// SAL-40 through api.New on the compose Postgres: the practitioner's view of the patient's history is their own
// visit and the other clinic's visit the patient shared, each with its signed document and diagnoses, the shared
// one marked with its clinic; the unshared visit is nowhere in it. The shared document opens for them. Nobody else
// has a history of the patient. Every view is audited, and once the patient stops sharing, the visit and its
// document are gone at once.
func TestNew_ServesThePractitionerThePatientsHistory(t *testing.T) {
	c := newConsultationApp(t)
	h := seedPatientHistory(t, c)
	path := "/patients/" + c.patientID + "/history"

	var page historyPage
	decode(t, c.d.practitioner.do(http.MethodGet, path, ""), &page)
	want := []string{"Composition/" + h.ownDoc, "Encounter/" + h.own, "Composition/" + h.sharedDoc, "Encounter/" + h.shared}
	if got := page.refs(); !slices.Equal(got, want) {
		t.Fatalf("history = %v, want %v", got, want)
	}
	for _, it := range page.Items {
		shared := it.EncounterID == h.shared
		if it.Shared != shared || (shared && (it.Clinic.Name != "Synthetic clinic D" || it.Specialty != "419192003")) {
			t.Errorf("item %+v: shared %v", it, shared)
		}
	}
	// The backfilled diagnosis reads as not confirmed; the shared one as confirmed, through history_confirmed_conditions.
	own, shared := page.Items[0].Diagnoses, page.Items[2].Diagnoses
	if len(own) != 1 || own[0].Code != "J06.9" || own[0].Label != "Placeholder label" || own[0].Confirmed ||
		len(shared) != 1 || shared[0].Code != "K21.0" || !shared[0].Confirmed {
		t.Errorf("diagnoses %+v and %+v, want J06.9 unconfirmed and K21.0 confirmed", own, shared)
	}
	body := c.d.practitioner.do(http.MethodGet, path, "").Body.String()
	if strings.Contains(body, h.unshared) || strings.Contains(body, h.unsharedDoc) || strings.Contains(body, "F32.1") {
		t.Errorf("the unshared visit is in the history: %s", body)
	}
	expectHTTP(t, c.d.practitioner, http.MethodGet, "/fhir/R4/Composition/"+h.sharedDoc, http.StatusOK, `"status":"final"`)
	expectHTTP(t, c.d.practitioner, http.MethodGet, "/fhir/R4/Composition/"+h.unsharedDoc, http.StatusNotFound)

	for _, cl := range []*client{c.d.admin, c.stranger, c.d.patient} {
		expectHTTP(t, cl, http.MethodGet, path, http.StatusNotFound, `"not_found"`)
		expectHTTP(t, cl, http.MethodGet, path+"/summary?appointment="+h.appointment, http.StatusNotFound, `"not_found"`)
	}

	var views, shown int
	if err := c.owner.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE entity_reference = $1 AND subtype_code = 'search-type' AND outcome = 0),
		count(*) FILTER (WHERE entity_reference = $2 AND outcome = 0) FROM audit_event`, "Patient/"+c.patientID, "Encounter/"+h.shared).Scan(&views, &shown); err != nil {
		t.Fatal(err)
	}
	if views != 2 || shown != 2 {
		t.Errorf("audited %d views and %d showings of the shared visit, want 2 and 2", views, shown)
	}

	expectHTTP(t, c.d.practitioner, http.MethodGet, path+"/summary?appointment="+h.appointment, http.StatusOK,
		`"lastVisit":{`, `"reference":"Encounter/`+h.own+`"`, `"allergies":[]`, `"medications":[]`)
	// Until SAL-41 is wired the stand-ins fail closed: both sections unavailable, never "none recorded". One
	// string per call: expectHTTP negates the status once per missing string, so two missing would cancel out.
	for _, want := range []string{`"allergiesUnavailable":true`, `"medicationsUnavailable":true`} {
		expectHTTP(t, c.d.practitioner, http.MethodGet, path+"/summary?appointment="+h.appointment, http.StatusOK, want)
	}

	if _, err := c.owner.Exec(t.Context(), `UPDATE history_consent_encounter SET removed_at = now() WHERE consent_id = $1`, h.consent); err != nil {
		t.Fatal(err)
	}
	page = historyPage{}
	decode(t, c.d.practitioner.do(http.MethodGet, path, ""), &page)
	if got := page.refs(); !slices.Equal(got, want[:2]) {
		t.Errorf("history after the removal = %v, want the own visit only", got)
	}
	expectHTTP(t, c.d.practitioner, http.MethodGet, "/fhir/R4/Composition/"+h.sharedDoc, http.StatusNotFound)
}

// A page of the history holds at most limit items and its next cursor continues it.
func TestNew_PagesThePatientsHistory(t *testing.T) {
	c := newConsultationApp(t)
	h := seedPatientHistory(t, c)
	path := "/patients/" + c.patientID + "/history"
	var first, second historyPage
	decode(t, c.d.practitioner.do(http.MethodGet, path+"?limit=3", ""), &first)
	if len(first.Items) != 3 || first.Next == "" {
		t.Fatalf("first page %v, next %q", first.refs(), first.Next)
	}
	decode(t, c.d.practitioner.do(http.MethodGet, path+"?limit=3&cursor="+first.Next, ""), &second)
	if got := second.refs(); !slices.Equal(got, []string{"Encounter/" + h.shared}) || second.Next != "" {
		t.Errorf("second page %v, next %q", got, second.Next)
	}
	expectHTTP(t, c.d.practitioner, http.MethodGet, path+"?cursor=nope", http.StatusBadRequest, `"invalid_request"`)
}
