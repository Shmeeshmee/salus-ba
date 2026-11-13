package appointment_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// joinA is patient A's request for role A1 of clinic A from today to the day of start, 08:00 to 12:00.
func joinA() appointment.NewEntry {
	return appointment.NewEntry{
		OrganizationID: clinicA, PractitionerRoleID: roleA1, Earliest: day(now), Latest: day(start),
		Times: appointment.Times{Start: 8 * 60, End: 12 * 60},
	}
}

// The patient joins for a role between two dates of the clinic's zone; the entry ends at the clinic's midnight after
// the latest day, and the join is audited.
func TestJoin_PutsThePatientOnTheClinicsWaitlist(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	got, err := w.svc.Join(t.Context(), me, joinA())
	if err != nil {
		t.Fatal(err)
	}
	endsAt := time.Date(2030, 1, 7, 23, 0, 0, 0, time.UTC) // midnight in Sarajevo
	if got.ID == "" || got.Status != appointment.WaitlistActive || got.PatientID != patientA || !got.EndsAt.Equal(endsAt) ||
		!got.Created.Equal(now) || got.Times != (appointment.Times{Start: 480, End: 720}) {
		t.Errorf("Join() = %+v, want patient A's active entry ending at %v", got, endsAt)
	}
	if want := []auditevent.AuditEvent{event(me, auditevent.CodeCreate, "Appointment", got.ID, auditevent.OutcomeSuccess)}; !slices.Equal(w.audit.events, want) {
		t.Errorf("audit %+v, want %+v", w.audit.events, want)
	}
	byspecialty := joinA()
	byspecialty.PractitionerRoleID, byspecialty.Specialty, byspecialty.Times = "", "394814009", appointment.Times{}
	if _, err := w.svc.Join(t.Context(), me, byspecialty); err != nil {
		t.Errorf("Join(a specialty, any time) = %v", err)
	}
}

func TestJoin_RefusesWhatTheRulesDoNotAllow(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		change  func(w *waitlistFixture, in *appointment.NewEntry)
		caller  appointment.Caller
		want    error
		name    string
		audited bool
	}{
		{name: "staff", caller: admin, want: appointment.ErrForbidden, audited: true},
		{name: "no profile", caller: noProfile, want: appointment.ErrProfileRequired, audited: true},
		{name: "a role and a specialty", change: func(_ *waitlistFixture, in *appointment.NewEntry) { in.Specialty = "394814009" }, want: appointment.ErrInvalidEntry},
		{name: "neither", change: func(_ *waitlistFixture, in *appointment.NewEntry) { in.PractitionerRoleID = "" }, want: appointment.ErrInvalidEntry},
		{name: "an unknown specialty", change: func(_ *waitlistFixture, in *appointment.NewEntry) {
			in.PractitionerRoleID, in.Specialty = "", "123456"
		}, want: appointment.ErrInvalidEntry},
		{name: "dates out of order", change: func(_ *waitlistFixture, in *appointment.NewEntry) { in.Earliest = in.Latest.AddDate(0, 0, 1) }, want: appointment.ErrInvalidEntry},
		{name: "times out of order", change: func(_ *waitlistFixture, in *appointment.NewEntry) { in.Times = appointment.Times{Start: 600, End: 540} }, want: appointment.ErrInvalidEntry},
		{name: "times past midnight", change: func(_ *waitlistFixture, in *appointment.NewEntry) { in.Times = appointment.Times{Start: 600, End: 1441} }, want: appointment.ErrInvalidEntry},
		{name: "a past day", change: func(_ *waitlistFixture, in *appointment.NewEntry) { in.Earliest = day(now).AddDate(0, 0, -1) }, want: appointment.ErrInvalidEntry, audited: true},
		{name: "more than 90 days ahead", change: func(_ *waitlistFixture, in *appointment.NewEntry) { in.Latest = day(now).AddDate(0, 0, 91) }, want: appointment.ErrInvalidEntry, audited: true},
		{name: "another clinic's role", change: func(_ *waitlistFixture, in *appointment.NewEntry) { in.PractitionerRoleID = roleB }, want: appointment.ErrInvalidEntry, audited: true},
		{name: "the waitlist off", change: func(w *waitlistFixture, _ *appointment.NewEntry) { w.settings.off[clinicA] = true }, want: appointment.ErrWaitlistOff, audited: true},
		{name: "an unlisted clinic", change: func(_ *waitlistFixture, in *appointment.NewEntry) {
			in.OrganizationID, in.PractitionerRoleID = clinicU, ""
			in.Specialty = "394814009"
		}, want: appointment.ErrWaitlistOff, audited: true},
		{name: "no consent", caller: other, change: func(w *waitlistFixture, _ *appointment.NewEntry) {
			delete(w.consents.state, patientB+clinicA)
		}, want: appointment.ErrConsentRequired, audited: true},
		{name: "five active entries", change: func(w *waitlistFixture, _ *appointment.NewEntry) {
			for range 5 {
				w.entry(patientA, roleA1, 0)
			}
		}, want: appointment.ErrTooManyEntries, audited: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, in, c := newWaitlistFixture(now), joinA(), me
			if tt.caller != (appointment.Caller{}) {
				c = tt.caller
			}
			if tt.change != nil {
				tt.change(w, &in)
			}
			entries := len(w.wl.entries)
			if _, err := w.svc.Join(t.Context(), c, in); !errors.Is(err, tt.want) {
				t.Errorf("Join() = %v, want %v", err, tt.want)
			}
			if len(w.wl.entries) != entries {
				t.Error("an entry was stored")
			}
			var want []auditevent.AuditEvent
			if tt.audited {
				want = []auditevent.AuditEvent{event(c, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure)}
			}
			if !slices.Equal(w.audit.events, want) {
				t.Errorf("audit %+v, want %+v", w.audit.events, want)
			}
		})
	}
}

