package auth_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
)

func withLanguages(a auth.Account, language, detected string) auth.Account {
	a.Language, a.DetectedLanguage = language, detected
	return a
}

func TestRegister_SavesTheChosenAndTheDetectedLanguage(t *testing.T) {
	for _, tt := range []struct{ language, detected string }{
		{"", ""}, {"", "de"}, {"ar", "de"}, {"sr-Cyrl", ""}, {"fr", "it"},
	} {
		f := newFixture()

		got, err := f.svc.Register(t.Context(), "ana@example.test", password, tt.language, tt.detected)

		stored := f.repo.accounts[accountID]
		if err != nil || stored.Language != tt.language || stored.DetectedLanguage != tt.detected || got.Language != tt.language {
			t.Errorf("Register(%q, %q) = %+v, %v; stored %q, %q", tt.language, tt.detected, got, err, stored.Language, stored.DetectedLanguage)
		}
	}
}

func TestLanguageWrites_RefuseATagOutsideTheListBeforeTouchingAnything(t *testing.T) {
	for _, tag := range []string{"sr", "en-GB", "BS", "xx", "de-DE", " bs"} {
		for name, call := range map[string]func(f fixture) error{
			"Register chosen": func(f fixture) error {
				_, err := f.svc.Register(t.Context(), "cleo@example.test", password, tag, "")
				return err
			},
			"Register detected": func(f fixture) error {
				_, err := f.svc.Register(t.Context(), "cleo@example.test", password, "", tag)
				return err
			},
			"Login chosen": func(f fixture) error {
				_, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP, tag, "")
				return err
			},
			"Login detected": func(f fixture) error {
				_, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP, "", tag)
				return err
			},
			"AcceptInvite": func(f fixture) error {
				_, err := f.svc.AcceptInvite(t.Context(), "token", password, tag, "")
				return err
			},
			"SetLanguage": func(f fixture) error {
				_, err := f.svc.SetLanguage(t.Context(), accountID, tag)
				return err
			},
		} {
			f := newFixture(patientWithoutMFA())
			if err := call(f); !errors.Is(err, auth.ErrInvalidLanguage) {
				t.Errorf("%s with %q: error = %v, want ErrInvalidLanguage", name, tag, err)
			}
			if f.hasher.hashes != 0 || len(f.audit.events) != 0 || len(f.repo.accounts) != 1 || f.repo.accounts[accountID] != patientWithoutMFA() {
				t.Errorf("%s with %q: hashed, audited or wrote before refusing", name, tag)
			}
		}
	}
}

func TestLogin_SavesTheLanguagesAndAuditsTheChangeWithTheLogin(t *testing.T) {
	f := newFixture(withLanguages(patientWithoutMFA(), "en", "en"))

	if _, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP, "tr", "de"); err != nil {
		t.Fatal(err)
	}

	if a := f.repo.accounts[accountID]; a.Language != "tr" || a.DetectedLanguage != "de" {
		t.Errorf("stored languages %q, %q; want tr, de", a.Language, a.DetectedLanguage)
	}
	e := f.audit.events
	if len(e) != 2 || e[0] != loginEvent(patientWithoutMFA(), auditevent.OutcomeSuccess) || e[1] != updateEvent(patientWithoutMFA(), auditevent.OutcomeSuccess) || !f.audit.inTx[1] {
		t.Errorf("audit events %+v; want the login, then the update, in the transaction", e)
	}
	if len(f.repo.outside) != 1 {
		t.Errorf("repository calls outside the transaction: %v, want the password pre-read only", f.repo.outside)
	}
}

func TestLogin_SavesTheLanguagesBeforeTheMFAStep(t *testing.T) {
	f := newFixture(enrolled(staff()))

	res, err := f.svc.Login(t.Context(), "ben@example.test", password, clientIP, "hr", "sr-Cyrl")

	if err != nil || res.Next != auth.NextVerify {
		t.Fatalf("Login() = %+v, %v; want the verify step", res, err)
	}
	if a := f.repo.accounts[accountID]; a.Language != "hr" || a.DetectedLanguage != "sr-Cyrl" {
		t.Errorf("stored languages %q, %q; want hr, sr-Cyrl", a.Language, a.DetectedLanguage)
	}
}

