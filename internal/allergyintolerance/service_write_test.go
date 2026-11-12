package allergyintolerance_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

func input() allergyintolerance.Input {
	return allergyintolerance.Input{Text: "  Placeholder substance ", Category: allergyintolerance.CategoryFood,
		Reaction: "Placeholder reaction", Severity: allergyintolerance.SeverityMild, ClinicalStatus: allergyintolerance.ClinicalActive}
}

// The patient adds an unconfirmed entry to their profile, recorded now, the text trimmed; a term picked from the
// terminology carries its ATC code. Audited as a create.
func TestAdd_ThePatientAddsToTheirProfile(t *testing.T) {
	t.Parallel()
	w := newWorld(newRepo(), sharing())
	in := input()
	in.TermID = termDrug
	a, err := w.svc.Add(t.Context(), patient, in)
	if err != nil {
		t.Fatal(err)
	}
	stored := w.repo.byID[a.ID]
	if !stored.Source.Patient() || !stored.InProfile || stored.VerificationStatus != allergyintolerance.VerificationUnconfirmed ||
		!stored.RecordedDate.Equal(now) || stored.Code.Text != "Placeholder substance" || stored.Code.TermID != termDrug ||
		stored.PatientID != patientP || a.VersionID != 1 {
		t.Errorf("stored %+v", stored)
	}
	if a.Code.ATC != "J01CA04" {
		t.Errorf("the ATC code = %q", a.Code.ATC)
	}
	if e := w.audit.events[0]; e.Subtype.Code != auditevent.CodeCreate || e.Action != auditevent.ActionCreate ||
		e.EntityReference != "AllergyIntolerance/"+a.ID || e.Purpose.Code != auditevent.CodePATRQT {
		t.Errorf("the event = %+v", e)
	}
	if w.repo.patient != patientP {
		t.Errorf("the scope = %q", w.repo.patient)
	}
}

// What the value sets and the table's lengths refuse is ErrInvalid; staff cannot add to a profile (ErrForbidden),
// audited as a refusal.
func TestAdd_RefusesAnInvalidInputAndStaff(t *testing.T) {
	t.Parallel()
	long := make([]rune, 201)
	for i := range long {
		long[i] = 'x'
	}
	for name, change := range map[string]func(*allergyintolerance.Input){
		"no text":                  func(in *allergyintolerance.Input) { in.Text = "  " },
		"text over 200 characters": func(in *allergyintolerance.Input) { in.Text = string(long) },
		"reaction over 200":        func(in *allergyintolerance.Input) { in.Reaction = string(long) },
		"an unknown category":      func(in *allergyintolerance.Input) { in.Category = "plant" },
		"an unknown severity":      func(in *allergyintolerance.Input) { in.Severity = "fatal" },
		"a severity, no reaction":  func(in *allergyintolerance.Input) { in.Reaction = " " },
		"no clinical status":       func(in *allergyintolerance.Input) { in.ClinicalStatus = "" },
		"a term that is no drug":   func(in *allergyintolerance.Input) { in.TermID = termAnatomy },
		"a term id that is no id":  func(in *allergyintolerance.Input) { in.TermID = "x" },
	} {
		w := newWorld(newRepo(), sharing())
		in := input()
		change(&in)
		if _, err := w.svc.Add(t.Context(), patient, in); !errors.Is(err, allergyintolerance.ErrInvalid) {
			t.Errorf("%s: %v, want ErrInvalid", name, err)
		}
	}
	w := newWorld(newRepo(), sharing())
	if _, err := w.svc.Add(t.Context(), practitioner, input()); !errors.Is(err, allergyintolerance.ErrForbidden) {
		t.Errorf("staff: %v, want ErrForbidden", err)
	}
	if len(w.repo.byID) != 0 || len(w.audit.events) != 1 || w.audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("stored %d, events %+v; want nothing stored and the refusal audited", len(w.repo.byID), w.audit.events)
	}
}

