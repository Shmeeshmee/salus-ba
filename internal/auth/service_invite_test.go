package auth_test

import (
	"encoding/base32"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

const inviteLifetime = 72 * time.Hour

var searchType = auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeSearchType}

func base32NoPad(b []byte) string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

func adminSession() auth.Session {
	return auth.Session{ID: sessionID, AccountID: accountID, OrganizationID: clinicID, MFAVerified: true}
}

func practitionerInvite() auth.Invite {
	return auth.Invite{Email: " Cy@Example.TEST ", Role: auth.RolePractitioner, GivenName: " Cy ", FamilyName: " Example "}
}

func inviteEvent(subtype auditevent.Coding, action auditevent.Action, agent, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type: rest, Subtype: subtype, Purpose: hoperat, Action: action, Outcome: outcome,
		AgentAccountID: agent, EntityReference: entity, OrganizationID: clinicID,
	}
}

func TestInvite_StoresAnOpenInviteOfTheAdminsClinicAndEnqueuesItsEmail(t *testing.T) {
	f := newFixture(staff())

	got, err := f.svc.Invite(t.Context(), adminSession(), practitionerInvite())

	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	want := auth.Invite{
		ID: got.ID, OrganizationID: clinicID, InvitedBy: accountID, Email: "cy@example.test", Role: auth.RolePractitioner,
		GivenName: "Cy", FamilyName: "Example", CreatedAt: now, ExpiresAt: now.Add(inviteLifetime),
	}
	if got.ID == "" || got != want || *f.repo.invites[got.ID] != want {
		t.Errorf("Invite() = %+v, stored %+v;\nwant %+v", got, f.repo.invites[got.ID], want)
	}
	if f.repo.clinic != clinicID {
		t.Errorf("clinic set %q, want the admin's", f.repo.clinic)
	}
	if len(f.jobs.jobs) != 1 || f.jobs.jobs[0] != (auth.InviteEmailArgs{InviteID: got.ID, OrganizationID: clinicID}) || !f.jobs.inTx[0] {
		t.Errorf("jobs %+v (in the transaction: %v), want one invite_email with the two ids, enqueued in the transaction", f.jobs.jobs, f.jobs.inTx)
	}
	if e := f.onlyEvent(t); e != inviteEvent(create, auditevent.ActionCreate, accountID, "Invite/"+got.ID, auditevent.OutcomeSuccess) {
		t.Errorf("audit event %+v", e)
	}
}

func TestInvite_ReplacesAnOpenInviteForTheSameEmail(t *testing.T) {
	f := newFixture(staff())
	first, err := f.svc.Invite(t.Context(), adminSession(), practitionerInvite())
	if err != nil {
		t.Fatal(err)
	}
	again := auth.Invite{Email: "cy@example.test", Role: auth.RoleAdmin}

	got, err := f.svc.Invite(t.Context(), adminSession(), again)

	if err != nil || got.ID != first.ID || len(f.repo.invites) != 1 || f.repo.invites[first.ID].Role != auth.RoleAdmin {
		t.Errorf("Invite() again = %+v, %v; %d invites stored; want the first one replaced", got, err, len(f.repo.invites))
	}
}

func TestInvite_RejectsAnInvalidInviteBeforeAnyAccess(t *testing.T) {
	long := strings.Repeat("x", 201)
	tests := []struct {
		want error
		in   auth.Invite
	}{
		{auth.ErrInvalidEmail, auth.Invite{Email: "not an e-mail", Role: auth.RoleAdmin}},
		{auth.ErrInvalidRole, auth.Invite{Email: "cy@example.test", Role: auth.RolePatient}},
		{auth.ErrInvalidRole, auth.Invite{Email: "cy@example.test"}},
		{auth.ErrInvalidName, auth.Invite{Email: "cy@example.test", Role: auth.RolePractitioner, GivenName: "Cy"}},
		{auth.ErrInvalidName, auth.Invite{Email: "cy@example.test", Role: auth.RolePractitioner, GivenName: " ", FamilyName: "Example"}},
		{auth.ErrInvalidName, auth.Invite{Email: "cy@example.test", Role: auth.RolePractitioner, GivenName: long, FamilyName: "Example"}},
		{auth.ErrInvalidName, auth.Invite{Email: "cy@example.test", Role: auth.RoleAdmin, FamilyName: "Example"}},
	}
	for _, tt := range tests {
		f := newFixture(staff())
		if _, err := f.svc.Invite(t.Context(), adminSession(), tt.in); !errors.Is(err, tt.want) {
			t.Errorf("Invite(%+v) = %v, want %v", tt.in, err, tt.want)
		}
		if len(f.repo.invites) != 0 || len(f.audit.events) != 0 || len(f.jobs.jobs) != 0 {
			t.Errorf("Invite(%+v) stored, audited or enqueued something", tt.in)
		}
	}
}

