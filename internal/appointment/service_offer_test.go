package appointment_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/payments/paymentstest"
)

// A freed slot is held for the first matching entry for the clinic's hold (15 minutes by default): the slot is
// busy-tentative, the offer stored with its expiry and the patient's notice enqueued, the offer audited with the job as
// agent. The match carries the slot's day and times in the clinic's zone.
func TestHandOn_OffersAFreedSlotToTheFirstMatchingEntry(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	first := w.entry(patientA, roleA1, 10)
	w.entry(patientB, roleA1, 0)
	w.wl.cands = []string{first, w.wl.entries[1].ID}
	if err := w.svc.HandOn(t.Context(), clinicA, slotFree); err != nil {
		t.Fatal(err)
	}
	if len(w.wl.offers) != 1 {
		t.Fatalf("offers %+v, want one", w.wl.offers)
	}
	o := w.wl.offers[0]
	if o.EntryID != first || o.PatientID != patientA || o.SlotID != slotFree || o.PractitionerRoleID != roleA1 || !o.ExpiresAt.Equal(now.Add(15*time.Minute)) ||
		!o.Start.Equal(start) || w.offered.status(slotFree) != slot.StatusTentative {
		t.Errorf("offer %+v, slot %s; want patient A's entry held 15 minutes, the slot busy-tentative", o, w.offered.status(slotFree))
	}
	wantJobs := []string{"offer_expire " + o.ID + " " + clinicA + " at " + now.Add(15*time.Minute).Format(time.RFC3339), "offer_notify " + o.ID + " " + clinicA}
	if !slices.Equal(w.jobs.enqueued, wantJobs) {
		t.Errorf("jobs %v, want %v", w.jobs.enqueued, wantJobs)
	}
	m := w.wl.matched[0]
	if !m.Day.Equal(day(start)) || m.Times != (appointment.Times{Start: 600, End: 630}) || m.PractitionerRoleID != roleA1 || m.SlotID != slotFree || !m.Now.Equal(now) {
		t.Errorf("match %+v, want 10:00 to 10:30 on the 7th in Sarajevo", m)
	}
	want := []auditevent.AuditEvent{
		jobEvent("waitlist_slot_freed", clinicA, auditevent.CodeSearchType, "", ""),
		jobEvent("waitlist_slot_freed", clinicA, auditevent.CodeUpdate, "Appointment", first),
	}
	if !slices.Equal(w.audit.events, want) {
		t.Errorf("audit %+v, want %+v", w.audit.events, want)
	}
	// A second job for the same slot finds it held and changes nothing.
	if err := w.svc.HandOn(t.Context(), clinicA, slotFree); err != nil || len(w.wl.offers) != 1 || len(w.jobs.enqueued) != 2 {
		t.Errorf("HandOn() again = %v, offers %+v; want nothing more", err, w.wl.offers)
	}
}

// An entry whose patient has no active care consent, or is inside their quiet hours, is passed over for the offer.
func TestHandOn_PassesOverWhoMayNotBeOfferedNow(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	a := w.entry(patientA, roleA1, 0)
	b := w.entry(patientB, roleA1, 0)
	w.wl.cands = []string{a, b}
	w.quiet.quiet[patientA] = true
	if err := w.svc.HandOn(t.Context(), clinicA, slotFree); err != nil || len(w.wl.offers) != 1 || w.wl.offers[0].EntryID != b {
		t.Fatalf("HandOn() = %v, offers %+v; want patient B's entry, A in quiet hours", err, w.wl.offers)
	}
	w = newWaitlistFixture(now)
	a = w.entry(patientA, roleA1, 0)
	w.wl.cands = []string{a}
	w.consents.state[patientA+clinicA] = access.Consent{}
	// The fake shows the entry as if it were still seen: the service checks the consent itself.
	w.wl.consents = &fakeConsents{state: map[string]access.Consent{patientA + clinicA: {Active: true}}}
	if err := w.svc.HandOn(t.Context(), clinicA, slotFree); err != nil || len(w.wl.offers) != 0 || w.offered.status(slotFree) != slot.StatusFree {
		t.Errorf("HandOn() = %v, offers %+v, slot %s; want none and the slot free: the consent is not active", err, w.wl.offers, w.offered.status(slotFree))
	}
}

