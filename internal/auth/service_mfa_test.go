package auth_test

import (
	"encoding/base32"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
)

var totpSecret = []byte("20 synthetic bytes!!") // 20 bytes

func updateEvent(a auth.Account, outcome auditevent.Outcome) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type: rest, Subtype: update, Purpose: hoperat, Action: auditevent.ActionUpdate, Outcome: outcome,
		AgentAccountID: a.ID, EntityReference: "Account/" + a.ID, OrganizationID: a.OrganizationID,
	}
}

// challenged logs a in and returns the challenge, with the login's audit event cleared.
func (f fixture) challenged(t *testing.T, a auth.Account) string {
	t.Helper()
	res, err := f.svc.Login(t.Context(), a.Email, password, clientIP, "", "")
	if err != nil || res.Challenge == "" {
		t.Fatalf("Login() = %+v, %v; want a challenge", res, err)
	}
	f.audit.events, f.audit.inTx = nil, nil
	return res.Challenge
}

// withSecret stores secret for the account, sealed, as Enrol does.
func (f fixture) withSecret(secret []byte) {
	f.repo.secrets[accountID], _ = f.box.Seal(secret, accountID)
}

func TestEnrol_StoresANewSealedSecretAndReturnsIt(t *testing.T) {
	f := newFixture(staff())
	challenge := f.challenged(t, staff())

	got, err := f.svc.Enrol(t.Context(), challenge, auth.Session{})

	if err != nil {
		t.Fatalf("Enrol() error = %v", err)
	}
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(got.Secret)
	if err != nil || len(secret) != 20 {
		t.Fatalf("secret %q is not 20 bytes of unpadded base32: %v", got.Secret, err)
	}
	if opened, err := f.box.Open(f.repo.secrets[accountID], accountID); err != nil || string(opened) != string(secret) {
		t.Error("the stored secret is not the returned one, sealed for the account")
	}
	u, err := url.Parse(got.URI)
	if err != nil || u.Scheme != "otpauth" || u.Host != "totp" || u.Path != "/Salus:ben@example.test" {
		t.Fatalf("URI %q, want otpauth://totp/Salus:<e-mail>", got.URI)
	}
	q := u.Query()
	if q.Get("secret") != got.Secret || q.Get("issuer") != "Salus" || q.Get("algorithm") != "SHA1" || q.Get("digits") != "6" || q.Get("period") != "30" {
		t.Errorf("URI parameters %v", q)
	}
	if staffAccount := f.repo.accounts[accountID]; staffAccount.MFAEnrolled() {
		t.Error("enrolment is marked before a code confirmed it")
	}
	if e := f.onlyEvent(t); e != updateEvent(staff(), auditevent.OutcomeSuccess) {
		t.Errorf("audit event %+v", e)
	}
	if _, ok := f.challenges.challenges[string(hashOf(t, challenge))]; !ok {
		t.Error("enrolment used up the challenge; confirm needs it")
	}
}

func TestEnrol_WorksFromAPatientsSession(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	res, _ := f.svc.Login(t.Context(), "ana@example.test", password, clientIP, "", "")

	if _, err := f.svc.Enrol(t.Context(), "", res.Session); err != nil {
		t.Fatalf("Enrol() error = %v", err)
	}
	if f.repo.secrets[accountID] == nil {
		t.Error("no secret stored")
	}
}

func TestEnrol_RefusesAnEnrolledAccountAndAuditsIt(t *testing.T) {
	f := newFixture(enrolled(staff()))
	challenge := f.challenged(t, enrolled(staff()))
	f.withSecret(totpSecret)

	if _, err := f.svc.Enrol(t.Context(), challenge, auth.Session{}); !errors.Is(err, auth.ErrAlreadyEnrolled) {
		t.Fatalf("Enrol() error = %v, want ErrAlreadyEnrolled", err)
	}
	if e := f.onlyEvent(t); e != updateEvent(enrolled(staff()), auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v", e)
	}
	if opened, _ := f.box.Open(f.repo.secrets[accountID], accountID); string(opened) != string(totpSecret) {
		t.Error("the enrolled secret was replaced")
	}
}