func TestInvite_RefusesAndAuditsAnyoneButAnAdminOfTheClinic(t *testing.T) {
	practitionerAccount := staff()
	practitionerAccount.Role = auth.RolePractitioner
	patientAccount := patientWithoutMFA()
	tests := []struct {
		name    string
		account auth.Account
		sess    auth.Session
	}{
		{name: "a practitioner", account: practitionerAccount, sess: adminSession()},
		{name: "a patient", account: patientAccount, sess: auth.Session{ID: sessionID, AccountID: accountID}},
		{name: "a session of another clinic", account: staff(), sess: auth.Session{ID: sessionID, AccountID: accountID, OrganizationID: "d0000000-0000-4000-8000-00000000000d"}},
		{name: "an account that is gone", account: auth.Account{ID: "0b6c8a52-0000-4000-8000-000000000000"}, sess: adminSession()},
	}
	for _, tt := range tests {
		f := newFixture(tt.account)
		_, err := f.svc.Invite(t.Context(), tt.sess, practitionerInvite())
		if !errors.Is(err, auth.ErrForbidden) || len(f.repo.invites) != 0 || len(f.jobs.jobs) != 0 {
			t.Errorf("%s: Invite() = %v with %d invites, %d jobs; want ErrForbidden and nothing stored", tt.name, err, len(f.repo.invites), len(f.jobs.jobs))
		}
		want := inviteEvent(create, auditevent.ActionCreate, accountID, "", auditevent.OutcomeMinorFailure)
		want.OrganizationID = tt.sess.OrganizationID
		if e := f.onlyEvent(t); e != want {
			t.Errorf("%s: audit event %+v,\nwant %+v", tt.name, e, want)
		}
	}
}

func TestInvite_FailsWhenTheJobCannotBeEnqueued(t *testing.T) {
	f := newFixture(staff())
	f.jobs.err = errors.New("river down")
	if _, err := f.svc.Invite(t.Context(), adminSession(), practitionerInvite()); err == nil {
		t.Error("Invite() with a failing enqueue: error = nil, want it, so the transaction rolls back")
	}
}

func TestInviteAdmin_InvitesAClinicsFirstAdminAsTheOperatorCommand(t *testing.T) {
	f := newFixture()

	got, err := f.svc.InviteAdmin(t.Context(), clinicID, "Dee@example.test")

	want := auth.Invite{ID: got.ID, OrganizationID: clinicID, Email: "dee@example.test", Role: auth.RoleAdmin, CreatedAt: now, ExpiresAt: now.Add(inviteLifetime)}
	if err != nil || got != want || f.repo.clinic != clinicID {
		t.Fatalf("InviteAdmin() = %+v, %v;\nwant %+v", got, err, want)
	}
	if len(f.jobs.jobs) != 1 || f.jobs.jobs[0].InviteID != got.ID || !f.jobs.inTx[0] {
		t.Errorf("jobs %+v, want the invite's e-mail enqueued in the transaction", f.jobs.jobs)
	}
	e := inviteEvent(create, auditevent.ActionCreate, "", "Invite/"+got.ID, auditevent.OutcomeSuccess)
	e.AgentName = auditevent.AgentOperator
	if got := f.onlyEvent(t); got != e {
		t.Errorf("audit event %+v,\nwant %+v", got, e)
	}
	if _, err := f.svc.InviteAdmin(t.Context(), clinicID, "not an e-mail"); !errors.Is(err, auth.ErrInvalidEmail) {
		t.Errorf("InviteAdmin(invalid e-mail) = %v, want ErrInvalidEmail", err)
	}
}

