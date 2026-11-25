package careplan_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/careplan"
)

func TestWrite_StoresADraftAndAuditsIt(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.svc.Write(t.Context(), admin, draft())
	if err != nil || got.Status != careplan.StatusDraft || len(got.Steps) != 2 || got.Steps[1].Position != 2 || got.Steps[1].DayOffset != 14 {
		t.Fatalf("Write = %+v, %v; want a two-step draft", got, err)
	}
	if want := []auditevent.AuditEvent{event(admin.AccountID, clinicA, auditevent.CodeCreate, got.ID, auditevent.OutcomeSuccess)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("audit %+v, want %+v", f.audit.events, want)
	}
}

func TestWrite_RefusesAndAuditsEachCase(t *testing.T) {
	t.Parallel()
	mutate := func(fn func(d *careplan.Draft)) careplan.Draft { d := draft(); fn(&d); return d }
	for name, tc := range map[string]struct {
		want  error
		c     careplan.Staff
		draft careplan.Draft
	}{
		"a receptionist":        {careplan.ErrForbidden, desk, draft()},
		"no title":              {careplan.ErrInvalid, admin, mutate(func(d *careplan.Draft) { d.Title = "  " })},
		"no steps":              {careplan.ErrInvalid, admin, mutate(func(d *careplan.Draft) { d.Steps = nil })},
		"a service not offered": {careplan.ErrInvalid, admin, mutate(func(d *careplan.Draft) { d.Steps[0].VisitMode = "video" })},
		"an unknown service":    {careplan.ErrInvalid, admin, mutate(func(d *careplan.Draft) { d.Steps[0].ServiceID = patientA })},
		"a day out of range":    {careplan.ErrInvalid, admin, mutate(func(d *careplan.Draft) { d.Steps[1].DayOffset = careplan.MaxDayOffset + 1 })},
		"no consent":            {careplan.ErrNotFound, admin, mutate(func(d *careplan.Draft) { d.PatientID = clinicA })},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			if _, err := f.svc.Write(t.Context(), tc.c, tc.draft); !errors.Is(err, tc.want) {
				t.Fatalf("Write: %v, want %v", err, tc.want)
			}
			if want := []auditevent.AuditEvent{event(tc.c.AccountID, clinicA, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure)}; len(f.repo.plans) != 0 || !slices.Equal(f.audit.events, want) {
				t.Errorf("stored %v, audit %+v; want nothing stored and the refusal audited", f.repo.plans, f.audit.events)
			}
		})
	}
}

func TestWrite_RefusesARestrictedRecord(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.svc = careplan.NewService(careplan.Ports{Repo: f.repo, Tx: fakeTx{f: f}, Audit: f.audit, Prices: fakePrices{}, Visits: f.visits, Clock: f.clock,
		Care: fakeCare{}, Roles: fakeRoles{}, Consents: fakeConsents{state: map[string]access.Consent{patientA + clinicA: {Active: true, Restricted: true}}}})
	if _, err := f.svc.Write(t.Context(), admin, draft()); !errors.Is(err, careplan.ErrNotFound) {
		t.Fatalf("Write: %v, want ErrNotFound", err)
	}
}

func TestChange_ChangesADraftAtItsVersionOnly(t *testing.T) {
	t.Parallel()
	f := newFixture()
	p, _ := f.svc.Write(t.Context(), admin, draft())
	d := draft()
	d.Title, d.Steps, d.Version = "Changed", d.Steps[:1], p.VersionID
	got, err := f.svc.Change(t.Context(), admin, p.ID, d)
	if err != nil || got.Title != "Changed" || len(got.Steps) != 1 || got.VersionID != 2 || got.PatientID != patientA {
		t.Fatalf("Change = %+v, %v", got, err)
	}
	if _, err := f.svc.Change(t.Context(), admin, p.ID, d); !errors.Is(err, careplan.ErrVersionConflict) {
		t.Errorf("Change at the old version: %v, want ErrVersionConflict", err)
	}
}

func TestBook_BooksEveryStepAndThePatientSeesIt(t *testing.T) {
	t.Parallel()
	f := newFixture()
	p, _ := f.svc.Write(t.Context(), admin, draft())
	if mine, _ := f.svc.Mine(t.Context(), me); len(mine) != 0 {
		t.Fatalf("Mine = %+v before booking, want no draft", mine)
	}
	day := time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC)
	got, err := f.svc.Book(t.Context(), admin, p.ID, careplan.Booking{StartsOn: day, SlotIDs: []string{"in-person@24", "video@360"}, Version: 1})
	if err != nil || got.Status != careplan.StatusActive || got.BookedAt == nil || !got.StartsOn.Equal(day) || got.Steps[0].AppointmentID == "" || got.Steps[1].AppointmentID == "" {
		t.Fatalf("Book = %+v, %v; want an active plan with its visits", got, err)
	}
	mine, err := f.svc.Mine(t.Context(), me)
	if err != nil || len(mine) != 1 || mine[0].ID != p.ID {
		t.Fatalf("Mine = %+v, %v; want the booked plan", mine, err)
	}
	if want := event(me.AccountID, "", auditevent.CodeRead, p.ID, auditevent.OutcomeSuccess); f.audit.events[len(f.audit.events)-1] != want {
		t.Errorf("last audit %+v, want the patient's read", f.audit.events[len(f.audit.events)-1])
	}
	if _, err := f.svc.Book(t.Context(), admin, p.ID, careplan.Booking{StartsOn: day, SlotIDs: []string{"in-person@24", "video@360"}, Version: 2}); !errors.Is(err, careplan.ErrNotDraft) {
		t.Errorf("a second booking: %v, want ErrNotDraft", err)
	}
}

