package careplan_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/careplan"
)

// A step whose service needs a device another visit holds then books nothing: the plan stays a draft and says why
// (RM1's resource_busy), and the refusal is audited.
func TestBook_ADeviceBusyBooksNothing(t *testing.T) {
	t.Parallel()
	f := newFixture()
	p, _ := f.svc.Write(t.Context(), admin, draft())
	f.audit.events = nil
	_, err := f.svc.Book(t.Context(), admin, p.ID, careplan.Booking{StartsOn: now, SlotIDs: []string{"in-person@24", "busy"}, Version: 1})
	if !errors.Is(err, careplan.ErrResourceBusy) {
		t.Fatalf("Book: %v, want ErrResourceBusy", err)
	}
	if want := []auditevent.AuditEvent{event(admin.AccountID, clinicA, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("audit %+v, want the refusal", f.audit.events)
	}
	if got, _ := f.svc.Read(t.Context(), admin, p.ID); got.Status != careplan.StatusDraft || len(f.visits.visits) != 0 {
		t.Errorf("plan %s, visits %v; want the draft and no visit", got.Status, f.visits.visits)
	}
}

// A practitioner works only with the plans of a patient they have or had a visit with (CLAUDE.md "GDPR and health
// data", access.CarePlan); another practitioner of the clinic is answered as for a patient they cannot see, and each
// refusal is audited. The admin works with any patient of the clinic.
func TestPlans_APractitionerWorksOnlyWithTheirOwnPatients(t *testing.T) {
	t.Parallel()
	f := newFixture()
	mine, err := f.svc.Write(t.Context(), doc, draft())
	if err != nil {
		t.Fatalf("Write by the practitioner of a visit: %v", err)
	}
	if _, err := f.svc.Read(t.Context(), doc, mine.ID); err != nil {
		t.Errorf("Read by the practitioner of a visit: %v", err)
	}
	if got, err := f.svc.ForPatient(t.Context(), admin, patientA); err != nil || len(got) != 1 {
		t.Errorf("ForPatient by the admin = %v, %v; want the plan", got, err)
	}
	f.audit.events = nil
	day := now.Add(24 * time.Hour)
	for name, call := range map[string]func() error{
		"write": func() error { _, err := f.svc.Write(t.Context(), other, draft()); return err },
		"read":  func() error { _, err := f.svc.Read(t.Context(), other, mine.ID); return err },
		"list":  func() error { _, err := f.svc.ForPatient(t.Context(), other, patientA); return err },
		"change": func() error {
			d := draft()
			d.Version = 1
			_, err := f.svc.Change(t.Context(), other, mine.ID, d)
			return err
		},
		"book": func() error {
			_, err := f.svc.Book(t.Context(), other, mine.ID, careplan.Booking{StartsOn: day, SlotIDs: []string{"in-person@24", "video@360"}, Version: 1})
			return err
		},
		"cancel": func() error { _, err := f.svc.Cancel(t.Context(), other, mine.ID, 1); return err },
	} {
		before := len(f.audit.events)
		if err := call(); !errors.Is(err, careplan.ErrNotFound) {
			t.Errorf("%s by a practitioner without a visit: %v, want ErrNotFound", name, err)
		}
		if got := f.audit.events[before:]; len(got) != 1 || got[0].Outcome != auditevent.OutcomeMinorFailure || got[0].AgentAccountID != other.AccountID {
			t.Errorf("%s: audit %+v, want the refusal", name, got)
		}
	}
	if got, _ := f.svc.Read(t.Context(), admin, mine.ID); got.Status != careplan.StatusDraft || got.VersionID != 1 || len(f.visits.visits) != 0 {
		t.Errorf("plan %+v, visits %v after the refusals; want it unchanged", got, f.visits.visits)
	}
}

// A patient the clinic does not have is not found (the contract's 404), not an empty list; audited.
func TestForPatient_APatientTheClinicDoesNotHaveIsNotFound(t *testing.T) {
	t.Parallel()
	f := newFixture()
	const stranger = "0b000000-0000-4000-8000-0000000000b9"
	if got, err := f.svc.ForPatient(t.Context(), admin, stranger); !errors.Is(err, careplan.ErrNotFound) {
		t.Fatalf("ForPatient = %v, %v; want ErrNotFound", got, err)
	}
	if want := []auditevent.AuditEvent{event(admin.AccountID, clinicA, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("audit %+v, want the refusal", f.audit.events)
	}
}

// Each step names its service as the clinic's price list names it, and once booked its visit's start, for the staff
// and for the patient, whose app has no price list of the clinic.
func TestRead_NamesEachStepsServiceAndItsVisit(t *testing.T) {
	t.Parallel()
	f := newFixture()
	p, _ := f.svc.Write(t.Context(), admin, draft())
	if p.Steps[0].ServiceName != "Synthetic exam" || p.Steps[1].ServiceName != "Synthetic video check" || p.Steps[0].VisitStart != nil {
		t.Errorf("written steps %+v, want their services named and no visit", p.Steps)
	}
	if _, err := f.svc.Book(t.Context(), admin, p.ID, careplan.Booking{StartsOn: now, SlotIDs: []string{"in-person@24", "video@360"}, Version: 1}); err != nil {
		t.Fatal(err)
	}
	f.visits.visits[0].Start = now.Add(400 * time.Hour) // the first step moved after the second
	got, err := f.svc.Read(t.Context(), admin, p.ID)
	if err != nil || got.Steps[0].ServiceName != "Synthetic video check" || got.Steps[0].VisitStart == nil ||
		!got.Steps[0].VisitStart.Equal(now.Add(360*time.Hour)) || !got.Steps[1].VisitStart.Equal(now.Add(400*time.Hour)) {
		t.Fatalf("Read = %+v, %v; want the steps named, in their visits' order, with their starts", got.Steps, err)
	}
	mine, err := f.svc.Mine(t.Context(), me)
	if err != nil || len(mine) != 1 || mine[0].Steps[0].ServiceName != "Synthetic video check" || mine[0].Steps[1].ServiceName != "Synthetic exam" {
		t.Errorf("Mine = %+v, %v; want the steps named", mine, err)
	}
}

// The patient's export of one clinic reads that clinic's booked plans only, never a draft (MineAt).
func TestMineAt_NarrowsToTheClinicAndNeverADraft(t *testing.T) {
	t.Parallel()
	f := newFixture()
	booked, _ := f.svc.Write(t.Context(), admin, draft())
	if _, err := f.svc.Book(t.Context(), admin, booked.ID, careplan.Booking{StartsOn: now, SlotIDs: []string{"in-person@24", "video@360"}, Version: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Write(t.Context(), admin, draft()); err != nil {
		t.Fatal(err)
	}
	if got, err := f.svc.MineAt(t.Context(), me, clinicA); err != nil || len(got) != 1 || got[0].ID != booked.ID {
		t.Errorf("MineAt(clinic A) = %+v, %v; want the booked plan alone", got, err)
	}
	if got, err := f.svc.MineAt(t.Context(), me, "0a000000-0000-4000-8000-0000000000af"); err != nil || len(got) != 0 {
		t.Errorf("MineAt(another clinic) = %+v, %v; want none", got, err)
	}
}