// An entry that expired for an active entry past its latest day; five expired entries do not count.
func TestJoin_ExpiredEntriesDoNotCount(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	for range 5 {
		id := w.entry(patientA, roleA1, 0)
		w.wl.entries[len(w.wl.entries)-1].EndsAt = now
		_ = id
	}
	if _, err := w.svc.Join(t.Context(), me, joinA()); err != nil {
		t.Errorf("Join() with five expired entries = %v", err)
	}
}

// The patient sees their own entries, newest first, each with its open offer, an active one past its last day expired;
// each is audited.
func TestMyWaitlist_GivesThePatientsEntriesWithTheirOffers(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	old := w.entry(patientA, roleA1, 0)
	w.wl.entries[0].EndsAt = now
	open := w.entry(patientA, roleA1, 0)
	w.entry(patientB, roleA1, 0)
	w.wl.offers = append(w.wl.offers, appointment.Offer{ID: "of", EntryID: open, PatientID: patientA, OrganizationID: clinicA, State: appointment.OfferOpen})
	got, err := w.svc.MyWaitlist(t.Context(), me)
	if err != nil || len(got) != 2 || got[0].ID != open || got[0].Offer.ID != "of" || got[1].ID != old || got[1].Status != appointment.WaitlistExpired {
		t.Fatalf("MyWaitlist() = %+v, %v; want the open one with its offer, then the expired one", got, err)
	}
	want := []auditevent.AuditEvent{
		event(me, auditevent.CodeSearchType, "Appointment", open, auditevent.OutcomeSuccess),
		event(me, auditevent.CodeSearchType, "Appointment", old, auditevent.OutcomeSuccess),
	}
	if !slices.Equal(w.audit.events, want) {
		t.Errorf("audit %+v, want %+v", w.audit.events, want)
	}
	if got, err := w.svc.MyWaitlist(t.Context(), noProfile); err != nil || len(got) != 0 {
		t.Errorf("MyWaitlist(no profile) = %v, %v; want none", got, err)
	}
	if _, err := w.svc.MyWaitlist(t.Context(), admin); !errors.Is(err, appointment.ErrForbidden) {
		t.Errorf("MyWaitlist(staff) = %v, want ErrForbidden", err)
	}
}