// openInvites stores, in the admin's clinic, an open invite, an expired one, an accepted one, and one of
// another clinic.
func openInvites(f fixture) {
	for id, inv := range map[string]auth.Invite{
		"1a000000-0000-4000-8000-000000000001": {Email: "cy@example.test", Role: auth.RoleAdmin, ExpiresAt: now.Add(time.Hour)},
		"1a000000-0000-4000-8000-000000000002": {Email: "dee@example.test", Role: auth.RoleAdmin, ExpiresAt: now},
		"1a000000-0000-4000-8000-000000000003": {Email: "eve@example.test", Role: auth.RoleAdmin, ExpiresAt: now.Add(time.Hour), AcceptedAt: now},
		"1a000000-0000-4000-8000-000000000004": {Email: "fay@example.test", Role: auth.RoleAdmin, ExpiresAt: now.Add(time.Hour), OrganizationID: "d0000000-0000-4000-8000-00000000000d"},
	} {
		inv.ID = id
		if inv.OrganizationID == "" {
			inv.OrganizationID = clinicID
		}
		f.repo.invites[id] = &inv
	}
}

func TestListInvites_GivesTheAdminTheOpenInvitesOfTheirClinicAuditingEach(t *testing.T) {
	f := newFixture(staff())
	openInvites(f)

	got, err := f.svc.ListInvites(t.Context(), adminSession())

	if err != nil || len(got) != 1 || got[0].ID != "1a000000-0000-4000-8000-000000000001" {
		t.Fatalf("ListInvites() = %+v, %v; want the one open invite", got, err)
	}
	want := inviteEvent(searchType, auditevent.ActionExecute, accountID, "Invite/"+got[0].ID, auditevent.OutcomeSuccess)
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v", e, want)
	}
}

func TestListInvites_AuditsAnEmptyListOnceWithoutAnEntity(t *testing.T) {
	f := newFixture(staff())

	got, err := f.svc.ListInvites(t.Context(), adminSession())

	if err != nil || len(got) != 0 {
		t.Fatalf("ListInvites() = %+v, %v; want none", got, err)
	}
	if e := f.onlyEvent(t); e != inviteEvent(searchType, auditevent.ActionExecute, accountID, "", auditevent.OutcomeSuccess) {
		t.Errorf("audit event %+v", e)
	}
}

func TestListInvites_RefusesAndAuditsAPractitioner(t *testing.T) {
	a := staff()
	a.Role = auth.RolePractitioner
	f := newFixture(a)
	openInvites(f)

	if _, err := f.svc.ListInvites(t.Context(), adminSession()); !errors.Is(err, auth.ErrForbidden) {
		t.Errorf("ListInvites() by a practitioner = %v, want ErrForbidden", err)
	}
	if e := f.onlyEvent(t); e != inviteEvent(searchType, auditevent.ActionExecute, accountID, "", auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v", e)
	}
}

// invited stores an open invite and issues its token, as the invite_email job does.
func invited(t *testing.T, f fixture, inv auth.Invite) (auth.Invite, string) {
	t.Helper()
	inv.OrganizationID, inv.ExpiresAt = clinicID, now.Add(time.Hour)
	id, _ := f.repo.PutInvite(t.Context(), inv)
	inv.ID = id
	email, token, err := f.svc.IssueInviteToken(t.Context(), id, clinicID)
	if err != nil || email != inv.Email || token == "" {
		t.Fatalf("IssueInviteToken() = %q, %q, %v; want the invite's e-mail and a token", email, token, err)
	}
	f.repo.outside, f.audit.events, f.audit.inTx = nil, nil, nil
	return inv, token
}