// A slot stays as it is while the clinic's waitlist is off, when it starts within the hold and an hour, is booked, is
// another clinic's, or an open offer holds it; with no entry matching, it stays free.
func TestHandOn_LeavesASlotThatCannotBeOffered(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		change func(w *waitlistFixture)
		name   string
		at     time.Time
		org    string
		want   slot.Status
	}{
		{name: "the waitlist off", change: func(w *waitlistFixture) { w.settings.off[clinicA] = true }, want: slot.StatusFree},
		{name: "starting within the hold and an hour", at: start.Add(-75*time.Minute + time.Second), want: slot.StatusFree},
		{name: "booked", change: func(w *waitlistFixture) { setStatus(w, slot.StatusBusy) }, want: slot.StatusBusy},
		{name: "another clinic's", org: clinicB, want: slot.StatusFree},
		{name: "held by an open offer", change: func(w *waitlistFixture) {
			setStatus(w, slot.StatusTentative)
			w.wl.offers = append(w.wl.offers, appointment.Offer{ID: "of", EntryID: "e", SlotID: slotFree, OrganizationID: clinicA, State: appointment.OfferOpen})
		}, want: slot.StatusTentative},
		{name: "no entry matching", change: func(w *waitlistFixture) { w.wl.cands = nil }, want: slot.StatusFree},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			at := now
			if !tt.at.IsZero() {
				at = tt.at
			}
			w := newWaitlistFixture(at)
			w.wl.cands = []string{w.entry(patientA, roleA1, 0)}
			if tt.change != nil {
				tt.change(w)
			}
			org := clinicA
			if tt.org != "" {
				org = tt.org
			}
			offers := len(w.wl.offers)
			if err := w.svc.HandOn(t.Context(), org, slotFree); err != nil {
				t.Fatal(err)
			}
			if len(w.wl.offers) != offers || w.offered.status(slotFree) != tt.want || slices.ContainsFunc(w.jobs.enqueued, func(j string) bool { return strings.HasPrefix(j, "offer_") }) {
				t.Errorf("offers %+v, slot %s, jobs %v; want no new offer and the slot %s", w.wl.offers, w.offered.status(slotFree), w.jobs.enqueued, tt.want)
			}
		})
	}
}

func fixedAt(t time.Time) clock.Fixed { return clock.Fixed{T: t} }

func setStatus(w *waitlistFixture, s slot.Status) {
	b := w.offered.slots[slotFree]
	b.Status = s
	w.offered.slots[slotFree] = b
}

// A held slot whose offer closed (declined, withdrawn) goes to the next entry; one never offered it before.
func TestHandOn_GivesAHeldSlotWhoseOfferClosedToTheNextEntry(t *testing.T) {
	t.Parallel()
	w := newWaitlistFixture(now)
	a := w.entry(patientA, roleA1, 0)
	b := w.entry(patientB, roleA1, 0)
	w.wl.cands = []string{a, b}
	setStatus(w, slot.StatusTentative)
	w.wl.offers = append(w.wl.offers, appointment.Offer{ID: "of", EntryID: a, SlotID: slotFree, PatientID: patientA, OrganizationID: clinicA, State: appointment.OfferDeclined})
	if err := w.svc.HandOn(t.Context(), clinicA, slotFree); err != nil {
		t.Fatal(err)
	}
	if len(w.wl.offers) != 2 || w.wl.offers[1].EntryID != b || w.offered.status(slotFree) != slot.StatusTentative {
		t.Errorf("offers %+v, slot %s; want patient B offered the held slot", w.wl.offers, w.offered.status(slotFree))
	}
}

// offered is the fixture with patient A's entry offered slotFree until 15 minutes after now, and patient B's entry next.
func offered(t *testing.T, at time.Time) (w *waitlistFixture, offer, next string) {
	t.Helper()
	w = newWaitlistFixture(now)
	a := w.entry(patientA, roleA1, 0)
	w.wl.cands = []string{a, w.entry(patientB, roleA1, 0)}
	if err := w.svc.HandOn(t.Context(), clinicA, slotFree); err != nil {
		t.Fatal(err)
	}
	w.jobs.enqueued, w.audit.events = nil, nil
	return w.with(func(p *appointment.Ports) { p.Clock = fixedAt(at) }), w.wl.offers[0].ID, w.wl.entries[1].ID
}