func TestMFA_RejectsAnUnknownChallengeOrNoSessionWithoutAuditing(t *testing.T) {
	f := newFixture(staff())
	unknown := strings.Repeat("A", 43)
	calls := map[string]func() error{
		"enrol, unknown challenge":   func() error { _, err := f.svc.Enrol(t.Context(), unknown, auth.Session{}); return err },
		"enrol, malformed challenge": func() error { _, err := f.svc.Enrol(t.Context(), "x", auth.Session{}); return err },
		"confirm":                    func() error { _, err := f.svc.Confirm(t.Context(), unknown, auth.Session{}, "000000"); return err },
		"verify":                     func() error { _, err := f.svc.Verify(t.Context(), unknown, "000000"); return err },
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, auth.ErrInvalidChallenge) {
			t.Errorf("%s: error = %v, want ErrInvalidChallenge", name, err)
		}
	}
	if _, err := f.svc.Enrol(t.Context(), "", auth.Session{}); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Errorf("enrol with neither challenge nor session: error = %v, want ErrUnauthenticated", err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("%d audit events, want none: no account was accessed", len(f.audit.events))
	}
}

func TestConfirm_EnrolsAndFromAChallengeOpensAStaffSession(t *testing.T) {
	f := newFixture(staff())
	challenge := f.challenged(t, staff())
	f.withSecret(totpSecret)

	got, err := f.svc.Confirm(t.Context(), challenge, auth.Session{}, auth.TOTPCode(totpSecret, now))

	if err != nil || got.Next != auth.NextDone || got.Token == "" {
		t.Fatalf("Confirm() = %+v, %v; want an open session", got, err)
	}
	s := f.repo.session(t)
	want := auth.Session{ID: s.ID, AccountID: accountID, OrganizationID: clinicID, MFAVerified: true, ExpiresAt: now.Add(12 * time.Hour)}
	if s.Session != want {
		t.Errorf("session %+v,\nwant %+v", s.Session, want)
	}
	if a := f.repo.accounts[accountID]; !a.MFAEnrolledAt.Equal(now) || f.repo.lastStep[accountID] != now.Unix()/30 {
		t.Errorf("enrolled at %v with last step %d, want now and the current step", a.MFAEnrolledAt, f.repo.lastStep[accountID])
	}
	if len(f.challenges.challenges) != 0 {
		t.Error("the challenge is still there after it opened a session")
	}
	if e := f.onlyEvent(t); e != updateEvent(staff(), auditevent.OutcomeSuccess) {
		t.Errorf("audit event %+v", e)
	}
}

func TestConfirm_FromASessionEnrolsWithoutANewSession(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	res, _ := f.svc.Login(t.Context(), "ana@example.test", password, clientIP, "", "")
	f.withSecret(totpSecret)

	got, err := f.svc.Confirm(t.Context(), "", res.Session, auth.TOTPCode(totpSecret, now))

	if err != nil || got != (auth.LoginResult{}) || len(f.repo.sessions) != 1 {
		t.Fatalf("Confirm() = %+v, %v with %d sessions; want no new session", got, err, len(f.repo.sessions))
	}
	if !f.repo.accounts[accountID].MFAEnrolled() {
		t.Error("not enrolled")
	}
}

func TestConfirm_RejectsAWrongCodeCountsItAndAuditsIt(t *testing.T) {
	f := newFixture(staff())
	challenge := f.challenged(t, staff())
	f.withSecret(totpSecret)

	_, err := f.svc.Confirm(t.Context(), challenge, auth.Session{}, auth.TOTPCode(totpSecret, now.Add(time.Hour)))

	if !errors.Is(err, auth.ErrWrongCode) {
		t.Fatalf("Confirm() error = %v, want ErrWrongCode", err)
	}
	if ch := f.challenges.challenges[string(hashOf(t, challenge))]; ch == nil || ch.fails != 1 {
		t.Errorf("challenge %+v, want one failure counted", ch)
	}
	if f.repo.accounts[accountID].MFAEnrolled() || len(f.repo.sessions) != 0 {
		t.Error("a wrong code enrolled the account or opened a session")
	}
	if e := f.onlyEvent(t); e != updateEvent(staff(), auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v", e)
	}
}