// The visit's practitioner records a confirmed entry in it, not in the profile; anyone else, and after a
// revocation, gets ErrNotFound.
func TestRecord_TheVisitsPractitionerRecords(t *testing.T) {
	t.Parallel()
	w := newWorld(newRepo(), sharing())
	a, err := w.svc.Record(t.Context(), practitioner, encA, input())
	if err != nil {
		t.Fatal(err)
	}
	stored := w.repo.byID[a.ID]
	want := allergyintolerance.Source{OrganizationID: clinicA, EncounterID: encA, RecorderRoleID: roleA}
	if stored.Source != want || stored.InProfile || stored.VerificationStatus != allergyintolerance.VerificationConfirmed ||
		stored.Confirmation != (allergyintolerance.Confirmation{At: now, RoleID: roleA, OrganizationID: clinicA}) || stored.PatientID != patientP {
		t.Errorf("stored %+v", stored)
	}
	if w.repo.clinic != clinicA {
		t.Errorf("the scope = %q", w.repo.clinic)
	}
	for name, c := range map[string]appointment.Caller{"a colleague": colleague, "the admin": admin, "the patient": patient} {
		if _, err := w.svc.Record(t.Context(), c, encA, input()); !errors.Is(err, allergyintolerance.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
	revoked := newWorld(newRepo(), sharingRevoked())
	if _, err := revoked.svc.Record(t.Context(), practitioner, encA, input()); !errors.Is(err, allergyintolerance.ErrNotFound) {
		t.Errorf("after a revocation: %v, want ErrNotFound", err)
	}
	if _, err := w.svc.Record(t.Context(), practitioner, encB, input()); !errors.Is(err, allergyintolerance.ErrNotFound) {
		t.Errorf("another clinic's encounter: %v, want ErrNotFound", err)
	}
}

func sharingRevoked() access.Consent { return access.Consent{RevokedAt: now} }

// The patient changes their own entry; a change to a confirmed one makes it unconfirmed; a clinic's entry is not
// theirs to change; a stale version is a conflict; an unchanged text without a term id keeps the term.
func TestUpdate_ThePatientsChanges(t *testing.T) {
	t.Parallel()
	w := newWorld(history(), sharing())
	in := input()
	in.Text = "Placeholder 1" // the own entry's text, without its term id
	a, err := w.svc.Update(t.Context(), patient, idOwn, 1, in)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.repo.byID[idOwn]; got.Code.TermID != termDrug || got.Reaction != "Placeholder reaction" || a.VersionID != 2 || !got.LastUpdated.Equal(now) {
		t.Errorf("the change: %+v", got)
	}
	in.Text = "Placeholder other"
	if _, err := w.svc.Update(t.Context(), patient, idOwnByB, 1, in); err != nil {
		t.Fatal(err)
	}
	if got := w.repo.byID[idOwnByB]; got.Code.TermID != "" || !got.Confirmation.At.IsZero() || got.VerificationStatus != allergyintolerance.VerificationUnconfirmed {
		t.Errorf("a change to the confirmed entry: %+v, want it unconfirmed", got)
	}
	if _, err := w.svc.Update(t.Context(), patient, idVisitA, 1, in); !errors.Is(err, allergyintolerance.ErrForbidden) {
		t.Errorf("a clinic's entry: %v, want ErrForbidden", err)
	}
	if _, err := w.svc.Update(t.Context(), patient, idOwn, 1, in); !errors.Is(err, allergyintolerance.ErrVersionConflict) {
		t.Errorf("a stale version: %v, want ErrVersionConflict", err)
	}
	if _, err := w.svc.Update(t.Context(), patient, idRemoved, 1, in); !errors.Is(err, allergyintolerance.ErrNotFound) {
		t.Errorf("a removed entry: %v, want ErrNotFound", err)
	}
	last := w.audit.events[len(w.audit.events)-2]
	if last.EntityReference != "AllergyIntolerance/"+idOwn || last.Outcome != auditevent.OutcomeMinorFailure || last.Action != auditevent.ActionUpdate {
		t.Errorf("the conflict's event = %+v, want the refusal naming the entry", last)
	}
}

// A practitioner changes the entries their clinic recorded: their clinic's confirmation stays, another clinic's is
// cleared; the patient's own entries and other clinics' are not theirs to change.
func TestUpdate_APractitionersChanges(t *testing.T) {
	t.Parallel()
	repo := history()
	byOther := repo.byID[idVisitA]
	byOther.InProfile, byOther.Confirmation = true, byB()
	repo.byID[idVisitA] = byOther
	w := newWorld(repo, sharing())
	if _, err := w.svc.Update(t.Context(), practitioner, idVisitA, 1, input()); err != nil {
		t.Fatal(err)
	}
	if got := w.repo.byID[idVisitA]; !got.Confirmation.At.IsZero() || got.VerificationStatus != allergyintolerance.VerificationUnconfirmed {
		t.Errorf("another clinic's confirmation: %+v, want it cleared", got)
	}
	w2 := newWorld(history(), sharing())
	if _, err := w2.svc.Update(t.Context(), practitioner, idVisitA, 1, input()); err != nil {
		t.Fatal(err)
	}
	if got := w2.repo.byID[idVisitA]; got.Confirmation != byA() || got.VerificationStatus != allergyintolerance.VerificationConfirmed {
		t.Errorf("their own clinic's confirmation: %+v, want it kept", got)
	}
	for _, id := range []string{idOwn, idShared} {
		if _, err := w2.svc.Update(t.Context(), practitioner, id, 1, input()); !errors.Is(err, allergyintolerance.ErrForbidden) {
			t.Errorf("%s: %v, want ErrForbidden", id, err)
		}
	}
}

// The patient's own unconfirmed entry is deleted; a confirmed one, and a clinic's, become entered-in-error; a
// practitioner removes only their clinic's entries.
func TestRemove(t *testing.T) {
	t.Parallel()
	w := newWorld(history(), sharing())
	for _, id := range []string{idOwn, idOwnByB, idVisitA} {
		if err := w.svc.Remove(t.Context(), patient, id, 1); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	if !slices.Equal(w.repo.deleted, []string{idOwn}) {
		t.Errorf("deleted %v, want the own unconfirmed entry", w.repo.deleted)
	}
	for _, id := range []string{idOwnByB, idVisitA} {
		if got := w.repo.byID[id]; got.VerificationStatus != allergyintolerance.VerificationEnteredInError || got.VersionID != 2 {
			t.Errorf("%s: %+v, want entered-in-error", id, got)
		}
	}
	if e := w.audit.events[0]; e.Action != auditevent.ActionDelete || e.EntityReference != "AllergyIntolerance/"+idOwn {
		t.Errorf("the event = %+v", e)
	}
	w2 := newWorld(history(), sharing())
	if err := w2.svc.Remove(t.Context(), practitioner, idVisitA, 1); err != nil {
		t.Fatal(err)
	}
	if got := w2.repo.byID[idVisitA]; got.VerificationStatus != allergyintolerance.VerificationEnteredInError {
		t.Errorf("the clinic's removal: %+v", got)
	}
	for _, id := range []string{idOwn, idShared} {
		if err := w2.svc.Remove(t.Context(), practitioner, id, 1); !errors.Is(err, allergyintolerance.ErrForbidden) {
			t.Errorf("%s: %v, want ErrForbidden", id, err)
		}
	}
}

// A practitioner who sees an entry confirms it as their clinic: never over another clinic's confirmation, never a
// shared visit's entry (read-only); the patient never confirms.
func TestConfirm(t *testing.T) {
	t.Parallel()
	w := newWorld(history(), sharing())
	a, err := w.svc.Confirm(t.Context(), practitioner, idOwn, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.repo.byID[idOwn]; got.Confirmation != (allergyintolerance.Confirmation{At: now, RoleID: roleA, OrganizationID: clinicA}) ||
		got.VerificationStatus != allergyintolerance.VerificationConfirmed || a.VersionID != 2 {
		t.Errorf("confirmed %+v", got)
	}
	if a, err := w.svc.Confirm(t.Context(), practitioner, idVisitA, 1); err != nil || a.VersionID != 1 {
		t.Errorf("their clinic's own confirmation: %d, %v; want it unchanged", a.VersionID, err)
	}
	for _, id := range []string{idOwnByB, idShared} {
		if _, err := w.svc.Confirm(t.Context(), practitioner, id, 1); !errors.Is(err, allergyintolerance.ErrForbidden) {
			t.Errorf("%s: %v, want ErrForbidden", id, err)
		}
	}
	if _, err := w.svc.Confirm(t.Context(), patient, idOwn, 2); !errors.Is(err, allergyintolerance.ErrForbidden) {
		t.Errorf("the patient: %v, want ErrForbidden", err)
	}
	if _, err := w.svc.Confirm(t.Context(), practitioner, idVisitB, 1); !errors.Is(err, allergyintolerance.ErrNotFound) {
		t.Errorf("an entry they do not see: %v, want ErrNotFound", err)
	}
}

// The patient adds a clinic's entry to their profile; one already there stays as it is; staff cannot.
func TestAddToProfile(t *testing.T) {
	t.Parallel()
	w := newWorld(history(), sharing())
	a, err := w.svc.AddToProfile(t.Context(), patient, idVisitB, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.repo.byID[idVisitB]; !got.InProfile || got.Confirmation != byB() || a.VersionID != 2 {
		t.Errorf("added %+v", got)
	}
	if a, err := w.svc.AddToProfile(t.Context(), patient, idOwn, 1); err != nil || a.VersionID != 1 {
		t.Errorf("an own entry: %d, %v; want it unchanged", a.VersionID, err)
	}
	if _, err := w.svc.AddToProfile(t.Context(), practitioner, idVisitA, 1); !errors.Is(err, allergyintolerance.ErrForbidden) {
		t.Errorf("staff: %v, want ErrForbidden", err)
	}
}
