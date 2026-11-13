//go:build e2e

package e2e_test

import (
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"
)

// TestHistory_VisitsAtTwoClinicsAppearNewestFirstAndNeverADraft (SAL-39, docs/plans/patient-history.md): a patient
// with a visit at each of two clinics sees both in their timeline, newest first, each with its clinic's name; the
// document signed in the first visit is there and the draft left open in the second is not; the clinic filter
// gives only that clinic's items.
func TestHistory_VisitsAtTwoClinicsAppearNewestFirstAndNeverADraft(t *testing.T) {
	first, second := newBookableClinic(t), newBookableClinic(t)
	p := newPatientWithProfile(t)
	signed := visitWithDraft(t, p, first, true)
	open := visitWithDraft(t, p, second, false)

	var page struct {
		Next  string
		Items []struct {
			Date                   time.Time
			Clinic                 struct{ ID, Name string }
			Type, Reference, Title string
			EncounterID            string `json:"encounterId"`
		}
	}
	p.must(http.StatusOK, http.MethodGet, "/me/history?limit=100", nil, &page)
	var refs []string
	for i, it := range page.Items {
		refs = append(refs, it.Reference)
		if i > 0 && it.Date.After(page.Items[i-1].Date) {
			t.Errorf("item %d (%s) is newer than the one before it: want newest first", i, it.Reference)
		}
		if want := map[string]string{first.id: first.name, second.id: second.name}[it.Clinic.ID]; it.Clinic.Name != want || want == "" {
			t.Errorf("item %s is at clinic %+v, want one of the two clinics by name", it.Reference, it.Clinic)
		}
	}
	visitFirst, visitSecond := "Encounter/"+signed.encounter, "Encounter/"+open.encounter
	if !slices.Contains(refs, visitFirst) || !slices.Contains(refs, visitSecond) || slices.Index(refs, visitSecond) > slices.Index(refs, visitFirst) {
		t.Errorf("the timeline is %v, want both visits, the second clinic's (newer) first", refs)
	}
	if !slices.Contains(refs, "Composition/"+signed.draft) {
		t.Errorf("the timeline is %v, want the signed document Composition/%s", refs, signed.draft)
	}
	if slices.Contains(refs, "Composition/"+open.draft) {
		t.Errorf("the timeline holds the open draft Composition/%s", open.draft)
	}
	var only struct {
		Items []struct{ Clinic struct{ ID string } }
	}
	p.must(http.StatusOK, http.MethodGet, "/me/history?organization="+second.id, nil, &only)
	for _, it := range only.Items {
		if it.Clinic.ID != second.id {
			t.Errorf("the second clinic's timeline holds an item of %s", it.Clinic.ID)
		}
	}
	if len(only.Items) == 0 {
		t.Error("the second clinic's timeline is empty, want its visit")
	}
	if status, code := first.practitioner.refusal(http.MethodGet, "/me/history", nil); status != http.StatusForbidden || code != "forbidden" {
		t.Errorf("a practitioner's GET /me/history = %d %q, want 403 forbidden", status, code)
	}
}

// visited is a visit's encounter and the draft written in it.
type visited struct{ encounter, draft string }

// visitWithDraft books the clinic, holds the consultation (the practitioner joins and ends it), drafts the clinic's
// template in it and signs the draft when sign is set.
func visitWithDraft(t *testing.T, p *patient, clinic *bookableClinic, sign bool) visited {
	t.Helper()
	appt := p.bookFirstTime(clinic)
	startAt(t, appt.ID, -time.Minute)
	recordSynthetic(t, clinic, p, appt.ID)
	p.giveTranscriptionConsent(clinic) // drafting needs it (SAL-12)
	var template struct{ ID string }
	clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/document-templates", map[string]any{
		"title": "Placeholder e2e history report " + suffix(t), "language": "en", "status": "draft", "typeCode": "11488-4",
		"fields": []map[string]any{{"key": "reason", "type": "text", "label": "Placeholder reason", "required": true}},
	}, &template)
	practitioner := connect(t, clinic.practitioner, appt.ID, false)
	eventually(t, "the Encounter in progress once the practitioner joined", func() bool {
		return encounterOf(t, p, appt.ID).Status == "in-progress"
	})
	v := visited{encounter: encounterOf(t, p, appt.ID).ID}
	var d draftEvent
	clinic.practitioner.must(http.StatusOK, http.MethodPost, "/encounters/"+v.encounter+"/drafts", map[string]string{"templateId": template.ID, "language": "en"}, &d)
	v.draft = d.DraftID
	if sign {
		values := []map[string]string{{"key": "reason", "value": "Placeholder reason"}}
		clinic.practitioner.mustWith(http.Header{"If-Match": {`W/"` + strconv.Itoa(d.VersionID) + `"`}}, http.StatusOK, http.MethodPost,
			"/drafts/"+d.DraftID+"/sign", map[string]any{"fields": values}, nil)
	}
	clinic.practitioner.must(http.StatusOK, http.MethodPost, "/appointments/"+appt.ID+"/consultation/end", nil, nil)
	practitioner.waitGone(t)
	return v
}