// An open offer whose hold is over expires and its slot goes to the next entry, the first entry staying active; before
// its end the job is not due; an offer that closed meanwhile only hands its slot on.
func TestExpireOffer_HandsTheSlotToTheNextEntry(t *testing.T) {
	t.Parallel()
	w, id, next := offered(t, now.Add(15*time.Minute))
	if err := w.svc.ExpireOffer(t.Context(), clinicA, id); err != nil {
		t.Fatal(err)
	}
	if w.wl.offers[0].State != appointment.OfferExpired || len(w.wl.offers) != 2 || w.wl.offers[1].EntryID != next ||
		w.wl.entries[0].Status != appointment.WaitlistActive || w.offered.status(slotFree) != slot.StatusTentative {
		t.Errorf("offers %+v, entries %+v; want the first expired and patient B offered the slot", w.wl.offers, w.wl.entries)
	}
	if want := jobEvent("waitlist_offer_expire", clinicA, auditevent.CodeUpdate, "Appointment", w.wl.entries[0].ID); w.audit.events[0] != want {
		t.Errorf("audit %+v, want first %+v", w.audit.events, want)
	}
	w, id, _ = offered(t, now.Add(15*time.Minute-time.Second))
	if err := w.svc.ExpireOffer(t.Context(), clinicA, id); !errors.Is(err, appointment.ErrOfferNotDue) || w.wl.offers[0].State != appointment.OfferOpen {
		t.Errorf("ExpireOffer() before the end = %v, offer %+v; want ErrOfferNotDue", err, w.wl.offers[0])
	}
	if err := w.svc.ExpireOffer(t.Context(), clinicA, "unknown"); err != nil {
		t.Errorf("ExpireOffer(gone) = %v, want nil", err)
	}
}

// Accepting books the offered slot: the appointment (booked via the waitlist) is stored, the slot busy, the offer
// accepted and the entry booked, the booking's jobs enqueued and the creation audited.
func TestAccept_BooksTheOfferedSlot(t *testing.T) {
	t.Parallel()
	w, id, _ := offered(t, now.Add(time.Minute))
	got, err := w.svc.Accept(t.Context(), me, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.BookedVia != appointment.BookedWaitlist || got.SlotID != slotFree || !got.Start.Equal(start) || got.PractitionerRoleID != roleA1 || got.PatientID != patientA {
		t.Errorf("Accept() = %+v, want patient A's appointment of the slot, booked via the waitlist", got)
	}
	if w.offered.status(slotFree) != slot.StatusBusy || w.wl.offers[0].State != appointment.OfferAccepted || w.wl.offers[0].AppointmentID != got.ID ||
		w.wl.entries[0].Status != appointment.WaitlistBooked {
		t.Errorf("slot %s, offer %+v, entry %+v; want busy, accepted, booked", w.offered.status(slotFree), w.wl.offers[0], w.wl.entries[0])
	}
	if !slices.Contains(w.jobs.enqueued, "booked "+got.ID+" "+clinicA) || !slices.Contains(w.jobs.enqueued, "questionnaire_request "+got.ID+" "+clinicA) {
		t.Errorf("jobs %v, want the booking's", w.jobs.enqueued)
	}
	if want := []auditevent.AuditEvent{event(me, auditevent.CodeCreate, "Appointment", got.ID, auditevent.OutcomeSuccess)}; !slices.Equal(w.audit.events, want) {
		t.Errorf("audit %+v, want %+v", w.audit.events, want)
	}
}

func TestAccept_RefusesWhatCannotBeBooked(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		change func(w *waitlistFixture, id string)
		caller appointment.Caller
		want   error
		name   string
		at     time.Duration
	}{
		{name: "staff", caller: admin, want: appointment.ErrForbidden},
		{name: "no profile", caller: noProfile, want: appointment.ErrProfileRequired},
		{name: "another patient's offer", caller: other, want: appointment.ErrNotFound},
		{name: "after the hold", at: 15 * time.Minute, want: appointment.ErrOfferExpired},
		{name: "a closed offer", change: func(w *waitlistFixture, _ string) { w.wl.offers[0].State = appointment.OfferExpired }, want: appointment.ErrOfferExpired},
		{name: "the consent revoked", change: func(w *waitlistFixture, _ string) {
			w.consents.state[patientA+clinicA] = access.Consent{RevokedAt: now}
		}, want: appointment.ErrConsentRequired},
		{name: "the slot no longer held", change: func(w *waitlistFixture, _ string) { setStatus(w, slot.StatusFree) }, want: appointment.ErrSlotUnavailable},
		{name: "the role booked at that time", change: func(w *waitlistFixture, _ string) {
			w.repo.appts = append(w.repo.appts, booked("a1000000-0000-4000-8000-000000000001", patientB, roleA1, clinicA, start))
		}, want: appointment.ErrSlotUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, id, _ := offered(t, now.Add(time.Minute+tt.at))
			c := me
			if tt.caller != (appointment.Caller{}) {
				c = tt.caller
			}
			if tt.change != nil {
				tt.change(w, id)
			}
			if _, err := w.svc.Accept(t.Context(), c, id); !errors.Is(err, tt.want) {
				t.Fatalf("Accept() = %v, want %v", err, tt.want)
			}
			if w.wl.entries[0].Status != appointment.WaitlistActive || slices.ContainsFunc(w.jobs.enqueued, func(j string) bool { return strings.HasPrefix(j, "booked") }) {
				t.Error("a refused acceptance booked")
			}
			if want := []auditevent.AuditEvent{event(c, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure)}; !slices.Equal(w.audit.events, want) {
				t.Errorf("audit %+v, want %+v", w.audit.events, want)
			}
		})
	}
}