func TestIssueInviteToken_StoresOnlyTheHashAndReplacesAnEarlierToken(t *testing.T) {
	f := newFixture()
	inv, first := invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RoleAdmin})
	if string(f.repo.hashes[inv.ID]) != string(hashOf(t, first)) || f.repo.clinic != clinicID {
		t.Fatalf("stored hash, clinic %q: want the token's SHA-256, in the invite's clinic", f.repo.clinic)
	}

	_, second, err := f.svc.IssueInviteToken(t.Context(), inv.ID, clinicID)

	if err != nil || second == first || string(f.repo.hashes[inv.ID]) != string(hashOf(t, second)) {
		t.Fatalf("a second IssueInviteToken() = %v; want a new token whose hash replaced the first", err)
	}
	// A retried job issues a new token, so the link in an earlier e-mail stops working (PROGRESS.md).
	if _, err := f.svc.AcceptInvite(t.Context(), first, password); !errors.Is(err, auth.ErrInvalidInvite) {
		t.Errorf("AcceptInvite() with the first token = %v, want ErrInvalidInvite", err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("issuing a token audited %d events, want none (SPEC.md lists creation and acceptance)", len(f.audit.events))
	}
}

func TestIssueInviteToken_RefusesAnInviteThatIsNotOpen(t *testing.T) {
	f := newFixture()
	openInvites(f)
	for _, id := range []string{"1a000000-0000-4000-8000-000000000002", "1a000000-0000-4000-8000-000000000003", "1a000000-0000-4000-8000-000000000004", "unknown"} {
		if _, _, err := f.svc.IssueInviteToken(t.Context(), id, clinicID); !errors.Is(err, auth.ErrInvalidInvite) || f.repo.hashes[id] != nil {
			t.Errorf("IssueInviteToken(%s) = %v, want ErrInvalidInvite and no token stored", id, err)
		}
	}
}

func TestAcceptInvite_CreatesThePractitionersAccountDirectoryEntriesAndPendingTOTPInOneTransaction(t *testing.T) {
	f := newFixture()
	inv, token := invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RolePractitioner, GivenName: "Cy", FamilyName: "Example"})

	got, err := f.svc.AcceptInvite(t.Context(), token, password)

	if err != nil {
		t.Fatalf("AcceptInvite() error = %v", err)
	}
	a := f.repo.accounts[accountID]
	want := auth.Account{ID: accountID, Email: "cy@example.test", Role: auth.RolePractitioner, OrganizationID: clinicID, PasswordHash: "hashed:" + strings.ToUpper(password)}
	if a != want || got.Account.ID != accountID || got.Account.PasswordHash != "" {
		t.Errorf("stored account %+v, returned %+v;\nwant %+v, returned without its hash", a, got.Account, want)
	}
	if f.repo.invites[inv.ID].AcceptedAt != now {
		t.Error("the invite is not marked accepted")
	}
	caller := practitioner.Caller{AccountID: accountID, OrganizationID: clinicID}
	if len(f.practitioners.practitioners) != 1 || f.practitioners.practitioners[0] != caller || f.practitioners.names[0] != [2]string{"Cy", "Example"} ||
		len(f.practitioners.roles) != 1 || f.practitioners.roles[0] != createdPractitioner ||
		f.practitioners.roleCallers[0] != (practitionerrole.Caller{AccountID: accountID, OrganizationID: clinicID}) ||
		f.repo.links[accountID] != createdPractitioner || f.practitioners.outside != 0 {
		t.Errorf("directory: %+v, links %v; want Practitioner and PractitionerRole created as the new account, linked, in the transaction", f.practitioners, f.repo.links)
	}
	secret, err := f.box.Open(f.repo.secrets[accountID], accountID)
	if err != nil || base32NoPad(secret) != got.Enrolment.Secret || !strings.HasPrefix(got.Enrolment.URI, "otpauth://totp/Salus:cy@example.test?") {
		t.Errorf("enrolment %+v, stored secret %v: want the pending secret, sealed for the account", got.Enrolment, err)
	}
	if ch, ok := f.challenges.challenges[string(hashOf(t, got.Challenge))]; !ok || ch.accountID != accountID {
		t.Error("no challenge for the new account: confirming the first code needs one")
	}
	if outside := f.repo.outside; len(outside) != 0 || f.hasher.inTx != 0 {
		t.Errorf("repository calls outside the transaction %v, hashes inside %d; want every write inside, the hash outside", outside, f.hasher.inTx)
	}
	if e := f.onlyEvent(t); e != inviteEvent(create, auditevent.ActionCreate, accountID, "Account/"+accountID, auditevent.OutcomeSuccess) {
		t.Errorf("audit event %+v", e)
	}

	// The first code confirms enrolment and opens the session, as at a first login.
	res, err := f.svc.Confirm(t.Context(), got.Challenge, auth.Session{}, auth.TOTPCode(secret, now))
	if err != nil || res.Next != auth.NextDone || res.Token == "" {
		t.Errorf("Confirm() after acceptance = %+v, %v; want the session", res, err)
	}
}