// The clinic sees its waitlist in offer order with the names access.Profile gives: the admin every name, a
// practitioner only the names of their own patients; each entry and name is audited.
func TestClinicWaitlist_NamesThePatientsAsProfileAllows(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	w.repo.appts = append(w.repo.appts, booked("a1000000-0000-4000-8000-000000000001", patientA, roleA1, clinicA, start))
	a := w.entry(patientA, roleA1, 0)
	b := w.entry(patientB, roleA2, 0)
	got, err := w.svc.ClinicWaitlist(t.Context(), admin, appointment.WaitlistFilter{})
	if err != nil || len(got) != 2 || !got[0].Named || got[0].Patient != nameA || got[1].Entry.ID != b || !got[1].Named {
		t.Fatalf("ClinicWaitlist(admin) = %+v, %v; want both named", got, err)
	}
	got, err = w.svc.ClinicWaitlist(t.Context(), pract, appointment.WaitlistFilter{})
	if err != nil || len(got) != 2 || !got[0].Named || got[1].Named {
		t.Fatalf("ClinicWaitlist(practitioner A1) = %+v, %v; want only their patient A named", got, err)
	}
	w.audit.events = nil
	got, err = w.svc.ClinicWaitlist(t.Context(), pract, appointment.WaitlistFilter{PractitionerRoleID: roleA1, Status: appointment.WaitlistActive})
	if err != nil || len(got) != 1 || got[0].Entry.ID != a {
		t.Fatalf("ClinicWaitlist(role A1) = %+v, %v; want patient A's entry", got, err)
	}
	want := []auditevent.AuditEvent{
		event(pract, auditevent.CodeSearchType, "Appointment", a, auditevent.OutcomeSuccess),
		event(pract, auditevent.CodeSearchType, "Patient", patientA, auditevent.OutcomeSuccess),
	}
	if !slices.Equal(w.audit.events, want) {
		t.Errorf("audit %+v, want %+v", w.audit.events, want)
	}
	if got, err := w.svc.ClinicWaitlist(t.Context(), admin, appointment.WaitlistFilter{Status: appointment.WaitlistBooked}); err != nil || len(got) != 0 {
		t.Errorf("ClinicWaitlist(booked) = %+v, %v; want none", got, err)
	}
	if got, err := w.svc.ClinicWaitlist(t.Context(), adminB, appointment.WaitlistFilter{}); err != nil || len(got) != 0 {
		t.Errorf("ClinicWaitlist(clinic B) = %+v, %v; want none", got, err)
	}
	if _, err := w.svc.ClinicWaitlist(t.Context(), me, appointment.WaitlistFilter{}); !errors.Is(err, appointment.ErrForbidden) {
		t.Errorf("ClinicWaitlist(a patient) = %v, want ErrForbidden", err)
	}
}

// The patient withdraws their entry and the clinic's staff remove one; its open offer is withdrawn and its slot handed
// on. Nobody else changes an entry.
func TestWithdraw_EndsTheEntryAndHandsItsOfferedSlotOn(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		caller appointment.Caller
		want   error
		name   string
	}{
		{name: "the patient", caller: me},
		{name: "the clinic's admin", caller: admin},
		{name: "a practitioner of the clinic", caller: pract2},
		{name: "another patient", caller: other, want: appointment.ErrNotFound},
		{name: "another clinic", caller: adminB, want: appointment.ErrNotFound},
		{name: "no profile", caller: noProfile, want: appointment.ErrNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newWaitlistFixture(now)
			id := w.entry(patientA, roleA1, 0)
			w.wl.offers = append(w.wl.offers, appointment.Offer{ID: "of", EntryID: id, SlotID: slotFree, PatientID: patientA, OrganizationID: clinicA, State: appointment.OfferOpen})
			err := w.svc.Withdraw(t.Context(), tt.caller, id)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Withdraw() = %v, want %v", err, tt.want)
			}
			outcome, ref := auditevent.OutcomeSuccess, id
			if tt.want != nil {
				outcome, ref = auditevent.OutcomeMinorFailure, ""
				if w.wl.entries[0].Status != appointment.WaitlistActive || w.wl.offers[0].State != appointment.OfferOpen || len(w.jobs.enqueued) != 0 {
					t.Error("a refused withdrawal changed something")
				}
			} else if w.wl.entries[0].Status != appointment.WaitlistWithdrawn || w.wl.offers[0].State != appointment.OfferWithdrawn ||
				!slices.Equal(w.jobs.enqueued, []string{"slot_freed " + slotFree + " " + clinicA}) {
				t.Errorf("entry %+v, offer %+v, jobs %v; want both withdrawn and the slot handed on", w.wl.entries[0], w.wl.offers[0], w.jobs.enqueued)
			}
			if want := []auditevent.AuditEvent{event(tt.caller, auditevent.CodeUpdate, "Appointment", ref, outcome)}; !slices.Equal(w.audit.events, want) {
				t.Errorf("audit %+v, want %+v", w.audit.events, want)
			}
		})
	}
}

