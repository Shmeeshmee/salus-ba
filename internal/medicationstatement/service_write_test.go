package medicationstatement_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
)

func input() medicationstatement.Input {
	return medicationstatement.Input{Text: "  Placeholder substance ", Status: medicationstatement.StatusActive,
		Dosage: medicationstatement.Dosage{DoseValue: 500, DoseUnit: "mg", Frequency: 2, Period: 1, PeriodUnit: "d"}}
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
	if !stored.Source.Patient() || !stored.InProfile || !stored.Confirmation.At.IsZero() ||
		!stored.DateAsserted.Equal(now) || stored.Code.Text != "Placeholder substance" || stored.Code.TermID != termDrug ||
		stored.PatientID != patientP || a.VersionID != 1 {
		t.Errorf("stored %+v", stored)
	}
	if a.Code.ATC != "J01CA04" {
		t.Errorf("the ATC code = %q", a.Code.ATC)
	}
	if e := w.audit.events[0]; e.Subtype.Code != auditevent.CodeCreate || e.Action != auditevent.ActionCreate ||
		e.EntityReference != "MedicationStatement/"+a.ID || e.Purpose.Code != auditevent.CodePATRQT {
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
	for name, change := range map[string]func(*medicationstatement.Input){
		"no text":                  func(in *medicationstatement.Input) { in.Text = "  " },
		"text over 200 characters": func(in *medicationstatement.Input) { in.Text = string(long) },
		"no status":                func(in *medicationstatement.Input) { in.Status = "" },
		"entered-in-error":         func(in *medicationstatement.Input) { in.Status = medicationstatement.StatusEnteredInError },
		"a dose without a unit":    func(in *medicationstatement.Input) { in.Dosage.DoseUnit = "" },
		"a unit without a dose":    func(in *medicationstatement.Input) { in.Dosage.DoseValue = 0 },
		"an unknown dose unit":     func(in *medicationstatement.Input) { in.Dosage.DoseUnit = "kg" },
		"a dose too large":         func(in *medicationstatement.Input) { in.Dosage.DoseValue = 100000 },
		"a negative dose":          func(in *medicationstatement.Input) { in.Dosage.DoseValue = -1 },
		"a timing without period":  func(in *medicationstatement.Input) { in.Dosage.Period = 0 },
		"a frequency of 0":         func(in *medicationstatement.Input) { in.Dosage.Frequency = 0 },
		"a frequency over 100":     func(in *medicationstatement.Input) { in.Dosage.Frequency = 101 },
		"a period over 1000":       func(in *medicationstatement.Input) { in.Dosage.Period = 1001 },
		"an unknown period unit":   func(in *medicationstatement.Input) { in.Dosage.PeriodUnit = "s" },
		"a term that is no drug":   func(in *medicationstatement.Input) { in.TermID = termAnatomy },
		"a term id that is no id":  func(in *medicationstatement.Input) { in.TermID = "x" },
	} {
		w := newWorld(newRepo(), sharing())
		in := input()
		change(&in)
		if _, err := w.svc.Add(t.Context(), patient, in); !errors.Is(err, medicationstatement.ErrInvalid) {
			t.Errorf("%s: %v, want ErrInvalid", name, err)
		}
	}
	w := newWorld(newRepo(), sharing())
	if _, err := w.svc.Add(t.Context(), practitioner, input()); !errors.Is(err, medicationstatement.ErrForbidden) {
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
	want := medicationstatement.Source{OrganizationID: clinicA, EncounterID: encA, RecorderRoleID: roleA}
	if stored.Source != want || stored.InProfile ||
		stored.Confirmation != (medicationstatement.Confirmation{At: now, RoleID: roleA, OrganizationID: clinicA}) || stored.PatientID != patientP {
		t.Errorf("stored %+v", stored)
	}
	if w.repo.clinic != clinicA {
		t.Errorf("the scope = %q", w.repo.clinic)
	}
	for name, c := range map[string]appointment.Caller{"a colleague": colleague, "the admin": admin, "the patient": patient} {
		if _, err := w.svc.Record(t.Context(), c, encA, input()); !errors.Is(err, medicationstatement.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
	revoked := newWorld(newRepo(), sharingRevoked())
	if _, err := revoked.svc.Record(t.Context(), practitioner, encA, input()); !errors.Is(err, medicationstatement.ErrNotFound) {
		t.Errorf("after a revocation: %v, want ErrNotFound", err)
	}
	if _, err := w.svc.Record(t.Context(), practitioner, encB, input()); !errors.Is(err, medicationstatement.ErrNotFound) {
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
	if got := w.repo.byID[idOwn]; got.Code.TermID != termDrug || got.Dosage.DoseValue != 500 || a.VersionID != 2 || !got.LastUpdated.Equal(now) {
		t.Errorf("the change: %+v", got)
	}
	in.Text = "Placeholder other"
	if _, err := w.svc.Update(t.Context(), patient, idOwnByB, 1, in); err != nil {
		t.Fatal(err)
	}
	if got := w.repo.byID[idOwnByB]; got.Code.TermID != "" || !got.Confirmation.At.IsZero() {
		t.Errorf("a change to the confirmed entry: %+v, want it unconfirmed", got)
	}
	if _, err := w.svc.Update(t.Context(), patient, idVisitA, 1, in); !errors.Is(err, medicationstatement.ErrForbidden) {
		t.Errorf("a clinic's entry: %v, want ErrForbidden", err)
	}
	if _, err := w.svc.Update(t.Context(), patient, idOwn, 1, in); !errors.Is(err, medicationstatement.ErrVersionConflict) {
		t.Errorf("a stale version: %v, want ErrVersionConflict", err)
	}
	if _, err := w.svc.Update(t.Context(), patient, idRemoved, 1, in); !errors.Is(err, medicationstatement.ErrNotFound) {
		t.Errorf("a removed entry: %v, want ErrNotFound", err)
	}
	last := w.audit.events[len(w.audit.events)-2]
	if last.EntityReference != "MedicationStatement/"+idOwn || last.Outcome != auditevent.OutcomeMinorFailure || last.Action != auditevent.ActionUpdate {
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
	if got := w.repo.byID[idVisitA]; !got.Confirmation.At.IsZero() {
		t.Errorf("another clinic's confirmation: %+v, want it cleared", got)
	}
	w2 := newWorld(history(), sharing())
	if _, err := w2.svc.Update(t.Context(), practitioner, idVisitA, 1, input()); err != nil {
		t.Fatal(err)
	}
	if got := w2.repo.byID[idVisitA]; got.Confirmation != byA() {
		t.Errorf("their own clinic's confirmation: %+v, want it kept", got)
	}
	for _, id := range []string{idOwn, idShared} {
		if _, err := w2.svc.Update(t.Context(), practitioner, id, 1, input()); !errors.Is(err, medicationstatement.ErrForbidden) {
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
		if got := w.repo.byID[id]; got.Status != medicationstatement.StatusEnteredInError || got.VersionID != 2 {
			t.Errorf("%s: %+v, want entered-in-error", id, got)
		}
	}
	if e := w.audit.events[0]; e.Action != auditevent.ActionDelete || e.EntityReference != "MedicationStatement/"+idOwn {
		t.Errorf("the event = %+v", e)
	}
	w2 := newWorld(history(), sharing())
	if err := w2.svc.Remove(t.Context(), practitioner, idVisitA, 1); err != nil {
		t.Fatal(err)
	}
	if got := w2.repo.byID[idVisitA]; got.Status != medicationstatement.StatusEnteredInError {
		t.Errorf("the clinic's removal: %+v", got)
	}
	for _, id := range []string{idOwn, idShared} {
		if err := w2.svc.Remove(t.Context(), practitioner, id, 1); !errors.Is(err, medicationstatement.ErrForbidden) {
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
	if got := w.repo.byID[idOwn]; got.Confirmation != (medicationstatement.Confirmation{At: now, RoleID: roleA, OrganizationID: clinicA}) || a.VersionID != 2 {
		t.Errorf("confirmed %+v", got)
	}
	if a, err := w.svc.Confirm(t.Context(), practitioner, idVisitA, 1); err != nil || a.VersionID != 1 {
		t.Errorf("their clinic's own confirmation: %d, %v; want it unchanged", a.VersionID, err)
	}
	for _, id := range []string{idOwnByB, idShared} {
		if _, err := w.svc.Confirm(t.Context(), practitioner, id, 1); !errors.Is(err, medicationstatement.ErrForbidden) {
			t.Errorf("%s: %v, want ErrForbidden", id, err)
		}
	}
	if _, err := w.svc.Confirm(t.Context(), patient, idOwn, 2); !errors.Is(err, medicationstatement.ErrForbidden) {
		t.Errorf("the patient: %v, want ErrForbidden", err)
	}
	if _, err := w.svc.Confirm(t.Context(), practitioner, idVisitB, 1); !errors.Is(err, medicationstatement.ErrNotFound) {
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
	if _, err := w.svc.AddToProfile(t.Context(), practitioner, idVisitA, 1); !errors.Is(err, medicationstatement.ErrForbidden) {
		t.Errorf("staff: %v, want ErrForbidden", err)
	}
}
