package appointment_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// A practitioner's status and the consultation's outcome are kept in the appointment's history, with who set them.
func TestHistory_KeepsEveryStatusSetAndWho(t *testing.T) {
	f := cancelFixture(now.Add(-time.Hour))
	if _, err := f.svc.SetStatus(t.Context(), pract, apptA, appointment.StatusNoShow); err != nil {
		t.Fatal(err)
	}
	err := inClinic(t, f, clinicA, func(ctx context.Context) error {
		return f.svc.RecordConsultationOutcome(ctx, "consultation_window_close", apptA2, true)
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []appointment.Change{
		{ID: "c0000001-0000-4000-8000-000000000000", At: now, AppointmentID: apptA, Kind: appointment.ChangeStatusSet, Via: appointment.ViaStaff,
			ByAccountID: practAcc, FromStatus: appointment.StatusBooked, ToStatus: appointment.StatusNoShow},
		{ID: "c0000002-0000-4000-8000-000000000000", At: now, AppointmentID: apptA2, Kind: appointment.ChangeStatusSet, Via: appointment.ViaSystem,
			Agent: "consultation_window_close", FromStatus: appointment.StatusBooked, ToStatus: appointment.StatusFulfilled},
	}
	if !slices.Equal(f.repo.changes, want) {
		t.Errorf("history %+v,\nwant %+v", f.repo.changes, want)
	}
}

// A refused status writes no history.
func TestHistory_NothingForARefusedStatus(t *testing.T) {
	f := cancelFixture(now.Add(time.Hour)) // not started
	if _, err := f.svc.SetStatus(t.Context(), pract, apptA, appointment.StatusNoShow); !errors.Is(err, appointment.ErrNotStarted) {
		t.Fatalf("SetStatus() = %v, want ErrNotStarted", err)
	}
	if len(f.repo.changes) != 0 {
		t.Errorf("history %+v, want none", f.repo.changes)
	}
}

// The history is the patient's own without who made each change, and the clinic's in full for those who see the
// appointment in full; anyone else is refused, audited.
func TestChanges_GivesTheHistoryToWhoeverSeesTheAppointmentInFull(t *testing.T) {
	f := cancelFixture(now.Add(-time.Hour))
	if _, err := f.svc.SetStatus(t.Context(), pract, apptA, appointment.StatusFulfilled); err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	for _, tt := range []struct {
		err    error
		name   string
		by     string
		caller appointment.Caller
		event  auditevent.AuditEvent
	}{
		{name: "the patient", caller: me, event: event(me, auditevent.CodeRead, "Appointment", apptA, auditevent.OutcomeSuccess)},
		{name: "the practitioner", caller: pract, by: practAcc, event: event(pract, auditevent.CodeRead, "Appointment", apptA, auditevent.OutcomeSuccess)},
		{name: "the admin", caller: admin, by: practAcc, event: event(admin, auditevent.CodeRead, "Appointment", apptA, auditevent.OutcomeSuccess)},
		{name: "another practitioner", caller: pract2, err: appointment.ErrNotFound, event: event(pract2, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure)},
		{name: "another patient", caller: other, err: appointment.ErrNotFound, event: event(other, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure)},
		{name: "another clinic", caller: adminB, err: appointment.ErrNotFound, event: event(adminB, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure)},
	} {
		f.audit.events = nil
		got, err := f.svc.Changes(t.Context(), tt.caller, apptA)
		switch {
		case tt.err != nil:
			if !errors.Is(err, tt.err) || got != nil {
				t.Errorf("%s: Changes() = %+v, %v; want %v", tt.name, got, err, tt.err)
			}
		case err != nil || len(got) != 1 || got[0].ToStatus != appointment.StatusFulfilled || got[0].ByAccountID != tt.by:
			t.Errorf("%s: Changes() = %+v, %v; want the status set, by %q", tt.name, got, err, tt.by)
		}
		checkEvents(t, tt.name, f.audit, tt.event)
	}
}

// A link is 32 random bytes in the URL, stored only as its SHA-256, bound to the appointment, its patient and account,
// expiring at the start, and audited with the job as agent.
func TestIssueLink_StoresTheHashOfAOneTimeLinkForABookedAppointment(t *testing.T) {
	f := cancelFixture(start)
	link := appointment.Link{AppointmentID: apptA, OrganizationID: clinicA, Language: "bs", Channel: appointment.ChannelEmail}
	url, err := f.svc.IssueLink(t.Context(), "appointment_reminder", link)
	if err != nil || !strings.HasPrefix(url, "https://api.salus.invalid/a/") {
		t.Fatalf("IssueLink() = %q, %v; want a link under the base", url, err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(url, "https://api.salus.invalid/a/"))
	if err != nil || len(raw) != 32 {
		t.Fatalf("token %q: %d bytes, %v; want 32", url, len(raw), err)
	}
	want := appointment.Token{
		CreatedAt: now, ExpiresAt: start, AppointmentID: apptA, OrganizationID: clinicA, PatientID: patientA, AccountID: accountA,
		Language: "bs", Channel: appointment.ChannelEmail, Hash: sha256.Sum256(raw),
	}
	if !slices.Equal(f.tokens.stored, []appointment.Token{want}) {
		t.Errorf("stored %+v,\nwant %+v", f.tokens.stored, want)
	}
	if f.repo.clinic != clinicA || f.repo.notice != apptA {
		t.Errorf("scope clinic %q notice %q, want the job's appointment in clinic A", f.repo.clinic, f.repo.notice)
	}
	checkEvents(t, "IssueLink", f.audit, jobEvent("appointment_reminder", clinicA, auditevent.CodeCreate, "Appointment", apptA))
	if again, _ := f.svc.IssueLink(t.Context(), "appointment_reminder", link); again == url {
		t.Error("two links share a token")
	}
}

func TestIssueLink_RefusesAnAppointmentThatIsNotBookedAheadOrNotTheClinics(t *testing.T) {
	for _, tt := range []struct {
		err   error
		name  string
		setup func(f *fixture)
		org   string
	}{
		{name: "cancelled", err: appointment.ErrClosed, org: clinicA, setup: func(f *fixture) { f.repo.appts[0].Status = appointment.StatusCancelled }},
		{name: "started", err: appointment.ErrClosed, org: clinicA, setup: func(f *fixture) { f.repo.appts[0].Start = now }},
		{name: "another clinic's", err: appointment.ErrNotFound, org: clinicB, setup: func(*fixture) {}},
	} {
		f := cancelFixture(start)
		tt.setup(f)
		_, err := f.svc.IssueLink(t.Context(), "appointment_reminder", appointment.Link{AppointmentID: apptA, OrganizationID: tt.org, Language: "bs", Channel: appointment.ChannelSMS})
		if !errors.Is(err, tt.err) || len(f.tokens.stored) != 0 {
			t.Errorf("%s: IssueLink() = %v with %d stored; want %v and none", tt.name, err, len(f.tokens.stored), tt.err)
		}
		refusal := jobEvent("appointment_reminder", tt.org, auditevent.CodeCreate, "", "")
		refusal.Outcome = auditevent.OutcomeMinorFailure
		checkEvents(t, tt.name, f.audit, refusal)
	}
}

func TestIssueLink_NeedsAChannel(t *testing.T) {
	f := cancelFixture(start)
	if _, err := f.svc.IssueLink(t.Context(), "appointment_reminder", appointment.Link{AppointmentID: apptA, OrganizationID: clinicA, Language: "bs"}); err == nil {
		t.Error("IssueLink() without a channel succeeded")
	}
}

func TestTokenPurger_DeletesTheLinksExpiredBefore(t *testing.T) {
	tokens := &fakeTokens{stored: []appointment.Token{{ExpiresAt: now.Add(-time.Hour)}, {ExpiresAt: now.Add(time.Hour)}}}
	n, err := appointment.NewTokenPurger(tokens).Purge(t.Context(), now)
	if err != nil || n != 1 || len(tokens.stored) != 1 {
		t.Errorf("Purge() = %d, %v with %d left; want 1 and 1", n, err, len(tokens.stored))
	}
}
