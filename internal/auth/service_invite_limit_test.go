package auth_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
)

// fillInvites stores n open invites of the admin's clinic, with synthetic e-mails n0@example.test, n1@...
func fillInvites(t *testing.T, f fixture, n int) {
	t.Helper()
	for i := range n {
		if _, err := f.svc.Invite(t.Context(), adminSession(), auth.Invite{Email: fmt.Sprintf("n%d@example.test", i), Role: auth.RoleAdmin}); err != nil {
			t.Fatalf("invite %d: %v", i, err)
		}
	}
	f.audit.events, f.audit.inTx, f.jobs.jobs, f.jobs.inTx = nil, nil, nil, nil
}

func TestInvite_RefusesAndAuditsThe51stOpenInviteOfAClinic(t *testing.T) {
	f := newFixture(staff())
	fillInvites(t, f, 50)

	_, err := f.svc.Invite(t.Context(), adminSession(), practitionerInvite())

	if !errors.Is(err, auth.ErrInviteLimit) || len(f.repo.invites) != 50 || len(f.jobs.jobs) != 0 {
		t.Errorf("Invite() = %v with %d invites, %d jobs; want ErrInviteLimit and nothing stored", err, len(f.repo.invites), len(f.jobs.jobs))
	}
	if e := f.onlyEvent(t); e != inviteEvent(create, auditevent.ActionCreate, accountID, "", auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v", e)
	}
}

func TestInvite_AllowsThe50thInviteAndReplacingAnOpenOneAtTheLimit(t *testing.T) {
	f := newFixture(staff())
	fillInvites(t, f, 49)
	if _, err := f.svc.Invite(t.Context(), adminSession(), practitionerInvite()); err != nil {
		t.Fatalf("the 50th Invite() = %v, want it stored", err)
	}

	_, err := f.svc.Invite(t.Context(), adminSession(), auth.Invite{Email: "n3@example.test", Role: auth.RoleReception})

	if err != nil || len(f.repo.invites) != 50 {
		t.Errorf("replacing an open invite at the limit = %v with %d invites, want it replaced", err, len(f.repo.invites))
	}
}

func TestInvite_CountsOnlyOpenInvites(t *testing.T) {
	f := newFixture(staff())
	fillInvites(t, f, 50)
	for _, inv := range f.repo.invites {
		if inv.Email == "n0@example.test" {
			inv.AcceptedAt = now
		}
		if inv.Email == "n1@example.test" {
			inv.ExpiresAt = now
		}
	}

	_, err1 := f.svc.Invite(t.Context(), adminSession(), practitionerInvite())
	_, err2 := f.svc.Invite(t.Context(), adminSession(), auth.Invite{Email: "dee@example.test", Role: auth.RoleAdmin})
	_, err3 := f.svc.Invite(t.Context(), adminSession(), auth.Invite{Email: "eve@example.test", Role: auth.RoleAdmin})

	if err1 != nil || err2 != nil || !errors.Is(err3, auth.ErrInviteLimit) {
		t.Errorf("with an accepted and an expired invite: %v, %v, %v; want two stored, then ErrInviteLimit", err1, err2, err3)
	}
}

func TestInvite_CountsTheOpenInvitesUnderTheClinicsLock(t *testing.T) {
	f := newFixture(staff())

	if _, err := f.svc.Invite(t.Context(), adminSession(), practitionerInvite()); err != nil {
		t.Fatal(err)
	}

	if len(f.repo.locks) != 1 || f.repo.locks[0] != clinicID || len(f.repo.counted) != 1 || !f.repo.counted[0] {
		t.Errorf("locks %v, counted under the lock %v; want the clinic locked before its open invites are counted", f.repo.locks, f.repo.counted)
	}
}
