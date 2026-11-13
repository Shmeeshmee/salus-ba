package appointment_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

const (
	tokA    = "70000000-0000-4000-8000-0000000000a1"
	tokB    = "70000000-0000-4000-8000-0000000000b1"
	address = "203.0.113.7"
)

// linkEvent is the audit event of interaction by patient A's account through a one-time link.
func linkEvent(interaction, resource, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event(me, interaction, resource, id, outcome)
	e.AgentName = "action_link"
	return e
}

func TestLink_ShowsTheClinicTheTimeAndTheActionsAndChangesNothing(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	f := newSelfFixture(a)
	token := f.link(a, tokA, "hr")
	for range 5 { // a mail scanner, a link preview, the patient
		got, err := f.svc.Link(t.Context(), token, address)
		sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
		if err != nil || got.Clinic != "Synthetic clinic" || !got.Start.Equal(start) || got.Start.Location().String() != sarajevo.String() ||
			got.Language != "hr" || got.Confirmed || !got.CanConfirm || !got.CanCancel || got.CancelClosed || got.Used != "" {
			t.Fatalf("Link() = %+v, %v; want clinic A's name, the start in Sarajevo, confirm and cancel offered", got, err)
		}
	}
	if f.repo.appts[0] != a || f.self.tokens[0].UsedAt != (time.Time{}) || len(f.repo.changes) != 0 || len(f.jobs.enqueued) != 0 {
		t.Errorf("appointment %+v, link %+v, history %v, jobs %v; want nothing changed by a GET", f.repo.appts[0], f.self.tokens[0], f.repo.changes, f.jobs.enqueued)
	}
	if len(f.audit.events) != 5 || f.audit.events[0] != linkEvent(auditevent.CodeRead, "Appointment", apptA, auditevent.OutcomeSuccess) {
		t.Errorf("audited %+v, want each read by patient A's account through the link", f.audit.events)
	}
}

func TestLink_InsideThe24HoursNoCancelAndAConfirmedOneOffersOnlyCancel(t *testing.T) {
	soon := booked(apptA, patientA, roleA1, clinicA, now.Add(5*time.Hour))
	f := newSelfFixture(soon)
	got, err := f.svc.Link(t.Context(), f.link(soon, tokA, "en"), address)
	if err != nil || !got.CanConfirm || got.CanCancel || !got.CancelClosed {
		t.Errorf("Link() inside the 24 hours = %+v, %v; want confirm only and the clinic to contact", got, err)
	}
	confirmed := booked(apptA, patientA, roleA1, clinicA, start)
	confirmed.ConfirmedAt, confirmed.ConfirmedVia = now.Add(-time.Hour), appointment.ViaApp
	f = newSelfFixture(confirmed)
	got, err = f.svc.Link(t.Context(), f.link(confirmed, tokA, "en"), address)
	if err != nil || !got.Confirmed || got.CanConfirm || !got.CanCancel {
		t.Errorf("Link() of a confirmed one = %+v, %v; want it shown confirmed with cancel offered", got, err)
	}
}