func TestSetPriority_StaffMoveAnEntryUp(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	id := w.entry(patientA, roleA1, 0)
	got, err := w.svc.SetPriority(t.Context(), admin, id, 40)
	if err != nil || got.Entry.Priority != 40 || !got.Named {
		t.Fatalf("SetPriority() = %+v, %v; want priority 40, named", got, err)
	}
	if want := event(admin, auditevent.CodeUpdate, "Appointment", id, auditevent.OutcomeSuccess); w.audit.events[0] != want {
		t.Errorf("audit %+v, want first %+v", w.audit.events, want)
	}
	for name, tt := range map[string]struct {
		caller   appointment.Caller
		priority int
		want     error
	}{
		"above 100":      {admin, 101, appointment.ErrInvalidPriority},
		"below 0":        {admin, -1, appointment.ErrInvalidPriority},
		"by the patient": {me, 10, appointment.ErrForbidden},
		"another clinic": {adminB, 10, appointment.ErrNotFound},
	} {
		if _, err := w.svc.SetPriority(t.Context(), tt.caller, id, tt.priority); !errors.Is(err, tt.want) {
			t.Errorf("SetPriority(%s) = %v, want %v", name, err, tt.want)
		}
	}
}

// After a revocation the job withdraws the patient's entries with the clinic and their offers, handing each slot on;
// a consent given again leaves them.
func TestWithdrawForRevokedConsent_EndsThePatientsEntriesThere(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	id := w.entry(patientA, roleA1, 0)
	w.entry(patientB, roleA1, 0)
	w.wl.offers = append(w.wl.offers, appointment.Offer{ID: "of", EntryID: id, SlotID: slotFree, PatientID: patientA, OrganizationID: clinicA, State: appointment.OfferOpen})
	if n, err := w.svc.WithdrawForRevokedConsent(t.Context(), patientA, clinicA); err != nil || n != 0 {
		t.Fatalf("WithdrawForRevokedConsent(active consent) = %d, %v; want none", n, err)
	}
	w.consents.state[patientA+clinicA] = access.Consent{RevokedAt: revokedAt}
	n, err := w.svc.WithdrawForRevokedConsent(t.Context(), patientA, clinicA)
	if err != nil || n != 1 || w.wl.entries[0].Status != appointment.WaitlistWithdrawn || w.wl.entries[1].Status != appointment.WaitlistActive ||
		w.wl.offers[0].State != appointment.OfferWithdrawn || !slices.Equal(w.jobs.enqueued, []string{"slot_freed " + slotFree + " " + clinicA}) {
		t.Fatalf("WithdrawForRevokedConsent() = %d, %v; entries %+v, offers %+v, jobs %v", n, err, w.wl.entries, w.wl.offers, w.jobs.enqueued)
	}
	if want := []auditevent.AuditEvent{jobEvent("consent_revoked", clinicA, auditevent.CodeUpdate, "Appointment", id)}; !slices.Equal(w.audit.events, want) {
		t.Errorf("audit %+v, want %+v", w.audit.events, want)
	}
}

func TestWaitlistPurger_DeletesEntriesAndOffers(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	if n, err := appointment.NewWaitlistEntryPurger(w.wl).Purge(t.Context(), now); err != nil || n != 2 {
		t.Errorf("entries: %d, %v", n, err)
	}
	if n, err := appointment.NewWaitlistOfferPurger(w.wl).Purge(t.Context(), now); err != nil || n != 3 {
		t.Errorf("offers: %d, %v", n, err)
	}
	w.wl.err = errBackend
	if _, err := appointment.NewWaitlistEntryPurger(w.wl).Purge(t.Context(), now); !errors.Is(err, errBackend) {
		t.Errorf("Purge() = %v, want the repository's error", err)
	}
}
