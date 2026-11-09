package auth_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
)

// A job that mails a patient reads the account's e-mail; the read is audited with the job as agent.
func TestAccountEmail_GivesTheJobTheAddressAndAuditsTheReadWithTheJobAsAgent(t *testing.T) {
	f := newFixture(staff())
	got, _, err := f.svc.AccountEmail(t.Context(), "appointment_booked_email", accountID)
	if err != nil || got != "ben@example.test" {
		t.Fatalf("AccountEmail() = %q, %v; want the account's e-mail", got, err)
	}
	want := auditevent.AuditEvent{
		Type: rest, Subtype: read, Purpose: hoperat, Action: auditevent.ActionRead, Outcome: auditevent.OutcomeSuccess,
		AgentName: "appointment_booked_email", EntityReference: "Account/" + accountID, OrganizationID: clinicID,
	}
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v", e, want)
	}
}

func TestAccountEmail_AuditsAnUnknownAccountAndPassesFailuresOn(t *testing.T) {
	f := newFixture()
	if _, _, err := f.svc.AccountEmail(t.Context(), "appointment_booked_email", accountID); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("AccountEmail() of an unknown account = %v, want ErrNotFound", err)
	}
	if e := f.onlyEvent(t); e.Outcome != auditevent.OutcomeMinorFailure || e.AgentName != "appointment_booked_email" || e.EntityReference != "" {
		t.Errorf("audit event %+v, want a minor failure by the job without an entity", e)
	}
	boom := errors.New("boom")
	for name, change := range map[string]func(*fixture){
		"repository": func(f *fixture) { f.repo.err = boom },
		"audit":      func(f *fixture) { f.audit.err = boom },
	} {
		f := newFixture(staff())
		change(&f)
		_, _, err := f.svc.AccountEmail(t.Context(), "appointment_booked_email", accountID)
		if !errors.Is(err, boom) || strings.Contains(err.Error(), "ben@example.test") {
			t.Errorf("%s fails: AccountEmail() error = %v, want it wrapped without the address", name, err)
		}
	}
}

// The e-mail's language is the account's chosen language, else the one its client last matched, else none
// (docs/i18n/SPEC.md §3 "The account's language fields"); the sender picks the fallback.
func TestAccountEmail_GivesTheLanguageTheAccountsEmailsAreWrittenIn(t *testing.T) {
	for _, tt := range []struct{ chosen, detected, want string }{
		{"hr", "de", "hr"},
		{"", "de", "de"},
		{"", "", ""},
	} {
		a := patientWithoutMFA()
		a.Language, a.DetectedLanguage = tt.chosen, tt.detected
		f := newFixture(a)
		_, got, err := f.svc.AccountEmail(t.Context(), "appointment_booked_email", accountID)
		if err != nil || got != tt.want {
			t.Errorf("chosen %q, detected %q: AccountEmail() language = %q, %v; want %q", tt.chosen, tt.detected, got, err, tt.want)
		}
	}
}

// The transcription agent reads only an account's language, to recognise the patient's speech in it (SAL-4):
// the chosen one, else the one its client last matched, else none. The read is audited with the agent as
// agent; no address leaves the domain.
func TestAccountLanguage_GivesTheLanguageAloneAndAuditsTheRead(t *testing.T) {
	for _, tt := range []struct{ chosen, detected, want string }{
		{"tr", "de", "tr"},
		{"", "ar", "ar"},
		{"", "", ""},
	} {
		a := patientWithoutMFA()
		a.Language, a.DetectedLanguage = tt.chosen, tt.detected
		f := newFixture(a)
		got, err := f.svc.AccountLanguage(t.Context(), "transcription_agent", accountID)
		if err != nil || got != tt.want {
			t.Errorf("chosen %q, detected %q: AccountLanguage() = %q, %v; want %q", tt.chosen, tt.detected, got, err, tt.want)
		}
		if e := f.onlyEvent(t); e.Action != auditevent.ActionRead || e.AgentName != "transcription_agent" || e.EntityReference != "Account/"+accountID || e.Outcome != auditevent.OutcomeSuccess {
			t.Errorf("audit event %+v, want the agent's read of the account", e)
		}
	}
}

func TestAccountLanguage_AuditsAnUnknownAccountAndPassesFailuresOn(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.AccountLanguage(t.Context(), "transcription_agent", accountID); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("AccountLanguage() of an unknown account = %v, want ErrNotFound", err)
	}
	if e := f.onlyEvent(t); e.Outcome != auditevent.OutcomeMinorFailure || e.EntityReference != "" {
		t.Errorf("audit event %+v, want a minor failure without an entity", e)
	}
	f = newFixture(staff())
	f.repo.err = errors.New("boom")
	if _, err := f.svc.AccountLanguage(t.Context(), "transcription_agent", accountID); err == nil {
		t.Error("AccountLanguage() with a failing repository = nil, want the error")
	}
}