func TestAcceptInvite_CreatesNoDirectoryEntryForAnAdmin(t *testing.T) {
	f := newFixture()
	_, token := invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RoleAdmin})

	if _, err := f.svc.AcceptInvite(t.Context(), token, password); err != nil {
		t.Fatalf("AcceptInvite() error = %v", err)
	}
	if f.repo.accounts[accountID].Role != auth.RoleAdmin || len(f.practitioners.practitioners) != 0 || len(f.practitioners.roles) != 0 || len(f.repo.links) != 0 {
		t.Errorf("account %+v, directory %+v: want an admin and no Practitioner", f.repo.accounts[accountID], f.practitioners)
	}
}

func TestAcceptInvite_RejectsAnUnknownTokenOrABadPasswordWithoutAuditing(t *testing.T) {
	f := newFixture()
	_, token := invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RoleAdmin})
	for _, tt := range []struct {
		want            error
		token, password string
	}{
		{auth.ErrInvalidInvite, "not-a-token", password},
		{auth.ErrInvalidInvite, challenge, password}, // well-formed, unknown
		{auth.ErrInvalidPassword, token, "too short"},
	} {
		if _, err := f.svc.AcceptInvite(t.Context(), tt.token, tt.password); !errors.Is(err, tt.want) {
			t.Errorf("AcceptInvite(%q) = %v, want %v", tt.token, err, tt.want)
		}
	}
	if len(f.repo.accounts) != 0 || len(f.audit.events) != 0 {
		t.Errorf("%d accounts, %d audit events; want none: nothing was accessed", len(f.repo.accounts), len(f.audit.events))
	}
}

func TestAcceptInvite_RefusesAndAuditsAnExpiredOrAcceptedInvite(t *testing.T) {
	for name, change := range map[string]func(*auth.Invite){
		"expired":  func(inv *auth.Invite) { inv.ExpiresAt = now },
		"accepted": func(inv *auth.Invite) { inv.AcceptedAt = now.Add(-time.Minute) },
	} {
		f := newFixture()
		inv, token := invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RoleAdmin})
		change(f.repo.invites[inv.ID])

		_, err := f.svc.AcceptInvite(t.Context(), token, password)

		if !errors.Is(err, auth.ErrInvalidInvite) || len(f.repo.accounts) != 0 {
			t.Errorf("%s: AcceptInvite() = %v with %d accounts, want ErrInvalidInvite and none", name, err, len(f.repo.accounts))
		}
		if e := f.onlyEvent(t); e != inviteEvent(create, auditevent.ActionCreate, "", "Invite/"+inv.ID, auditevent.OutcomeMinorFailure) {
			t.Errorf("%s: audit event %+v", name, e)
		}
	}
}

func TestAcceptInvite_RefusesAndAuditsATakenEmailLeavingTheInviteOpen(t *testing.T) {
	existing := patientWithoutMFA()
	existing.ID, existing.Email = "0b6c8a52-0000-4000-8000-0000000000ff", "cy@example.test"
	f := newFixture(existing)
	inv, token := invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RolePractitioner, GivenName: "Cy", FamilyName: "Example"})

	_, err := f.svc.AcceptInvite(t.Context(), token, password)

	if !errors.Is(err, auth.ErrEmailTaken) || !f.repo.invites[inv.ID].AcceptedAt.IsZero() || len(f.practitioners.practitioners) != 0 {
		t.Errorf("AcceptInvite() = %v; want ErrEmailTaken, the invite still open and no Practitioner", err)
	}
	if e := f.onlyEvent(t); e != inviteEvent(create, auditevent.ActionCreate, "", "Invite/"+inv.ID, auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v", e)
	}
}