func TestConfirm_RefusesBeforeEnrolAndForAnEnrolledAccount(t *testing.T) {
	f := newFixture(staff())
	challenge := f.challenged(t, staff())
	if _, err := f.svc.Confirm(t.Context(), challenge, auth.Session{}, "000000"); !errors.Is(err, auth.ErrNotEnrolled) {
		t.Errorf("Confirm() before Enrol: error = %v, want ErrNotEnrolled", err)
	}
	if e := f.onlyEvent(t); e.Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("audit event %+v, want a failure", e)
	}

	g := newFixture(enrolled(staff()))
	challenge = g.challenged(t, enrolled(staff()))
	g.withSecret(totpSecret)
	if _, err := g.svc.Confirm(t.Context(), challenge, auth.Session{}, auth.TOTPCode(totpSecret, now)); !errors.Is(err, auth.ErrAlreadyEnrolled) {
		t.Errorf("Confirm() when enrolled: error = %v, want ErrAlreadyEnrolled", err)
	}
}

func TestVerify_OpensASessionForTheRightCode(t *testing.T) {
	f := newFixture(enrolled(staff()))
	challenge := f.challenged(t, enrolled(staff()))
	f.withSecret(totpSecret)

	got, err := f.svc.Verify(t.Context(), challenge, auth.TOTPCode(totpSecret, now))

	if err != nil || got.Next != auth.NextDone || got.Token == "" {
		t.Fatalf("Verify() = %+v, %v; want an open session", got, err)
	}
	if s := f.repo.session(t); !s.MFAVerified || s.OrganizationID != clinicID {
		t.Errorf("session %+v, want an MFA-verified session in the clinic", s.Session)
	}
	if len(f.challenges.challenges) != 0 {
		t.Error("the challenge is still there after it opened a session")
	}
	if e := f.onlyEvent(t); e != loginEvent(enrolled(staff()), auditevent.OutcomeSuccess) {
		t.Errorf("audit event %+v", e)
	}
}

func TestVerify_AcceptsOneStepOfDriftEitherWay(t *testing.T) {
	for drift, wantOK := range map[time.Duration]bool{-30 * time.Second: true, 30 * time.Second: true, -60 * time.Second: false, 60 * time.Second: false} {
		f := newFixture(enrolled(staff()))
		challenge := f.challenged(t, enrolled(staff()))
		f.withSecret(totpSecret)
		_, err := f.svc.Verify(t.Context(), challenge, auth.TOTPCode(totpSecret, now.Add(drift)))
		if (err == nil) != wantOK {
			t.Errorf("a code %v off: error = %v, want accepted %v", drift, err, wantOK)
		}
	}
}

func TestVerify_RejectsACodeWhoseStepWasUsed(t *testing.T) {
	f := newFixture(enrolled(staff()))
	f.withSecret(totpSecret)
	f.repo.lastStep[accountID] = now.Unix() / 30 // used by the last sign-in
	challenge := f.challenged(t, enrolled(staff()))

	_, err := f.svc.Verify(t.Context(), challenge, auth.TOTPCode(totpSecret, now))

	if !errors.Is(err, auth.ErrWrongCode) {
		t.Fatalf("Verify() with a used code: error = %v, want ErrWrongCode", err)
	}
	if len(f.repo.sessions) != 0 {
		t.Error("a used code opened a session")
	}
	if e := f.onlyEvent(t); e != loginEvent(enrolled(staff()), auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v", e)
	}
}

func TestVerify_FiveWrongCodesEndTheChallenge(t *testing.T) {
	f := newFixture(enrolled(staff()))
	challenge := f.challenged(t, enrolled(staff()))
	f.withSecret(totpSecret)
	wrong := auth.TOTPCode(totpSecret, now.Add(time.Hour))
	for i := range 5 {
		if _, err := f.svc.Verify(t.Context(), challenge, wrong); !errors.Is(err, auth.ErrWrongCode) {
			t.Fatalf("wrong code %d: error = %v, want ErrWrongCode", i+1, err)
		}
	}
	if _, err := f.svc.Verify(t.Context(), challenge, auth.TOTPCode(totpSecret, now)); !errors.Is(err, auth.ErrInvalidChallenge) {
		t.Errorf("the right code after five wrong ones: error = %v, want ErrInvalidChallenge", err)
	}
}

