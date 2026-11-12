package composition_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

const (
	encB     = "ec000000-0000-4000-8000-0000000000b1" // clinic B's visit, shared with clinic A
	signedB  = "d0000000-0000-4000-8000-0000000000b1"
	draftB   = "d0000000-0000-4000-8000-0000000000b2"
	signedA2 = "d0000000-0000-4000-8000-0000000000a2"
)

// withSharedVisit adds clinic B's finished visit with a signed document and an unsigned draft, shared with the
// practitioner of clinic A, and a signed document of visit A.
func withSharedVisit(f *fixture) encounter.Encounter {
	b := encounter.Encounter{ID: encB, OrganizationID: clinicB, PatientID: patientA, Status: encounter.StatusFinished}
	f.encounters.shared = map[string]encounter.Encounter{practAcc + encB: b}
	f.repo.drafts[signedB] = composition.Draft{ID: signedB, EncounterID: encB, Status: composition.StatusFinal, SignedAt: now.Add(-48 * 3600e9), Title: "Placeholder report"}
	f.repo.drafts[draftB] = composition.Draft{ID: draftB, EncounterID: encB, Status: composition.StatusPreliminary}
	f.repo.drafts[signedA2] = composition.Draft{ID: signedA2, EncounterID: encA, Status: composition.StatusFinal, SignedAt: now}
	return b
}

func draftIDs(ds []composition.Document) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.Draft.ID
	}
	return out
}

// History gives the signed documents of the visits the history shows, each with its visit, in the caller's clinic
// scope: never a draft, nothing of a visit not given. Unaudited: the history audits what it shows (Disclosed).
func TestHistory_GivesTheSignedDocumentsOfTheVisits(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.drafted(t) // a draft of visit A, not signed
	b := withSharedVisit(f)
	f.audit.events = nil
	a := encounter.Visit{Encounter: f.encounters.enc}
	got, err := f.svc.History(t.Context(), pract, []encounter.Visit{a, {Encounter: b, Shared: true}})
	if err != nil || !slices.Equal(draftIDs(got), []string{signedA2, signedB}) {
		t.Fatalf("History() = %v, %v; want the two signed documents", draftIDs(got), err)
	}
	if got[1].Encounter.ID != encB {
		t.Errorf("History()[1].Encounter = %s, want visit B", got[1].Encounter.ID)
	}
	if got, err := f.svc.History(t.Context(), pract, []encounter.Visit{a}); err != nil || !slices.Equal(draftIDs(got), []string{signedA2}) {
		t.Errorf("History(visit A) = %v, %v; want its signed document only", draftIDs(got), err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audit = %q, want none", f.codes())
	}
	if _, err := f.svc.History(t.Context(), admin, []encounter.Visit{a}); !errors.Is(err, composition.ErrForbidden) {
		t.Errorf("History(admin) = %v, want ErrForbidden", err)
	}
	f.repo.err = errBackend
	if _, err := f.svc.History(t.Context(), pract, []encounter.Visit{a}); !errors.Is(err, errBackend) {
		t.Errorf("History(down) = %v, want the failure", err)
	}
}

// Disclosed audits each document the history shows as a search of it.
func TestDisclosed_AuditsEachDocument(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if err := f.svc.Disclosed(t.Context(), pract, []string{signedA2, signedB}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"search-type 0 Composition/" + signedA2, "search-type 0 Composition/" + signedB}; !slices.Equal(f.codes(), want) {
		t.Errorf("audit = %q, want %q", f.codes(), want)
	}
	for _, e := range f.audit.events {
		if e.OrganizationID != clinicA || e.AgentAccountID != practAcc || e.Purpose.Code != "TREAT" {
			t.Errorf("event %+v, want clinic A's practitioner for treatment", e)
		}
	}
}

// A signed document of another clinic's visit the patient shared opens for the practitioner the history shows it
// to, audited under their clinic; never its drafts, and nobody else's.
func TestRead_ASharedVisitsSignedDocument(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	withSharedVisit(f)
	got, err := f.svc.Composition(t.Context(), pract, signedB)
	if err != nil || got.Draft.ID != signedB || got.Encounter.ID != encB {
		t.Fatalf("Composition(shared) = %+v, %v", got.Draft.ID, err)
	}
	if !slices.Equal(f.codes(), []string{"read 0 Composition/" + signedB}) || f.audit.events[0].OrganizationID != clinicA {
		t.Errorf("audit = %q under %s, want the read under clinic A", f.codes(), f.audit.events[0].OrganizationID)
	}
	if docs, err := f.svc.Signed(t.Context(), pract, encB); err != nil || !slices.Equal(draftIDs(docs), []string{signedB}) {
		t.Errorf("Signed(shared) = %v, %v; want its signed document", draftIDs(docs), err)
	}
	for name, read := range map[string]func() error{
		"its draft":            func() error { _, err := f.svc.Composition(t.Context(), pract, draftB); return err },
		"its drafts":           func() error { _, err := f.svc.Drafts(t.Context(), pract, encB); return err },
		"another practitioner": func() error { _, err := f.svc.Composition(t.Context(), pract2, signedB); return err },
		"the admin":            func() error { _, err := f.svc.Composition(t.Context(), admin, signedB); return err },
	} {
		if err := read(); !errors.Is(err, composition.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
}