func TestAcceptInvite_FailsWhenTheDirectoryOrTheChallengeFails(t *testing.T) {
	f := newFixture()
	_, token := invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RolePractitioner, GivenName: "Cy", FamilyName: "Example"})
	f.practitioners.err = errors.New("directory down")
	if _, err := f.svc.AcceptInvite(t.Context(), token, password); err == nil {
		t.Error("AcceptInvite() with a failing directory: error = nil")
	}

	f = newFixture()
	_, token = invited(t, f, auth.Invite{Email: "cy@example.test", Role: auth.RoleAdmin})
	f.challenges.err = errors.New("redis down")
	if _, err := f.svc.AcceptInvite(t.Context(), token, password); err == nil {
		t.Error("AcceptInvite() with a failing challenge store: error = nil")
	}
}

func TestResetMFA_ClearsEnrolmentRevokesSessionsAndAuditsTheOperator(t *testing.T) {
	a := staff()
	a.MFAEnrolledAt = now.Add(-time.Hour)
	f := newFixture(a)
	f.withSecret(totpSecret)
	f.repo.lastStep[accountID] = 1
	open, _ := f.repo.CreateSession(t.Context(), auth.Session{AccountID: accountID, OrganizationID: clinicID, ExpiresAt: now.Add(time.Hour)}, []byte("hash"), now)

	if err := f.svc.ResetMFA(t.Context(), accountID); err != nil {
		t.Fatalf("ResetMFA() error = %v", err)
	}
	if f.repo.accounts[accountID].MFAEnrolled() || f.repo.secrets[accountID] != nil {
		t.Error("the account is still enrolled or keeps its secret")
	}
	if _, last := f.repo.lastStep[accountID]; last {
		t.Error("the last used TOTP step was kept")
	}
	if s := f.repo.session(t); s.ID != open || s.revokedAt != now {
		t.Errorf("session revoked at %v, want %v", s.revokedAt, now)
	}
	want := auditevent.AuditEvent{
		Type: rest, Subtype: update, Purpose: hoperat, Action: auditevent.ActionUpdate, Outcome: auditevent.OutcomeSuccess,
		AgentName: auditevent.AgentOperator, EntityReference: "Account/" + accountID, OrganizationID: clinicID,
	}
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v", e, want)
	}
	// The next login asks for enrolment again.
	if res, err := f.svc.Login(t.Context(), a.Email, password, clientIP); err != nil || res.Next != auth.NextEnrol {
		t.Errorf("Login() after the reset = %+v, %v; want next: enrol", res, err)
	}
}

func TestResetMFA_RefusesAPatientAndAnUnknownAccount(t *testing.T) {
	a := patientWithoutMFA()
	a.MFAEnrolledAt = now.Add(-time.Hour)
	f := newFixture(a)

	if err := f.svc.ResetMFA(t.Context(), accountID); !errors.Is(err, auth.ErrNotStaff) || !f.repo.accounts[accountID].MFAEnrolled() {
		t.Errorf("ResetMFA(patient) = %v; want ErrNotStaff and the account unchanged", err)
	}
	want := auditevent.AuditEvent{
		Type: rest, Subtype: update, Purpose: hoperat, Action: auditevent.ActionUpdate, Outcome: auditevent.OutcomeMinorFailure,
		AgentName: auditevent.AgentOperator, EntityReference: "Account/" + accountID,
	}
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v", e, want)
	}

	f = newFixture()
	if err := f.svc.ResetMFA(t.Context(), accountID); !errors.Is(err, auth.ErrNotFound) || len(f.audit.events) != 0 {
		t.Errorf("ResetMFA(unknown) = %v with %d events; want ErrNotFound, not audited: no account was accessed", err, len(f.audit.events))
	}
}