func TestBook_BooksNoneWhenAStepIsRefused(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want  error
		slots []string
	}{
		"a slot taken":           {careplan.ErrSlotUnavailable, []string{"in-person@24", "taken"}},
		"not one slot per step":  {careplan.ErrInvalid, []string{"in-person@24"}},
		"a slot in another mode": {careplan.ErrInvalid, []string{"in-person@24", "in-person@48"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			p, _ := f.svc.Write(t.Context(), admin, draft())
			if _, err := f.svc.Book(t.Context(), admin, p.ID, careplan.Booking{StartsOn: now, SlotIDs: tc.slots, Version: 1}); !errors.Is(err, tc.want) {
				t.Fatalf("Book: %v, want %v", err, tc.want)
			}
			got, _ := f.svc.Read(t.Context(), admin, p.ID)
			if got.Status != careplan.StatusDraft || len(f.visits.visits) != 0 {
				t.Errorf("plan %s, visits %v; want the draft and no visit", got.Status, f.visits.visits)
			}
		})
	}
}

func TestRead_OrdersStepsByTheirVisitsAndCompletesAPastPlan(t *testing.T) {
	t.Parallel()
	f := newFixture()
	p, _ := f.svc.Write(t.Context(), admin, draft())
	p, _ = f.svc.Book(t.Context(), admin, p.ID, careplan.Booking{StartsOn: now, SlotIDs: []string{"in-person@24", "video@360"}, Version: 1})
	f.visits.visits[0].Start = now.Add(400 * time.Hour) // the first step moved after the second
	f.visits.visits[0].End = f.visits.visits[0].Start.Add(30 * time.Minute)
	got, err := f.svc.Read(t.Context(), admin, p.ID)
	if err != nil || got.Steps[0].ServiceID != serviceVd || got.Steps[0].Position != 1 || got.Status != careplan.StatusActive {
		t.Fatalf("Read = %+v, %v; want the moved step second", got, err)
	}
	f.visits.visits[1].Status = appointment.StatusCancelled // the desk cancelled the second step's visit
	f.visits.visits[1].Start = now.Add(600 * time.Hour)
	f.visits.visits[1].End = f.visits.visits[1].Start.Add(30 * time.Minute)
	f.clock.T = now.Add(500 * time.Hour)
	if got, _ := f.svc.Read(t.Context(), admin, p.ID); got.Status != careplan.StatusCompleted {
		t.Errorf("status %s once every visit is past or cancelled, want completed", got.Status)
	}
	if _, err := f.svc.Cancel(t.Context(), admin, p.ID, p.VersionID); !errors.Is(err, careplan.ErrEnded) {
		t.Errorf("Cancel of a completed plan: %v, want ErrEnded", err)
	}
}

func TestCancel_RevokesThePlanAndCancelsItsFutureVisits(t *testing.T) {
	t.Parallel()
	f := newFixture()
	p, _ := f.svc.Write(t.Context(), admin, draft())
	p, _ = f.svc.Book(t.Context(), admin, p.ID, careplan.Booking{StartsOn: now, SlotIDs: []string{"in-person@24", "video@360"}, Version: 1})
	got, err := f.svc.Cancel(t.Context(), admin, p.ID, p.VersionID)
	if err != nil || got.Status != careplan.StatusRevoked || got.RevokedAt == nil || len(f.visits.cancelled) != 2 {
		t.Fatalf("Cancel = %+v, %v; cancelled %v", got, err, f.visits.cancelled)
	}
	if _, err := f.svc.Cancel(t.Context(), admin, p.ID, got.VersionID); !errors.Is(err, careplan.ErrEnded) {
		t.Errorf("a second cancel: %v, want ErrEnded", err)
	}
}

func TestForPatient_ListsTheClinicsPlansAndAuditsEach(t *testing.T) {
	t.Parallel()
	f := newFixture()
	a, _ := f.svc.Write(t.Context(), admin, draft())
	b, _ := f.svc.Write(t.Context(), admin, draft())
	f.audit.events = nil
	got, err := f.svc.ForPatient(t.Context(), admin, patientA)
	if err != nil || len(got) != 2 || got[0].ID != b.ID || got[1].ID != a.ID {
		t.Fatalf("ForPatient = %+v, %v; want both, newest first", got, err)
	}
	search := event(admin.AccountID, clinicA, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
	search.EntityReference = "Patient/" + patientA
	if len(f.audit.events) != 3 || f.audit.events[0] != search || f.audit.events[1] != event(admin.AccountID, clinicA, auditevent.CodeRead, b.ID, auditevent.OutcomeSuccess) {
		t.Errorf("audit %+v, want the search and each plan read", f.audit.events)
	}
	f.audit.events = nil
	if got, err := f.svc.Mine(t.Context(), me); err != nil || len(got) != 0 || len(f.audit.events) != 1 {
		t.Errorf("Mine = %v, %v with audit %+v; want no plan and the search audited", got, err, f.audit.events)
	}
	if _, err := f.svc.ForPatient(t.Context(), desk, patientA); !errors.Is(err, careplan.ErrForbidden) {
		t.Errorf("a receptionist: %v, want ErrForbidden", err)
	}
}