func TestLogin_WritesAndAuditsNothingWhenTheLanguagesAreUnchangedOrUnsent(t *testing.T) {
	for _, tt := range []struct{ language, detected string }{{"", ""}, {"en", ""}, {"", "de"}, {"en", "de"}} {
		f := newFixture(withLanguages(patientWithoutMFA(), "en", "de"))

		if _, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP, tt.language, tt.detected); err != nil {
			t.Fatal(err)
		}

		if a := f.repo.accounts[accountID]; a.Language != "en" || a.DetectedLanguage != "de" || len(f.audit.events) != 1 {
			t.Errorf("Login(%q, %q): stored %q, %q with %d audit events; want unchanged and the login only", tt.language, tt.detected, a.Language, a.DetectedLanguage, len(f.audit.events))
		}
	}
}

func TestLogin_DoesNotSaveAPatientLanguageOnAStaffAccount(t *testing.T) {
	f := newFixture(enrolled(staff()))

	if _, err := f.svc.Login(t.Context(), "ben@example.test", password, clientIP, "de", "ar"); err != nil {
		t.Fatalf("Login() error = %v; a language the web app does not offer must not stop a sign-in", err)
	}

	if a := f.repo.accounts[accountID]; a.Language != "" || a.DetectedLanguage != "" || len(f.audit.events) != 1 {
		t.Errorf("stored %q, %q with %d audit events; want nothing saved", a.Language, a.DetectedLanguage, len(f.audit.events))
	}
}

func TestLogin_SavesNoLanguageForAWrongPassword(t *testing.T) {
	f := newFixture(patientWithoutMFA())

	if _, err := f.svc.Login(t.Context(), "ana@example.test", "the wrong password", clientIP, "de", "de"); !errors.Is(err, auth.ErrWrongCredentials) {
		t.Fatalf("Login() error = %v, want ErrWrongCredentials", err)
	}

	if a := f.repo.accounts[accountID]; a.Language != "" || a.DetectedLanguage != "" {
		t.Errorf("stored %q, %q after a wrong password", a.Language, a.DetectedLanguage)
	}
}

func TestLogin_PassesALanguageWriteFailureOn(t *testing.T) {
	boom := errors.New("boom")
	f := newFixture(patientWithoutMFA())
	f.repo.languageErr = boom

	if _, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP, "de", ""); !errors.Is(err, boom) {
		t.Errorf("Login() error = %v, want it wrapped", err)
	}
}

func TestAcceptInvite_SavesAStaffAccountsLanguages(t *testing.T) {
	f := newFixture()
	_, token := invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RoleAdmin})

	got, err := f.svc.AcceptInvite(t.Context(), token, password, "sr-Latn", "en")

	if a := f.repo.accounts[accountID]; err != nil || a.Language != "sr-Latn" || a.DetectedLanguage != "en" || got.Account.Language != "sr-Latn" {
		t.Errorf("AcceptInvite() = %+v, %v; stored %q, %q; want sr-Latn, en", got.Account, err, a.Language, a.DetectedLanguage)
	}
}

func TestAcceptInvite_DoesNotSaveAPatientLanguage(t *testing.T) {
	f := newFixture()
	_, token := invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RoleAdmin})

	if _, err := f.svc.AcceptInvite(t.Context(), token, password, "it", "tr"); err != nil {
		t.Fatalf("AcceptInvite() error = %v; a language the web app does not offer must not stop the acceptance", err)
	}

	if a := f.repo.accounts[accountID]; a.Language != "" || a.DetectedLanguage != "" {
		t.Errorf("stored %q, %q; want nothing", a.Language, a.DetectedLanguage)
	}
}