func TestResetMFA_FailsOnARepositoryError(t *testing.T) {
	f := newFixture(staff())
	f.repo.sessionErr = errors.New("connection reset")
	if err := f.svc.ResetMFA(t.Context(), accountID); err == nil || errors.Is(err, auth.ErrNotFound) {
		t.Errorf("ResetMFA() with a failing repository = %v, want its error", err)
	}
}

func TestIsClinicAdmin_IsTrueOnlyForAnAdminOfTheSessionsClinicAndAuditsNothing(t *testing.T) {
	practitionerAccount := staff()
	practitionerAccount.Role = auth.RolePractitioner
	tests := []struct {
		name    string
		account auth.Account
		sess    auth.Session
		want    bool
	}{
		{name: "the clinic's admin", account: staff(), sess: adminSession(), want: true},
		{name: "a practitioner", account: practitionerAccount, sess: adminSession()},
		{name: "a patient", account: patientWithoutMFA(), sess: auth.Session{ID: sessionID, AccountID: accountID}},
		{name: "a session of another clinic", account: staff(), sess: auth.Session{ID: sessionID, AccountID: accountID, OrganizationID: "d0000000-0000-4000-8000-00000000000d"}},
		{name: "an account that is gone", account: auth.Account{ID: "0b6c8a52-0000-4000-8000-000000000000"}, sess: adminSession()},
	}
	for _, tt := range tests {
		f := newFixture(tt.account)
		got, err := f.svc.IsClinicAdmin(t.Context(), tt.sess)
		if err != nil || got != tt.want {
			t.Errorf("%s: IsClinicAdmin() = %v, %v; want %v, nil", tt.name, got, err, tt.want)
		}
		if n := len(f.audit.events); n != 0 {
			t.Errorf("%s: %d audit events, want none", tt.name, n)
		}
	}
}

func TestIsClinicAdmin_PassesARepositoryErrorOn(t *testing.T) {
	f := newFixture(staff())
	f.repo.err = errors.New("db down")

	if _, err := f.svc.IsClinicAdmin(t.Context(), adminSession()); !errors.Is(err, f.repo.err) {
		t.Errorf("IsClinicAdmin() error = %v, want the repository's", err)
	}
}

func TestClinicMember_GivesTheAdminFlagAndThePractitionerOfTheSessionsClinicAndAuditsNothing(t *testing.T) {
	const practitionerID = "a3000000-0000-4000-8000-0000000000a1"
	practitionerAccount := staff()
	practitionerAccount.Role, practitionerAccount.PractitionerID = auth.RolePractitioner, practitionerID
	tests := []struct {
		name    string
		account auth.Account
		sess    auth.Session
		want    auth.Member
	}{
		{name: "the clinic's admin", account: staff(), sess: adminSession(), want: auth.Member{Admin: true}},
		{name: "a practitioner", account: practitionerAccount, sess: adminSession(), want: auth.Member{PractitionerID: practitionerID}},
		{name: "a patient", account: patientWithoutMFA(), sess: auth.Session{ID: sessionID, AccountID: accountID}},
		{name: "a session of another clinic", account: practitionerAccount, sess: auth.Session{ID: sessionID, AccountID: accountID, OrganizationID: "d0000000-0000-4000-8000-00000000000d"}},
		{name: "an account that is gone", account: auth.Account{ID: "0b6c8a52-0000-4000-8000-000000000000"}, sess: adminSession()},
	}
	for _, tt := range tests {
		f := newFixture(tt.account)
		got, err := f.svc.ClinicMember(t.Context(), tt.sess)
		if err != nil || got != tt.want {
			t.Errorf("%s: ClinicMember() = %+v, %v; want %+v, nil", tt.name, got, err, tt.want)
		}
		if n := len(f.audit.events); n != 0 {
			t.Errorf("%s: %d audit events, want none", tt.name, n)
		}
	}

	f := newFixture(staff())
	f.repo.err = errors.New("db down")
	if _, err := f.svc.ClinicMember(t.Context(), adminSession()); !errors.Is(err, f.repo.err) {
		t.Errorf("ClinicMember() error = %v, want the repository's", err)
	}
}