// A priced acceptance is charged before its transaction under the offer's key, and refunded when the booking then
// fails (pkg/payments' fake; nothing has a price before SAL-23).
func TestAccept_ChargesBeforeAndRefundsWhenTheBookingFails(t *testing.T) {
	t.Parallel()
	w, id, _ := offered(t, now.Add(time.Minute))
	provider := &paymentstest.Provider{}
	w.with(func(p *appointment.Ports) { p.Prices, p.Payments = fakePrices{amount: 3000}, provider })
	setStatus(w, slot.StatusFree) // the booking fails after the charge
	if _, err := w.svc.Accept(t.Context(), me, id); !errors.Is(err, appointment.ErrSlotUnavailable) {
		t.Fatalf("Accept() = %v, want ErrSlotUnavailable", err)
	}
	got := provider.Charged()
	if len(got) != 1 || got[0].Key != "waitlist:"+id || got[0].AmountMinor != 3000 || provider.Refunded(got[0].ID) != 3000 {
		t.Errorf("charged %+v, want 3000 under the offer's key, refunded", got)
	}
}

// Declining ends the offer and hands the slot on; the entry stays active. Only the patient's own open offer.
func TestDecline_HandsTheSlotOn(t *testing.T) {
	t.Parallel()
	w, id, _ := offered(t, now.Add(time.Minute))
	if err := w.svc.Decline(t.Context(), other, id); !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("Decline(another patient's) = %v, want ErrNotFound", err)
	}
	if err := w.svc.Decline(t.Context(), me, id); err != nil {
		t.Fatal(err)
	}
	if w.wl.offers[0].State != appointment.OfferDeclined || w.wl.entries[0].Status != appointment.WaitlistActive ||
		!slices.Equal(w.jobs.enqueued, []string{"slot_freed " + slotFree + " " + clinicA}) {
		t.Errorf("offer %+v, entry %+v, jobs %v; want declined, active, the slot handed on", w.wl.offers[0], w.wl.entries[0], w.jobs.enqueued)
	}
	if err := w.svc.Decline(t.Context(), me, id); !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("Decline(again) = %v, want ErrNotFound", err)
	}
	if err := w.svc.Decline(t.Context(), admin, id); !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("Decline(staff) = %v, want ErrNotFound", err)
	}
}

// The notifier reads the offer and its patient's account, audited with the job as agent.
func TestOfferNotice_GivesTheOfferAndTheAccount(t *testing.T) {
	t.Parallel()
	w, id, _ := offered(t, now)
	got, err := w.svc.OfferNotice(t.Context(), "waitlist_offer_notify", clinicA, id)
	if err != nil || got.AccountID != accountA || got.Offer.ID != id {
		t.Fatalf("OfferNotice() = %+v, %v; want the offer and patient A's account", got, err)
	}
	want := []auditevent.AuditEvent{
		jobEvent("waitlist_offer_notify", clinicA, auditevent.CodeRead, "Appointment", w.wl.entries[0].ID),
		jobEvent("waitlist_offer_notify", clinicA, auditevent.CodeRead, "Patient", patientA),
	}
	if !slices.Equal(w.audit.events, want) {
		t.Errorf("audit %+v, want %+v", w.audit.events, want)
	}
	if _, err := w.svc.OfferNotice(t.Context(), "waitlist_offer_notify", clinicB, id); !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("OfferNotice(another clinic) = %v, want ErrNotFound", err)
	}
}