func TestUseLink_ConfirmsOnceAsThePatientThroughTheLink(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	f := newSelfFixture(a)
	token := f.link(a, tokA, "de")
	if err := f.svc.UseLink(t.Context(), token, address, appointment.LinkConfirm, false); err != nil {
		t.Fatalf("UseLink(confirm) = %v", err)
	}
	if got := f.repo.appts[0]; !got.ConfirmedAt.Equal(now) || got.ConfirmedVia != appointment.ViaLink {
		t.Errorf("appointment %+v, want confirmed through the link now", got)
	}
	if tok := f.self.tokens[0]; !tok.UsedAt.Equal(now) || tok.UsedAction != appointment.LinkConfirm {
		t.Errorf("link %+v, want used for confirm", tok)
	}
	if len(f.repo.changes) != 1 || f.repo.changes[0].Via != appointment.ViaLink || f.repo.changes[0].ByAccountID != accountA {
		t.Errorf("history %+v, want the confirmation via the link by patient A's account", f.repo.changes)
	}
	// A second POST (a double tap, a scanner after the patient) is refused, and the page shows the outcome.
	for _, action := range []appointment.LinkAction{appointment.LinkConfirm, appointment.LinkCancel} {
		if err := f.svc.UseLink(t.Context(), token, address, action, true); !errors.Is(err, appointment.ErrLinkInvalid) && !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("UseLink(%s) again = %v, want refused", action, err)
		}
	}
	if f.repo.appts[0].Status != appointment.StatusBooked || len(f.repo.changes) != 1 {
		t.Errorf("appointment %+v, history %d; want it still booked, nothing more recorded", f.repo.appts[0], len(f.repo.changes))
	}
	got, err := f.svc.Link(t.Context(), token, address)
	if err != nil || got.Used != appointment.LinkConfirm || got.CanConfirm || got.CanCancel || got.Clinic == "" || !got.Start.Equal(start) {
		t.Errorf("Link() after use = %+v, %v; want only the outcome, with the clinic and the time", got, err)
	}
	checkEvents(t, "UseLink", f.audit,
		linkEvent(auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess),
		linkEvent(auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure),
		linkEvent(auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure),
		linkEvent(auditevent.CodeRead, "Appointment", apptA, auditevent.OutcomeSuccess))
}

func TestUseLink_CancelsWithTheTickedBoxRecordedViaTheLink(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	f := newSelfFixture(a)
	token := f.link(a, tokA, "en")
	if err := f.svc.UseLink(t.Context(), token, address, appointment.LinkCancel, false); !errors.Is(err, appointment.ErrConfirmCancelRequired) {
		t.Fatalf("UseLink(cancel) without the box = %v, want ErrConfirmCancelRequired", err)
	}
	if f.repo.appts[0].Status != appointment.StatusBooked || !f.self.tokens[0].UsedAt.IsZero() {
		t.Fatalf("a cancel without the box changed %+v / %+v", f.repo.appts[0], f.self.tokens[0])
	}
	if err := f.svc.UseLink(t.Context(), token, address, appointment.LinkCancel, true); err != nil {
		t.Fatalf("UseLink(cancel) = %v", err)
	}
	got := f.repo.appts[0]
	if got.Status != appointment.StatusCancelled || got.Reason != appointment.ReasonPatient || got.CancelledBy != accountA {
		t.Errorf("appointment %+v, want cancelled by patient A (pat)", got)
	}
	if len(f.repo.changes) != 1 || f.repo.changes[0].Via != appointment.ViaLink || f.repo.changes[0].ToStatus != appointment.StatusCancelled {
		t.Errorf("history %+v, want the cancellation via the link", f.repo.changes)
	}
	if !strings.Contains(strings.Join(f.jobs.enqueued, ","), "cancelled "+apptA) {
		t.Errorf("enqueued %v, want the cancellation e-mail", f.jobs.enqueued)
	}
	if tok := f.self.tokens[0]; tok.UsedAction != appointment.LinkCancel {
		t.Errorf("link %+v, want used for cancel", tok)
	}
}

func TestUseLink_ARefusedCancelLeavesTheLinkUnused(t *testing.T) {
	soon := booked(apptA, patientA, roleA1, clinicA, now.Add(5*time.Hour))
	f := newSelfFixture(soon)
	token := f.link(soon, tokA, "en")
	if err := f.svc.UseLink(t.Context(), token, address, appointment.LinkCancel, true); !errors.Is(err, appointment.ErrCancellationWindowClosed) {
		t.Fatalf("UseLink(cancel) inside the 24 hours = %v, want ErrCancellationWindowClosed", err)
	}
	if !f.self.tokens[0].UsedAt.IsZero() || f.repo.appts[0].Status != appointment.StatusBooked {
		t.Errorf("link %+v, appointment %+v; want both unchanged", f.self.tokens[0], f.repo.appts[0])
	}
	if err := f.svc.UseLink(t.Context(), token, address, appointment.LinkConfirm, false); err != nil {
		t.Errorf("UseLink(confirm) after = %v, want the link still usable", err)
	}
	checkEvents(t, "UseLink", f.audit,
		linkEvent(auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure),
		linkEvent(auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess))
}

