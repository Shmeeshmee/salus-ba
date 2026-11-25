package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
)

var deleteCoding = auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeDelete}

// storedOpenInvite stores one open invite of the admin's clinic and returns its id, with the audit trail cleared.
func storedOpenInvite(t *testing.T, f fixture) string {
	t.Helper()
	inv, err := f.svc.Invite(t.Context(), adminSession(), practitionerInvite())
	if err != nil {
		t.Fatal(err)
	}
	f.audit.events, f.audit.inTx, f.jobs.jobs, f.jobs.inTx = nil, nil, nil, nil
	return inv.ID
}

func TestWithdrawInvite_EndsTheOpenInviteAndAuditsTheDelete(t *testing.T) {
	f := newFixture(staff())
	id := storedOpenInvite(t, f)

	if err := f.svc.WithdrawInvite(t.Context(), adminSession(), id); err != nil {
		t.Fatalf("WithdrawInvite() error = %v", err)
	}

	if inv := f.repo.invites[id]; !inv.ExpiresAt.Equal(now) {
		t.Errorf("expires at %v, want the invite ended now (%v)", inv.ExpiresAt, now)
	}
	if e := f.onlyEvent(t); e != inviteEvent(deleteCoding, auditevent.ActionDelete, accountID, "Invite/"+id, auditevent.OutcomeSuccess) {
		t.Errorf("audit event %+v", e)
	}
	if len(f.audit.inTx) != 1 || !f.audit.inTx[0] {
		t.Errorf("audit in transaction = %v, want it in the withdrawal's", f.audit.inTx)
	}
}

func TestWithdrawInvite_RefusesAndAuditsAPractitionerAndAnInviteThatIsNotOpen(t *testing.T) {
	practitioner := staff()
	practitioner.Role = auth.RolePractitioner
	tests := []struct {
		want    error
		name    string
		id      string
		account auth.Account
	}{
		{name: "a practitioner", account: practitioner, want: auth.ErrForbidden},
		{name: "an unknown invite", account: staff(), id: "1a000000-0000-4000-8000-000000000999", want: auth.ErrInvalidInvite},
	}
	for _, tt := range tests {
		f := newFixture(tt.account)
		f.repo.invites["1a000000-0000-4000-8000-000000000001"] = &auth.Invite{
			ID: "1a000000-0000-4000-8000-000000000001", OrganizationID: clinicID, Email: "cy@example.test", ExpiresAt: now.Add(72 * time.Hour),
		}
		id := tt.id
		if id == "" {
			id = "1a000000-0000-4000-8000-000000000001"
		}

		err := f.svc.WithdrawInvite(t.Context(), adminSession(), id)

		if !errors.Is(err, tt.want) {
			t.Errorf("%s: WithdrawInvite() = %v, want %v", tt.name, err, tt.want)
		}
		if e := f.onlyEvent(t); e != inviteEvent(deleteCoding, auditevent.ActionDelete, accountID, "", auditevent.OutcomeMinorFailure) {
			t.Errorf("%s: audit event %+v", tt.name, e)
		}
		if inv := f.repo.invites["1a000000-0000-4000-8000-000000000001"]; !inv.ExpiresAt.After(now) {
			t.Errorf("%s: the open invite was ended", tt.name)
		}
	}
}
