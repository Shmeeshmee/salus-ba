package medicationstatement_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// provenanceOf is what a practitioner's write of the entry id at version writes: their role on behalf of clinic A.
func provenanceOf(id string, version int32, activity, agent string) medicationstatement.Provenance {
	return medicationstatement.Provenance{Recorded: now, EntryID: id, RoleID: roleA, OrganizationID: clinicA,
		Activity: activity, AgentType: agent, Version: version}
}

// written is the repository's Provenances without their random ids.
func written(r *fakeRepo) []medicationstatement.Provenance {
	out := make([]medicationstatement.Provenance, len(r.provenances))
	for i, p := range r.provenances {
		p.ID = ""
		out[i] = p
	}
	return out
}

// Every clinical write writes a Provenance of the version it produced, in the write's transaction, and audits it:
// recording (create, the author), a change (update, the author), a confirmation (update, the verifier), a removal
// (nullify: entered-in-error).
func TestClinicalWrites_WriteTheirProvenance(t *testing.T) {
	t.Parallel()
	w := newWorld(history(), sharing())
	a, err := w.svc.Record(t.Context(), practitioner, encA, input())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.Update(t.Context(), practitioner, idVisitA, 1, input()); err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.Confirm(t.Context(), practitioner, idOwn, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.svc.Remove(t.Context(), practitioner, idVisitA, 2); err != nil {
		t.Fatal(err)
	}
	want := []medicationstatement.Provenance{
		provenanceOf(a.ID, 1, fhir.ActivityCreate, fhir.AgentAuthor),
		provenanceOf(idVisitA, 2, fhir.ActivityUpdate, fhir.AgentAuthor),
		provenanceOf(idOwn, 2, fhir.ActivityUpdate, fhir.AgentVerifier),
		provenanceOf(idVisitA, 3, fhir.ActivityNullify, fhir.AgentAuthor),
	}
	if got := written(w.repo); !slices.Equal(got, want) {
		t.Errorf("the Provenances:\n got %+v\nwant %+v", got, want)
	}
	for _, p := range w.repo.provenances {
		if !fhir.IsUUID(p.ID) {
			t.Errorf("a Provenance id %q, want a UUID", p.ID)
		}
		if !slices.ContainsFunc(w.audit.events, func(e auditevent.AuditEvent) bool {
			return e.EntityReference == "Provenance/"+p.ID && e.Action == auditevent.ActionCreate && e.Outcome == auditevent.OutcomeSuccess &&
				e.OrganizationID == clinicA && e.Purpose.Code == auditevent.CodeTREAT
		}) {
			t.Errorf("the creation of Provenance/%s is not audited: %v", p.ID, w.audit.refs())
		}
	}
}

// The patient's own writes are patient-reported (the entry names them as asserter, the AuditEvent records the
// write): no Provenance. Neither does a refused write, nor a confirmation that changes nothing.
func TestWritesWithoutProvenance(t *testing.T) {
	t.Parallel()
	w := newWorld(history(), sharing())
	if _, err := w.svc.Add(t.Context(), patient, input()); err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.Update(t.Context(), patient, idOwnByB, 1, input()); err != nil {
		t.Fatal(err)
	}
	if err := w.svc.Remove(t.Context(), patient, idVisitB, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.AddToProfile(t.Context(), patient, idVisitA, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.Confirm(t.Context(), practitioner, idVisitA, 2); err != nil { // confirmed by clinic A already
		t.Fatal(err)
	}
	if _, err := w.svc.Update(t.Context(), practitioner, idShared, 1, input()); !errors.Is(err, medicationstatement.ErrForbidden) {
		t.Fatalf("a shared visit's entry: %v, want ErrForbidden", err)
	}
	if len(w.repo.provenances) != 0 {
		t.Errorf("Provenances %+v, want none", w.repo.provenances)
	}
}

// A Provenance names the clinic that wrote, so it is read only by the patient and by staff of that clinic who see
// the entry (access.Attributed), audited as a read of the Provenance; a refusal is not found, audited, naming
// nothing. An id that is no medication's Provenance is not this service's (ok false, nothing audited): another
// resource's Provenance may have it.
func TestProvenance_OnlyThePatientAndTheWritingClinicRead(t *testing.T) {
	t.Parallel()
	repo := history()
	const byA, byB, ofRemoved, ofColleague = "9a000000-0000-4000-8000-000000000001", "9a000000-0000-4000-8000-000000000002",
		"9a000000-0000-4000-8000-000000000003", "9a000000-0000-4000-8000-000000000004"
	repo.provenances = []medicationstatement.Provenance{
		{ID: byA, EntryID: idVisitA, RoleID: roleA, OrganizationID: clinicA, Activity: fhir.ActivityCreate, AgentType: fhir.AgentAuthor, Version: 1},
		{ID: byB, EntryID: idOwnByB, RoleID: roleB, OrganizationID: clinicB, Activity: fhir.ActivityUpdate, AgentType: fhir.AgentVerifier, Version: 1},
		{ID: ofRemoved, EntryID: idRemoved, RoleID: roleA, OrganizationID: clinicA, Activity: fhir.ActivityUpdate, AgentType: fhir.AgentVerifier, Version: 1},
		{ID: ofColleague, EntryID: idColleague, RoleID: roleC, OrganizationID: clinicA, Activity: fhir.ActivityCreate, AgentType: fhir.AgentAuthor, Version: 1},
	}
	w := newWorld(repo, sharing())
	for _, tt := range []struct {
		want error
		name string
		id   string
		c    medicationstatement.Caller
	}{
		{nil, "the patient, clinic A's", byA, patient},
		{nil, "the patient, clinic B's", byB, patient},
		{nil, "the practitioner, their clinic's", byA, practitioner},
		{medicationstatement.ErrNotFound, "the practitioner, another clinic's on a profile entry", byB, practitioner},
		{medicationstatement.ErrNotFound, "the practitioner, a removed entry's", ofRemoved, practitioner},
		{medicationstatement.ErrNotFound, "the practitioner, a colleague's visit's", ofColleague, practitioner},
		{medicationstatement.ErrNotFound, "the colleague, the practitioner's visit's", byA, colleague},
		{medicationstatement.ErrNotFound, "the admin", byA, admin},
	} {
		before := len(w.audit.events)
		p, ok, err := w.svc.Provenance(t.Context(), tt.c, tt.id)
		if !ok || !errors.Is(err, tt.want) {
			t.Errorf("%s: ok %v, %v; want ok, %v", tt.name, ok, err, tt.want)
			continue
		}
		events := w.audit.events[before:]
		if len(events) != 1 || events[0].Action != auditevent.ActionRead || events[0].Subtype.Code != auditevent.CodeRead {
			t.Errorf("%s: events %+v, want one read", tt.name, events)
			continue
		}
		switch e := events[0]; {
		case tt.want == nil && (p.ID != tt.id || e.EntityReference != "Provenance/"+tt.id || e.Outcome != auditevent.OutcomeSuccess):
			t.Errorf("%s: %+v, event %+v; want the Provenance, audited", tt.name, p, e)
		case tt.want != nil && (p != (medicationstatement.Provenance{}) || e.EntityReference != "" || e.Outcome != auditevent.OutcomeMinorFailure):
			t.Errorf("%s: %+v, event %+v; want nothing, the refusal audited naming nothing", tt.name, p, e)
		}
	}
	before := len(w.audit.events)
	for name, id := range map[string]string{"another resource's id": "9a000000-0000-4000-8000-0000000000ff", "no id": "x"} {
		if _, ok, err := w.svc.Provenance(t.Context(), practitioner, id); ok || err != nil {
			t.Errorf("%s: ok %v, %v; want not this service's", name, ok, err)
		}
	}
	if _, ok, err := w.svc.Provenance(t.Context(), medicationstatement.Caller{}, byA); ok || err != nil {
		t.Errorf("nobody: ok %v, %v; want not this service's", ok, err)
	}
	if len(w.audit.events) != before {
		t.Errorf("audited %+v for ids that are not this service's, want nothing", w.audit.events[before:])
	}
}