func TestLink_LinksThatNoLongerWork(t *testing.T) {
	for _, tc := range []struct {
		setup func(f *selfFixture, a *appointment.Appointment) string
		name  string
	}{
		{name: "unknown", setup: func(*selfFixture, *appointment.Appointment) string {
			return "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		}},
		{name: "not a token", setup: func(*selfFixture, *appointment.Appointment) string { return "../etc" }},
		{name: "expired", setup: func(f *selfFixture, a *appointment.Appointment) string {
			tok := f.link(*a, tokA, "en")
			f.self.tokens[0].StartAt = now.Add(-time.Minute)
			return tok
		}},
		{name: "cancelled", setup: func(f *selfFixture, a *appointment.Appointment) string {
			f.repo.appts[0].Status = appointment.StatusCancelled
			return f.link(*a, tokA, "en")
		}},
		{name: "rescheduled since", setup: func(f *selfFixture, a *appointment.Appointment) string {
			tok := f.link(*a, tokA, "en")
			f.repo.appts[0].Start = start.Add(48 * time.Hour)
			return tok
		}},
		{name: "consent revoked", setup: func(f *selfFixture, a *appointment.Appointment) string {
			f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: revokedAt}
			return f.link(*a, tokA, "en")
		}},
		{name: "another appointment's id in the link", setup: func(f *selfFixture, a *appointment.Appointment) string {
			tok := f.link(*a, tokA, "en")
			f.self.tokens[0].AppointmentID = apptB // patient B's appointment: the link's patient does not see it
			return tok
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := booked(apptA, patientA, roleA1, clinicA, start)
			f := newSelfFixture(a, booked(apptB, patientB, roleA1, clinicA, start.Add(time.Hour)))
			token := tc.setup(f, &a)
			if _, err := f.svc.Link(t.Context(), token, address); !errors.Is(err, appointment.ErrLinkInvalid) {
				t.Errorf("Link() = %v, want ErrLinkInvalid", err)
			}
			for _, action := range []appointment.LinkAction{appointment.LinkConfirm, appointment.LinkCancel} {
				if err := f.svc.UseLink(t.Context(), token, address, action, true); !errors.Is(err, appointment.ErrLinkInvalid) {
					t.Errorf("UseLink(%s) = %v, want ErrLinkInvalid", action, err)
				}
			}
			if len(f.repo.changes) != 0 || f.repo.appts[1].Status != appointment.StatusBooked || (len(f.self.tokens) > 0 && !f.self.tokens[0].UsedAt.IsZero()) {
				t.Errorf("history %v, B %+v, links %+v; want nothing changed", f.repo.changes, f.repo.appts[1], f.self.tokens)
			}
			for _, e := range f.audit.events {
				if e.Outcome == auditevent.OutcomeSuccess {
					t.Errorf("audited %+v, want refusals only", e)
				}
			}
		})
	}
}

func TestLink_RateLimitedPerAddressUnderAnHMAC(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	f := newSelfFixture(a)
	token := f.link(a, tokA, "en")
	var limited int
	for range 40 {
		if _, err := f.svc.Link(t.Context(), token, address); errors.Is(err, appointment.ErrLinkRateLimited) {
			limited++
		}
	}
	if limited != 10 {
		t.Errorf("%d of 40 lookups limited, want the 10 past 30", limited)
	}
	if err := f.svc.UseLink(t.Context(), token, address, appointment.LinkConfirm, false); !errors.Is(err, appointment.ErrLinkRateLimited) {
		t.Errorf("UseLink() past the limit = %v, want ErrLinkRateLimited", err)
	}
	if _, err := f.svc.Link(t.Context(), token, "198.51.100.1"); err != nil {
		t.Errorf("Link() from another address = %v, want it allowed", err)
	}
	for key := range f.limiter.counts {
		if strings.Contains(key, address) || strings.Contains(key, token) {
			t.Errorf("counter key %q carries the address or the token", key)
		}
	}
}

func TestLink_ATokenIsNeverInAnAuditEvent(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	f := newSelfFixture(a)
	token := f.link(a, tokA, "en")
	_, _ = f.svc.Link(t.Context(), token, address)
	_ = f.svc.UseLink(t.Context(), token, address, appointment.LinkConfirm, false)
	for _, e := range f.audit.events {
		if s := strings.Join([]string{e.EntityReference, e.AgentName, e.AgentAccountID, e.OrganizationID}, " "); strings.Contains(s, token) || strings.Contains(s, tokA) {
			t.Errorf("audit event %+v names the link", e)
		}
	}
}

func TestUseLink_ADatabaseFailureIsNotARefusal(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	f := newSelfFixture(a)
	token := f.link(a, tokB, "en")
	f.clinics.err = errBackend
	if _, err := f.svc.Link(t.Context(), token, address); !errors.Is(err, errBackend) {
		t.Errorf("Link() = %v, want the backend's error", err)
	}
	if err := f.svc.UseLink(t.Context(), token, address, appointment.LinkConfirm, false); !errors.Is(err, errBackend) {
		t.Errorf("UseLink() = %v, want the backend's error", err)
	}
	f.clinics.err, f.limiter.err = nil, errBackend
	if _, err := f.svc.Link(t.Context(), token, address); !errors.Is(err, errBackend) {
		t.Errorf("Link() with Redis down = %v, want the backend's error", err)
	}
}

// A reschedule that commits after the page was read and before the link's action leaves the link issued for the old
// time useless: its use reads the appointment under its lock.
func TestUseLink_ARescheduleCommittedFirstLeavesTheOldLinkUseless(t *testing.T) {
	for _, action := range []appointment.LinkAction{appointment.LinkConfirm, appointment.LinkCancel} {
		a := booked(apptA, patientA, roleA1, clinicA, start)
		f := newSelfFixture(a)
		token := f.link(a, tokA, "en")
		f.self.onLock = func() {
			f.repo.appts[0].Start, f.repo.appts[0].End = start.Add(48*time.Hour), start.Add(48*time.Hour+30*time.Minute)
		}
		if err := f.svc.UseLink(t.Context(), token, address, action, true); !errors.Is(err, appointment.ErrLinkInvalid) {
			t.Errorf("UseLink(%s) after a reschedule = %v, want ErrLinkInvalid", action, err)
		}
		if got := f.repo.appts[0]; got.Status != appointment.StatusBooked || !got.ConfirmedAt.IsZero() || !f.self.tokens[0].UsedAt.IsZero() {
			t.Errorf("UseLink(%s) acted on the moved appointment: %+v, link %+v", action, got, f.self.tokens[0])
		}
	}
}

// IPv6 lookups count per /64, the block one subscriber holds; IPv4 per address.
func TestLink_RateLimitedPerIPv6Slash64(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	f := newSelfFixture(a)
	token := f.link(a, tokA, "en")
	for _, addr := range []string{"2001:db8:1:2::1", "2001:db8:1:2:ffff::9", "2001:db8:1:3::1", "203.0.113.7", "203.0.113.8"} {
		_, _ = f.svc.Link(t.Context(), token, addr)
	}
	if len(f.limiter.counts) != 4 {
		t.Errorf("counters %v, want one for the shared /64, one for the other /64 and one per IPv4 address", f.limiter.counts)
	}
	for key, n := range f.limiter.counts {
		if strings.Contains(key, "2001") || strings.Contains(key, "203.0") {
			t.Errorf("counter key %q (%d) carries an address", key, n)
		}
	}
}