func TestSetLanguage_SetsOrClearsTheChoiceAndAuditsIt(t *testing.T) {
	tests := []struct {
		account  auth.Account
		language string
	}{
		{patientWithoutMFA(), "ar"},
		{withLanguages(patientWithoutMFA(), "ar", "de"), ""},
		{staff(), "sr-Cyrl"},
		{withLanguages(staff(), "en", ""), ""},
	}
	for _, tt := range tests {
		f := newFixture(tt.account)

		got, err := f.svc.SetLanguage(t.Context(), accountID, tt.language)

		if err != nil || got.Language != tt.language || got.PasswordHash != "" || got.Email != tt.account.Email {
			t.Fatalf("SetLanguage(%q) = %+v, %v; want the account with the language and without its hash", tt.language, got, err)
		}
		if a := f.repo.accounts[accountID]; a.Language != tt.language || a.DetectedLanguage != tt.account.DetectedLanguage {
			t.Errorf("stored %q, %q; want %q and the detected language kept", a.Language, a.DetectedLanguage, tt.language)
		}
		if e := f.onlyEvent(t); e != updateEvent(tt.account, auditevent.OutcomeSuccess) {
			t.Errorf("audit event %+v", e)
		}
		if len(f.repo.outside) != 0 {
			t.Errorf("repository calls outside the transaction: %v", f.repo.outside)
		}
	}
}

func TestSetLanguage_RefusesAPatientLanguageForStaffAndAuditsTheRefusal(t *testing.T) {
	for _, tag := range []string{"de", "ar", "tr", "it", "fr"} {
		f := newFixture(withLanguages(staff(), "hr", ""))

		_, err := f.svc.SetLanguage(t.Context(), accountID, tag)

		if !errors.Is(err, auth.ErrLanguageUnavailable) {
			t.Errorf("SetLanguage(%q) for staff = %v, want ErrLanguageUnavailable", tag, err)
		}
		if f.repo.accounts[accountID].Language != "hr" {
			t.Errorf("SetLanguage(%q) for staff changed the language", tag)
		}
		if e := f.onlyEvent(t); e != updateEvent(staff(), auditevent.OutcomeMinorFailure) {
			t.Errorf("audit event %+v", e)
		}
	}
}

func TestSetLanguage_AuditsAnUnknownAccount(t *testing.T) {
	f := newFixture()

	_, err := f.svc.SetLanguage(t.Context(), accountID, "de")

	if !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("SetLanguage() error = %v, want ErrNotFound", err)
	}
	if e := f.onlyEvent(t); e != updateEvent(auth.Account{ID: accountID}, auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v", e)
	}
}

func TestSetLanguage_PassesFailuresOnWithoutTheirDetails(t *testing.T) {
	boom := errors.New("boom")
	for name, change := range map[string]func(*fixture){
		"read":  func(f *fixture) { f.repo.err = boom },
		"write": func(f *fixture) { f.repo.languageErr = boom },
		"audit": func(f *fixture) { f.audit.err = boom },
	} {
		f := newFixture(patientWithoutMFA())
		change(&f)
		_, err := f.svc.SetLanguage(t.Context(), accountID, "de")
		if !errors.Is(err, boom) || strings.Contains(err.Error(), "de") {
			t.Errorf("%s fails: SetLanguage() error = %v, want it wrapped without the language", name, err)
		}
	}
}

func TestSetLanguage_TakesEveryTagOfTheEnumForAPatientAndTheFiveStaffLanguagesForStaff(t *testing.T) {
	for _, tag := range []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"} {
		if _, err := newFixture(patientWithoutMFA()).svc.SetLanguage(t.Context(), accountID, tag); err != nil {
			t.Errorf("SetLanguage(%q) for a patient = %v", tag, err)
		}
	}
	for _, tag := range []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en"} {
		if _, err := newFixture(staff()).svc.SetLanguage(t.Context(), accountID, tag); err != nil {
			t.Errorf("SetLanguage(%q) for staff = %v", tag, err)
		}
	}
}