func TestVerify_RefusesAnAccountThatIsNotEnrolled(t *testing.T) {
	f := newFixture(staff())
	challenge := f.challenged(t, staff()) // an enrol challenge

	if _, err := f.svc.Verify(t.Context(), challenge, "000000"); !errors.Is(err, auth.ErrNotEnrolled) {
		t.Errorf("Verify() error = %v, want ErrNotEnrolled", err)
	}
	if e := f.onlyEvent(t); e != loginEvent(staff(), auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v", e)
	}
}

func TestMFA_PassesFailuresOn(t *testing.T) {
	boom := errors.New("boom")
	type call func(f fixture, challenge string) error
	enrol := func(f fixture, ch string) error { _, err := f.svc.Enrol(t.Context(), ch, auth.Session{}); return err }
	confirm := func(f fixture, ch string) error {
		_, err := f.svc.Confirm(t.Context(), ch, auth.Session{}, auth.TOTPCode(totpSecret, now))
		return err
	}
	verify := func(f fixture, ch string) error {
		_, err := f.svc.Verify(t.Context(), ch, auth.TOTPCode(totpSecret, now))
		return err
	}
	tests := []struct {
		change  func(f *fixture)
		call    call
		name    string
		account auth.Account
	}{
		{name: "enrol, read", account: staff(), call: enrol, change: func(f *fixture) { f.repo.err = boom }},
		{name: "enrol, seal", account: staff(), call: enrol, change: func(f *fixture) { f.box.err = boom }},
		{name: "enrol, audit", account: staff(), call: enrol, change: func(f *fixture) { f.audit.err = boom }},
		{name: "enrol, challenge", account: staff(), call: enrol, change: func(f *fixture) { f.challenges.getErr = boom }},
		{name: "confirm, read", account: staff(), call: confirm, change: func(f *fixture) { f.repo.err = boom }},
		{name: "confirm, open", account: staff(), call: confirm, change: func(f *fixture) { f.box.err = boom }},
		{name: "confirm, session", account: staff(), call: confirm, change: func(f *fixture) { f.repo.sessionErr = boom }},
		{name: "confirm, audit", account: staff(), call: confirm, change: func(f *fixture) { f.audit.err = boom }},
		{name: "verify, read", account: enrolled(staff()), call: verify, change: func(f *fixture) { f.repo.err = boom }},
		{name: "verify, open", account: enrolled(staff()), call: verify, change: func(f *fixture) { f.box.err = boom }},
		{name: "verify, session", account: enrolled(staff()), call: verify, change: func(f *fixture) { f.repo.sessionErr = boom }},
		{name: "verify, audit", account: enrolled(staff()), call: verify, change: func(f *fixture) { f.audit.err = boom }},
		{name: "verify, challenge", account: enrolled(staff()), call: verify, change: func(f *fixture) { f.challenges.err = boom }},
	}
	for _, tt := range tests {
		f := newFixture(tt.account)
		challenge := f.challenged(t, tt.account)
		f.withSecret(totpSecret)
		tt.change(&f)
		if err := tt.call(f, challenge); !errors.Is(err, boom) {
			t.Errorf("%s fails: error = %v, want it wrapped", tt.name, err)
		}
	}
}

func TestMFA_WrongCodeCountingFailureIsPassedOn(t *testing.T) {
	f := newFixture(enrolled(staff()))
	challenge := f.challenged(t, enrolled(staff()))
	f.withSecret(totpSecret)
	boom := errors.New("boom")
	f.challenges.err = boom
	if _, err := f.svc.Verify(t.Context(), challenge, "000000"); !errors.Is(err, boom) {
		t.Errorf("Verify() error = %v, want the challenge store's failure", err)
	}
}
